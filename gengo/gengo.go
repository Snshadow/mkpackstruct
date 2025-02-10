// Package gengo generates go file with the given struct informations.
package gengo

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Snshadow/mkpackstruct/parsestruct"
)

const (
	packedFieldFormat = "%s: *(*%s)(unsafe.Pointer(&buf[%d])),\n"
)

// handle repeated struct in array
type repeatedStruct struct {
	typeStr string
	name    string
	count   int
}

func parseRepeatedStruct(s string) *repeatedStruct {
	count, name := 0, ""
	n, err := fmt.Sscanf(s, "[%d]%s", &count, &name)

	if err != nil || n != 2 {
		return nil
	}

	return &repeatedStruct{
		typeStr: s,
		name:    name,
		count:   count,
	}
}

func writePackedBytes(fldInfo []*parsestruct.FieldInfo, fldPrefix string, addIndent int, rpSt *repeatedStruct) string {
	var b strings.Builder

	indent := strings.Repeat("\t", 1+addIndent)
	fieldIndent := indent

	if rpSt != nil {
		indexVarName := rpSt.name + "Index"
		lastDot := strings.LastIndexByte(fldPrefix, '.')
		fldPrefix = fldPrefix[:lastDot] + "[" + indexVarName + "]" + fldPrefix[lastDot:]

		fmt.Fprintf(&b, indent+"for %[1]s := 0; %[1]s < %[2]d; %[1]s++ {\n", indexVarName, rpSt.count)
		fieldIndent = strings.Repeat("\t", 2+addIndent) // strings.Repeat returns optimized value with '\t'
	}

	for _, field := range fldInfo {
		if field.StructInfo != nil {
			b.WriteString(writePackedBytes(field.StructInfo.Fields, fldPrefix+field.Name+".", addIndent+1, parseRepeatedStruct(field.Type)))
		} else {
			fieldExpr := fldPrefix + field.Name

			fmt.Fprintf(&b, fieldIndent+"b.Write(unsafe.Slice((*byte)(unsafe.Pointer(&%s)), %d))\n", fieldExpr, field.Size)
		}
	}

	if rpSt != nil {
		b.WriteString(indent + "}\n")
	}

	return b.String()
}

func writeUnpackedFields(stInfo *parsestruct.StructInfo, baseOffset int64, addIndent int, nested string, rpSt *repeatedStruct) string {
	var b strings.Builder
	var startStruct, endStruct string

	indent := strings.Repeat("\t", 2+addIndent)

	// start struct declaration
	if nested == "" {
		startStruct = fmt.Sprintf(indent+"result = %s{\n", stInfo.StructName)
		endStruct = indent + "}\n"
	} else {
		fieldTypeStr := stInfo.StructName
		if rpSt != nil {
			fieldTypeStr = rpSt.typeStr
		}

		startStruct = fmt.Sprintf(indent+"%s: %s{\n", nested, fieldTypeStr)
		endStruct = indent + "},\n"
	}

	b.WriteString(startStruct)

	fieldIndent := strings.Repeat("\t", 3+addIndent)

	rpCount := 1 // for non-array field, it is always one

	var rpLeft, rpRight string // used for enclosing structs in array

	if rpSt != nil {
		rpCount = rpSt.count
		rpLeft = fieldIndent + "{\n"
		rpRight = fieldIndent + "},\n"
		fieldIndent = strings.Repeat("\t", 4+addIndent)
	}

	for i := 0; i < rpCount; i++ {
		if i != 0 {
			// increment for next offset
			baseOffset += stInfo.StructSize
		}

		if rpLeft != "" {
			b.WriteString(rpLeft)
		}

		for _, field := range stInfo.Fields {
			if field.StructInfo != nil {
				// nested struct
				b.WriteString(writeUnpackedFields(field.StructInfo, baseOffset+field.Offset, addIndent+1, field.Name, parseRepeatedStruct(field.Type)))
			} else {
				fmt.Fprintf(&b, fieldIndent+packedFieldFormat, field.Name, field.Type, baseOffset+field.Offset)
			}
		}

		if rpRight != "" {
			b.WriteString(rpRight)
		}
	}

	// end struct declaration
	b.WriteString(endStruct)

	return b.String()
}

type Generator struct {
	buf      bytes.Buffer
	packInfo parsestruct.GoPackInfo
}

func (g *Generator) Printf(format string, args ...any) {
	fmt.Fprintf(&g.buf, format, args...)
}

func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: formatting failed: %s", err)
		return g.buf.Bytes()
	}

	return src
}

