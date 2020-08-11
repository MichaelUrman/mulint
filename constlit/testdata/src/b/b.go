// Package b tests simple literal replacements with a renamed import
package b

import zz "image/png"

var nah = -2

var bs zz.CompressionLevel = -2 // want "Replace `-2` with `zz.BestSpeed`"

func Compress(level zz.CompressionLevel) {}

func main() {
	Compress(-3) // want "Replace `-3` with `zz.BestCompression`"
}
