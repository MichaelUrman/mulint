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
		// println("NOPE, nil")
		return
	}
	if fi.ssa == nil {
		//println("NOPE, passe")
		return // already classified; skip
	}
	// fmt.Printf("CLASSIFY: %*s%s (%s) %t %d %d\n", 2*depth, "", fn.Name(), pass.Fset.Position(fn.Pos()).Filename, fi.ssa == nil, depth, passCt)

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
			//fmt.Printf("CLASSIFY-CON: %*s%s (%s) %d %d\n", 2*depth, "", obj.Name(), pass.Fset.Position(fn.Pos()).Filename, depth, passCt)

			if !pass.ImportObjectFact(obj, ci) {
				cfn := obj.(*types.Func)
				var ok bool
				ci, ok = m[cfn]
				if !ok {
					// println("NOT TRACKED", call.String())
					continue // assume uninteresting
				}
				if cfn == nil {
					println("NILFN", call.String())
					continue
				}
				if ci.ssa != nil && ci.Locks != nil {
					// println("RECURSE", cfn.String(), "to", call.String())
					continue // assume uninteresting
				}
				//println("ENTER", fn.Name(), "classifyFunc", cfn.Name(), depth+1)
				classifyFunc(pass, m, cfn, ci, depth+1)
				//println("EXIT", fn.Name(), "classifyFunc", cfn.Name(), ci.Locks.String())
			} else {
				//println("IMP", fn.Name(), "classifyFunc", obj.Name(), ci.Locks.String())
			}
			for path, next := range ci.Locks {
				// print("CONVERT ", fn.FullName(), ": ", path.Path())
				path := CallerPath(path, fi.ssa, call)
				if path == nil {
					// println(" TO NIL")
					continue
				}
				// println(" TO", path.Path())
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

	// for path, lock := range fi.Locks {
	// 	println("FUNC", path.Path(), lock.String())
	// }
	if len(fi.Locks) > 0 {
		pass.ExportObjectFact(fi.ssa.Object().(*types.Func), fi)
	}

	fi.ssa = nil
}
