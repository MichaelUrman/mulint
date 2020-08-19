package a

import "sync"

type composeMutex struct {
	mu sync.Mutex
}

func (c *composeMutex) Good1() { // want Good1:`c.mu:Locker`
	c.lock()
}

func (c *composeMutex) Good2() { // want Good2:`c.mu:Locker`
	c.mu.Lock()
	c.mu.Unlock()

	c.lock()
}

func (c *composeMutex) Good3() { // want Good3:`c.mu:LockUnlocker`
	c.mu.Lock()
	c.weird()
	c.mu.Unlock()
}

func (c *composeMutex) Good4() { // want Good4:`c.mu:LockUnlocker`
	c.iffy()
	c.iffy()
}

func (c *composeMutex) Bad1() { // want Bad1:`c.mu:Locker`
	c.mu.Lock()
	c.mu.Lock()
}

func (c *composeMutex) Bad2() { // want Bad2:`c.mu:Locker`
	c.mu.Lock()
	c.lock()
}

func (c *composeMutex) Bad3() { // want Bad3:`c.mu:Locker`
	c.lock()
	c.lock()
}

// iffy locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *composeMutex) iffy() { // want iffy:`c.mu:LockUnlocker`
	c.mu.Lock()
	c.mu.Unlock()
}

// lock locks but does not unlock.
func (c *composeMutex) lock() { // want lock:`c.mu:Locker`
	c.mu.Lock()
}

// unlock unlocks but does not lock.
func (c *composeMutex) unlock() { // want unlock:`c.mu:Unlocker`
	c.mu.Unlock()
}

// weird unlocks then locks. It's not really all that weird.
func (c *composeMutex) weird() { // want weird:`c.mu:UnlockLocker`
	c.mu.Unlock()
	c.mu.Lock()
}
