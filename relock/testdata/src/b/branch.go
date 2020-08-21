package b

import (
	"sync"
)

var mu sync.Mutex

type branches struct{}

func (branches) Good1(which bool) { // want Good1:`b:mu:"Ll"`
	if which {
		mu.Lock()
	} else {
		mu.Lock()
	}
	mu.Unlock()
}

func (branches) Good2(which bool) { // want Good2:`b:mu:"Ll"`
	mu.Lock()
	if which {
		mu.Unlock()
	} else {
		mu.Unlock()
	}
}

func (branches) Good3(which bool) { // want Good3:`b:mu:"L[|]l"`
	if which {
		mu.Lock()
	} else {
		mu.Unlock()
	}
}

func (branches) Good4(which bool) { // want Good4:`b:mu:"Ll"`
	mu.Lock()
	if which {
		defer mu.Unlock()
	} else {
		mu.Unlock()
	}
}

func (branches) Bad1(which bool) { // want Bad1:`b:mu:"Ll"`
	mu.Lock()
	if which {
		defer mu.Unlock()
	} else {
		mu.Unlock()
	}
	mu.Lock() // want `Locks locked b:mu`
}
