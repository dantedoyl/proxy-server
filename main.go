package main

import (
	"log"
)

func main() {
	server := NewServer(":8080", InitDatabase())
	log.Fatal(server.ListenAndServe())
}
