package a

import "sync"

type embedMutex struct {
	sync.Mutex
}

func (c *embedMutex) Good1() { // want Good1:`c.Mutex:"L"`
	c.lock()
}

func (c *embedMutex) Good2() { // want Good2:`c.Mutex:"L"`
	c.Lock()
	c.Unlock()

	c.lock()
}

func (c *embedMutex) Good3() { // want Good3:`c.Mutex:"Ll"`
	c.Lock()
	c.weird()
	c.Unlock()
}

func (c *embedMutex) Good4() { // want Good4:`c.Mutex:"Ll"`
	c.iffy()
	c.iffy()
}

func (c *embedMutex) Bad1() { // want Bad1:`c.Mutex:"L"`
	c.Lock()
	c.Lock() // want `Locks locked c.Mutex`
}

func (c *embedMutex) Bad2() { // want Bad2:`c.Mutex:"L"`
	c.Lock()
	c.lock() // want `Locks locked c.Mutex`
}

func (c *embedMutex) Bad3() { // want Bad3:`c.Mutex:"L"`
	c.lock()
	c.lock() // want `Locks locked c.Mutex`
}

// iffy locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *embedMutex) iffy() { // want iffy:`c.Mutex:"Ll"`
	c.Lock()
	c.Unlock()
}

// lock locks but does not unlock.
func (c *embedMutex) lock() { // want lock:`c.Mutex:"L"`
	c.Lock()
}

// unlock unlocks but does not lock.
func (c *embedMutex) unlock() { // want unlock:`c.Mutex:"l"`
	c.Unlock()
}

// weird unlocks then locks. It's not really all that weird.
func (c *embedMutex) weird() { // want weird:`c.Mutex:"lL"`
	c.Unlock()
	c.Lock()
}
