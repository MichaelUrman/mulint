package a

type likeMutex struct {
}

func (c *likeMutex) Lock()   {} // want Lock:`c:"L"`
func (c *likeMutex) Unlock() {} // want Unlock:`c:"l"`

func (c *likeMutex) Good1() { // want Good1:`c:"L"`
	c.lock()
}

func (c *likeMutex) Good2() { // want Good2:`c:"L"`
	c.Lock()
	c.Unlock()

	c.lock()
}

func (c *likeMutex) Good3() { // want Good3:`c:"Ll"`
	c.Lock()
	c.weird()
	c.Unlock()
}

func (c *likeMutex) Good4() { // want Good4:`c:"Ll"`
	c.iffy()
	c.iffy()
}

func (c *likeMutex) Bad1() { // want Bad1:`c:"L"`
	c.Lock()
	c.Lock()
}

func (c *likeMutex) Bad2() { // want Bad2:`c:"L"`
	c.Lock()
	c.lock()
}

func (c *likeMutex) Bad3() { // want Bad3:`c:"L"`
	c.lock()
	c.lock()
}

// iffy locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *likeMutex) iffy() { // want iffy:`c:"Ll"`
	c.Lock()
	c.Unlock()
}

// lock locks but does not unlock.
func (c *likeMutex) lock() { // want lock:`c:"L"`
	c.Lock()
}

// unlock unlocks but does not lock.
func (c *likeMutex) unlock() { // want unlock:`c:"l"`
	c.Unlock()
}

// weird unlocks then locks. It's not really all that weird.
func (c *likeMutex) weird() { // want weird:`c:"lL"`
	c.Unlock()
	c.Lock()
}
