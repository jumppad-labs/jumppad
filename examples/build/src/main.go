package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Println("Hello World Arse")

		fmt.Fprint(rw, "Erik is a big girl")
	})

	http.ListenAndServe(":9090", nil)
}
