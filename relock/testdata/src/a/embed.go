package a

import "sync"

type embedMutex struct {
	sync.Mutex
}

func (c *embedMutex) Good1() { // want Good1:`c.Mutex:Locker`
	c.lock()
}

func (c *embedMutex) Good2() { // want Good2:`c.Mutex:Locker`
	c.Lock()
	c.Unlock()

	c.lock()
}

func (c *embedMutex) Good3() { // want Good3:`c.Mutex:LockUnlocker`
	c.Lock()
	c.weird()
	c.Unlock()
}

func (c *embedMutex) Good4() { // want Good4:`c.Mutex:LockUnlocker`
	c.iffy()
	c.iffy()
}

func (c *embedMutex) Bad1() { // want Bad1:`c.Mutex:Locker`
	c.Lock()
	c.Lock()
}

func (c *embedMutex) Bad2() { // want Bad2:`c.Mutex:Locker`
	c.Lock()
	c.lock()
}

func (c *embedMutex) Bad3() { // want Bad3:`c.Mutex:Locker`
	c.lock()
	c.lock()
}

// iffy locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *embedMutex) iffy() { // want iffy:`c:LockUnlocker`
	c.Lock()
	c.Unlock()
}

// lock locks but does not unlock.
func (c *embedMutex) lock() { // want lock:`c:Locker`
	c.Lock()
}

// unlock unlocks but does not lock.
func (c *embedMutex) unlock() { // want unlock:`c:Unlocker`
	c.Unlock()
}

// weird unlocks then locks. It's not really all that weird.
func (c *embedMutex) weird() { // want weird:`c:UnlockLocker`
	c.Unlock()
	c.Lock()
}
