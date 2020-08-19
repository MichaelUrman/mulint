package a

import "sync"

type rwMutex struct {
	mu sync.RWMutex
}

func (c *rwMutex) Good1() { // want Good1:`c.mu:RLocker`
	c.lock()
}

func (c *rwMutex) Good2() { // want Good2:`c.mu:RLocker`
	c.mu.RLock()
	c.mu.RUnlock()

	c.lock()
}

func (c *rwMutex) Good3() { // want Good3:`c.mu:RLockRUnlocker`
	c.mu.RLock()
	c.weird()
	c.mu.RUnlock()
}

func (c *rwMutex) Good4() { // want Good4:`c.mu:RLockRUnlocker`
	c.iffy()
	c.iffy()
}

func (c *rwMutex) Bad1() { // want Bad1:`c.mu:RLocker`
	c.mu.RLock()
	c.mu.RLock()
}

func (c *rwMutex) Bad2() { // want Bad2:`c.mu:RLocker`
	c.mu.RLock()
	c.lock()
}

func (c *rwMutex) Bad3() { // want Bad3:`c.mu:RLocker`
	c.lock()
	c.lock()
}

// iffy locks and unlocks. It's called iffy due to its author's preference to
// lock only in exported methods.
func (c *rwMutex) iffy() { // want iffy:`c.mu@r:RLockRUnlocker`
	c.mu.RLock()
	c.mu.RUnlock()
}

// lock locks but does not unlock.
func (c *rwMutex) lock() { // want lock:`c.mu@r:RLocker`
	c.mu.RLock()
}

// unlock unlocks but does not lock.
func (c *rwMutex) unlock() { // want unlock:`c.mu@r:RUnlocker`
	c.mu.RUnlock()
}

// weird unlocks then locks. It's not really all that weird.
func (c *rwMutex) weird() { // want weird:`c.mu@r:RUnlockRLocker`
	c.mu.RUnlock()
	c.mu.RLock()
}
