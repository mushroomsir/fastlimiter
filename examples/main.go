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

	limiter := fastlimiter.New(&fastlimiter.Options{})

	http.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		policy := []int32{3, 30000, 2, 60000}
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
