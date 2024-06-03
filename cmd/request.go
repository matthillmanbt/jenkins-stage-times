package cmd

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/spf13/viper"
)

func jenkinsRequest(path string, query ...map[string]string) (*http.Response, error) {
	var (
		vHost = viper.Get("host")
		vUser = viper.Get("user")
		vKey  = viper.Get("key")
	)

	verbose("Using host [%s]", host)
	verbose("Using user [%s] and key [***]", vUser)

	client := &http.Client{}
	apiKey := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", vUser, vKey)))

	url := fmt.Sprintf("%s/%s", vHost, path)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", apiKey))

	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query[0] {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	verbose("Calling jenkins API [%s]", req.URL)

	return client.Do(req)
}
