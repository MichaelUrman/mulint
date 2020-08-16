// Package b also tests exceptions such as 1*time.Second but with dot imports
package b

import . "time"

var Tdot Duration = 1 // want "Replace `1` with `Nanosecond`"
var Udot Duration = 1 * Second
var Vdot = Second * 1
var Wdot = 3*Second + 1 // want "Replace `1` with `Nanosecond`"

var Adot int = 2 // OK; don't use Wed
