package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

var srcDir = flag.String("src", "pb", "source file")
var pkgName = flag.String("pkg", "pb", "dst package name")
var dstDir = flag.String("dst", "pb_Sep", "dst dir")

func main() {
	flag.Parse()
	arg := flag.Arg(0)
	if arg != "" {
		srcDir = &arg
	}
	if arg == "version" {
		fmt.Println("0.1.1")
		os.Exit(0)
	}
	sp := NewSegregatedPackage(*pkgName)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *srcDir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	ast.Walk(sp, f)
	err = sp.MakePackage(*dstDir)
	if err != nil {
		panic(err)
	}
}
