package a

type deferUnlock struct{}

func (*deferUnlock) Good1() { // want Good1:`a:pkgMu:"Ll"`
	pkgMu.Lock()
	defer pkgMu.Unlock()
}

func (*deferUnlock) Bad1() { // want Bad1:`a:pkgMu:"Ll"`
	pkgMu.Lock()
	defer pkgMu.Unlock()
	pkgMu.Lock() // want `Locks locked a:pkgMu`
}

func (*deferUnlock) Bad2() { // want Bad2:`a:pkgMu:"Ll"`
	pkgMu.Lock()
	defer pkgMu.Unlock() // want `Unlocks unlocked a:pkgMu`
	pkgMu.Lock()         // want `Locks locked a:pkgMu`
	defer pkgMu.Unlock()
}
