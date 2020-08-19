package relock

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

var Analyzer = &analysis.Analyzer{
	Name:      "relock",
	Doc:       "reports likely deadlocks",
	Run:       run,
	FactTypes: []analysis.Fact{(*FuncInfo)(nil)},
	Requires:  []*analysis.Analyzer{buildssa.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// for id, obj := range pass.TypesInfo.Defs {
	// 	fn, ok := obj.(*types.Func)
	// 	if !ok {
	// 		continue
	// 	}

	// 	println(id.Name, "is a func", fn.Type().String())
	// }

	functions := classify(pass)
	_ = functions

	// for fn, info := range functions {
	// 	for path, lock := range info.Locks {
	// 		println(fn.String(), path.Path(), lock.String())
	// 	}
	// }

	return nil, nil
}

type FuncInfo struct {
	Locks PathLocker    // non-nil during processing, and if relevant
	ssa   *ssa.Function // nil once final
}

func (*FuncInfo) AFact() {}

func (fi FuncInfo) String() string {
	return fi.Locks.String()
}

type PathLocker map[Pather]LockInfo

func (p PathLocker) String() string {
	sb := strings.Builder{}
	for path, locker := range p {
		if sb.Len() > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString(path.Path())
		sb.WriteRune(':')
		sb.WriteString(locker.String())
	}
	return sb.String()
}

type LockInfo string

func (li LockInfo) String() string { return `"` + string(li) + `"` }

var thenSimplify = strings.NewReplacer(
	"LL", "L", "RR", "R",
	"ll", "l", "rr", "r",
	"LlL", "L", "RrR", "R",
	"lLl", "l", "rRr", "r",
).Replace

func (a LockInfo) Then(b LockInfo) LockInfo {
	c := string(a + b)
	for {
		d := thenSimplify(c)
		if d == c {
			break
		}
		c = d
	}
	return LockInfo(c)
}

var passCt = 0

func classify(pass *analysis.Pass) map[*types.Func]*FuncInfo {
	passCt++
	ssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	m := make(map[*types.Func]*FuncInfo, len(ssa.SrcFuncs))
	for _, fn := range ssa.SrcFuncs {
		obj, ok := fn.Object().(*types.Func)
		if !ok {
			continue // typically runtime.*
		}
		if obj.Type().(*types.Signature).Params().Len() == 0 {
			var li LockInfo
			switch obj.Name() {
			case "Lock":
				li = "L"
			case "RLock":
				li = "R"
			case "Unlock":
				li = "l"
			case "RUnlock":
				li = "r"
			}
			if li != "" {
				fi := &FuncInfo{
					Locks: PathLocker{
						FunctionReceiver(fn): li,
					},
				}
				pass.ExportObjectFact(obj, fi)
				m[obj] = fi
				continue
			}
		}
		m[obj] = &FuncInfo{
			ssa: fn,
		}
	}

	for fn, fi := range m {
		classifyFunc(pass, m, fn, fi, 0)
	}
	return m
}

func classifyFunc(pass *analysis.Pass, m map[*types.Func]*FuncInfo, fn *types.Func, fi *FuncInfo, depth int) {
	if fn == nil {
		return
	}
	if fi.ssa == nil {
		return // already classified; skip
	}

	// if fn.Name() == "Good5" {
	// 	buf := &bytes.Buffer{}
	// 	ssa.WriteFunction(buf, fi.ssa)
	// 	print(string(buf.Bytes()))
	// }

	fi.Locks = make(PathLocker)

	type callee struct {
		caller     *ssa.Function
		callee     *ssa.Call
		calledInfo *FuncInfo
	}
	blockLocks := make(map[*ssa.BasicBlock]PathLocker)

	// look for calls to classified functions
	for _, block := range fi.ssa.Blocks {
		li := make(PathLocker)
		for _, inst := range block.Instrs {
			call, ok := inst.(*ssa.Call)
			if !ok {
				continue
			}

			cf, ok := call.Call.Value.(*ssa.Function)
			if !ok || cf == nil {
				continue
			}
			obj := cf.Object()
			if obj == nil {
				continue
			}
			ci := new(FuncInfo)

			if !pass.ImportObjectFact(obj, ci) {
				cfn := obj.(*types.Func)
				var ok bool
				ci, ok = m[cfn]
				if !ok {
					continue // assume uninteresting
				}
				if cfn == nil {
					println("NILFN", call.String())
					continue
				}
				if ci.ssa != nil && ci.Locks != nil {
					continue // assume uninteresting
				}
				classifyFunc(pass, m, cfn, ci, depth+1)
			}
			for path, next := range ci.Locks {
				path := CallerPath(path, fi.ssa, call)
				if path == nil {
					continue
				}
				//pass.Reportf(call.Pos(), "%s:%s:%s", cf.Name(), path, next)
				li[path] = li[path].Then(next)
			}
		}
		blockLocks[block] = li
	}

	// TODO: combine blocks properly
	for _, locks := range blockLocks {
		for path, lock := range locks {
			fi.Locks[path] = fi.Locks[path].Then(lock)
		}
	}

	if len(fi.Locks) > 0 {
		pass.ExportObjectFact(fi.ssa.Object().(*types.Func), fi)
	}

	fi.ssa = nil
}
