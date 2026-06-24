// Package parsestruct parses a file and returns name and information of every struct.
package parsestruct

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Snshadow/mkpackstruct/internal/sizes"
)

var stripRe *regexp.Regexp // regex for stripping parsed package name

type GoPackInfo struct {
	PackageName string
	Imports     []*types.Package
	StructInfo  []*StructInfo
}

type FieldInfo struct {
	Name       string
	Offset     int64
	Size       int64
	Type       string
	StructInfo *StructInfo // if not struct, nil
}

type StructInfo struct {
	StructName  string
	Fields      []*FieldInfo
	PackAlign   int64
	StructAlign int64
	StructSize  int64 // types.Sizes interface returns int64
}

// cleanStructString removes package name prefix from field type from nameless struct type string.
func cleanStructString(s string) string {
	parts := strings.Fields(s)
	for i, part := range parts {
		if stripRe.MatchString(part) {
			if idx := strings.LastIndex(part, "."); idx >= 0 {
				parts[i] = part[idx+1:]
			}
		}
	}
	return strings.Join(parts, " ")
}

// getTypeName returns type name without package name prefix
func getTypeName(t types.Type) string {
	str := types.TypeString(t, func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Name()
	})
	if !stripRe.MatchString(str) { // return imported type from external package as is
		return str
	}

	switch tt := t.(type) {
	case *types.Named:
		return tt.Obj().Name()
	case *types.Array:
		return "[" + strconv.FormatInt(tt.Len(), 10) + "]" + getTypeName(tt.Elem())
	case *types.Slice:
		return "[]" + getTypeName(tt.Elem())
	case *types.Pointer:
		return "*" + getTypeName(tt.Elem())
	case *types.Chan:
		switch tt.Dir() {
		case types.SendOnly:
			return "chan<- " + getTypeName(tt.Elem())
		case types.RecvOnly:
			return "<-chan " + getTypeName(tt.Elem())
		case types.SendRecv:
			return "chan " + getTypeName(tt.Elem())
		}
	case *types.Map:
		return "map[" + getTypeName(tt.Key()) + "]" + getTypeName(tt.Elem())
	}

	return str
}

type namedPackFunc func(*types.TypeName) int64

func packAlignForType(t types.Type, inherited int64, namedPack namedPackFunc) int64 {
	if named, ok := t.(*types.Named); ok {
		if _, ok := named.Underlying().(*types.Struct); ok {
			return namedPack(named.Obj())
		}
	}
	return inherited
}

// getStructInfo returns infomation of a struct and its fields.
func getStructInfo(st *types.Struct, sizeInfo *sizes.PackedSizes, name string, packAlign int64, namedPack namedPackFunc) StructInfo {
	var stInfo StructInfo

	numField := st.NumFields()
	fields := make([]*types.Var, 0, numField)
	fldInfos := make([]*FieldInfo, 0, numField)

	for i := 0; i < numField; i++ {
		fields = append(fields, st.Field(i))
	}

	stSizes := sizeInfo.WithMaxAlign(packAlign)
	offsets := stSizes.Offsetsof(fields)

	for i, field := range fields {
		fldInfo := FieldInfo{
			Name:   field.Name(),
			Offset: offsets[i],
		}

		t := field.Type()

		switch ut := t.Underlying().(type) {
		case *types.Struct:
			innerPackAlign := packAlignForType(t, packAlign, namedPack)
			innerSt := getStructInfo(ut, sizeInfo, getTypeName(t), innerPackAlign, namedPack)
			fldInfo.StructInfo = &innerSt
			fldInfo.Size = stSizes.Sizeof(t)
		case *types.Array:
			elemType := ut.Elem()
			if est, ok := elemType.Underlying().(*types.Struct); ok {
				innerPackAlign := packAlignForType(elemType, packAlign, namedPack)
				innerSt := getStructInfo(est, sizeInfo, getTypeName(elemType), innerPackAlign, namedPack)
				fldInfo.StructInfo = &innerSt
			}
			fldInfo.Size = stSizes.Sizeof(t)
			fldInfo.Type = getTypeName(t)
		default:
			fldInfo.Size = stSizes.Sizeof(t)
			fldInfo.Type = getTypeName(t) // use named type string for assignment
		}

		fldInfos = append(fldInfos, &fldInfo)
	}

	stInfo.StructName = cleanStructString(name)
	stInfo.Fields = fldInfos
	stInfo.PackAlign = packAlign
	stInfo.StructAlign = stSizes.Alignof(st)
	stInfo.StructSize = stSizes.Sizeof(st)

	return stInfo
}

