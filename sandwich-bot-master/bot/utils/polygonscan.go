package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

type GasPriceDataType struct {
	LastUpdated time.Time
	Result      Result
	mu          sync.Mutex
}

type Result struct {
	Status string `json:"status"`
	Result struct {
		SafeGasPrice    string `json:"SafeGasPrice"`
		ProposeGasPrice string `json:"ProposeGasPrice"`
		FastGasPrice    string `json:"FastGasPrice"`
		SuggestBaseFee  string `json:"suggestBaseFee"`
		UsdPrice        string `json:"UsdPrice"`
	} `json:"result"`
}

var GasPriceData GasPriceDataType

func GetGasPrice() (*Result, error) {
	GasPriceData.mu.Lock()
	defer GasPriceData.mu.Unlock()

	// If the gas price data is less than 5 seconds old, return it
	if time.Since(GasPriceData.LastUpdated) < 10*time.Second {
		return &GasPriceData.Result, nil
	}

	// log.Println("request to polygonscan for gas price")
	polygonscan_api_key := os.Getenv("POLYGONSCAN_API_KEY")
	resp, err := http.Get("https://api.polygonscan.com/api?module=gastracker&action=gasoracle&apikey=" + polygonscan_api_key)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// bodyString := string(bodyBytes)
	// fmt.Println("PolyScan Gas: ", bodyString)
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	defer resp.Body.Close()

	var result Result
	// bodyBytes, _ := ioutil.ReadAll(resp.Body)
	// bodyString := string(bodyBytes)
	// fmt.Println(bodyString)
	// resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Update the gas price data and its last updated time
	GasPriceData.Result = result
	GasPriceData.LastUpdated = time.Now()

	return &GasPriceData.Result, nil
}
