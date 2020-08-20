package c

import "sync"

var mu sync.Mutex

type loops struct{}

func (loops) Stretch1() { // want Stretch1:`c:mu:"Ll"`
	for i := 0; i < 1; i++ {
		mu.Lock()
	}
	mu.Unlock()
}

func (loops) Stretch2() { // want Stretch2:`c:mu:"Ll"`
	mu.Lock()
	for i := 0; i < 1; i++ {
		mu.Unlock()
	}
}

func (loops) Stretch3() { // want Stretch3:`c:mu:"L"`
	for i := 0; i < 3; i++ {
		mu.Lock() // want `Locks locked c:mu`
	}
}

func (loops) Stretch4() { // -want Stretch4:`(mus.[012].:"Ll" ?){3}`
	mus := make([]sync.Mutex, 3)
	for i := range mus {
		mus[i].Lock()
	}
	for i := range mus {
		x := mus[i]
		x.Unlock()
	}
}
