package relock

import (
	"go/types"
	"sort"
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

func (a LockInfo) Simplified() LockInfo {
	simplify := func(a, chars string) string {
		first := strings.IndexAny(a, chars)
		if first == -1 {
			return ""
		}
		last := strings.LastIndexAny(a, chars)
		if a[first] == a[last] {
			return a[first : first+1]
		}
		return a[first:first+1] + a[last:last+1]
	}

	srcs := strings.Split(strings.Trim(string(a), "|"), "|")
	for i, src := range srcs {
		srcs[i] = simplify(src, "Ll") + simplify(src, "Rr")
	}
	sort.Strings(srcs)
	i := 0
	for j := 1; j < len(srcs); j++ {
		if srcs[j] != srcs[i] {
			i++
			srcs[i] = srcs[j]
		}
	}
	srcs = srcs[:i+1]
	return LockInfo(strings.Join(srcs, "|"))
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
		sig := obj.Type().(*types.Signature)
		if sig.Params().Len() == 0 && sig.Results().Len() == 0 {
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

var msgs = map[rune]string{
	'L': "Locks locked %s",
	'R': "RLocks locked %s",
	'l': "Unlocks unlocked %s",
	'r': "RUnlocks unlocked %s",
}

func classifyFunc(pass *analysis.Pass, m map[*types.Func]*FuncInfo, fn *types.Func, fi *FuncInfo, depth int) {
	if fn == nil || fi.ssa == nil || len(fi.ssa.Blocks) == 0 {
		return
	}
	defer func() { fi.ssa = nil }()

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
		defers := make(map[Pather][]ssa.Instruction)
		deferred := make(PathLocker)
		li := make(PathLocker)
		for _, inst := range block.Instrs {
			switch inst := inst.(type) {
			case *ssa.Call:
				ci := classifyCall(pass, inst, inst.Call, m, depth)
				if ci == nil {
					continue
				}
				for path, next := range ci.Locks {
					path := CallerPath(path, fi.ssa, inst)
					if path == nil {
						continue
					}
					li[path] = combineLocks(pass, inst, nil, path, li[path], next)
				}
			case *ssa.Defer:
				ci := classifyCall(pass, inst, inst.Call, m, depth)
				if ci == nil {
					continue
				}
				for path, prev := range ci.Locks {
					path := CallerPath(path, fi.ssa, inst)
					if path == nil {
						continue
					}
					deferred[path] = prev + deferred[path]
					insts := make([]ssa.Instruction, len(prev))
					for i := range insts {
						insts[i] = inst
					}
					defers[path] = append(insts, defers[path]...)
				}
			case *ssa.RunDefers:
				for path, next := range deferred {
					li[path] = combineLocks(pass, inst, defers[path], path, li[path], next)
				}
			default:
				//println("INSTR", reflect.TypeOf(inst).String(), inst.String())
			}
		}
		blockLocks[block] = li
	}

	if len(blockLocks) == 0 {
		return
	}

	// TODO: combine blocks properly
	type transition struct{ From, To *ssa.BasicBlock }
	type graph struct {
		Block *ssa.BasicBlock
		Locks PathLocker
	}

	mergeBlockLocks := func(p, n int, at, next PathLocker) PathLocker {
		elts := len(at)
		for elt := range next {
			if _, ok := at[elt]; !ok {
				elts++
			}
		}
		m := make(PathLocker, elts)
		for path, locks := range at {
			m[path] = locks
		}
		for path, locks := range next {
			m[path] += locks
		}
		return m
	}

	seen := make(map[transition]int)
	queue := []graph{{Block: &ssa.BasicBlock{Succs: []*ssa.BasicBlock{fi.ssa.Blocks[0]}}}}
	for len(queue) > 0 {
		at := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		queued := false
		for _, next := range at.Block.Succs {
			t := transition{at.Block, next}
			count := seen[t]
			if count < 2 {
				seen[t]++
				queued = true
				queue = append(queue, graph{
					Block: next,
					Locks: mergeBlockLocks(at.Block.Index, next.Index, at.Locks, blockLocks[next]),
				})
			}
		}
		if !queued {
			for path, locks := range at.Locks {
				fi.Locks[path] += "|" + locks
			}
		}
	}

	for path, lock := range fi.Locks {
		fi.Locks[path] = lock.Simplified()
	}
	if len(fi.Locks) > 0 {
		pass.ExportObjectFact(fi.ssa.Object().(*types.Func), fi)
	}
}

func classifyCall(pass *analysis.Pass, inst ssa.Instruction, call ssa.CallCommon, m map[*types.Func]*FuncInfo, depth int) *FuncInfo {
	cf, ok := call.Value.(*ssa.Function)
	if !ok || cf == nil {
		return nil
	}
	obj := cf.Object()
	if obj == nil {
		return nil
	}
	ci := new(FuncInfo)

	if !pass.ImportObjectFact(obj, ci) {
		cfn := obj.(*types.Func)
		var ok bool
		ci, ok = m[cfn]
		if !ok {
			return nil // assume uninteresting
		}
		if cfn == nil {
			println("NILFN", inst.String())
			return nil
		}
		if ci.ssa != nil && ci.Locks != nil {
			return nil // assume uninteresting
		}
		classifyFunc(pass, m, cfn, ci, depth+1)
	}
	return ci
}

func combineLocks(pass *analysis.Pass, inst ssa.Instruction, insts []ssa.Instruction, path Pather, prev, next LockInfo) LockInfo {
	states := make(map[rune]rune, 2)
	for i, x := range prev + next {
		track := []rune(strings.ToUpper(string(x)))[0]
		old, ok := states[track]
		states[track] = x
		if ok && old == x && i >= len(prev) {
			if insts != nil {
				inst = insts[i-len(prev)]
			}
			pass.Reportf(inst.Pos(), msgs[x], path.Path())
		}
	}
	return prev + next
}
