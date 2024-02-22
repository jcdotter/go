// Copyright 2023 james dotter.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://github.com/jcdotter/go/LICENSE
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inspect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/jcdotter/go/data"
	"github.com/jcdotter/go/path"
)

// Inspect parses the package content in the path provided and
// returns the package object for inspection, or an error if
// the package cannot be parsed.
func Inspect(PkgPath string) (*Package, error) {
	p := NewPackage(PkgPath)
	if err := p.Parse(); err != nil {
		return nil, err
	}
	for _, f := range p.Files.List() {
		if err := f.(*File).Inspect(); err != nil {
			return nil, err
		}
	}
	p.i = true
	return p, nil
}

// Parse parses the package content if not already parsed. If Entites are provided,
// the package will only parse the provided entities, otherwise the package will
// parse all entities in the package. Returns an error if the package cannot be parsed.
// TODO: Make file parsing concurrent.
func (p *Package) Parse() (err error) {
	// parse each file in the package
	for _, f := range path.Files(p.Path) {
		var file *File

		// parse file name
		n := f[strings.LastIndex(f, "/")+1 : strings.LastIndex(f, ".")]

		// check if file is already parsed
		// else add a new file to the package
		if f := p.Files.Get(n); f != nil {
			return
		}
		file = NewFile(p, n)
		p.Files.Add(file)

		// parse file to abstract syntax tree
		file.t, err = parser.ParseFile(token.NewFileSet(), f, nil, parser.SkipObjectResolution)
		if err != nil {
			return
		}
	}

	// parse package name
	if p.Files.Len() > 0 {
		p.Name = p.Files.Index(0).(*File).t.Name.Name
	}
	return
}

// Inspect inspects the declared entities in the file and
// adds them to the package.
func (f *File) Inspect() (err error) {
	if f.t != nil {
		for _, d := range f.t.Decls {

			// route declaration to appropriate
			// inspection method
			switch decl := d.(type) {
			case *ast.FuncDecl:
				err = f.InspectFunc(decl)
			case *ast.GenDecl:
				switch decl.Tok {
				case token.CONST:
					err = f.InspectValues(CONST, decl.Specs)
				case token.VAR:
					err = f.InspectValues(VAR, decl.Specs)
				case token.TYPE:
					err = f.InspectType(decl.Specs)
				case token.IMPORT:
					err = f.InspectImports(decl.Specs)
				}
			}
		}
	}
	return
}

// InspectImports inspects the import declarations in the file and
// adds them to the package.
func (f *File) InspectImports(specs []ast.Spec) (err error) {
	for _, s := range specs {

		// create and add import to file
		i := s.(*ast.ImportSpec)
		imp := &Import{file: f, name: i.Name.Name}
		f.i.Add(imp)

		// get package by path if already imported another file,
		// otherwise create a new imported package and add it to
		// the current package
		pkgPath := i.Path.Value
		if pkg := f.p.Imports.Get(pkgPath); pkg != nil {
			imp.pkg = pkg.(*Package)
		} else {
			imp.pkg = NewPackage(pkgPath)
			f.p.Imports.Add(imp.pkg)
		}
	}
	return
}

