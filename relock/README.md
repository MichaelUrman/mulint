# Relock

Relock is an analysis package that identifies function paths that appear to call `Lock()` on the same object twice in a row. This is useful to find code that could deadlock.

Since this requires deeper knowledge either of the code, its use, or the conventions, it does not recommend a specific fix.

## Example

```go

func (q *quux) Outer() {
    q.mu.Lock()
    q.inner() // Warns that this call locks the already-locked q.mu
}

func (q *quux) inner() {
    q.mu.Lock()
}
```

## Notes
