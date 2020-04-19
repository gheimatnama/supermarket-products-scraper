package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)


func GetHostFromUrl(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.Hostname()
}


func GetPathFromUrl(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.Path
}


func GetJson(url string, target interface{}) error {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, target)
	if err != nil {
		return err
	}
	return nil
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return true, err
}


func IsNumber(str string) bool {
	if _, err := strconv.Atoi(str); err == nil {
		return true
	}
	return false
}


func StringToNumber(str string) int {
	if number, err := strconv.Atoi(str); err == nil {
		return number
	}
	return 0
}


func HashString(str string) string {
	md5HashInBytes := md5.Sum([]byte("Sum returns bytes"))
	md5HashInString := hex.EncodeToString(md5HashInBytes[:])
	return md5HashInString
}
