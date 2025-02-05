// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Snshadow/mkpackstruct/parsestruct"
	"github.com/Snshadow/mkpackstruct/gengo"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "mkpackstruct parses a go file and writes methods for packing structs into bytes buffer and vice versa.\nFor more information, see \"github.com/Snshadow/mkpackstruct\"\n\n")

	flag.Usage()
}

func main() {
	var filename, output string

	flag.StringVar(&filename, "filename", "", "file name to parse")
	flag.StringVar(&output, "output", "", "output file name; default srcdir/<go_filename>_gopack.go")

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
			output = strings.TrimSuffix(filename, ".go") + "_gopack.go"
		}
	}

	packInfo, err := parsestruct.GetPackInfo(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse source file: %v\n", err)
		os.Exit(2)
	}

	if err = gengo.GenPackStructGo(packInfo, output); err != nil {
		fmt.Fprintf(os.Stderr, "write generated go file: %v\n", err)
		os.Exit(2)
	}
}
