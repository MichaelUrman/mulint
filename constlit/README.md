# constlit

ConstLit is an analysis package that identifies literal expressions that should be replaced by local or imported constants.

## Before

```go
import "image/png"

var enc := png.Encoder{CompressionLevel: -2}
```

## After

```go
import "image/png"

var enc := png.Encoder{CompressionLevel: png.BestSpeed}
```
