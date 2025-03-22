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
	StructName string
	Fields     []*FieldInfo
	StructSize int64 // types.Sizes interface returns int64
}

// cleanStructString removes package name prefix from field type from nameless struct type string.
func cleanStructString(s string) string {
	parts := strings.Split(s, " ")
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

// getStructInfo returns infomation of a struct and its fields.
func getStructInfo(st *types.Struct, sizes types.Sizes, name string) StructInfo {
	var stInfo StructInfo

	numField := st.NumFields()
	fields := make([]*types.Var, 0, numField)
	fldInfos := make([]*FieldInfo, 0, numField)

	for i := 0; i < numField; i++ {
		fields = append(fields, st.Field(i))
	}

	offsets := sizes.Offsetsof(fields)

	for i, field := range fields {
		fldInfo := FieldInfo{
			Name:   field.Name(),
			Offset: offsets[i],
		}

		t := field.Type()

		switch ut := t.Underlying().(type) {
		case *types.Struct:
			innerSt := getStructInfo(ut, sizes, getTypeName(t))
			fldInfo.StructInfo = &innerSt
			fldInfo.Size = sizes.Sizeof(ut)
		case *types.Array:
			elemType := ut.Elem()
			if est, ok := elemType.Underlying().(*types.Struct); ok {
				innerSt := getStructInfo(est, sizes, getTypeName(elemType))
				fldInfo.StructInfo = &innerSt
			}
			fldInfo.Size = sizes.Sizeof(ut)
			fldInfo.Type = getTypeName(t)
		default:
			fldInfo.Size = sizes.Sizeof(ut)
			fldInfo.Type = getTypeName(t) // use named type string for assignment
		}

		fldInfos = append(fldInfos, &fldInfo)
	}

	stInfo.StructName = cleanStructString(name)
	stInfo.Fields = fldInfos
	stInfo.StructSize = sizes.Sizeof(st)

	return stInfo
}

// GetPackInfo returns required information including package name and
// struct names and informations from a file by parsing files in
// searchFiles or all file in the same directory if not specified.
func GetPackInfo(filename string, wordSize int64, parsedFiles ...string) (GoPackInfo, error) {
	fset := token.NewFileSet()

	targetFile, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
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
		pkgs, err := parser.ParseDir(fset, filepath.Dir(filename), func(fi os.FileInfo) bool {
			if len(parsedFiles) != 0 {
				if !slices.Contains(parsedFiles, fi.Name()) {
					return false
				}
			}
			return !strings.Contains(fi.Name(), "_packstruct")
		}, parser.SkipObjectResolution)
		if err != nil {
			return GoPackInfo{}, err
		}

		for _, f := range pkgs[targetFile.Name.Name].Files {
			files = append(files, f)
		}
	} else {
		// parse single target file
		files = append(files, targetFile)
	}

	if len(files) == 0 {
		return GoPackInfo{}, fmt.Errorf("no files found in package %s", targetFile.Name.Name)
	}

	sizes := sizes.NewPackedSizes(wordSize)

	conf := types.Config{
		Importer: importer.ForCompiler(fset, "source", nil),
		Sizes:    sizes,
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
							stInfo := getStructInfo(st, sizes, typeName.Name())
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
