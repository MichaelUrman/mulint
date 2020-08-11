package methodlambda

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "methodlambda",
	Doc:  "reports function literals that could be method expressions",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ml := methodLambda{Pass: pass}
		ast.Inspect(file, ml.inspect)
		if ml.err != nil {
			return nil, ml.err
		}
	}
	return nil, nil
}

type methodLambda struct {
	*analysis.Pass
	imports []*ast.ImportSpec
	err     error
}

func (ml *methodLambda) inspect(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.ImportSpec:
		ml.imports = append(ml.imports, n)
	case *ast.FuncLit:
		match, err := ml.matchFn(n)
		if err != nil {
			var note errDiagnostic
			if !errors.As(err, &note) {
				ml.err = err
				return false
			}
			//fmt.Printf("methodlambda: %v @ %-s\n", err, sourcer{ml.Fset, n})
		}
		if match != "" {
			ml.Reportf(n.Pos(), "replace `%s` with `%s`", sourcer{ml.Fset, n.Type}, match)
		}
		return false
	}
	return true
}

func (ml *methodLambda) qualify(pkg *types.Package) string {
	if pkg == ml.Pkg {
		return ""
	}
	for _, spec := range ml.imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if path == pkg.Path() {
			if spec.Name == nil {
				i := strings.LastIndex(path, "/")
				if i == -1 {
					return path
				}
				return path[i+1:]
			}
		}
	}
	return pkg.Path()
}

func (ml *methodLambda) matchFn(fn *ast.FuncLit) (string, error) {
	if fn.Body == nil {
		return "", nil
	}
	switch len(fn.Body.List) {
	case 1:
		switch stmt := fn.Body.List[0].(type) {
		case *ast.ReturnStmt:
			if len(stmt.Results) != 1 {
				return "", nil
			}
			return ml.matchCall(fn, stmt.Results[0], true)

		case *ast.ExprStmt:
			return ml.matchCall(fn, stmt.X, false)

		default:
			return "", nil
		}
	case 2:
		return "", nil // consider handling x = blah(); return x
	}

	return "", nil
}

func (ml *methodLambda) matchCall(fn *ast.FuncLit, call ast.Expr, returned bool) (string, error) {
	if fnHasResults := fn.Type.Results != nil && len(fn.Type.Results.List) > 0; fnHasResults != returned {
		return "", diag("return mismatches of results: %v vs %v", returned, !returned)
	}

	switch call := call.(type) {
	case *ast.CallExpr:
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			return ml.matchParams(fn, fun, append([]ast.Expr{fun.X}, call.Args...), returned)
		}
	default:
		// ast.Print(pass.Fset, call)
	}
	return "", diag("unexpected call type %T", call)
}

func (ml *methodLambda) matchParams(fn *ast.FuncLit, sel *ast.SelectorExpr, args []ast.Expr, returned bool) (string, error) {
	i := 0
	seln := ml.TypesInfo.Selections[sel]
	if seln == nil {
		return "", diag("selection not found")
	}
	meth := seln.Obj()
	sig := meth.Type().(*types.Signature)
	_, ptrRecv := sig.Recv().Type().(*types.Pointer)
	if ptrRecv && !seln.Indirect() {
		return "", diag("method requires %v, but has %v", sig.Recv().Type(), seln.Recv())
	}
	if sig.Results().Len() != 0 != returned {
		return "", diag("method return mismatched lambda return: %v vs %v", sig.Results().Len(), returned)
	}

	for _, param := range fn.Type.Params.List {
		for _, name := range param.Names {
			switch arg := args[i].(type) {
			case *ast.Ident:
				if arg.Name != name.Name {
					return "", diag("name for %v mismatch %q vs %q", i, arg.Name, name.Name)
				}
				if arg.Obj != nil && arg.Obj.Decl != nil {
					// println("arg")
					// ast.Print(pass.Fset, arg)
					switch decl := arg.Obj.Decl.(type) {
					case *ast.Field:
						for j, name := range decl.Names {
							if name.Name == arg.Name && name.Obj != arg.Obj {
								ast.Print(ml.Fset, arg)
								return "", diag("argument object %v.%v didn't match param object", i, j)
							}
						}
					default:
						return "", diag("unexpected decl type %T", decl)
					}
				}
				tv := ml.TypesInfo.Types[param.Type]
				if tv.Type == nil {
					return "", diag("no type for param %v", i)
				}
				if i == 0 && seln.Recv() != tv.Type {
					return "", diag("type %v mismatch %s vs %s", i, seln.Recv(), tv.Type)
				} else if i > 0 && sig.Params().At(i-1).Type() != tv.Type {
					return "", diag("type %v mismatch %s vs %s", i, sig.Params().At(i-1).Type(), tv.Type)
				}
			default:
				return "", diag("unexpected arg type %T", arg)
			}
			i++
		}
	}

	if i == len(args) {
		switch t := seln.Recv().(type) {
		case *types.Named:
			return types.TypeString(t, ml.qualify) + "." + sel.Sel.Name, nil
		case *types.Pointer:
			return "(" + types.TypeString(t, ml.qualify) + ")." + sel.Sel.Name, nil
		default:
			return "", fmt.Errorf("unexpected selection recv %T", t)
		}
	}
	return "", diag("length mismatch %v vs %v", i, len(args))
}

type errDiagnostic error

func diag(format string, a ...interface{}) error {
	return errDiagnostic(fmt.Errorf(format, a...))
}
