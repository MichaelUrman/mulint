package constlit

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/loader"
)

var Analyzer = &analysis.Analyzer{
	Name:     "constlit",
	Doc:      "reports literals that should be constants",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	cl := constLit{Pass: pass}
	return cl.Run()
}

type constLit struct {
	*analysis.Pass

	imported    imports
	importnames map[importpath]importname
	ids         map[*ast.Ident]constant.Value
	local       constants
}

type imports map[importpath]constants

func (i imports) Import(path importpath) {
	i[path] = constants{}
}
func (i imports) Load(eval func(*ast.Ident) (constant.Value, error)) error {
	if len(i) == 0 {
		return nil
	}
	var cfg loader.Config
	for path := range i {
		cfg.Import(string(path))
	}

	prog, err := cfg.Load()
	if err != nil {
		return err
	}

	for _, info := range prog.Imported {
		for id, obj := range info.Defs {
			if !id.IsExported() || id.Obj == nil || id.Obj.Kind != ast.Con {
				continue
			}
			if obj == nil || obj.Type() == nil {
				continue
			}

			v, err := eval(id)
			if err != nil || v.Kind() == constant.Unknown {
				continue
			}
			c := i[importpath(obj.Pkg().Path())]
			t := obj.Type()
			if !untyped(t) {
				c.Add(id, t, v)
			}
		}
	}
	for path, constants := range i {
		if len(constants) == 0 {
			delete(i, path)
		}
	}
	return nil
}

type importpath string
type importname string
type constants map[cval]*ast.Ident
type cval struct {
	Type  string
	Value constant.Value
}

func (c constants) Add(id *ast.Ident, ctype types.Type, value constant.Value) {
	c[cval{ctype.String(), value}] = id
}

func (c constants) Lookup(ctype types.Type, value constant.Value) *ast.Ident {
	return c[cval{ctype.String(), value}]
}

func (cl *constLit) Run() (interface{}, error) {
	inspect := cl.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	importFilter := []ast.Node{
		(*ast.ImportSpec)(nil),
	}

	cl.imported = make(map[importpath]constants)
	inspect.Preorder(importFilter, func(n ast.Node) {
		if n, ok := n.(*ast.ImportSpec); ok {
			cl.imported.Import(cl.importPath(n))
		}
	})

	cl.ids = make(map[*ast.Ident]constant.Value)
	if err := cl.imported.Load(cl.EvalID); err != nil {
		return nil, err
	}
	cl.ids = make(map[*ast.Ident]constant.Value)

	cl.local = make(constants)
	for id, obj := range cl.TypesInfo.Defs {
		if obj == nil {
			continue
		}
		if id.Obj == nil || id.Obj.Kind != ast.Con {
			continue
		}
		if obj == nil || obj.Type() == nil {
			continue
		}

		v, err := cl.Eval(id, nil)
		if err != nil || v.Kind() == constant.Unknown {
			continue
		}
		t := obj.Type()
		if !untyped(t) {
			cl.local.Add(id, t, v)
		}
	}

	literalFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.ImportSpec)(nil),
		(*ast.UnaryExpr)(nil),
		(*ast.BasicLit)(nil),
	}
	inspect.WithStack(literalFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		switch n := n.(type) {
		case *ast.File:
			cl.importnames = make(map[importpath]importname)
			return true

		case *ast.ImportSpec:
			// imports let us know what other constants we can use
			name := ""
			if n.Name != nil {
				name = n.Name.Name
			}

			cl.importnames[cl.importPath(n)] = importname(name)
			return false

		case *ast.UnaryExpr:
			// literals like -1 are a UnaryExpr around a BasicLit
			lit, ok := n.X.(*ast.BasicLit)
			if !ok || (n.Op != token.ADD && n.Op != token.SUB) {
				return true
			}

			// ensure the expression has a known type
			xtype := cl.typeof(stack)
			if xtype == nil {
				return true
			}

			// try to evaluate the expression as a constant
			litval := constant.UnaryOp(n.Op, constant.MakeFromLiteral(lit.Value, lit.Kind, 0), 0)
			if litval.Kind() == constant.Unknown {
				return true
			}

			// match against known constants, don't recurse the rest of this AST
			cl.Check(n, xtype, litval, stack)
			return false

		case *ast.BasicLit:
			xtype := cl.typeof(stack)
			if xtype == nil {
				return false
			}

			litval := constant.MakeFromLiteral(n.Value, n.Kind, 0)
			cl.Check(n, xtype, litval, stack)
			return false
		}

		return true
	})
	cl.ids = nil
	return nil, nil
}