// InspectValues inspects the value declarations in the file and
// adds them to the package.
func (f *File) InspectValues(k byte, specs []ast.Spec) (err error) {

	// the prior type is used when only one type
	// is used for a var or const declaration.
	var priorType *Type

	// iterate through specs and create values for each
	for _, s := range specs {

		// assert value spec
		var vals = s.(*ast.ValueSpec)
		var num = len(vals.Names)
		var names = make([]*ast.Ident, num)

		// check if the value has already been
		// added to the package, if so, skip it
		for i, n := range vals.Names {
			if f.p.Values.Get(n.Name) == nil {
				names[i] = n
				continue
			}
			num--
		}
		if num == 0 {
			continue
		}

		// get declared type if it exists. if not,
		// the prior declared type will set the
		// type of this value if there are no
		// assginment expressions in this value.
		var typ *Type
		if vals.Type != nil {
			typ = f.TypeExpr(vals.Type)
			priorType = typ
		} else if priorType != nil && len(vals.Values) == 0 {
			typ = priorType
		}

		// if the type is not declared and the value
		// is function call, store the output types
		// to be applied to the ValueSpec.Values
		var types []*Type
		if typ == nil && num > 1 && len(vals.Values) == 1 {
			types = make([]*Type, num)
			if t := f.TypeExpr(vals.Values[0]); t != nil && t.kind == FUNC && t.object != nil {
				for i, t := range t.object.(*Func).out.List() {
					types[i] = t.(*Type)
				}
			}
		}

		// iterate through and create value for each named
		// value in the value spec, using the declared type
		// if it exists, or deriving the type from the
		// value expression if it exists.
		for i, n := range names {

			// if value already exists, name will be nil
			// so skip it and continue to the next value
			if n == nil {
				continue
			}

			// create and add value to package
			val := &Value{file: f, kind: k, name: n.Name}
			f.p.Values.Add(val)

			// set value type if already declared
			// or derrived from function call output,
			// otherwise derive it from value expression
			switch {
			case typ != nil:
				val.typ = typ
			case len(types) > i:
				val.typ = types[i]
			case len(vals.Values) > i:
				val.typ = f.TypeExpr(vals.Values[i])
			}

			// print test
			f.PrintValue(val)
		}

	}
	return
}

func (f *File) InspectValue(k byte, v *ast.ValueSpec) (err error) {
	return f.InspectValues(k, []ast.Spec{v})
}

func (f *File) PrintValue(v *Value) {
	var tname string
	var tkind byte
	if v.typ != nil {
		tname = v.typ.name
		tkind = v.typ.kind
	}
	fmt.Println("VALUE:",
		v.kind,
		v.name,
		tname,
		tkind,
	)
}

func (f *File) InspectType(t []ast.Spec) (err error) {
	/* for _, s := range t {
		fmt.Println("TYPE:", s.(*ast.TypeSpec).Name.Name)
	} */
	return
}

func (f *File) InspectFunc(fn *ast.FuncDecl) (err error) {
	//fmt.Println("FUNC DECL:", fn.Name.Name)
	return
}

// ----------------------------------------------------------------------------
// Type Evaluation Methods

// GetType returns the type of the name provided by first checking
// builtin types, then declared types, and lastly inspecting the
// ident object if it exists.
func (f *File) GetType(name string) (typ *Type, err error) {

	// check builtin types
	if t := BuiltinTypes.Get(name); t != nil {
		return t.(*Type), nil
	}

	// check declared types
	// TODO: if declared type has not been parsed,
	// follow the ident object to import the
	// declared type in the current package
	if t := f.p.Types.Get(name); t != nil {
		return t.(*Type), nil
	}

	// if an imported type, get the type from the
	// imported package and return it. if the imported
	// package has not been parsed, parse it and return
	// the type if it exists.
	if parts := strings.Split(name, "."); len(parts) > 1 {
		// get imported package if type contains a period
		if imp := f.i.Get(parts[0]); imp != nil {
			i := imp.(*Import)
			// get type from imported package
			t := i.pkg.Types.Get(parts[1])
			if t == nil {
				if err = i.pkg.Parse(); err != nil {
					return
				}
				t = i.pkg.Types.Get(parts[1])
			}
			if t != nil {
				return t.(*Type), nil
			}
		}
		return nil, ErrNotType
	}

	// TODO: parse complex data types by
	// cascading through the type Ident and
	// fetching or parsing the type component
	// and add them to the package types

	return nil, nil
}

