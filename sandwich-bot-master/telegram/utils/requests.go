package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Response struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Status  string      `json:"status"`
}

func InternalRouter(endpoint, method string, headers, payload interface{}) (*Response, int, error) {
	var headersMap map[string]interface{}
	if headers == nil {
		headersMap = make(map[string]interface{})
	} else {
		var ok bool
		headersMap, ok = headers.(map[string]interface{})
		if !ok {
			return nil, 0, errors.New("headers must be of type map[string]interface{}")
		}
	}

	if payload == nil {
		payload = map[string]interface{}{}
	}

	_payload, _err := json.Marshal(payload)
	if _err != nil {
		return nil, 0, _err
	}

	resp, respCode, err := ForwardRequest(method, endpoint, &headersMap, _payload)
	log.Println("Error", err, respCode == nil)
	if err != nil {
		return nil, *respCode, err
	}

	return resp, *respCode, err
}

func ForwardRequest(httpMethod string, url string, headers *map[string]interface{}, payload []byte) (*Response, *int, error) {
	_http := &http.Client{}

	var request *http.Request
	var err error
	if len(payload) > 4 {
		request, err = http.NewRequest(httpMethod, url, bytes.NewReader(payload))
		if err != nil {
			return nil, nil, err
		}
	} else {
		request, err = http.NewRequest(httpMethod, url, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	if headers != nil {
		for key, value := range *headers {
			request.Header.Set(key, value.(string))
		}
	}

	response, err := _http.Do(request)
	fmt.Println("Error request", err)
	if err != nil {
		return nil, nil, err
	}

	defer response.Body.Close()

	var resp Response

	_body, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, nil, err
	}

	if _err := json.Unmarshal(_body, &resp); _err != nil {
		return nil, nil, err
	}

	return &resp, &response.StatusCode, nil
}
