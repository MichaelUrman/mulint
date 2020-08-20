package d

import "sync"

type sliceMutex struct {
	mus []*sync.Mutex
}

func (r *sliceMutex) Good1() { // want Good1:`r.mus.[*].:"L"`
	for _, x := range r.mus {
		x.Lock()
	}
}

func (r *sliceMutex) Good2() { // want Good2:`r.mus.[*].:"l"`
	for _, x := range r.mus {
		x.Unlock()
	}
}

func (r *sliceMutex) Good3() { // want Good3:`r.mus.[*].:"Ll"`
	for _, x := range r.mus {
		x.Lock()
	}
	for _, x := range r.mus {
		x.Unlock()
	}
}

func (r *sliceMutex) Bad1() { // want Bad1:`r.mus.[*].:"L"`
	for _, x := range r.mus {
		x.Lock()
	}
	for _, x := range r.mus {
		x.Lock()
	}
}

type mapMutex struct {
	mus map[string]*sync.Mutex
}

func (r *mapMutex) Good1() { // want Good1:`r.mus.[*].:"L"`
	for _, x := range r.mus {
		x.Lock()
	}
}

func (r *mapMutex) Good2() { // want Good2:`r.mus.[*].:"l"`
	for _, x := range r.mus {
		x.Unlock()
	}
}

func (r *mapMutex) Good3() { // want Good3:`r.mus.[*].:"Ll"`
	for _, x := range r.mus {
		x.Lock()
	}
	for _, x := range r.mus {
		x.Unlock()
	}
}

func (r *mapMutex) Bad1() { // want Bad1:`r.mus.[*].:"L"`
	for _, x := range r.mus {
		x.Lock()
	}
	for _, x := range r.mus {
		x.Lock()
	}
}
