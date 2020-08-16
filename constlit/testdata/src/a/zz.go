// Package a also tests simple literal replacements with a renamed import
package a

import zz "image/png"

var zznah = -2

var zzbs zz.CompressionLevel = -2 // want "Replace `-2` with `zz.BestSpeed`"

func zzCompress(level zz.CompressionLevel) {}

func zzmain() {
	zzCompress(-3) // want "Replace `-3` with `zz.BestCompression`"
}
