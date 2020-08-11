// Package d tests exceptions such as 1*time.Second
package d

import "time"

var T time.Duration = 1 // want "Replace `1` with `time.Nanosecond`"
var U time.Duration = 1 * time.Second
var V = time.Second * 1
var W = 3*time.Second + 1 // want "Replace `1` with `time.Nanosecond`"

var A int = 2 // OK; don't use time.Wed
