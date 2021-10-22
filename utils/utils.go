package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// DecodeReq decodes a json request body into an interface
func DecodeReq(r *http.Request, model interface{}) error {
	defer r.Body.Close()
	b, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(b, model)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	return err
}

// HashPassword returns an encrypted form of a password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash compares a plain password with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenPasscode generates a new passcode
func GenPasscode() int {
	return rand.Intn(9999-939) + 939
}

// Now returns the current formatted timestamp
func Now() string {
	return time.Now().Format("01-02-2006 15:04:05")
}

// PostToChain ...
func PostToChain(endpoint string, payload interface{}) (*http.Response, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	return client.Do(req)
}
