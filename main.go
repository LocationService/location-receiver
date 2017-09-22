package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
	"net/http"
	"os"
	"strings"
)

type Device struct {
	gorm.Model
	Imei string
}

type Location struct {
	gorm.Model
	Lat      float64
	Lng      float64
	DeviceID uint
}

const SIGN_KEY = "ea86ec783c52d9e26607d11a1247485a"
const AUTH_TOKEN = "f5ee5dee5f9ded00a624ff4bf34eb3d3"

var addr = flag.String("addr", "0.0.0.0:8083", "Address to server handle")
var dbUser = flag.String("db-user", "root", "Database User")
var dbPassword = flag.String("db-password", "password", "Database Password")

func mysqlUrl() string {
	url := fmt.Sprintf("%s:%s@/location_receiver?charset=utf8&parseTime=True&loc=Local", *dbUser, *dbPassword)
	return url
}

func init() {
	var device Device
	var location Location

	flag.Parse()

	db, err := gorm.Open("mysql", mysqlUrl())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	db.AutoMigrate(device)
	db.AutoMigrate(location)
}

func main() {
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

func parseSignedBody(signedBody string) (map[string]interface{}, bool) {
	var params interface{}
	raw := strings.Split(signedBody, ".")
	if len(raw) < 2 {
		log.Println("signed_body must contain sign and body only")
		return nil, false
	}
	body := strings.Join(raw[1:], ".")
	err := json.Unmarshal([]byte(body), &params)

	if err != nil {
		log.Println(err)
		return nil, false
	}
	return params.(map[string]interface{}), true
}

func handler(w http.ResponseWriter, r *http.Request) {
	var signedBody string
	var device Device
	var location Location

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

	params, ok := parseSignedBody(signedBody)

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	db, err := gorm.Open("mysql", mysqlUrl())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	db.Where("imei = ?", params["imei"]).First(&device)
	if db.NewRecord(device) {
		device.Imei = params["imei"].(string)
		db.Create(&device)
	}

	location.Lat = params["lat"].(float64)
	location.Lng = params["lng"].(float64)
	location.DeviceID = device.ID

	db.Create(&location)
}

func authRequest(r *http.Request) bool {
	authHeaders := r.Header["Authorization"]
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
