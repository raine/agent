package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	acceptLogs(os.Stdout)
}

func acceptLogs(out io.Writer) {
	http.HandleFunc("/frames", func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(out, string(dump))
		w.WriteHeader(200)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
