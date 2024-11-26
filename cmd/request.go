package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spf13/viper"
)

var client = &http.Client{}

func jenkinsRequest(path string, query ...map[string]string) (*http.Response, error) {
	return jenkinsRequestWithMethod(http.MethodGet, path, query...)
}

func jenkinsRequestWithMethod(method string, path string, query ...map[string]string) (*http.Response, error) {
	var (
		vHost = viper.Get("host")
		vUser = viper.Get("user")
		vKey  = viper.Get("key")
	)

	verbose("Using host [%s]", vHost)
	verbose("Using user [%s] and key [***]", vUser)

	apiKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", vUser, vKey)))

	location := fmt.Sprintf("%s/%s", vHost, path)

	var body io.Reader = nil
	if len(query) > 0 && method != http.MethodGet {
		q := url.Values{}
		for k, v := range query[0] {
			q.Add(k, v)
		}
		data := q.Encode()
		vVerbose("setting post data [%s]", data)
		body = bytes.NewBuffer([]byte(data))
	}

	req, err := http.NewRequest(method, location, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", apiKey))

	if body != nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if len(query) > 0 && method == http.MethodGet {
		q := req.URL.Query()
		for k, v := range query[0] {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	verbose("Calling jenkins API [%s][%s]", req.Method, req.URL)

	return client.Do(req)
}
