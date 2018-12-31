//+build !login

package main

import (
	"io/ioutil"
	"net/http"
)

func FindEmptyInstance(token string) string {
	resp, err := http.Get("http://instance-data/latest/meta-data/public-hostname")
	if err == nil {
		body, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		return string(body)
	} else {
		return "127.0.0.1"
	}
}

func LogoutInstance(token string) {
}
