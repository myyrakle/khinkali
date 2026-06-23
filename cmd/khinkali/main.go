package main

import (
	"log"
	"net/http"
	"os"

	"github.com/myyrakle/khinkali/converter"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.Handle("/convert", converter.NewHandler(converter.New()))

	log.Printf("khinkali listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
