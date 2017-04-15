package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	acceptLogs(os.Stdout)
}

func acceptLogs(out io.Writer) {
	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		io.Copy(out, r.Body)
		w.WriteHeader(200)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
