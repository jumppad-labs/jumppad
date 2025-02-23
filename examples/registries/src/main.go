package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	// get the upstream url if present
	upstream_url := os.Getenv("UPSTREAM_URL")

	listen_addr := ":9090"
	if os.Getenv("LISTEN_ADDR") != "" {
		listen_addr = os.Getenv("LISTEN_ADDR")
	}

	message := "Hello world"
	if os.Getenv("MESSAGE") != "" {
		message = os.Getenv("MESSAGE")
	}

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "%s\n", message)

		if upstream_url != "" {
			resp, err := http.Get(upstream_url)
			if err != nil {
				http.Error(rw, fmt.Sprintf("unable to contact upstream: %s", err), http.StatusInternalServerError)
				return
			}

			b, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(rw, "Response from upstream: %s", string(b))
		}
	})

	fmt.Println(http.ListenAndServe(listen_addr, nil))
}
