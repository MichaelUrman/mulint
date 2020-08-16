package constlit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"math/bits"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

/*
The constlit Analyzer determines the values of constants in local and dependency packages, then looks for literals of the same type that can be replaced with the constant.

To find constants in dependency packages, it exports a fact of type *constantValues to its other runs. This appears to be the only way facts cross (analyzed) package boundaries. (A prior incarnation of this split the constant evaluation into a separate analyzer. Those constants could only be consumed via Requires/ResultOf, and then the main analyzer had to re-export a fact.)
*/
var Analyzer = &analysis.Analyzer{
	Name:      "constlit",
	Doc:       "reports literals that should be constants",
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	FactTypes: []analysis.Fact{(*constantValues)(nil)},
	Run:       run,
}

// constantValues are exported as a fact to cross package boundaries
type constantValues []constantValue

func (*constantValues) AFact() {}

type constantValue struct {
	id  *ast.Ident
	obj types.Object
	val constant.Value
}

func (v constantValue) tv() typedValue { return typedValue{v.obj.Type(), v.val} }
func (v constantValue) String() string { return v.obj.Name() + "=" + v.val.String() }

type typedValue struct {
	Type  types.Type
	Value constant.Value
}

type byValue map[typedValue][]types.Object

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	imports := make(map[string]string)

	// load local constants first, so they are preferred as replacements
	constants := localConstants(pass)
	// merge imported constants, so they are less preferred
	mergeImportedConstants(pass, constants)

	literalFilter := []ast.Node{
		(*ast.ImportSpec)(nil), // for proper selector names, or excluded packages
		(*ast.UnaryExpr)(nil),  // for, e.g., -1, which is a &UnaryExpr{X: &BasicLit}
		(*ast.BasicLit)(nil),   // for e.g., 20
	}
	inspect.WithStack(literalFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			// imports let us know what other constants we can use
			name := ""
			if n.Name != nil {
				name = n.Name.Name
			}

			imports[constant.StringVal(constant.MakeFromLiteral(n.Path.Value, n.Path.Kind, 0))] = name
			return false

		case *ast.UnaryExpr:
			// literals like -1 are a &UnaryExpr{X: &BasicLit{}}
			lit, ok := n.X.(*ast.BasicLit)
			if !ok {
				// recurse for more complicated cases, like &UnaryExpr{X: &BinaryExpr{}}
				return true
			}

			// ensure the expression has a known type
			xtype := typeof(stack, pass.TypesInfo)
			if xtype == nil {
				return true
			}

			// try to evaluate the expression as a constant
			prec := uint(0)
			if basic, ok := xtype.Underlying().(*types.Basic); ok && n.Op == token.XOR {
				switch basic.Kind() {
				case types.Int8, types.Uint8:
					prec = 8
				case types.Int16, types.Uint16:
					prec = 16
				case types.Int32, types.Uint32:
					prec = 32
				case types.Int64, types.Uint64:
					prec = 64
				case types.Int, types.Uint:
					prec = bits.UintSize
				}
			}

			litval := constant.UnaryOp(n.Op, constant.MakeFromLiteral(lit.Value, lit.Kind, 0), prec)
			if litval.Kind() == constant.Unknown {
				return true
			}

			// match against known constants, don't recurse the rest of this AST
			check(pass, constants[typedValue{xtype, litval}], n, stack, imports)
			return false

		case *ast.BasicLit:
			xtype := typeof(stack, pass.TypesInfo)
			if xtype == nil {
				return false
			}

			litval := constant.MakeFromLiteral(n.Value, n.Kind, 0)
			check(pass, constants[typedValue{xtype, litval}], n, stack, imports)
			return false
		}

		return true
	})
	return nil, nil
}

func localConstants(pass *analysis.Pass) byValue {
	constants := byValue{}
	var exported constantValues

	for id, obj := range pass.TypesInfo.Defs {
		if !id.IsExported() || id.Obj == nil || id.Obj.Kind != ast.Con {
			continue
		}
		if obj == nil || obj.Type() == nil {
			continue
		}
		if strings.HasPrefix(obj.Pkg().Path(), "internal/") {
			continue
		}

		v, err := EvalExpr(pass, id, constant.MakeUnknown())
		if err != nil || v.Kind() == constant.Unknown {
			continue
		}

		tv := typedValue{obj.Type(), v}
		constants[tv] = append(constants[tv], obj)
		if obj.Exported() /*&& !untyped(obj.Type())*/ && !builtin(obj.Type()) {
			exported = append(exported, constantValue{id, obj, v})
		}
	}
	if len(exported) != 0 {
		pass.ExportPackageFact(&exported)
	}

	// println(pass.Pkg.Path(), "has", len(constants), "constants,", len(exported) " are exported")

	return constants
}

