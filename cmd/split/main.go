package main

import (
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"golang.org/x/tools/go/ast/astutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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

type SegregatedPackage struct {
	Fset    *token.FileSet
	Files   map[string]*ast.File
	Decls   []*Decl
	PkgName string
}

func NewSegregatedPackage(pkgName string) *SegregatedPackage {
	return &SegregatedPackage{
		Fset:    token.NewFileSet(),
		Files:   map[string]*ast.File{},
		PkgName: pkgName,
		Decls:   []*Decl{},
	}
}

func (s *SegregatedPackage) Visit(n ast.Node) (m ast.Visitor) {
	if n != nil {
		switch node := n.(type) {
		case *ast.GenDecl:
			s.addDeclFromGD(node)
			return nil
		case *ast.FuncDecl:
			s.addDeclFromFD(node)
			return nil
		}
	}
	return s
}

func (s *SegregatedPackage) addFile(name string) *ast.File {
	f := &ast.File{
		Name: &ast.Ident{
			Name: s.PkgName,
		},
		Scope: ast.NewScope(nil),
	}

	s.Files[name] = f
	s.Fset.AddFile(name, s.Fset.Base(), 0)
	return f
}

func (s *SegregatedPackage) addDeclFromGD(decl *ast.GenDecl) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(astutil.NodeDescription(decl))
			spew.Dump(decl)
			fmt.Println(r)
			os.Exit(1)
		}
	}()

	switch decl.Tok {
	case token.TYPE:
		typeSpec := decl.Specs[0].(*ast.TypeSpec)
		s.Decls = append(s.Decls, &Decl{
			GD:   decl,
			Name: typeSpec.Name.String(),
			kind: "type",
		})
		fmt.Printf("type %s\n", typeSpec.Name)
	case token.VAR:
		valueSpec := decl.Specs[0].(*ast.ValueSpec)
		ident := valueSpec.Names[0]
		d := &Decl{
			GD:      decl,
			Name:    ident.Name,
			VarType: getTypeName(valueSpec),
			kind:    "var",
		}
		s.Decls = append(s.Decls, d)
		fmt.Printf("var %s %s\n", d.Name, d.VarType)
	case token.CONST:
		valueSpec := decl.Specs[0].(*ast.ValueSpec)
		ident := valueSpec.Names[0]
		d := &Decl{
			GD:      decl,
			Name:    ident.Name,
			VarType: getTypeName(valueSpec),
			kind:    "const",
		}
		s.Decls = append(s.Decls, d)
		fmt.Printf("const %s %s\n", d.Name, d.VarType)
	case token.IMPORT:
		spew.Dump(decl)
		for _, spec := range decl.Specs {
			im, ok := spec.(*ast.ImportSpec)
			if ok {
				d := &Decl{
					GD:   decl,
					kind: "import",
				}
				if im.Name != nil {
					d.Name = im.Name.String()
				}
				d.path = im.Path.Value
				s.Decls = append(s.Decls, d)
				fmt.Printf("import %s %s\n", d.Name, d.path)
			}
		}
	}
}

func (s *SegregatedPackage) addDeclFromFD(decl *ast.FuncDecl) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(astutil.NodeDescription(decl))
			spew.Dump(decl)
			fmt.Println(r)
			os.Exit(1)
		}
	}()
	d := &Decl{
		DF:   decl,
		kind: "func",
	}
	if decl.Recv != nil {
		switch t := decl.Recv.List[0].Type.(type) {
		case *ast.Ident:
			d.receiver = t.Name
		case *ast.StarExpr:
			d.receiver = t.X.(*ast.Ident).Name
		}
	}
	d.Name = decl.Name.String()
	fmt.Printf("func (%s) %s \n", d.receiver, d.Name)
	s.Decls = append(s.Decls, d)
}

func (s *SegregatedPackage) MakePackage(dir string) error {
	inDir, err := filepath.Glob(fmt.Sprintf("%s/*", dir))
	if err != nil {
		return err
	}
	for _, f := range inDir {
		_ = os.Remove(f)
	}

	common := s.addFile("common")
	for _, decl := range s.Decls {
		if decl.Kind() == "type" {
			s.addFile(decl.Name)
		}
	}
	for fileName, file := range s.Files {
		for _, decl := range s.Decls {
			if decl.Kind() == "import" {
				fmt.Printf("add import %s %s to file %s\n", decl.Name, decl.path, fileName)
				if !astutil.AddNamedImport(s.Fset, file, decl.Name, strings.Trim(decl.path, "\"")) {
					fmt.Println("false")
				}
			}
		}
	}

	for fileName, file := range s.Files {
		if fileName != "common" {
			for _, decl := range s.Decls {
				t := decl.Kind()
				if decl.Used {
					continue
				}
				hasPrefixName := strings.HasPrefix(decl.Name, fileName)
				hasPrefixType := strings.HasPrefix(decl.VarType, fileName)
				hasPrefixXXXName := strings.HasPrefix(decl.Name, "xxx_messageInfo_"+fileName)
				hasPrefixNewName := strings.HasPrefix(decl.Name, "New"+fileName)
				hasPrefixRegister := strings.HasPrefix(decl.Name, "Register"+strings.TrimSuffix(fileName, "Server"))
				hasPrefixUnderscore := strings.HasPrefix(decl.Name, "_"+strings.TrimSuffix(fileName, "Server"))
				hasPrefixRequest := strings.HasPrefix(decl.Name, "request_"+strings.TrimSuffix(fileName, "Server"))
				hasPrefixLocalRequest := strings.HasPrefix(decl.Name, "local_request_"+strings.TrimSuffix(fileName, "Server"))
				hasPrefixFilter := strings.HasPrefix(decl.Name, "filter_"+strings.TrimSuffix(fileName, "Server"))
				hasPrefixReceiver := strings.HasPrefix(decl.receiver, fileName)

				switch {
				case t == "type" && hasPrefixName:
					file.Decls = append(file.Decls, decl.Node())
					decl.Used = true
				case (t == "var" || t == "const") && (hasPrefixName || hasPrefixType || hasPrefixXXXName || hasPrefixUnderscore || hasPrefixFilter):
					file.Decls = append(file.Decls, decl.Node())
					decl.Used = true
				case t == "func" && (hasPrefixReceiver || hasPrefixNewName || hasPrefixRegister || hasPrefixUnderscore || hasPrefixRequest || hasPrefixLocalRequest):
					file.Decls = append(file.Decls, decl.Node())
					decl.Used = true
				}
			}
		}
	}

	for _, decl := range s.Decls {
		if !decl.Used && decl.Kind() != "import" {
			common.Decls = append(common.Decls, decl.Node())
			decl.Used = true
		}
	}

	for fileName, file := range s.Files {
		fn := path.Join(dir, fmt.Sprintf("%s.go", fileName))
		fmt.Printf("write %s \n", fn)
		err := s.writeFile(fn, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SegregatedPackage) writeFile(fn string, file *ast.File) error {
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	return printer.Fprint(f, s.Fset, file)
}

func (s *SegregatedPackage) parrallel(cb func(fileName string, file *ast.File) error) {
	wg := sync.WaitGroup{}
	sem := make(chan struct{}, 10)
	for fileName, file := range s.Files {
		wg.Add(1)
		sem <- struct{}{}
		go func(fn string, f *ast.File) {
			defer func() {
				wg.Done()
				<-sem
			}()
			err := cb(fn, f)
			if err != nil {
				panic(err)
			}
		}(fileName, file)
	}
	wg.Wait()
}

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
