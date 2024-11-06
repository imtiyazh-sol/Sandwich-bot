package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"telegram/config"
	"telegram/utils"
)

var (
	RetrieveUser = func(qParams map[string]interface{}, scope ...string) (interface{}, error) {
		endpoint, err := config.InternalEndpoint("auth", "retrieve_user", qParams)
		if err != nil {
			return nil, errors.New("user not found")
		}

		__resp, _, _err := utils.InternalRouter(endpoint.String(), "GET", nil, nil)
		if _err != nil {
			return nil, _err
		}

		if __resp.Status != "success" {
			return nil, errors.New("user not found")
		}

		user, ok := __resp.Data.(map[string]interface{})
		if !ok {
			return nil, errors.New("malformed response from auth service retrieve_user")
		}

		if len(scope) > 0 {
			valueToReturn, ok := user[scope[0]]
			if !ok {
				return nil, errors.New("respnonse does not provide selected scope")
			}
			return valueToReturn, nil
		}

		// return full scope of user data
		return __resp, nil
	}

	CreateUser = func(payload map[string]interface{}) (interface{}, error) {
		endpoint, err := config.InternalEndpoint("auth", "create_user")
		if err != nil {
			return nil, errors.New("user not found")
		}

		__resp, _respCode, _err := utils.InternalRouter(endpoint.String(), "PUT", nil, payload)
		if _err != nil {
			return nil, _err
		}

		if _respCode == 409 {
			return nil, errors.New("user already exists")
		}

		if data, ok := __resp.Data.(map[string]interface{}); ok {
			if mnemonic, exists := data["mnemonic"]; exists && mnemonic != nil {
				return mnemonic, nil
			}
		}

		return nil, errors.New("internal error while creating the user")
	}

	CreateAccess = func(payload map[string]interface{}) (interface{}, error) {
		endpoint, err := config.InternalEndpoint("auth", "create_access")
		if err != nil {
			return nil, err
		}

		__resp, _respCode, _err := utils.InternalRouter(endpoint.String(), "PUT", nil, payload)
		if _err != nil {
			return nil, err
		}

		if _respCode != http.StatusCreated || __resp == nil {
			return nil, errors.New("internal error while creating access")
		}

		return __resp, nil
	}

	GenericRequest = func(method, service, endppoint string, payload map[string]interface{}) (*utils.Response, error) {
		endpoint, err := config.InternalEndpoint(service, endppoint)
		if err != nil {
			return nil, err
		}
		if method == "GET" || method == "DELETE" {
			queryParams := url.Values{}
			for _k, _v := range payload {
				queryParams.Add(_k, fmt.Sprintf("%v", _v))
			}
			endpoint.RawQuery = queryParams.Encode()
			payload = map[string]interface{}{}
		}

		__resp, _, _err := utils.InternalRouter(endpoint.String(), method, nil, payload)
		if _err != nil {
			return nil, err
		}

		return __resp, nil
	}
)
