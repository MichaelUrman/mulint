package a

import "sync"

type multisMutex struct {
	a, b sync.Mutex
}

func (c *multisMutex) Good1ab() { // want Good1ab:`(c.[ab]:"L" ?){2}`
	c.locka()
	c.lockb()
}

func (c *multisMutex) Good2ab() { // want Good2ab:`(c.[ab]:"L" ?){2}`
	c.a.Lock()
	c.lockb()
	c.a.Unlock()

	c.locka()
}

func (c *multisMutex) Good2ba() { // want Good2ba:`(c.[ab]:"L" ?){2}`
	c.b.Lock()
	c.locka()
	c.b.Unlock()

	c.lockb()
}

func (c *multisMutex) Good3ab() { // want Good3ab:`(c.[ab]:"Ll" ?){2}`
	c.a.Lock()
	c.weirda()
	c.a.Unlock()
	c.b.Lock()
	c.weirdb()
	c.b.Unlock()
}

func (c *multisMutex) Good4ab() { // want Good4ab:`(c.[ab]:"Ll" ?){2}`
	c.iffya()
	c.iffyb()
}

func (c *multisMutex) Good4ba() { // want Good4ba:`(c.[ab]:"Ll" ?){2}`
	c.iffyb()
	c.iffya()
	c.iffyb()
	c.iffya()
}

func (c *multisMutex) Good5a() { // want Good5a:`c.a:"Ll"`
	a := c.a
	a.Lock()
	a.Unlock()
}

func (c *multisMutex) Good5axx() { // want Good5axx:`c.a:"Ll"`
	d := c
	b := d.a
	a := b
	a.Lock()
	a.Unlock()
}

func (c *multisMutex) Good5ab() { // want Good5ab:`(c.[ab]:"Ll" ?){2}`
	a := c.a
	b := c.b
	a.Lock()
	b.Lock()
	b.Unlock()
	a.Unlock()
}

func (c *multisMutex) Bad1a() { // want Bad1a:`c.a:"L"`
	c.a.Lock()
	c.a.Lock() // want `Locks locked c.a`
}

func (c *multisMutex) Bad1b() { // want Bad1b:`c.b:"L"`
	c.b.Lock()
	c.b.Lock() // want `Locks locked c.b`
}

func (c *multisMutex) Bad1ab() { // want Bad1ab:`(c.[ab]:"L" ?){2}`
	c.a.Lock()
	c.b.Lock()
	c.b.Lock() // want `Locks locked c.b`
	c.a.Lock() // want `Locks locked c.a`
}

func (c *multisMutex) Bad2a() { // want Bad2a:`c.a:"L"`
	c.a.Lock()
	c.locka() // want `Locks locked c.a`
}

func (c *multisMutex) Bad2b() { // want Bad2b:`c.b:"L"`
	b := c.b
	b.Lock()
	c.lockb() // want `Locks locked c.b`
}

func (c *multisMutex) Bad2ab() { // want Bad2ab:`(c.[ab]:"L" ?){2}`
	c.a.Lock()
	c.locka() // want `Locks locked c.a`
	c.lockb()
	c.b.Lock() // want `Locks locked c.b`
}

func (c *multisMutex) Bad3a() { // want Bad3a:`c.a:"L"`
	c.locka()
	c.locka() // want `Locks locked c.a`
}
func (c *multisMutex) Bad3b() { // want Bad3b:`c.b:"L"`
	c.lockb()
	c.lockb() // want `Locks locked c.b`
}
func (c *multisMutex) Bad3ab() { // want Bad3ab:`(c.[ab]:"L" ?){2}`
	c.locka()
	c.lockb()
	c.lockb() // want `Locks locked c.b`
	c.locka() // want `Locks locked c.a`
}

// iffya locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *multisMutex) iffya() { // want iffya:`c.a:"Ll"`
	c.a.Lock()
	c.a.Unlock()
}

// iffyb locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *multisMutex) iffyb() { // want iffyb:`c.b:"Ll"`
	c.b.Lock()
	c.b.Unlock()
}

// locka locks but does not unlock.
func (c *multisMutex) locka() { // want locka:`c.a:"L"`
	c.a.Lock()
}

// lockb locks but does not unlock.
func (c *multisMutex) lockb() { // want lockb:`c.b:"L"`
	c.b.Lock()
}

// unlocka unlocks but does not lock.
func (c *multisMutex) unlocka() { // want unlocka:`c.a:"l"`
	c.a.Unlock()
}

// unlockb unlocks but does not lock.
func (c *multisMutex) unlockb() { // want unlockb:`c.b:"l"`
	c.b.Unlock()
}

// weirda unlocks then locks. It's not really all that weird.
func (c *multisMutex) weirda() { // want weirda:`c.a:"lL"`
	c.a.Unlock()
	c.a.Lock()
}

// weirdb unlocks then locks. It's not really all that weird.
func (c *multisMutex) weirdb() { // want weirdb:`c.b:"lL"`
	c.b.Unlock()
	c.b.Lock()
}
