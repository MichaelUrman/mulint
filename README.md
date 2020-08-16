# mulint

**mulint** is a collection of linters I've written to handle things that have come up in code reviews, but could be automated. They're not necessarily great to apply across the board, so these are intended to help find instances that might benefit.

Many thanks to [Fatih's blog article](https://arslan.io/2019/06/13/using-go-analysis-to-write-a-custom-linter/), the [types tutorial](https://github.com/golang/example/tree/master/gotypes) and all contributors to [golang.org/x/tools/go/analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis) for making this possible.

## Linters

* [constlit](constlit/README.md): Identify literal expressions that should be replaced by local or imported constants
* [methodlambda](methodlambda/README.md): Identify anonymous functions that could be replaced by method expressions

## Use

Install with `go get -u github.com/MichaelUrman/mulint`; this should install cmd/mulint into your `GOPATH/bin`.

Then invoke like any other `golang.org/x/tools/go/analysis` checker. To run all included linters against the packages under the current directory, run `mulint ./...`. To run just one linter, add `-lintername=False` for all other linters. If you like all its recommendations, you can run with `-fix` to apply them. See `mulint -h` for details.

## Troubleshooting Notes

If you see errors including `could not import package C`, install gcc. This appears to be a bug or limitation in `golang.org/x/tools/go/analysis` as of when this was originally written. Once addressed, I'm open to fixing here, but expect it would just work.

(See [dominikh/go-tools#52](https://github.com/dominikh/go-tools/issues/572) and [golang/go#34229](https://github.com/golang/go/issues/34229) for updates.)
