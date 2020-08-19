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

		switch param := (*ops[1]).(type) {
		case *ssa.Global:
			return GlobalPath(param.Object().Pkg().Path() + ":" + param.Object().Name() + subpath)
		case *ssa.Parameter:
			if caller.Signature.Recv() != nil && param == caller.Params[0] {
				return path
			}
			return ParamPath(param.Name() + subpath)
		case *ssa.FieldAddr:
			subpath = "." + param.X.
				Type().Underlying().(*types.Pointer).
				Elem().Underlying().(*types.Struct).
				Field(param.Field).Name() + subpath

			switch oparam := param.X.(type) {
			case *ssa.Parameter:
				if caller.Signature.Recv() != nil && oparam == caller.Params[0] {
					return RecvPath(oparam.Name() + subpath)
				}
			default:
				println("FA", reflect.TypeOf(param.X).String(), param.String())
				return FieldAddrPath(oparam.Name() + subpath)
			}
		case *ssa.IndexAddr:
			var subpath string
			switch index := param.Index.(type) {
			case *ssa.Const:
				subpath = "[" + index.Value.String() + "]"
			default:
				println("IA INDEX", reflect.TypeOf(index).String())
				return nil
			}

			switch indexed := param.X.(type) {
			case *ssa.FieldAddr:
				subpath = "." + indexed.X.
					Type().Underlying().(*types.Pointer).
					Elem().Underlying().(*types.Struct).
					Field(indexed.Field).Name() + subpath

				switch oparam := indexed.X.(type) {
				case *ssa.Parameter:
					if caller.Signature.Recv() != nil && oparam == caller.Params[0] {
						return RecvPath(oparam.Name() + subpath)
					}
				default:
					println("IA FA", reflect.TypeOf(param.X).String(), param.String())
					return FieldAddrPath(oparam.Name() + subpath)
				}
				return FieldAddrPath(indexed.Name() + subpath)
			default:
				println("IA", reflect.TypeOf(param.X).String(), param.String())
				return IndexAddrPath(indexed.Name() + subpath)
			}
		default:
			println("RECV", reflect.TypeOf(param).String(), param.String())
		}
	default:
		println("PATHER", reflect.TypeOf(path).String())
	}
	return nil
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
