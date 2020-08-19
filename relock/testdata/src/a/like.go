package a

type likeMutex struct {
}

func (c *likeMutex) Lock()   {} // want Lock:"c:Locker"
func (c *likeMutex) Unlock() {} // want Unlock:"c:Unlocker"

func (c *likeMutex) Good1() { // want Good1:`c:Locker`
	c.lock()
}

func (c *likeMutex) Good2() { // want Good2:`c:Locker`
	c.Lock()
	c.Unlock()

	c.lock()
}

func (c *likeMutex) Good3() { // want Good3:`c:LockUnlocker`
	c.Lock()
	c.weird()
	c.Unlock()
}

func (c *likeMutex) Good4() { // want Good4:`c:LockUnlocker`
	c.iffy()
	c.iffy()
}

func (c *likeMutex) Bad1() { // want Bad1:`c:Locker`
	c.Lock()
	c.Lock()
}

func (c *likeMutex) Bad2() { // want Bad2:`c:Locker`
	c.Lock()
	c.lock()
}

func (c *likeMutex) Bad3() { // want Bad3:`c:Locker`
	c.lock()
	c.lock()
}

// iffy locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *likeMutex) iffy() { // want iffy:`c:LockUnlocker`
	c.Lock()
	c.Unlock()
}

// lock locks but does not unlock.
func (c *likeMutex) lock() { // want lock:`c:Locker`
	c.Lock()
}

// unlock unlocks but does not lock.
func (c *likeMutex) unlock() { // want unlock:`c:Unlocker`
	c.Unlock()
}

// weird unlocks then locks. It's not really all that weird.
func (c *likeMutex) weird() { // want weird:`c:UnlockLocker`
	c.Unlock()
	c.Lock()
}