func mergeImportedConstants(pass *analysis.Pass, constants byValue) {
	for _, pkg := range pass.Pkg.Imports() {
		var exported constantValues
		if !pass.ImportPackageFact(pkg, &exported) {
			continue
		}

		for _, con := range exported {
			constants[con.tv()] = append(constants[con.tv()], con.obj)
		}
	}
}

func check(pass *analysis.Pass, constants []types.Object, n ast.Expr, stack []ast.Node, imports map[string]string) {
	for _, obj := range constants {
		if obj.Pkg() == pass.Pkg {
			if declares(stack, obj) {
				continue
			}
		} else if !obj.Exported() {
			continue
		}

		orig := &bytes.Buffer{}
		printer.Fprint(orig, pass.Fset, n)

		var fix string
		edits := []analysis.TextEdit{
			{
				Pos: n.Pos(),
				End: n.End(),
			},
		}

		imp, ok := imports[obj.Pkg().Path()]
		if imp == "" {
			imp = obj.Pkg().Name()
		}
		if obj.Pkg() == pass.Pkg {
			imp = "."
		}
		switch imp {
		case ".":
			fix = fmt.Sprintf("Replace `%s` with `%s`", orig, obj.Name())
			edits[0].NewText = []byte(obj.Name())
		case "_":
			// don't offer ignored modules' constants
			continue
		default:
			if ok {
				fix = fmt.Sprintf("Replace `%s` with `%s.%s`", orig, imp, obj.Name())
				edits[0].NewText = []byte(imp + "." + obj.Name())
			} else {
				fix = fmt.Sprintf("Consider importing \"%s\" and replacing `%s` with `%s.%s`", obj.Pkg().Path(), orig, obj.Pkg().Name(), obj.Name())
			}
		}
		if timesTypedConstant(stack, obj.Type(), pass.TypesInfo) {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:     n.Pos(),
			Message: fix, //fmt.Sprintf("literal with known constant: %v", orig),
			SuggestedFixes: []analysis.SuggestedFix{
				{Message: fix, TextEdits: edits},
			},
		})
	}
}

func typeof(stack []ast.Node, ti *types.Info) types.Type {
	for n := len(stack); n > 0; n-- {
		switch x := stack[n-1].(type) {
		case *ast.BinaryExpr, *ast.UnaryExpr, *ast.BasicLit:
			xtype := ti.TypeOf(x.(ast.Expr))
			if !untyped(xtype) {
				return xtype
			}
		default:
			return nil
		}
	}
	return nil
}

// timesTypedConstant returns true if the leaf-most literal is multiplied by a constant of the type we need. This prevents recommending `1 * time.Second` become `time.Nanosecond * times.Second`.
func timesTypedConstant(stack []ast.Node, want types.Type, ti *types.Info) bool {
	for n := len(stack); n > 0; n-- {
		switch x := stack[n-1].(type) {
		case *ast.UnaryExpr:
			continue
		case *ast.BinaryExpr:
			if x.Op != token.MUL || n >= len(stack) {
				return false
			}

			inner := stack[n].(ast.Expr)
			other := x.X
			if x.X == inner {
				other = x.Y
			}

			// prevent, e.g.,  1 * time.Second from flagging 1 -> time.Nanosecond
			switch o := other.(type) {
			case *ast.Ident:
				return isTypedConstant(ti.Uses[o], want)

			case *ast.SelectorExpr:
				if sel := ti.Selections[o]; sel != nil {
					return isTypedConstant(sel.Obj(), want)
				}
				return isTypedConstant(ti.Uses[o.Sel], want)
			}
			break
		}
	}
	return false
}

func isTypedConstant(obj types.Object, want types.Type) bool {
	if want != nil && obj.Type() != want {
		return false
	}
	_, ok := obj.(*types.Const)
	return ok
}

func untyped(o types.Type) bool {
	b, ok := o.(*types.Basic)
	return ok && b.Kind() >= types.UntypedBool && b.Kind() <= types.UntypedNil
}

func builtin(o types.Type) bool {
	_, ok := o.(*types.Basic)
	return ok
}

func declares(stack []ast.Node, obj types.Object) bool {
	if len(stack)-2 < 0 {
		return false
	}
	vs, ok := stack[len(stack)-2].(*ast.ValueSpec)
	if !ok {
		return false
	}
	return vs.Pos() <= obj.Pos() && vs.End() >= obj.Pos()
}
