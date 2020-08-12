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

## Notes

It's not always clear whether using a method expression is a win over the anonymous function it replaces, for at least two reasons.

- It's easy to not have learned the method expression syntax
- The syntax for a pointer receiver, `(*Type).Method`, is a bit clunky
