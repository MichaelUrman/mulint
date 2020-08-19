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

	for fn, info := range functions {
		for path, lock := range info.Locks {
			println(fn.String(), path.String(), lock.String())
		}
	}

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

type PathLocker map[Path]LockInfo

func (p PathLocker) String() string {
	sb := strings.Builder{}
	for path, locker := range p {
		if sb.Len() > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString(path.String())
		sb.WriteRune(':')
		sb.WriteString(locker.String())
	}
	return sb.String()
}

type LockInfo struct {
	Locks, Locked     bool // locks or leaves locked
	Unlocks, Unlocked bool // unlocks or leaves unlocked
}

func (li LockInfo) String() string {
	if li.Locks && li.Locked {
		return "Locker"
	}
	if li.Unlocks && li.Unlocked {
		return "Unlocker"
	}
	if li.Locks && li.Unlocked {
		return "LockUnlocker"
	}
	if li.Unlocks && li.Locked {
		return "UnlockLocker"
	}
	l := map[bool]byte{true: 'L', false: '_'}
	u := map[bool]byte{true: 'U', false: '_'}
	return string([]byte{
		l[li.Locks],
		u[li.Unlocks],
		u[li.Unlocked],
		l[li.Locked],
	})
}

func classify(pass *analysis.Pass) map[*types.Func]*FuncInfo {
	ssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	m := make(map[*types.Func]*FuncInfo, len(ssa.SrcFuncs))
	for _, fn := range ssa.SrcFuncs {
		obj, ok := fn.Object().(*types.Func)
		if !ok {
			continue // typically runtime.*
		}
		if obj.Type().(*types.Signature).Params().Len() == 0 {
			switch obj.Name() {
			case "Lock":
				m[obj] = &FuncInfo{
					Locks: PathLocker{
						makeRecv(fn, nil): {true, true, false, false},
					},
				}
				pass.ExportObjectFact(obj, m[obj])
				continue
			case "Unlock":
				m[obj] = &FuncInfo{
					Locks: PathLocker{
						makeRecv(fn, nil): {false, false, true, true},
					},
				}
				pass.ExportObjectFact(obj, m[obj])
				continue
			}
		}
		m[obj] = &FuncInfo{
			ssa: fn,
		}

	}

	for fn, fi := range m {
		classifyFunc(pass, m, fn, fi)
	}
	return m
}

func classifyFunc(pass *analysis.Pass, m map[*types.Func]*FuncInfo, fn *types.Func, fi *FuncInfo) {
	if fi.ssa == nil {
		return // already classified; skip
	}
	if fn == nil {
		return
	}

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
				classifyFunc(pass, m, call.Call.Method, ci)
				println("REC", fn.Name(), "classifyFunc", cfn.Name(), ci.Locks.String())
			}
			for path, next := range ci.Locks {
				print("CONVERT ", fn.FullName(), ": ", path.String())
				path := path.ToCaller(fi.ssa, call)
				if path == nil {
					println(" TO NIL")
					continue
				}
				println(" TO", path.String())
				li[path] = combineLockInfo(li[path], next)
			}
		}
		blockLocks[block] = li
	}

	// TODO: combine blocks properly
	for _, locks := range blockLocks {
		for path, lock := range locks {
			fi.Locks[path] = combineLockInfo(fi.Locks[path], lock)
		}
	}

	for path, lock := range fi.Locks {
		println("FUNC", path.String(), lock.String())
	}
	if len(fi.Locks) > 0 {
		pass.ExportObjectFact(fi.ssa.Object().(*types.Func), fi)
	}

	fi.ssa = nil
}

func combineLockInfo(prev, next LockInfo) LockInfo {
	prev.Locks = prev.Locks || (next.Locks && !prev.Unlocks)
	prev.Unlocks = prev.Unlocks || (next.Unlocks && !prev.Locks)
	prev.Unlocked = prev.Unlocked && !next.Locks || next.Unlocked
	prev.Locked = prev.Locked && !next.Unlocks || next.Locked
	return prev
}
