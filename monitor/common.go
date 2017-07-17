package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func JSONDecode(data []byte, to interface{}) error {
	err := json.Unmarshal(data, &to)

	if err != nil {
		return err
	}

	return nil
}

func StripHTTPPrefix(url string) string {
	if strings.Contains(url, "http://") {
		return strings.TrimPrefix(url, "http://")
	} else {
		return strings.TrimPrefix(url, "https://")
	}
}

func GetOnlineOffline(online bool) string {
	if !online {
		return "OFFLINE"
	}
	return "ONLINE"
}

func ReadFile(path string) ([]byte, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func SendHTTPGetRequest(url string, jsonDecode bool) (result interface{}, contentSize, httpCode int, err error) {
	res, err := http.Get(url)

	if err != nil {
		return
	}

	httpCode = res.StatusCode
	if httpCode != 200 && httpCode != 400 {
		log.Printf("HTTP status code: %d\n", httpCode)
		err = errors.New("Status code was not 200.")
		return
	}

	contents, err := ioutil.ReadAll(res.Body)
	contentSize = len(contents)

	if err != nil {
		return
	}

	defer res.Body.Close()

	if jsonDecode {
		err := JSONDecode(contents, &result)

		if err != nil {
			return result, contentSize, httpCode, err
		}
	} else {
		result = string(contents)
	}

	return
}

func GetSecondsElapsed(timestamp int64) int64 {
	tm := time.Unix(timestamp, 0)
	return int64(time.Since(tm).Seconds())
}
