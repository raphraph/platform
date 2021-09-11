package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

const socket = "/var/snap/platform/common/api.socket"

func do(method string, url string, data interface{}) (*string, error) {
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}
	var dataReader io.Reader
	if data != nil {
		requestJson, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		dataReader = bytes.NewReader(requestJson)
	}
	request, err := http.NewRequest(method, fmt.Sprintf("http://unix/%s", url), dataReader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	responseString, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(responseString.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if responseString.StatusCode != 200 {
		return nil, fmt.Errorf("unable to connect to %s with error code: %d\n", socket, responseString.StatusCode)
	}

	var response Response
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf("service error: %s\n", response.Message)
	}
	return response.Data, nil
}

func GetAppDir(app string) (*string, error) {
	return do(http.MethodGet, fmt.Sprintf("/app/install_path?name=%s", app), nil)
}

func Restart(service string) (*string, error) {
	return do(http.MethodPost, "/service/restart", &ServiceRestart{Name: service})
}

func SetDkimKey(dkimKey string) (*string, error) {
	return do(http.MethodPost, "/config/set_dkim_key", &DkimKey{dkimKey})
}

func GetDkimKey() (*string, error) {
	return do(http.MethodGet, "/config/get_dkim_key", nil)
}

func GetDataDir(app string) (*string, error) {
	return do(http.MethodGet, fmt.Sprintf("/app/data_path?name=%s", app), nil)
}

func GetAppUrl(app string) (*string, error) {
	return do(http.MethodGet, fmt.Sprintf("/app/url?name=%s", app), nil)
}
