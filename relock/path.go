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

func CallerPath(path Pather, caller *ssa.Function, callee ssa.Instruction) Pather {
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
		var lastBefore ssa.Value
		var lastBlock *ssa.BasicBlock
		for _, ref := range *param.Referrers() {
			// ref block should be between last and pos (inclusive)
			blockOK := ref.Block().Dominates(pos.Block())
			if lastBlock != nil {
				blockOK = blockOK && lastBlock.Dominates(ref.Block())
			}
			// ref instr should be between last and pos (exclusive)
			instrOK := ref.Pos() < pos.Pos() || ref.Block() != pos.Block()
			if lastBlock != nil {
				if lastBlock != ref.Block() {
					instrOK = instrOK && lastBlock.Dominates(ref.Block())
				} else {
					instrOK = instrOK && lastBefore.Pos() < ref.Pos()
				}
			}
			if blockOK && instrOK {
				switch ref := ref.(type) {
				case *ssa.Call: // ignore
				case *ssa.UnOp: // ignore
				case *ssa.Store:
					lastBefore = ref.Val
					lastBlock = ref.Block()
				default:
					println("ALLOC REF", reflect.TypeOf(ref).String(), ref.String())
				}
			}
		}
		if lastBefore != nil {
			return makePath(lastBefore, scope, pos, subpath, makeCtx+"_ALLOC")
		}

	case *ssa.IndexAddr:
		subpath = makeIndex(param.Index, makeCtx+"_IA") + subpath
		return makePath(param.X, scope, pos, subpath, makeCtx+"_IA")

	case *ssa.UnOp:
		return makePath(param.X, scope, pos, subpath, makeCtx+"_UN")

	case *ssa.Index:
		subpath = makeIndex(param.Index, makeCtx+"_IN") + subpath
		return makePath(param.X, scope, pos, subpath, makeCtx+"_IN")

	default:
		println(scope.String(), "MP", makeCtx, reflect.TypeOf(param).String(), param.String())
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