func validatePackAlign(n int64) bool {
	switch n {
	case 1, 2, 4, 8, 16:
		return true
	default:
		return false
	}
}

func parsePackAlign(s string) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid pack alignment %q", s)
	}
	if !validatePackAlign(n) {
		return 0, fmt.Errorf("unsupported pack alignment %d", n)
	}
	return n, nil
}

func directiveLines(text string) []string {
	text = strings.TrimSpace(text)
	if comment, singleLine := strings.CutPrefix(text, "//"); singleLine {
		return []string{strings.TrimSpace(comment)}
	}

	if strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/") {
		text = strings.TrimPrefix(strings.TrimSuffix(text, "*/"), "/*")
		lines := strings.Split(text, "\n")
		for i := range lines {
			lines[i] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "*"))
		}
		return lines
	}
	return nil
}

func applyPackDirective(fset *token.FileSet, c *ast.Comment, current *int64, stack *[]int64) error {
	const prefix = "mkpackstruct:pack"

	for _, line := range directiveLines(c.Text) {
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if !strings.HasPrefix(rest, "(") || !strings.HasSuffix(rest, ")") {
			return fmt.Errorf("%s: malformed mkpackstruct pack directive", fset.Position(c.Pos()))
		}

		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(rest, "("), ")"))
		if body == "" {
			*current = sizes.DefaultPackAlign
			continue
		}

		parts := strings.Split(body, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		switch {
		case len(parts) == 1 && parts[0] == "push":
			*stack = append(*stack, *current)
		case len(parts) == 2 && parts[0] == "push":
			n, err := parsePackAlign(parts[1])
			if err != nil {
				return fmt.Errorf("%s: %w", fset.Position(c.Pos()), err)
			}
			*stack = append(*stack, *current)
			*current = n
		case len(parts) == 1 && parts[0] == "pop":
			if len(*stack) == 0 {
				return fmt.Errorf("%s: mkpackstruct pack pop with empty stack", fset.Position(c.Pos()))
			}
			last := len(*stack) - 1
			*current = (*stack)[last]
			*stack = (*stack)[:last]
		case len(parts) == 1:
			n, err := parsePackAlign(parts[0])
			if err != nil {
				return fmt.Errorf("%s: %w", fset.Position(c.Pos()), err)
			}
			*current = n
		default:
			return fmt.Errorf("%s: malformed mkpackstruct pack directive", fset.Position(c.Pos()))
		}
	}

	return nil
}

func collectStructPackAligns(fset *token.FileSet, files []*ast.File) (map[string]int64, error) {
	packByName := make(map[string]int64)

	for _, file := range files {
		current := sizes.DefaultPackAlign
		var stack []int64
		commentIdx := 0
		prevEnd := file.Package

		processCommentsBefore := func(limit token.Pos) error {
			for commentIdx < len(file.Comments) && file.Comments[commentIdx].Pos() < limit {
				cg := file.Comments[commentIdx]
				if cg.Pos() > prevEnd && cg.End() < limit {
					for _, c := range cg.List {
						if err := applyPackDirective(fset, c, &current, &stack); err != nil {
							return err
						}
					}
				}
				commentIdx++
			}
			return nil
		}

		for _, decl := range file.Decls {
			if err := processCommentsBefore(decl.Pos()); err != nil {
				return nil, err
			}

			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ts.Doc != nil {
						for _, c := range ts.Doc.List {
							if err := applyPackDirective(fset, c, &current, &stack); err != nil {
								return nil, err
							}
						}
					}
					if _, ok := ts.Type.(*ast.StructType); ok {
						packByName[ts.Name.Name] = current
					}
				}
			}

			prevEnd = decl.End()
		}

		if err := processCommentsBefore(token.Pos(int(^uint(0) >> 1))); err != nil {
			return nil, err
		}
	}

	return packByName, nil
}

