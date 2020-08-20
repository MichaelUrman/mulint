package a

import "sync"

type multirMutex struct {
	mus [2]sync.Mutex
}

func (c *multirMutex) Good1ab() { // want Good1ab:`(c.mus.[01].:"L" ?){2}`
	c.locka()
	c.lockb()
}

func (c *multirMutex) Good2ab() { // want Good2ab:`(c.mus.[01].:"L" ?){2}`
	c.mus[0].Lock()
	c.lockb()
	c.mus[0].Unlock()

	c.locka()
}

func (c *multirMutex) Good2ba() { // want Good2ba:`(c.mus.[01].:"L" ?){2}`
	c.mus[1].Lock()
	c.locka()
	c.mus[1].Unlock()

	c.lockb()
}

func (c *multirMutex) Good3ab() { // want Good3ab:`(c.mus.[01].:"Ll" ?){2}`
	c.mus[0].Lock()
	c.weirda()
	c.mus[0].Unlock()
	c.mus[1].Lock()
	c.weirdb()
	c.mus[1].Unlock()
}

func (c *multirMutex) Good4ab() { // want Good4ab:`(c.mus.[01].:"Ll" ?){2}`
	c.iffya()
	c.iffyb()
}

func (c *multirMutex) Good4ba() { // want Good4ba:`(c.mus.[01].:"Ll" ?){2}`
	c.iffyb()
	c.iffya()
	c.iffyb()
	c.iffya()
}

func (c *multirMutex) Good5a() { // want Good5a:`c.mus.[0].:"Ll"`
	a := c.mus[0]
	a.Lock()
	a.Unlock()
}

func (c *multirMutex) Good5ab() { // want Good5ab:`(c.mus.[01].:"Ll" ?){2}`
	a := c.mus[0]
	b := c.mus[1]
	a.Lock()
	b.Lock()
	b.Unlock()
	a.Unlock()
}

func (c *multirMutex) Bad1a() { // want Bad1a:`c.mus.0.:"L"`
	c.mus[0].Lock()
	c.mus[0].Lock() // want `Locks locked c.mus.0.`
}

func (c *multirMutex) Bad1b() { // want Bad1b:`c.mus.1.:"L"`
	c.mus[1].Lock()
	c.mus[1].Lock() // want `Locks locked c.mus.1.`
}

func (c *multirMutex) Bad1ab() { // want Bad1ab:`(c.mus.[01].:"L" ?){2}`
	c.mus[0].Lock()
	c.mus[1].Lock()
	c.mus[1].Lock() // want `Locks locked c.mus.1.`
	c.mus[0].Lock() // want `Locks locked c.mus.0.`
}

func (c *multirMutex) Bad2a() { // want Bad2a:`c.mus.0.:"L"`
	c.mus[0].Lock()
	c.locka() // want `Locks locked c.mus.0.`
}

func (c *multirMutex) Bad2b() { // want Bad2b:`c.mus.1.:"L"`
	b := c.mus[1]
	b.Lock()
	c.lockb() // want `Locks locked c.mus.1.`
}

func (c *multirMutex) Bad2ab() { // want Bad2ab:`(c.mus.[01].:"L" ?){2}`
	c.mus[0].Lock()
	c.locka() // want `Locks locked c.mus.0.`
	c.lockb()
	c.mus[1].Lock() // want `Locks locked c.mus.1.`
}

func (c *multirMutex) Bad3a() { // want Bad3a:`c.mus.0.:"L"`
	c.locka()
	c.locka() // want `Locks locked c.mus.0.`
}
func (c *multirMutex) Bad3b() { // want Bad3b:`c.mus.1.:"L"`
	c.lockb()
	c.lockb() // want `Locks locked c.mus.1.`
}
func (c *multirMutex) Bad3ab() { // want Bad3ab:`(c.mus.[01].:"L" ?){2}`
	c.locka()
	c.lockb()
	c.lockb() // want `Locks locked c.mus.1.`
	c.locka() // want `Locks locked c.mus.0.`
}

// iffya locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *multirMutex) iffya() { // want iffya:`c.mus.0.:"Ll"`
	c.mus[0].Lock()
	c.mus[0].Unlock()
}

// iffyb locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *multirMutex) iffyb() { // want iffyb:`c.mus.1.:"Ll"`
	c.mus[1].Lock()
	c.mus[1].Unlock()
}

// locka locks but does not unlock.
func (c *multirMutex) locka() { // want locka:`c.mus.0.:"L"`
	c.mus[0].Lock()
}

// lockb locks but does not unlock.
func (c *multirMutex) lockb() { // want lockb:`c.mus.1.:"L"`
	c.mus[1].Lock()
}

// unlocka unlocks but does not lock.
func (c *multirMutex) unlocka() { // want unlocka:`c.mus.0.:"l"`
	c.mus[0].Unlock()
}

// unlockb unlocks but does not lock.
func (c *multirMutex) unlockb() { // want unlockb:`c.mus.1.:"l"`
	c.mus[1].Unlock()
}

// weirda unlocks then locks. It's not really all that weird.
func (c *multirMutex) weirda() { // want weirda:`c.mus.0.:"lL"`
	c.mus[0].Unlock()
	c.mus[0].Lock()
}

// weirdb unlocks then locks. It's not really all that weird.
func (c *multirMutex) weirdb() { // want weirdb:`c.mus.1.:"lL"`
	c.mus[1].Unlock()
	c.mus[1].Lock()
}
