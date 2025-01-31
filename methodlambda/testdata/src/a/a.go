package a

type foo struct {
	k string
}

type bar struct{}

func (f foo) bar(i int) {}

func (f foo) bar2(i int, s string) {}

func (f foo) bar3(i int) string { return "" }

func (f foo) bar4(i, j int) {}

func (f *foo) ptrRecv() {}

func methodCalls() {
	f := foo{}

	// unnamed parameters cannot simplify
	_ = func(foo, int) { f.bar(10) }

	// non-parameter argument doesn't simplify
	_ = func(g foo) { g.bar(20) }

	// added return doesn't simplify
	_ = func(g foo, i int) int {
		g.bar(i)
		return i
	}

	// non-trivial function doesn't simplify
	_ = func(g foo, j int) {
		g.bar(j)
		g.bar(j)
	}

	// missing a parameter doesn't simplify
	_ = func(g foo, k int) {
		g.bar2(k, "")
	}

	// missing a return doesn't simplify
	_ = func(g foo, m int) {
		g.bar3(m)
	}

	// non-trivial (modify return) doesn't simplify
	_ = func(g foo, m int) string {
		return g.bar3(m) + "_"
	}

	// TODO: support simplification of slightly less trivial
	_ = func(g foo, m int) string {
		x := g.bar3(m)
		return x
	}

	// non-pointer doesn't simplify a pointer receiver method
	_ = func(g foo) { g.ptrRecv() }
}

func methodExprCalls() {
	_ = func(f foo, i int) { // want "replace `func[(]f foo, i int[)]` with `foo.bar`"
		f.bar(i)
	}
	_ = func(f foo, i int, s string) { // want "replace `func[(]f foo, i int, s string[)]` with `foo.bar2`"
		f.bar2(i, s)
	}
	_ = func(f foo, i int) string { // want "replace `func[(]f foo, i int[)] string` with `foo.bar3`"
		return f.bar3(i)
	}
	_ = func(f foo, m int, n int) { // want "replace `func[(]f foo, m int, n int[)]` with `foo.bar4`"
		f.bar4(m, n)
	}
	_ = func(f foo, m, n int) { // want "replace `func[(]f foo, m, n int[)]` with `foo.bar4`"
		f.bar4(m, n)
	}

	_ = func(f *foo, i int) { // want "replace `func[(].*[)]` with `[(][*]foo[)].bar`"
		f.bar(i)
	}
	_ = func(f *foo, i int, s string) { // want "replace `func[(].*[)]` with `[(][*]foo[)].bar2`"
		f.bar2(i, s)
	}
	_ = func(f *foo, i int) string { // want "replace `func[(].*[)] string` with `[(][*]foo[)].bar3`"
		return f.bar3(i)
	}

	_ = func(f *foo) { // want "replace `func[(]f [*]foo[)]` with `[(][*]foo[)].ptrRecv`"
		f.ptrRecv()
	}
}