// collectUsedImports analyzes struct fields to determine which imports are actually used
func collectUsedImports(info parsestruct.GoPackInfo) []string {
	imported := make(map[string]struct{}) // collect used imported packages

	// always needed for pointer operations
	imported["unsafe"] = struct{}{}
	// needed for buffer operations in ToPackedByte
	imported["bytes"] = struct{}{}
	// needed for error handling in ToStruct
	imported["fmt"] = struct{}{}

	// helper function to check if a type uses a package
	checkType := func(typeName string) {
		for _, pkg := range info.Imports {
			pkgPath := pkg.Path()
			pkgName := pkg.Name()

			// look for exact package usage patterns
			patterns := [...]string{
				pkgName + ".",             // import declaration
				"*" + pkgName + ".",       // pointer
				"[]" + pkgName + ".",      // slice 
				"[]*" + pkgName + ".",     // slice of pointers
				"map[" + pkgName + ".",    // map with package key type
				"]" + pkgName + ".",       // map with package value type
				"chan " + pkgName + ".",   // channel of package type
				"<-chan " + pkgName + ".", // receive-only channel
				"chan<- " + pkgName + ".", // send-only channel
			}

			// check for array pattern like [N]pkg.Type where N is a number
			if strings.Contains(typeName, "[") && strings.Contains(typeName, "]"+pkgName+".") {
				if idx := strings.Index(typeName, "["); idx >= 0 {
					if closeIdx := strings.Index(typeName[idx:], "]"); closeIdx >= 0 {
						// verify there's a number between [ and ]
						numStr := typeName[idx+1 : idx+closeIdx]
						if _, err := strconv.Atoi(numStr); err == nil {
							imported[pkgPath] = struct{}{}
							continue
						}
					}
				}
			}

			// check other patterns
			for _, pattern := range patterns {
				if strings.Contains(typeName, pattern) {
					imported[pkgPath] = struct{}{}
					break
				}
			}
		}
	}

	// analyze each struct's fields
	for _, st := range info.StructInfo {
		var analyzeFields func([]*parsestruct.FieldInfo)
		analyzeFields = func(fields []*parsestruct.FieldInfo) {
			for _, field := range fields {
				// check the field's type
				if field.Type != "" {
					checkType(field.Type)
				}

				// if it's a nested struct, analyze struct and its fields
				if field.StructInfo != nil {
					checkType(field.StructInfo.StructName)
					analyzeFields(field.StructInfo.Fields)
				}
			}
		}
		analyzeFields(st.Fields)
	}

	// convert map to sorted slice
	var result []string
	for imp := range imported {
		result = append(result, imp)
	}
	slices.Sort(result)

	return result
}

func writeImportHeader(info parsestruct.GoPackInfo) string {
	imported := collectUsedImports(info)

	var b strings.Builder
	b.WriteString("\nimport (")

	for _, p := range imported {
		fmt.Fprintf(&b, "\n\t\"%s\"", p)
	}

	b.WriteString("\n)\n\n")

	return b.String()
}

func (g *Generator) writeHeader() {
	g.Printf("// Code generated by \"mkgopack %s\"; DO NOT EDIT.\n", strings.Join(os.Args[1:], " "))
	g.Printf("\npackage %s\n", g.packInfo.PackageName)
	g.buf.WriteString(writeImportHeader(g.packInfo))
}

func (g *Generator) writeStructPackerFunctions() {
	for _, info := range g.packInfo.StructInfo {
		g.Printf("func (s *%s) ToPackedByte() []byte {\n", info.StructName)
		g.buf.WriteString("\tvar b bytes.Buffer\n\n")

		g.buf.WriteString(writePackedBytes(info.Fields, "s.", 0, nil))

		g.buf.WriteString("\n\treturn b.Bytes()\n}\n\n")
	}
}

// write an interface representing struct to be used with packing
func (g *Generator) writeStructInterface() {
	g.buf.WriteString("type PackedStruct interface {\n\t")

	for i, st := range g.packInfo.StructInfo {
		g.buf.WriteString(st.StructName)

		if i+1 != len(g.packInfo.StructInfo) {
			g.buf.WriteString(" | ")
		}
	}

	g.buf.WriteString("}\n")
}

// write a function to get packed size for comparision
func (g *Generator) writeGetPackedSize() {
	g.buf.WriteString("func GetPackedSize[P PackedStruct](st P) int {\tswitch any(st).(type) {\n")

	for _, info := range g.packInfo.StructInfo {
		g.Printf("\tcase %s:\n\t\treturn %d\n", info.StructName, info.StructSize)
	}

	g.buf.WriteString("\t}\n\n\treturn 0 // can't happen\n}\n\n")
}

func (g *Generator) writeGenericStructUnpacker() {
	g.buf.WriteString("func ToStruct[P PackedStruct](buf []byte) (P, error) {\n")
	g.buf.WriteString("\tvar st P // empty value used for type switch and returning error\n")
	g.buf.WriteString("\tvar result any // empty interface for holding generated struct before assertion\n")
	g.buf.WriteString("\n\tswitch sst := any(st).(type) { // convert to any for type switch\n")
	for _, info := range g.packInfo.StructInfo {
		if info == nil {
			fmt.Fprintln(os.Stderr, "struct info has nil")
			os.Exit(2)
		}

		g.Printf("\tcase %s:\n", info.StructName)

		g.buf.WriteString("\t\tif GetPackedSize(sst) != len(buf) {\n")
		g.buf.WriteString("\t\t\t return st, fmt.Errorf(\"the size of buffer does not match the size of struct\")\n\t\t\t}\n\n")

		g.buf.WriteString(writeUnpackedFields(info, 0, 0, "", nil))
	}

	g.Printf("\t}\n\n\treturn result.(P), nil\n}\n")
}

// GenPackStructGo generates go code into file using [parsestruct.GoPackInfo]
func GenPackStructGo(info parsestruct.GoPackInfo, output string) error {
	g := &Generator{
		packInfo: info,
	}

	g.writeHeader()
	g.writeStructPackerFunctions()
	g.writeStructInterface()
	g.writeGetPackedSize()
	g.writeGenericStructUnpacker()

	if err := os.WriteFile(output, g.format(), 0644); err != nil {
		return fmt.Errorf("writing output: %v", err)
	}

	return nil
}
