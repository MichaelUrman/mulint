package a

// falseLockA has Lock and Unlock methods that don't match the Lock() and Unlock() interface
type falseLockA struct{}

func (f *falseLockA) Lock(a bool)   {}
func (f *falseLockA) Unlock(a bool) {}

func (f *falseLockA) Good() {
	f.Lock(true)
	f.Lock(false)
	f.Unlock(true)
	f.Unlock(false)
}

// falseLockB has Lock and Unlock methods that don't match the Lock() and Unlock() interface
type falseLockB struct{}

func (f *falseLockB) Lock() bool   { return true }
func (f *falseLockB) Unlock() bool { return false }

func (f *falseLockB) Good() {
	f.Lock()
	f.Lock()
	f.Unlock()
	f.Unlock()
}

// falseLockC has Lock and Unlock methods that don't match the Lock() and Unlock() interface, but call normal ones.
type falseLockC struct{}

func (f *falseLockC) Lock(a bool)   { pkgMu.Lock() }   // want Lock:`a:pkgMu:"L"`
func (f *falseLockC) Unlock(a bool) { pkgMu.Unlock() } // want Unlock:`a:pkgMu:"l"`

func (f *falseLockC) Bad() { // want Bad:`a:pkgMu:"Ll"`
	f.Lock(true)
	f.Lock(false) // want `Locks locked a:pkgMu`
	f.Unlock(true)
	f.Unlock(false) // want `Unlocks unlocked a:pkgMu`
}
