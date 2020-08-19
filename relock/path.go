package relock

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type Path interface {
	aPath()
	String() string
	ToCaller(caller *ssa.Function, callee *ssa.Call) Path
}

type Recv struct {
	path []Path
	fn   *ssa.Function
}

var knownRecv = make(map[*ssa.Function][]*Recv)

func makeRecv(fn *ssa.Function, path []Path) *Recv {
	for _, recv := range knownRecv[fn] {
		if pathMatch(recv.path, path) {
			return recv
		}
	}
	recv := &Recv{path, fn}
	knownRecv[fn] = append(knownRecv[fn])
	return recv
}

var _ Path = (*Recv)(nil)

func (*Recv) aPath() {}
func (r *Recv) String() string {
	name := "?"
	if r.fn != nil {
		name = r.fn.Params[0].Name()
	}
	return fmt.Sprintf("RECV(%p):%s%s", r, name, pathString(r.path))
}

func (r *Recv) ToCaller(caller *ssa.Function, callee *ssa.Call) Path {
	ops := callee.Operands(nil)
	if len(ops) < 2 || ops[1] == nil {
		return nil
	}

	switch param := (*ops[1]).(type) {
	case *ssa.Global:
		return makeGlobal(param, r.path)
	case *ssa.Parameter:
		if caller.Signature.Recv() != nil && param == caller.Params[0] {
			path := r.path
			if len(r.path) == 0 {
				path = nil
			}
			fmt.Printf("<r2r: %p %p %s>", caller, path, pathString(path))
			return makeRecv(caller, path)
		}
		return makeParam(param, r.path)
	case *ssa.FieldAddr:
		switch oparam := param.X.(type) {
		case *ssa.Parameter:
			if caller.Signature.Recv() != nil && oparam == caller.Params[0] {
				return makeRecv(caller, []Path{makeFieldAddr(param, r.path)})
			}
		default:
			println("FA", reflect.TypeOf(param.X).String(), param.String())
		}
		return makeFieldAddr(param, r.path)
	default:
		println("RECV", reflect.TypeOf(param).String(), param.String())
	}

	return nil
}

type Global struct {
	glob *ssa.Global
	path []Path
}

var knownGlobal = make(map[*ssa.Global][]*Global)

func makeGlobal(glob *ssa.Global, path []Path) *Global {
	for _, g := range knownGlobal[glob] {
		if pathMatch(g.path, path) {
			return g
		}
	}
	g := &Global{glob, path}
	knownGlobal[glob] = append(knownGlobal[glob], g)
	return g
}

var _ Path = (*Global)(nil)

func (*Global) aPath()         {}
func (*Global) String() string { return "Global" }
func (*Global) ToCaller(caller *ssa.Function, callee *ssa.Call) Path {
	return nil
}

type FieldAddr struct {
	field *ssa.FieldAddr
	path  []Path
}

var knownFieldAddr = make(map[*ssa.FieldAddr][]*FieldAddr)

func makeFieldAddr(field *ssa.FieldAddr, path []Path) *FieldAddr {
	for _, fa := range knownFieldAddr[field] {
		if pathMatch(fa.path, path) {
			return fa
		}
	}
	fa := &FieldAddr{field, path}
	knownFieldAddr[field] = append(knownFieldAddr[field], fa)
	return fa
}

var _ Path = (*FieldAddr)(nil)

func (*FieldAddr) aPath() {}
func (f *FieldAddr) String() string {
	return ".(FLD:)" + f.field.X.
		Type().Underlying().(*types.Pointer).
		Elem().Underlying().(*types.Struct).
		Field(f.field.Field).Name() + pathString(f.path)
}
func (*FieldAddr) ToCaller(caller *ssa.Function, callee *ssa.Call) Path {
	return nil
}

type Param struct {
	param *ssa.Parameter
	path  []Path
}

var knownParams = make(map[*ssa.Parameter][]*Param)

func makeParam(param *ssa.Parameter, path []Path) *Param {
	for _, p := range knownParams[param] {
		if pathMatch(p.path, path) {
			return p
		}
	}
	p := &Param{param, path}
	knownParams[param] = append(knownParams[param], p)
	return p
}

var _ Path = (*Param)(nil)

func (*Param) aPath() {}
func (p *Param) String() string {
	return "PARM" + p.param.Name() + pathString(p.path)
}
func (*Param) ToCaller(caller *ssa.Function, callee *ssa.Call) Path {
	return nil
}

func pathString(p []Path) string {
	sb := &strings.Builder{}
	for _, p := range p {
		switch p := p.(type) {
		case *FieldAddr:
			sb.WriteRune('.')
			sb.WriteString(p.field.X.
				Type().Underlying().(*types.Pointer).
				Elem().Underlying().(*types.Struct).
				Field(p.field.Field).Name())
		}
	}
	return sb.String()
}

func pathMatch(a, b []Path) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
