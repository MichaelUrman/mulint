// Package a tests simple literal replacements
package a

import "image/png"

var anah = -2

var abs png.CompressionLevel = -2 // want "Replace `-2` with `png.BestSpeed`"

func aCompress(level png.CompressionLevel) {}

func amain() {
	aCompress(-3) // want "Replace `-3` with `png.BestCompression`"
}
