package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

var srcDir = flag.String("src", "pb", "source dir package")
var pkgName = flag.String("pkg", "pb", "dst package name")
var dstDir = flag.String("dst", "pb_Sep", "source dir package")

func main() {
	flag.Parse()
	arg := flag.Arg(0)
	if arg != "" {
		srcDir = &arg
	}
	if arg == "version" {
		fmt.Println("0.1.0")
		os.Exit(0)
	}
	sp := NewSegregatedPackage(*pkgName)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *srcDir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		ast.Walk(sp, pkg)
	}
	err = sp.MakePackage(*dstDir)
	if err != nil {
		panic(err)
	}
}
