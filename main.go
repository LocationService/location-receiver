package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"log"
	"net/http"
	"strings"
)

const SIGN_KEY = "ea86ec783c52d9e26607d11a1247485a"
const AUTH_TOKEN = "f5ee5dee5f9ded00a624ff4bf34eb3d3"

type Location struct {
	gorm.Model
	Lat      float64
	Lng      float64
	Provider string
	Imei     string
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

	db, err := gorm.Open("sqlite3", "location.db")
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	db.AutoMigrate(location)
	db.Create(location)
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
