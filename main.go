package main

import (
	"net/http"
	//"fmt"
	//"os"
	//"time"
	"flag"
	"log"
	"crypto/hmac"
	"crypto/sha1"
	"strings"
	"encoding/hex"
	"encoding/json"
)

const SIGN_KEY = "test"
const AUTH_TOKEN = "123"

type Location struct {
	Lat float64
	Lng float64
	Provider string
	Imei string
}

var addr = flag.String("addr", "localhost:8082", "Address to server handle")

func main() {
	flag.Parse()
	http.HandleFunc("/", handler)
	log.Println("Starting on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func makeSign(data string) []byte {
	key := []byte(SIGN_KEY)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func checkSignedBody(signedBody string) bool {
	raw := strings.Split(signedBody, ".")
	if len(raw) < 2 {
		log.Println("signed_body must contain sign and body only")
		return false
	}

	hexSign := raw[0]
	body := strings.Join(raw[1:], ".")

	expSign, err := hex.DecodeString(hexSign)
	if err != nil {
		log.Println(err)
		return false
	}
	sign := makeSign(body)
	return hmac.Equal(expSign, sign)
}

func parseSignedBody(signedBody string) (*Location, bool) {
	var location Location
	raw := strings.Split(signedBody, ".")
	if len(raw) < 2 {
		log.Println("signed_body must contain sign and body only")
		return nil, false
	}
	body := strings.Join(raw[1:], ".")
	err := json.Unmarshal([]byte(body), &location)
	if err != nil {
		log.Println(err)
		return nil, false
	}
	return &location, true
}

func handler(w http.ResponseWriter, r *http.Request) {
	var signedBody string

	if !authRequest(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	log.Println("Request with", r.Form)

	signedBody = r.Form["signed_body"][0]

	if !checkSignedBody(signedBody) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	location, ok := parseSignedBody(signedBody)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println(location)

	//lat := r.Form["lat"]
	//lng := r.Form["lng"]
	//str := fmt.Sprintf("%v\tlat: %s\tlng: %s\n", time.Now(), lat, lng)
	//
	//f, err := os.OpenFile("locations.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	//defer f.Close()
	//
	//if err != nil {
	//	log.Println("Error while open file:", err)
	//	http.Error(w, "Internal server error", 500)
	//	return
	//}
	//n, err := f.WriteString(str)
	//if err != nil {
	//	log.Println("Error while write to file:", err)
	//	http.Error(w, "Internal server error", 500)
	//	return
	//}
	//
	//log.Println("Wrote", n, "bytes to file")
}

func authRequest(r *http.Request) bool {
	authHeaders := r.Header["Authentication"]
	if len(authHeaders) < 1 {
		return false
	}

	authHeaders = strings.Split(authHeaders[0], " ")
	if len(authHeaders) != 2 {
		return false
	}

	authToken := authHeaders[1]
	if authToken != AUTH_TOKEN {
		return false
	}

	return true
}