package main

import (
	"net/http"
	"fmt"
	"os"
	"time"
	"flag"
	"log"
)

var addr = flag.String("addr", "localhost:8082", "Address to server handle")

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	log.Println("Starting on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Println("Request with", r.Form)

	lat := r.Form["lat"]
	lng := r.Form["lng"]
	str := fmt.Sprintf("%v\tlat: %s\tlng: %s\n", time.Now(), lat, lng)

	f, err := os.OpenFile("locations.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	if err != nil {
		log.Println("Error while open file:", err)
		http.Error(w, "Internal server error", 500)
		return
	}
	n, err := f.WriteString(str)
	if err != nil {
		log.Println("Error while write to file:", err)
		http.Error(w, "Internal server error", 500)
		return
	}

	log.Println("Wrote", n, "bytes to file")
}