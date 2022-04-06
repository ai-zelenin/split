package main

import "go/ast"

type Decl struct {
	GD       *ast.GenDecl
	DF       *ast.FuncDecl
	Name     string
	VarType  string
	path     string
	receiver string
	Used     bool
	kind     string
}

func (d *Decl) Kind() string {
	return d.kind
}

func (d *Decl) Node() ast.Decl {
	switch {
	case d.DF != nil:
		return d.DF
	case d.GD != nil:
		return d.GD
	}
	panic("empty decl")
}
