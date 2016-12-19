fastlimiter
=====
The fastest abstract rate limiter, base on memory

[![Build Status](http://img.shields.io/travis/mushroomsir/fastlimiter.svg?style=flat-square)](https://travis-ci.org/mushroomsir/fastlimiter)
[![Coverage Status](http://img.shields.io/coveralls/mushroomsir/fastlimiter.svg?style=flat-square)](https://coveralls.io/r/mushroomsir/fastlimiter)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/mushroomsir/fastlimiter/master/LICENSE)
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/mushroomsir/fastlimiter)
## Installation

```bash
go get github.com/mushroomsir/fastlimiter
```

## Example

Try into github.com/teambition/fastlimiter directory:

```bash
go run examples/main.go
```
Visit: http://127.0.0.1:8080/a

```go
package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/mushroomsir/fastlimiter"
)

func main() {

	limiter := fastlimiter.New(fastlimiter.Options{})

	http.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		policy := []int{3, 30000, 2, 60000}
		res, err := limiter.Get(r.URL.Path, policy...)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		header := w.Header()
		header.Set("X-Ratelimit-Limit", strconv.FormatInt(int64(res.Total), 10))
		header.Set("X-Ratelimit-Remaining", strconv.FormatInt(int64(res.Remaining), 10))
		header.Set("X-Ratelimit-Reset", strconv.FormatInt(res.Reset.Unix(), 10))

		if res.Remaining >= 0 {
			w.WriteHeader(200)
			fmt.Fprintf(w, "Path: %q\n", html.EscapeString(r.URL.Path))
			fmt.Fprintf(w, "Remaining: %d\n", res.Remaining)
			fmt.Fprintf(w, "Total: %d\n", res.Total)
			fmt.Fprintf(w, "Duration: %v\n", res.Duration)
			fmt.Fprintf(w, "Reset: %v\n", res.Reset)
		} else {
			after := int64(res.Reset.Sub(time.Now())) / 1e9
			header.Set("Retry-After", strconv.FormatInt(after, 10))
			w.WriteHeader(429)
			fmt.Fprintf(w, "Rate limit exceeded, retry in %d seconds.\n", after)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

## Benchmark and Test
```sh
go test -bench="."
```
```sh
BenchmarkGet-4                            100000               409 ns/op              96 B/op          3 allocs/op
BenchmarkGetAndEexceeding-4               100000               379 ns/op              96 B/op          3 allocs/op
BenchmarkGetAndParallel-4                 100000               389 ns/op              96 B/op          3 allocs/op
BenchmarkGetAndClean-4                     10000               399 ns/op              96 B/op          3 allocs/op
BenchmarkGetForDifferentUser-4             10000              1600 ns/op             466 B/op          8 allocs/op
PASS
ok      github.com/mushroomsir/fastlimiter      6.121s
```