func parsePackageFiles(fset *token.FileSet, filename string, packageName string, parsedFiles []string) ([]*ast.File, error) {
	dir := filepath.Dir(filename)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if len(parsedFiles) != 0 && !slices.Contains(parsedFiles, name) {
			continue
		}
		if strings.Contains(name, "_packstruct") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		if file.Name.Name == packageName {
			files = append(files, file)
		}
	}

	return files, nil
}

// GetPackInfo returns required information including package name and
// struct names and informations from a file by parsing files in
// searchFiles or all file in the same directory if not specified.
func GetPackInfo(filename string, wordSize int64, parsedFiles ...string) (GoPackInfo, error) {
	fset := token.NewFileSet()

	targetFile, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return GoPackInfo{}, err
	}

	// get base name of parsed files
	for i, name := range parsedFiles {
		parsedFiles[i] = filepath.Base(name)
	}

	// optimize single file parsing
	var singleFile bool

	if len(parsedFiles) == 1 && slices.ContainsFunc(parsedFiles, func(parsed string) bool {
		return filepath.Base(filename) == parsed
	}) {
		singleFile = true
	}

	// find the package containing the target file
	var files []*ast.File

	if !singleFile {
		// parse .go files in the same directory, including specified files and excluding generated files
		files, err = parsePackageFiles(fset, filename, targetFile.Name.Name, parsedFiles)
		if err != nil {
			return GoPackInfo{}, err
		}
	} else {
		// parse single target file
		files = append(files, targetFile)
	}

	if len(files) == 0 {
		return GoPackInfo{}, fmt.Errorf("no files found in package %s", targetFile.Name.Name)
	}

	structPackAligns, err := collectStructPackAligns(fset, files)
	if err != nil {
		return GoPackInfo{}, err
	}

	typeCheckSizes := sizes.NewPackedSizes(wordSize)

	conf := types.Config{
		Importer: importer.ForCompiler(fset, "source", nil),
		Sizes:    typeCheckSizes,
	}

	pkg, err := conf.Check(targetFile.Name.Name, fset, files, nil)
	if err != nil {
		return GoPackInfo{}, err
	}

	re, err := regexp.Compile(fmt.Sprintf(`[^\w]*\b%s\.`, targetFile.Name.Name))
	if err != nil {
		return GoPackInfo{}, err
	}
	stripRe = re

	namedPack := func(obj *types.TypeName) int64 {
		if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != pkg.Path() {
			return sizes.DefaultPackAlign
		}
		if a, ok := structPackAligns[obj.Name()]; ok {
			return a
		}
		return sizes.DefaultPackAlign
	}
	layoutSizes := sizes.NewPackedSizesWithNamedPack(wordSize, namedPack)

	structInfos := make([]*StructInfo, 0)

	// get position info for the target file to filter structs
	targetFilePos := fset.File(targetFile.Pos())
	if targetFilePos == nil {
		return GoPackInfo{}, fmt.Errorf("could not get position info for target file")
	}
	targetFilename := filepath.ToSlash(targetFilePos.Name())

	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

		if typeName, ok := obj.(*types.TypeName); ok {
			if st, ok := typeName.Type().Underlying().(*types.Struct); ok {
				// only include structs defined in the target file
				if obj.Pos() != token.NoPos {
					posInfo := fset.File(obj.Pos())
					if posInfo != nil {
						posName := filepath.ToSlash(posInfo.Name())
						if posName == targetFilename {
							stInfo := getStructInfo(st, layoutSizes, typeName.Name(), namedPack(typeName), namedPack)
							stInfo.StructName = name
							structInfos = append(structInfos, &stInfo)
						}
					}
				}
			}
		}
	}

	return GoPackInfo{
		PackageName: targetFile.Name.Name,
		Imports:     pkg.Imports(),
		StructInfo:  structInfos,
	}, nil
}
