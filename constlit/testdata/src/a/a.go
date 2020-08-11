// Package a tests simple literal replacements
package a

import "image/png"

var nah = -2

var bs png.CompressionLevel = -2 // want "Replace `-2` with `png.BestSpeed`"

func Compress(level png.CompressionLevel) {}

func main() {
	Compress(-3) // want "Replace `-3` with `png.BestCompression`"
}
