package constlit

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// EvalExpr evaluates an expression as though it's a constant value.
// data must supply the iota value for ValueSpec's that use it.
// If the expression is invalid or non-constant, EvalExpr can return either
// an Unknown constant, an error, or both.
func EvalExpr(pass *analysis.Pass, expr ast.Expr, data constant.Value) (constant.Value, error) {
	unk := constant.MakeUnknown()
	switch x := expr.(type) {
	case *ast.BasicLit:
		return constant.MakeFromLiteral(x.Value, x.Kind, 0), nil
	case *ast.UnaryExpr:
		xval, err := EvalExpr(pass, x.X, data)
		if err != nil {
			return unk, err
		}
		return constant.UnaryOp(x.Op, xval, 0), nil
	case *ast.BinaryExpr:
		xval, xerr := EvalExpr(pass, x.X, data)
		yval, yerr := EvalExpr(pass, x.Y, data)
		if err := Errors(xerr, yerr); err != nil {
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
		// if v, ok := cl.ids[x]; ok {
		// 	return v, nil
		// }
		return EvalObject(pass, x.Obj)
	default:
		return unk, fmt.Errorf("eval: unhandled expr %T", x)
	}
}

// EvalObject evaluates an *ast.EvalObject for its value
// On error it can return an unknown constant, an error, or both.
func EvalObject(pass *analysis.Pass, o *ast.Object) (constant.Value, error) {
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
			return EvalExpr(pass, decl.Values[0], iotaVal)
		default:
			return unk, fmt.Errorf("unexpected decl.Values length %v", len(decl.Values))
		}
	case *ast.AssignStmt:
		return unk, fmt.Errorf("unhandled %T", decl)
	default:
		return unk, fmt.Errorf("unhandled %T", decl)
	}
}
