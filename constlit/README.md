# constlit

Constlit is an analysis package that identifies literal expressions that should be replaced by local or imported constants.

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

## Notes

Constlit can get pretty memory hungry, hopefully due to misuse of `golang.org/x/tools/go/loader`. (If you know a better approach, I'd love any pointers or pull requests.)

Constlit uses some arbitrary heuristics to avoid making horrible recommendations.

- Only recommend constants from the local package, or from packages it already imports
- Only recommend constants from imported packages that are of a defined type (not untyped, or plain int, etc.)
- Don't recommend constants to replace multiplied literals (i.e. 1 * time.Minute)
