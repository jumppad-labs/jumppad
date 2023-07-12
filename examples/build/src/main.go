package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Println("Hello World")

		fmt.Fprint(rw, "Hello cunt face")
	})

	http.ListenAndServe(":9090", nil)
}
