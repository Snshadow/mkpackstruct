// Package parsestruct parses a file and returns name and information of every struct.
package parsestruct

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/Snshadow/mkpackstruct/internal/sizes"
)

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

// cleanStructString removes package name prefix from field type from nameless struct type string
func cleanStructString(s string) string {
	parts := strings.Split(s, " ")
	for i, part := range parts {
		if idx := strings.LastIndex(part, "."); idx >= 0 {
			parts[i] = part[idx+1:]
		}
	}
	return strings.Join(parts, " ")
}

// getTypeName returns type name without package name prefix
func getTypeName(t types.Type) string {
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

	return t.String()
}

// getStructInfo returns an error if a struct contains go specific type(slice,
// map, chan, interface, function signature) as a field or in an array
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
// struct names and informations from a file, if any struct contains go
// specific type(slice, map, chan, interface, function signature) which
// can't be directly converted to raw bytes, it will return an error
func GetPackInfo(filename string) (GoPackInfo, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		return GoPackInfo{}, err
	}

	sizes := &sizes.PackedSizes{}

	conf := types.Config{
		Importer: importer.Default(),
		Sizes:    sizes,
	}

	pkg, err := conf.Check(f.Name.Name, fset, []*ast.File{f}, nil)
	if err != nil {
		return GoPackInfo{}, err
	}

	structInfos := make([]*StructInfo, 0)

	scope := pkg.Scope()

	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

		if typeName, ok := obj.(*types.TypeName); ok {
			if st, ok := typeName.Type().Underlying().(*types.Struct); ok {
				stInfo := getStructInfo(st, sizes, typeName.Name())

				stInfo.StructName = name
				structInfos = append(structInfos, &stInfo)
			}
		}
	}

	return GoPackInfo{
		PackageName: f.Name.Name,
		Imports:     pkg.Imports(),
		StructInfo:  structInfos,
	}, nil
}
