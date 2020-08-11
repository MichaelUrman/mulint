# MethodLambda

MethodLambda is an analysis package that identifies anonymous functions that could be replaced with method expressions.

## Before

```go
    bar := func(q quux, a, b int) int {
        return q.bar(a, b)
    }
    baz := func(q *quux, c, d float) int {
        return q.baz(c, d)
    }
```

## After

```go
    bar := quux.bar
    baz := (*quux).baz
```