func (f *File) TypeExpr(e ast.Expr) *Type {

	// TODO: build out expression type evaluation
	// using the expresssion list below

	switch t := e.(type) {
	case *ast.BasicLit:
		// literal expression of int, float, rune, string
		return TypeToken(t.Kind)
	case *ast.ParenExpr:
		return f.TypeExpr(t.X)
	case *ast.Ident:
		return f.TypeIdent(t)
	case *ast.StarExpr:
		return f.TypePointer(t)
	case *ast.UnaryExpr:
		return f.TypeUnary(t)
	case *ast.BinaryExpr:
		return f.TypeBinary(t)
	case *ast.CallExpr:
		return f.TypeCall(t)
	case *ast.FuncLit:
		return f.TypeFunc(t)
	case *ast.CompositeLit:
		switch c := t.Type.(type) {
		case *ast.Ident:
			return f.TypeIdent(c)
		case *ast.SelectorExpr:
			// TODO: return f.TypeSelector(c)
		case *ast.ArrayType:
			return f.TypeArray(c)
		case *ast.MapType:
			return f.TypeMap(c)
		case *ast.StructType:
			return f.TypeStruct(c)
		}
	case *ast.SelectorExpr:
		// call to an external package function, value or type
		// or call to internal package method or struct field
	default:
		// case *ast.TypeAssertExpr:
		// case *ast.IndexExpr:
		// case *ast.SliceExpr:
	}
	fmt.Println("EXPR:", reflect.TypeOf(e))
	return nil

	/* switch t := v.Values[0].(type) {
	case *ast.TypeAssertExpr:
		// TODO: evaluate type assert expression
	case *ast.Ident:
		val.typ = TypeToken(token.IDENT)
	case *ast.BasicLit:
		val.typ = TypeToken(t.Kind)
	case *ast.UnaryExpr:
		// TODO: evaluate unary expression (&, *, etc.)
	case *ast.BinaryExpr:
		// TODO: evaluate binary expression
	case *ast.CallExpr:
		// TODO: evaluate call expression
	case *ast.FuncLit:
		// TODO: evaluate function literal
	case *ast.CompositeLit:
		switch t.Type.(type) {
		case *ast.Ident:
			val.typ = TypeToken(token.IDENT)
		case *ast.SelectorExpr:
			// TODO: evaluate selector expression
		case *ast.ArrayType:
			// TODO: evaluate array type
		case *ast.MapType:
			// TODO: evaluate map type
		case *ast.StructType:
			// TODO: evaluate struct type
		case *ast.InterfaceType:
			// TODO: evaluate interface type
		case *ast.ChanType:
			// TODO: evaluate chan type
		default:
			// TODO: check for other types
		}
	} */
}

// TypeIdent returns the type of the identifier in the file
// by first checking builtin values, then declared types,
// and lastly inspecting the ident object if it exists.
func (f *File) TypeIdent(i *ast.Ident) (typ *Type) {

	// check builtin values and parsed types
	if v := BuiltinValues.Get(i.Name); v != nil {
		return v.(*Value).typ
	}
	if typ, _ = f.GetType(i.Name); typ != nil {
		return
	}

	// if ident is not in types and has an object,
	// inspect the object and return the type
	if i.Obj != nil {
		switch i.Obj.Kind {
		case ast.Var:
			if err := f.InspectValue(VAR, i.Obj.Decl.(*ast.ValueSpec)); err == nil {
				return f.p.Values.Get(i.Name).(*Value).typ
			}
		case ast.Con:
			if err := f.InspectValue(CONST, i.Obj.Decl.(*ast.ValueSpec)); err == nil {
				return f.p.Values.Get(i.Name).(*Value).typ
			}
		case ast.Typ:
			if err := f.InspectType([]ast.Spec{i.Obj.Decl.(*ast.TypeSpec)}); err == nil {
				return f.p.Types.Get(i.Name).(*Type)
			}
		case ast.Fun:
			if err := f.InspectFunc(i.Obj.Decl.(*ast.FuncDecl)); err == nil {
				return f.p.Funcs.Get(i.Name).(*Func).typ
			}
		}
	}
	return
}

