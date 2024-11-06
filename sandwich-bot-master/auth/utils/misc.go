package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// PriceInfo holds the price data and the last update timestamp
type PriceInfo struct {
	MATICinUSDT float64
	LastUpdate  time.Time
	Processing  bool
	// sync.Mutex
}

// Global variable to hold the price
var MaticPriceInfo = PriceInfo{}

// func init() {
// 	FetchMATICPrice()
// }

// fetchMATICPrice updates the globalPriceInfo with the latest MATIC price in USDT
func FetchMATICPrice() {
	url := "https://api.coingecko.com/api/v3/simple/price?ids=matic-network&vs_currencies=usd"
	MaticPriceInfo.Processing = true

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch MATIC price: %v", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode response: %v", err)
		return
	}

	// MaticPriceInfo.Lock() defer MaticPriceInfo.Unlock()
	// Assuming the API response structure, update the global price
	fmt.Println(result)
	if usdtPrice, ok := result["matic-network"]["usd"]; ok {
		MaticPriceInfo.MATICinUSDT = usdtPrice
		MaticPriceInfo.LastUpdate = time.Now()
		log.Printf("Updated MATIC price: %f USDT", usdtPrice)

	} else {
		log.Println("MATIC price in USDT not found in response")
	}
	MaticPriceInfo.Processing = false
}

// GetMaticPrice checks if the current price is older than 5 seconds and fetches a new one if necessary
func GetMaticPrice() float64 {
	// MaticPriceInfo.Lock()
	// defer MaticPriceInfo.Unlock()

	// fmt.Println(time.Since(MaticPriceInfo.LastUpdate) > 5*time.Second, MaticPriceInfo.MATICinUSDT == 0)
	if time.Since(MaticPriceInfo.LastUpdate) > 5*time.Second || MaticPriceInfo.MATICinUSDT == 0 {
		// Price is older than 5 seconds, fetch a new one
		// Background task
		if !MaticPriceInfo.Processing {
			go FetchMATICPrice() // Fetch in a goroutine to not block the GetMaticPrice call
		}
		return MaticPriceInfo.MATICinUSDT // Return the last known price while the new one is being fetched
	}

	// Price is recent, return it
	return MaticPriceInfo.MATICinUSDT
}

// func CheckPrice() {
// 	client, err := ethclient.Dial("wss://soft-damp-thunder.matic.quiknode.pro/f6508ae4e35a9831d01bd715348a775770c12568/"
// 	if err != nil {
// 		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
// 	}

// 	contractAddress := common.HexToAddress()
// 	contractABI, err := abi.JSON(strings.NewReader("CONTRACT_ABI"))
// 	if err != nil {
// 		log.Fatalf("Failed to parse contract ABI: %v", err)
// 	}

// 	// Assuming the contract has a function `getLatestPrice` that returns the latest MATIC/USDT price
// 	result, err := contractABI.Pack("getLatestPrice")
// 	if err != nil {
// 		log.Fatalf("Failed to pack data for getLatestPrice: %v", err)
// 	}

// 	msg := ethereum.CallMsg{
// 		To:   &contractAddress,
// 		Data: result,
// 	}
// 	output, err := client.CallContract(context.Background(), msg, nil)
// 	if err != nil {
// 		log.Fatalf("Failed to call contract: %v", err)
// 	}

// 	// Assuming the price is returned as a uint256
// 	var price *big.Int
// 	err = contractABI.Unpack(&price, "getLatestPrice", output)
// 	if err != nil {
// 		log.Fatalf("Failed to unpack output: %v", err)
// 	}

// 	fmt.Printf("The latest MATIC/USDT price is: %s\n", price)
// }


