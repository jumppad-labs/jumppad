package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	// get the upstream url if present
	url := os.Getenv("UPSTREAM_URL")

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Hello world\n")

		if url != "" {
			resp, err := http.Get(url)
			if err != nil {
				http.Error(rw, fmt.Sprintf("unable to contact upstream: %s", err), http.StatusInternalServerError)
				return
			}

			b, _ := ioutil.ReadAll(resp.Body)
			fmt.Fprintf(rw, "Response from upstream: %s", string(b))
		}
	})

	fmt.Println(http.ListenAndServe(":9090", nil))
}
