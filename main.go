package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
)

var srcDir = flag.String("src", "pb", "source dir package")
var pkgName = flag.String("pkg", "pb", "dst package name")
var dstDir = flag.String("dst", "pb_Sep", "source dir package")

func main() {
	flag.Parse()
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
