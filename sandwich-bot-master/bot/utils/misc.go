package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/oklog/ulid/v2"
	"github.com/shopspring/decimal"
)

// PriceInfo holds the price data and the last update timestamp

type SafePairPrice struct {
	Data       map[string]*PriceInfo
	LastUpdate time.Time
}

type PriceInfo struct {
	PairPrice  float64
	Processing bool
	Mutex      sync.Mutex
}

func (spp *SafePairPrice) ToggleLock(from string, processing bool) {
	priceInfo, ok := spp.Data[from]
	if !ok {
		// If the key is not present in the map, create a new PriceInfo
		priceInfo = &PriceInfo{
			Processing: processing,
		}
	} else {
		priceInfo.Mutex.Lock()
		defer priceInfo.Mutex.Unlock()
		priceInfo.Processing = processing
	}
	spp.Data[from] = priceInfo
}

// Global variable to hold the price

var PairPriceInfo SafePairPrice

func RetrievePairPrice(from, to []string) {
	_from := strings.Join(from, ",")
	_to := strings.Join(to, ",")

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=%s", _from, _to)

	for _, _f := range from {
		PairPriceInfo.ToggleLock(_f, true)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch %s-%s price: %v", from, to, err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}

	// log.Printf("Response Body: %s", string(bodyBytes))

	var result map[string]map[string]float64

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		log.Printf("Failed to decode response: %v", err)
		return
	}

	for _, _f := range from {
		if usdtPrice, ok := result[_f]["usd"]; ok {
			priceInfo, ok := PairPriceInfo.Data[_f]
			if !ok {
				log.Printf("Price info not found for %s", _f)
				continue
			}
			priceInfo.Mutex.Lock()
			priceInfo.PairPrice = usdtPrice
			priceInfo.Mutex.Unlock()
			// log.Printf("Updated %s price: %f USDT", _f, usdtPrice)
		} else {
			log.Printf("%s price in USDT not found in response", _f)
		}
	}

	PairPriceInfo.LastUpdate = time.Now()

	for _, _f := range from {
		PairPriceInfo.ToggleLock(_f, false)
	}
}

// GetMaticPrice checks if the current price is older than 5 seconds and fetches a new one if necessary
func GetPairPrice(from, to []string) SafePairPrice {
	if PairPriceInfo.Data == nil {
		PairPriceInfo.Data = make(map[string]*PriceInfo)
	}
	for _, _f := range from {
		if _, exists := PairPriceInfo.Data[_f]; !exists {
			PairPriceInfo.Data[_f] = &PriceInfo{}
		}
		PairPriceInfo.Data[_f].Mutex.Lock()
		defer PairPriceInfo.Data[_f].Mutex.Unlock()
	}

	if time.Since(PairPriceInfo.LastUpdate) > 20*time.Second {
		// Price is older than 5 seconds, fetch a new one
		// Background task
		_from := []string{}
		for _, _f := range from {
			if !PairPriceInfo.Data[_f].Processing {
				_from = append(_from, _f)
			}
		}
		if len(_from) > 0 {
			// fmt.Println("CALL IS MADE")
			go RetrievePairPrice(_from, to)
		}
	}

	// Price is recent, return it
	return PairPriceInfo
}

var MapContains = func(myMap map[string]string, value string) (*string, *string, bool) {

	for _k, _v := range myMap {
		if strings.EqualFold(_v, value) || strings.EqualFold(_k, value) {
			return &_k, &_v, true
		}
	}

	return nil, nil, false
}

var MapContainsV2 = func(myMap map[string][]interface{}, value string) (coin *string, address *string, decimals *int32, contains bool) {

	for _k, _v := range myMap {
		if strings.EqualFold(_v[0].(string), value) {
			addr := strings.ToLower(_v[0].(string))
			decInt32 := _v[1].(int32)
			// decInt32 := int32(_v[1].(decimal.Decimal).IntPart())

			return &_k, &addr, &decInt32, true
		}
	}

	return nil, nil, nil, false
}

var HexToECDSA = func(hexKey string) (*ecdsa.PrivateKey, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}

	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = elliptic.P256()
	privateKey.D = new(big.Int).SetBytes(bytes)

	publicKeyX, publicKeyY := privateKey.PublicKey.Curve.ScalarBaseMult(privateKey.D.Bytes())

	privateKey.PublicKey.X, privateKey.PublicKey.Y = publicKeyX, publicKeyY

	return privateKey, nil
}

var HexToECDSAV2 = func(hexKey string) (*ecdsa.PrivateKey, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}

	return crypto.ToECDSA(bytes)
}

var DecodePath = func(path string) ([]string, error) {
	// The path is encoded as a hexadecimal string
	decoded, err := hex.DecodeString(strings.TrimPrefix(path, "0x"))
	if err != nil {
		return nil, err
	}

	// The decoded path is a byte slice, each token address is 20 bytes
	var tokens []string
	for i := 0; i < len(decoded); i += 20 {
		address := decoded[i : i+20]
		tokens = append(tokens, hex.EncodeToString(address))
	}
	fmt.Println("TOKENS", tokens)
	return tokens, nil
}

var StringToUlid = func(str string) ulid.ULID {
	uid, err := ulid.Parse(str)
	if err != nil {
		log.Fatalf("Error: could not convert str to ulid: %v", err)
	}
	return uid
}

var StringToDecimal = func(str string) *decimal.Decimal {
	decimal, err := decimal.NewFromString(str)
	if err != nil {
		log.Fatalf("Error: could not convert str to decimal: %v", err)
	}
	return &decimal
}

var IntToUint = func(i int) *uint {
	u := uint(i)
	return &u
}
var FormatLikeClause = func(input interface{}) string {
	switch v := input.(type) {
	case string:
		return fmt.Sprintf("%%%s%%", v) // Safe for SQL LIKE clause
	case int, int64:
		return fmt.Sprintf("%%%d%%", v) // Convert integer to string in LIKE clause
	case float64:
		return fmt.Sprintf("%%%s%%", strconv.FormatFloat(v, 'f', -1, 64))
	default:
		return "%" // Default fallback if type is unknown or nil
	}
}

// Define a custom type for int24
type Int24 int

// Constants defining the range of int24
const (
	MinInt24 = -1 << 23
	MaxInt24 = 1<<23 - 1
)

// ValidateInt24 checks if the value is within the int24 range
func ValidateInt24(value Int24) error {
	if value < MinInt24 || value > MaxInt24 {
		return errors.New("value is not within the int24 range")
	}
	return nil
}

func ConvertDecimalToInt24(decimalValue *decimal.Decimal) (Int24, error) {
	if decimalValue == nil {
		return 0, fmt.Errorf("decimal value is nil")
	}

	// Convert decimal to float64 for comparison
	floatValue, _ := decimalValue.Float64()

	// Check if the decimal value is within the range of an int24
	if floatValue < float64(MinInt24) || floatValue > float64(MaxInt24) {
		return 0, fmt.Errorf("decimal value %s is out of int24 range", decimalValue.String())
	}

	// Round the decimal value to the nearest integer and convert to Int24
	intValue := Int24(decimalValue.Round(0).IntPart())
	return intValue, nil
}

var IntToInt32 = func(i int) *int32 {
	_int32 := int32(i)
	return &_int32
}

var StringToInt32 = func(s string) *int32 {
	i, _ := strconv.ParseInt(s, 10, 32)
	i32 := int32(i)
	return &i32
}
