// Package c tests local constants and literals in more complex positions
package c

type foo int

const (
	Zero foo = iota
	One
	Two
	Four foo = 4
)

const Eight = Four + 4 // want "Replace `4` with `Four`"

var g0 foo = 0 // want "Replace `0` with `Zero`"
var (
	g1 foo = 1 // want "Replace `1` with `One`"
	g2 foo = 2 // want "Replace `2` with `Two`"
	g3 foo = 3
	g4 foo = 4 // want "Replace `4` with `Four`"
	g5 foo = 5
)

func Foo(f foo) {}

type bar struct {
	f foo
}

func f() {
	Foo(4) // want "Replace `4` with `Four`"
	Foo(5)
	if g0 == 2 { // want "Replace `2` with `Two`"
		g0 = 4      // want "Replace `4` with `Four`"
		g0 = g0 + 4 // want "Replace `4` with `Four`"
		g0 += 4     // want "Replace `4` with `Four`"
		g0 = 4 - 3  // want "Replace `4` with `Four`"
		g0 = 3 - 4  // want "Replace `4` with `Four`"
	}
	m := map[foo]foo{4: 5} // want "Replace `4` with `Four`"
	m = map[foo]foo{5: 4}  // want "Replace `4` with `Four`"
	m[8] = 5               // want "Replace `8` with `Eight`"
	m[3] = 5
	m[5] = 8 // want "Replace `8` with `Eight`"

	_ = bar{4}    // want "Replace `4` with `Four`"
	_ = bar{f: 4} // want "Replace `4` with `Four`"
}
