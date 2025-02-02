package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"strings"

	"github.com/Snshadow/mkpackstruct/parsestruct"
)

const (
	packedFieldFormat = "%s: *(*%s)(unsafe.Pointer(&buf[%d])),\n"
)

// handle repeated struct in array
type repeatedStruct struct {
	typeStr string
	count   int
}

func parseRepeatedStruct(s string) *repeatedStruct {
	count := 0
	n, err := fmt.Sscanf(s, "[%d]", &count)

	if err != nil || n != 1 {
		return nil
	}

	return &repeatedStruct{
		typeStr: s,
		count:   count,
	}
}

func writePackedBytes(fldInfo []*parsestruct.FieldInfo, fldPrefix string, addIndent int, rpSt *repeatedStruct) string {
	var b strings.Builder

	indent := strings.Repeat("\t", 1+addIndent)
	fieldIndent := indent

	if rpSt != nil {
		lastDot := strings.LastIndexByte(fldPrefix, '.')
		fldPrefix = fldPrefix[:lastDot] + "[i]" + fldPrefix[lastDot:]

		fmt.Fprintf(&b, indent+"for i := 0; i < %d; i++ {\n", rpSt.count)
		fieldIndent = strings.Repeat("\t", 2+addIndent) // strings.Repeat returns optimized value with '\t
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
		startStruct = fmt.Sprintf(indent+"st = %s{\n", stInfo.StructName)
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

	rpCount := 1         // for non-array field, it is always one
	rpOffset := int64(0) // for non-array field, it is always zero

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
			rpOffset += stInfo.StructSize
		}

		if rpLeft != "" {
			b.WriteString(rpLeft)
		}

		for _, field := range stInfo.Fields {
			if field.StructInfo != nil {
				// nested struct
				b.WriteString(writeUnpackedFields(field.StructInfo, field.Offset+rpOffset, addIndent+1, field.Name, parseRepeatedStruct(field.Type)))
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

func (g *Generator) writeHeader() {
	// g.Printf("// Code generated by \"mkgopack %s\"; DO NOT EDIT.\n", strings.Join(os.Args[1:], " "))
	g.Printf("// Code generated by \"mkgopack %s\"\n", strings.Join(os.Args[1:], " ")) // TEST
	g.Printf("\npackage %s\n", g.packInfo.PackageName)
	g.Printf("\nimport (\n\t\"bytes\"\n\t\"fmt\"\n\t\"unsafe\"\n)\n\n") // used for conversion between struct and buffer
}

func (g *Generator) writeStructPackerFunctions() {
	for _, info := range g.packInfo.StructInfo {
		g.Printf("func (s %s) ToPackedByte() []byte {\n", info.StructName)
		g.Printf("\tvar b bytes.Buffer\n\n")

		g.buf.WriteString(writePackedBytes(info.Fields, "s.", 0, nil))

		g.Printf("\n\treturn b.Bytes()\n}\n\n")
	}
}

func (g *Generator) writeStructInterface() {
	g.Printf("type PackedStruct interface {\n\tToPackedByte() []byte\n}\n")
}

func (g *Generator) writeGenericStructUnpacker() {
	g.Printf("func ToStruct[P PackedStruct](buf []byte) (P, error) {\n")
	g.Printf("\tvar st P\n\n\tswitch st := any(st).(type) { // convert to any for type switch\n")
	for _, info := range g.packInfo.StructInfo {
		if info == nil {
			fmt.Fprintln(os.Stderr, "struct info has nil")
			os.Exit(2)
		}

		g.Printf("\tcase %s:\n", info.StructName)

		g.Printf("\t\tif int(unsafe.Sizeof(st)) != len(buf) {\n")
		g.Printf("\t\t\t return st, fmt.Errorf(\"the size of buffer does not match the size of struct\")\n\t\t\t}\n\n")

		g.buf.WriteString(writeUnpackedFields(info, 0, 0, "", nil))
	}

	g.Printf("\t}\n\n\treturn st, nil\n}\n")
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "mkgopack parses a go file and writes methods for packing structs into bytes buffer and vice versa.\nFor more information, see \"github.com/Snshadow/mkgopack\"\n\n")

	flag.Usage()
}

func main() {
	var filename, output string

	flag.StringVar(&filename, "filename", "", "file name to parse")
	flag.StringVar(&output, "output", "", "output file name; default srcdir/<filename>_gopack.go")

	flag.Usage = usage

	flag.Parse()

	if filename == "" {
		if filename = flag.Arg(0); filename == "" {
			flag.Usage()
			os.Exit(1)
		}
	}

	if output == "" {
		if output = flag.Arg(1); output == "" {
			output = filename + "_gopack.go"
		}
	}

	info, err := parsestruct.GetPackInfo(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	g := &Generator{
		packInfo: info,
	}

	g.writeHeader()
	g.writeStructPackerFunctions()
	g.writeStructInterface()
	g.writeGenericStructUnpacker()

	if err = os.WriteFile(output, g.format(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "writing output: %v\n", err)
		os.Exit(2)
	}
}
