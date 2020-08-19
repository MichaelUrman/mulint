package relock

import (
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type Pather interface {
	Path() string
}

func CallerPath(path Pather, caller *ssa.Function, callee *ssa.Call) Pather {
	switch path := path.(type) {
	case GlobalPath:
		return path
	case RecvPath:
		var subpath string
		if sub := strings.IndexAny(string(path), ".[("); sub > 0 {
			subpath = string(path)[sub:]
		}
		ops := callee.Operands(nil)
		if len(ops) < 2 || ops[1] == nil {
			return nil
		}

		return makePath(*ops[1], caller, callee, subpath, "")
	default:
		println("PATHER", reflect.TypeOf(path).String(), path.Path())
		return nil
	}
}

func makePath(param ssa.Value, scope *ssa.Function, pos ssa.Instruction, subpath string, makeCtx string) Pather {
	// scope = caller
	switch param := param.(type) {
	case *ssa.Global:
		return GlobalPath(param.Object().Pkg().Path() + ":" + param.Object().Name() + subpath)
	case *ssa.Parameter:
		if scope.Signature.Recv() != nil && param == scope.Params[0] {
			return RecvPath(scope.Object().(*types.Func).Type().(*types.Signature).Recv().Name() + subpath)
		}
		return ParamPath(param.Name() + subpath)
	case *ssa.FieldAddr:
		subpath = "." + param.X.
			Type().Underlying().(*types.Pointer).
			Elem().Underlying().(*types.Struct).
			Field(param.Field).Name() + subpath

		return makePath(param.X, scope, pos, subpath, makeCtx+"_FA")
	case *ssa.Alloc:
		for _, ref := range *param.Referrers() {
			if ref.Block() == pos.Block() {
				if ref.Pos() >= pos.Pos() {
					continue
				}
				switch ref := ref.(type) {
				case *ssa.Call: // ignore
				case *ssa.Store:
					return makePath(ref.Val, scope, pos, subpath, makeCtx+"_ALLOCSTORE")
				default:
					println("ALLOC REF", reflect.TypeOf(ref).String(), ref.String())
				}
				for _, inst := range ref.Block().Instrs {
					if inst == pos {
						break
					}

				}
			}
			println("RECV: *ssa.Alloc ref", ref.String(), reflect.TypeOf(ref).String(), ref.Pos(), pos.Pos())
		}
		println()

	case *ssa.IndexAddr:
		subpath = makeIndex(param.Index, makeCtx+"_IA") + subpath
		return makePath(param.X, scope, pos, subpath, makeCtx+"_IA")
	case *ssa.UnOp:
		return makePath(param.X, scope, pos, subpath, makeCtx+"_UN")
	case *ssa.Index:
		subpath = makeIndex(param.Index, makeCtx+"_IN") + subpath
		return makePath(param.X, scope, pos, subpath, makeCtx+"_IN")

	default:
		println("MP", makeCtx, reflect.TypeOf(param).String(), param.String())
	}
	return nil
}

func makeIndex(index ssa.Value, makeCtx string) string {
	switch index := index.(type) {
	case *ssa.Const:
		return "[" + index.Value.String() + "]"
	default:
		println("MI+", makeCtx, reflect.TypeOf(index).String())
		return ""
	}
}

type RecvPath string

func (r RecvPath) Path() string {
	return string(r)
}

func FunctionReceiver(fn *ssa.Function) RecvPath {
	return RecvPath(fn.Object().(*types.Func).Type().(*types.Signature).Recv().Name())
}

type GlobalPath string

func (r GlobalPath) Path() string {
	return string(r)
}

type ParamPath string

func (r ParamPath) Path() string {
	return string(r)
}

type FieldAddrPath string

func (r FieldAddrPath) Path() string {
	return string(r)
}

type IndexAddrPath string

func (r IndexAddrPath) Path() string {
	return string(r)
}
