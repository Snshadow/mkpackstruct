// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/Snshadow/mkpackstruct/gengo"
	"github.com/Snshadow/mkpackstruct/parsestruct"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "mkpackstruct parses a go file and writes methods for packing structs into bytes buffer and vice versa.\nFor more information, see \"github.com/Snshadow/mkpackstruct\"\n\n")

	flag.PrintDefaults()
}

func main() {
	var filename, parsedFiles, output string
	var wordSize int64

	flag.StringVar(&filename, "filename", "", "name of file used to create packed struct")
	flag.StringVar(&parsedFiles, "parsed", "", "files to be parsed within a package, separated with commas, if empty, parse all file in the same directory")
	flag.StringVar(&output, "output", "", "output file name; default srcdir/<go_filename>_gopack_${GOARCH}.go")
	flag.Int64Var(&wordSize, "word-size", 0, "word size to be used for parsing structs, default to word size of a running architecture")

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
			output = fmt.Sprintf("%s_packstruct_%s.go", strings.TrimSuffix(filename, ".go"), runtime.GOARCH)
		}
	}

	var resolvedParsed []string
	if parsedFiles != "" {
		resolvedParsed = strings.Split(parsedFiles, ",")
	}

	packInfo, err := parsestruct.GetPackInfo(filename, wordSize, resolvedParsed...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse source file: %v\n", err)
		os.Exit(2)
	}

	if err = gengo.GenPackStructGo(packInfo, output); err != nil {
		fmt.Fprintf(os.Stderr, "write generated go file: %v\n", err)
		os.Exit(2)
	}
}
