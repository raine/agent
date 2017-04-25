package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func AcceptLogs(out io.Writer) {
	http.HandleFunc("/frames", func(w http.ResponseWriter, r *http.Request) {
		// dump, err := httputil.DumpRequest(r, true)
		dump, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintln(out, string(dump))
		log.Println("handling request")
		w.WriteHeader(200)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
