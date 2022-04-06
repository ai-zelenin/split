package main

import "go/ast"

func getTypeName(vs *ast.ValueSpec) string {
	if vs.Type != nil {
		switch t := vs.Type.(type) {
		case *ast.StarExpr:
			return t.X.(*ast.Ident).Name
		case *ast.Ident:
			return t.Name
		}
	}
	if len(vs.Values) > 0 {
		switch t := vs.Values[0].(type) {
		case *ast.UnaryExpr:
			cl, ok := t.X.(*ast.CompositeLit)
			if ok {
				id, ok := cl.Type.(*ast.Ident)
				if ok {
					return id.Name
				}
			}
		case *ast.CompositeLit:
			id, ok := t.Type.(*ast.Ident)
			if ok {
				return id.Name
			}
		}
	}
	return ""
}
