package a

import "sync"

var pkgMu sync.Mutex

type pkgMutex struct{}

func (c *pkgMutex) Good1() { // want Good1:`a:pkgMu:"L"`
	c.lock()
}

func (c *pkgMutex) Good2() { // want Good2:`a:pkgMu:"L"`
	pkgMu.Lock()
	pkgMu.Unlock()

	c.lock()
}

func (c *pkgMutex) Good3() { // want Good3:`a:pkgMu:"Ll"`
	pkgMu.Lock()
	c.weird()
	pkgMu.Unlock()
}

func (c *pkgMutex) Good4() { // want Good4:`a:pkgMu:"Ll"`
	c.iffy()
	c.iffy()
}

func (c *pkgMutex) Bad1() { // want Bad1:`a:pkgMu:"L"`
	pkgMu.Lock()
	pkgMu.Lock() // want `Locks locked a:pkgMu`
}

func (c *pkgMutex) Bad2() { // want Bad2:`a:pkgMu:"L"`
	pkgMu.Lock()
	c.lock() // want `Locks locked a:pkgMu`
}

func (c *pkgMutex) Bad3() { // want Bad3:`a:pkgMu:"L"`
	c.lock()
	c.lock() // want `Locks locked a:pkgMu`
}

// iffy locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *pkgMutex) iffy() { // want iffy:`a:pkgMu:"Ll"`
	pkgMu.Lock()
	pkgMu.Unlock()
}

// lock locks but does not unlock.
func (c *pkgMutex) lock() { // want lock:`a:pkgMu:"L"`
	pkgMu.Lock()
}

// unlock unlocks but does not lock.
func (c *pkgMutex) unlock() { // want unlock:`a:pkgMu:"l"`
	pkgMu.Unlock()
}

// weird unlocks then locks. It's not really all that weird.
func (c *pkgMutex) weird() { // want weird:`a:pkgMu:"lL"`
	pkgMu.Unlock()
	pkgMu.Lock()
}
