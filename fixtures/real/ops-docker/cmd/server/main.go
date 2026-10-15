// Command server answers with a greeting.
package main

import (
	"fmt"
	"net/http"

	"example.com/real/opsdocker/internal/greet"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, greet.Hello(r.URL.Query().Get("name")))
	})
	_ = http.ListenAndServe(":8080", nil)
}