// typePointer returns the pointer type of the type provided.
func (f *File) typePointer(t *Type) (typ *Type) {
	n := "*" + t.name
	if t := f.p.Types.Get(n); t != nil {
		return t.(*Type)
	}
	typ = &Type{
		file:   t.file,
		name:   "*" + t.name,
		kind:   POINTER,
		object: &Pointer{elem: t},
	}
	typ.object.(*Pointer).typ = typ
	f.p.Types.Add(typ)
	return
}

// TypePointer returns the pointer type of the type provided.
func (f *File) TypePointer(p *ast.StarExpr) (typ *Type) {
	return f.typePointer(f.TypeExpr(p.X))
}

// TypeUnary returns the unary type of the type provided.
// Typically used for &Expr expressions.
func (f *File) TypeUnary(u *ast.UnaryExpr) (typ *Type) {
	switch u.Op {
	case token.AND:
		return f.typePointer(f.TypeExpr(u.X))
	}
	return
}

// TypeBinary returns the binary type of the types provided.
// Typically used for equation expressions (0+0 or ""+"").
func (f *File) TypeBinary(b *ast.BinaryExpr) (typ *Type) {
	// return type of the left expression if it is not
	// a basic literal. Go only adapts the type of basic
	// literals, so we can assume the type of a non-basic
	// literal is the type of the binary expression.
	x := b.X
	if p, ok := x.(*ast.ParenExpr); ok {
		x = p.X
	}
	if _, ok := x.(*ast.BasicLit); !ok {
		return f.TypeExpr(x)
	}
	return f.TypeExpr(b.Y)
}

// TypeCall returns the type of the call expression provided.
func (f *File) TypeCall(c *ast.CallExpr) (typ *Type) {
	return f.TypeExpr(c.Fun)
}

// TypeFunc returns the type of the function literal provided.
func (f *File) TypeFunc(fn *ast.FuncLit) (typ *Type) {
	// TODO: implement function literal as a function in the package.
	// currently only stored as a value in the package.
	fnc := &Func{file: f}
	typ = &Type{
		file:   f,
		kind:   FUNC,
		object: fnc,
	}
	fnc.typ = typ
	f.TypeFuncParams(fn.Type.Params, fnc.in)
	f.TypeFuncParams(fn.Type.Results, fnc.out)
	return
}

// TypeFuncParams adds the type of the function
// parameters provided to the data list provided.
func (f *File) TypeFuncParams(from *ast.FieldList, to *data.Data) {
	if from != nil {
		for _, field := range from.List {
			t := f.TypeExpr(field.Type)
			for range field.Names {
				to.Add(t)
			}
		}
	}
}

// TypeArray returns the type of the array expression provided.
func (f *File) TypeArray(a *ast.ArrayType) (typ *Type) {
	arr := &Array{elem: f.TypeExpr(a.Elt)}
	typ = &Type{
		file:   f,
		object: arr,
	}
	arr.typ = typ
	if a.Len == nil {
		typ.kind = SLICE
		typ.name = "[]" + arr.elem.name
	} else if _, ok := a.Len.(*ast.Ellipsis); ok {
		typ.kind = ELIPS
		typ.name = "..." + arr.elem.name
	} else {
		typ.kind = ARRAY
		l := a.Len.(*ast.BasicLit).Value
		arr.len, _ = strconv.Atoi(l)
		typ.name = "[" + l + "]" + arr.elem.name
	}
	return
}

// TypeMap returns the type of the map expression provided.
func (f *File) TypeMap(m *ast.MapType) (typ *Type) {
	mp := &Map{key: f.TypeExpr(m.Key), elem: f.TypeExpr(m.Value)}
	typ = &Type{
		file:   f,
		kind:   MAP,
		object: mp,
	}
	mp.typ = typ
	typ.name = "map[" + mp.key.name + "]" + mp.elem.name
	return
}

// TypeStruct returns the type of the struct expression provided.
func (f *File) TypeStruct(s *ast.StructType) (typ *Type) {
	typ = &Type{
		file: f,
		kind: STRUCT,
	}
	str := NewStruct(typ)
	typ.object = str
	// TODO: loop fields and add them to the struct
	// if field is func, add as method, else add as field
	return
}