func (cl *constLit) Check(n ast.Expr, xt types.Type, val constant.Value, stack []ast.Node) {
	id := cl.local.Lookup(xt, val)
	if id != nil && id.Obj != nil {
		decl, ok := id.Obj.Decl.(ast.Node)
		if !ok || !decl.Pos().IsValid() || decl.Pos() > n.Pos() ||
			!decl.End().IsValid() || decl.End() < n.End() {
			fix := fmt.Sprintf("Replace `%s` with `%s`", cl.Render(n), id.Name)
			cl.Report(analysis.Diagnostic{
				Pos:     n.Pos(),
				Message: fix,
				SuggestedFixes: []analysis.SuggestedFix{{
					Message: fix,
					TextEdits: []analysis.TextEdit{{
						Pos:     n.Pos(),
						End:     n.End(),
						NewText: []byte(id.Name),
					}},
				}},
			})
			return
		}
	}

	for imppath, constants := range cl.imported {
		id := constants.Lookup(xt, val)
		if id == nil {
			continue
		}

		orig := cl.Render(n)
		var fix string
		edits := []analysis.TextEdit{
			{
				Pos: n.Pos(),
				End: n.End(),
			},
		}
		imp, ok := cl.importnames[imppath]
		if !ok {
			imp = "_"
		}
		if imp == "" {
			imp = importname(path.Base(string(imppath)))
		}
		switch imp {
		case ".":
			fix = fmt.Sprintf("Replace `%s` with `%s`", orig, id.Name)
			edits[0].NewText = []byte(id.Name)
		case "_":
			// don't offer ignored modules' constants
			continue
		default:
			if ok {
				fix = fmt.Sprintf("Replace `%s` with `%s.%s`", orig, imp, id.Name)
				edits[0].NewText = []byte(string(imp) + "." + id.Name)
			} else {
				fix = fmt.Sprintf("Consider importing \"%s\" and replacing `%s` with `%s.%s`", imppath, orig, path.Base(string(imppath)), id.Name)
			}
		}
		if cl.unlessTimes(stack, string(imp)) {
			return
		}
		cl.Report(analysis.Diagnostic{
			Pos:     n.Pos(),
			Message: fix, //fmt.Sprintf("literal with known constant: %v", orig),
			SuggestedFixes: []analysis.SuggestedFix{
				{Message: fix, TextEdits: edits},
			},
		})
	}
}

func (cl *constLit) EvalID(id *ast.Ident) (constant.Value, error) {
	return cl.Eval(id, nil)
}

func (cl *constLit) Eval(x ast.Expr, data constant.Value) (constant.Value, error) {
	unk := constant.MakeUnknown()
	switch x := x.(type) {
	case *ast.BasicLit:
		return constant.MakeFromLiteral(x.Value, x.Kind, 0), nil
	case *ast.UnaryExpr:
		xval, err := cl.Eval(x.X, data)
		if err != nil {
			return unk, err
		}
		return constant.UnaryOp(x.Op, xval, 0), nil
	case *ast.BinaryExpr:
		xval, xerr := cl.Eval(x.X, data)
		yval, yerr := cl.Eval(x.Y, data)
		if err := wraperr(xerr, yerr); err != nil {
			return unk, err
		}
		switch x.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
			return constant.MakeBool(constant.Compare(xval, x.Op, yval)), nil
		case token.SHL, token.SHR:
			if sh, exact := constant.Int64Val(yval); exact {
				return constant.Shift(yval, x.Op, uint(sh)), nil
			}
			return unk, nil
		default:
			return constant.BinaryOp(xval, x.Op, yval), nil
		}
	case *ast.Ident:
		if x.Name == "iota" {
			if data != nil {
				return data, nil
			}
			return unk, nil
		}
		if v, ok := cl.ids[x]; ok {
			return v, nil
		}
		return cl.EvalObj(x.Obj)
	default:
		return unk, fmt.Errorf("eval: unhandled expr %T", x)
	}
}

func (cl *constLit) EvalObj(o *ast.Object) (constant.Value, error) {
	unk := constant.MakeUnknown()
	if o == nil || o.Decl == nil || o.Kind != ast.Con {
		return unk, nil
	}

	data, ok := o.Data.(int)
	if !ok {
		return unk, fmt.Errorf("obj data %T", o.Data)
	}
	iotaVal := constant.Make(int64(data))

	switch decl := o.Decl.(type) {
	case *ast.ValueSpec:
		switch len(decl.Values) {
		case 0:
			return iotaVal, nil
		case 1:
			return cl.Eval(decl.Values[0], iotaVal)
		default:
			return unk, fmt.Errorf("unexpected decl.Values length %v", len(decl.Values))
		}
	case *ast.AssignStmt:
		return unk, fmt.Errorf("unhandled %T", decl)
	default:
		return unk, fmt.Errorf("unhandled %T", decl)
	}
}

func (cl *constLit) Render(x ast.Expr) string {
	switch x := x.(type) {
	case *ast.BasicLit:
		return x.Value
	case *ast.UnaryExpr:
		return x.Op.String() + cl.Render(x.X)
	}
	return ""
}

func (cl *constLit) saveid(id *ast.Ident, val constant.Value) constant.Value {
	cl.ids[id] = val
	return val
}

func (cl *constLit) importPath(imp *ast.ImportSpec) importpath {
	return importpath(constant.StringVal(constant.MakeFromLiteral(imp.Path.Value, imp.Path.Kind, 0)))
}

func (cl *constLit) typeof(stack []ast.Node) types.Type {
	for n := len(stack); n > 0; n-- {
		switch x := stack[n-1].(type) {
		case *ast.BinaryExpr, *ast.UnaryExpr, *ast.BasicLit:
			xtype := cl.TypesInfo.TypeOf(x.(ast.Expr))
			if !untyped(xtype) {
				return xtype
			}
		default:
			return nil
		}
	}
	return nil
}

func (cl *constLit) unlessTimes(stack []ast.Node, imp string) bool {
	for n := len(stack); n > 0; n-- {
		switch x := stack[n-1].(type) {
		case *ast.UnaryExpr:
			continue
		case *ast.BinaryExpr:
			if x.Op != token.MUL {
				return false
			}

			if n < len(stack) {
				inner := stack[n].(ast.Expr)
				other := x.X
				if x.X == inner {
					other = x.Y
				}

				// prevent, e.g.,  1 * time.Second from flagging 1 -> time.Nanosecond
				switch o := other.(type) {
				case *ast.SelectorExpr:
					switch ox := o.X.(type) {
					case *ast.Ident:
						if ox.Name == imp {
							return true
						}
					}
				}
			}

			return false
		}
	}
	return false
}

func untyped(o types.Type) bool {
	b, ok := o.(*types.Basic)
	return ok && b.Kind() >= types.UntypedBool && b.Kind() <= types.UntypedNil
}
