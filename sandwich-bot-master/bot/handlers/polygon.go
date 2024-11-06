package handlers

import (
	"bot/controllers"
	"bot/models"
	"bot/utils"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

//TODO: weird cases
// 1. https://polygonscan.com/tx/0x471b4f75c6a00c09cd892d16050f574f4e21efe2547c6379c355de0f9292d8b5
// 3. https://polygonscan.com/tx/0xec1537d503700c26366b1118881c81d542c77b96dabbd6e89184e3b520ee6f74
// 4. https://polygonscan.com/tx/0x62f47ccfe88879075735ad940384b2ebf87e237cce744824021299337c9c936e
// 2. https://polygonscan.com/tx/0x0ba34f745afdc3f199bf2094bbb85adaa04dddcd3aebd338ea9b457295bfe3f9
// // Contract creation
// 5. https://polygonscan.com/tx/0xc688ae356bce4b8f67797246939e35da6b1f3d84e280d273f6785c9659cd1a84
// // Uniswap V3
// 6. https://polygonscan.com/tx/0x59be9124aa4f605b7554f4897ea8407c660a05f1b3b2d5f5d4f55628471992e6

// TRASH polygonMethods is a map that holds the method IDs of different functions
var polygonMethods = map[string]string{
	"0x18cbafe5": "swapExactTokensForETH",
	"0x7ff36ab5": "swapExactETHForTokens",
	"0x791ac947": "swapExactTokensForETHSupportingFeeOnTransferTokens",
	"0x5c11d795": "swapExactTokensForTokensSupportingFeeOnTransferTokens",
	"0x38ed1739": "swapExactTokensForTokens",
	"0x8803dbee": "swapTokensForExactTokens",
}

// GLOBAL CONSTANTS
var CHAIN_ID = big.NewInt(137)
var ZERO_BIG_INT = big.NewInt(0)

type Token interface {
	BalanceOf(owner common.Address) (*big.Int, error)
}

type ERC20Token struct {
	address  common.Address
	contract *bind.BoundContract
	client   *ethclient.Client
	decimals *int32
}

func NewERC20Token(address common.Address, client *ethclient.Client, abi abi.ABI, decimals *int32) *ERC20Token {
	return &ERC20Token{
		address:  address,
		contract: bind.NewBoundContract(address, abi, client, client, client),
		client:   client,
		decimals: decimals,
	}
}

func (t *ERC20Token) BalanceOf(owner common.Address) ([]interface{}, error) {
	balance := []interface{}{}
	err := t.contract.Call(nil, &balance, "balanceOf", owner)
	if err != nil {
		return nil, err
	}
	return balance, nil
}
func GetTokenDecimals(client *ethclient.Client, tokenAddress common.Address) (*int32, error) {
	// Define the contract ABI for the decimals function
	if client == nil {
		p := Polygon{}
		p.GetNode(true)

		nodeSupportPool, ok := p.NodeSupportPool.([]string)
		if !ok {
			log.Fatal("Failed to assert NodeSupportPool as []string")
			return nil, errors.New("Failed to assert NodeSupportPool as []string")
		}

		node := nodeSupportPool[rand.Intn(len(nodeSupportPool))]
		client = p.GetClient(node)
		defer client.Close()
	}

	const decimalsABI = `[{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"}]`

	// Parse the ABI to create an ABI instance
	parsedABI, err := abi.JSON(strings.NewReader(decimalsABI))
	if err != nil {
		return nil, err
	}

	// Create a call message
	callMsg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: parsedABI.Methods["decimals"].ID,
	}

	// Call the contract
	result, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return nil, err
	}

	// Decode the result
	decimalSlice, err := parsedABI.Methods["decimals"].Outputs.Unpack(result)
	if err != nil {
		return nil, err
	}

	if len(decimalSlice) == 0 {
		return nil, errors.New("decimalSlice is empty")
	}
	firstElement, ok := decimalSlice[0].(uint8)
	if !ok {
		return nil, errors.New("type assertion to uint8 failed")
	}
	decimalValue := int32(firstElement)

	return &decimalValue, nil
}

// Approve spendings
func (t *ERC20Token) Approve(spender common.Address, signer *bind.TransactOpts, value *big.Int, nonce *uint64) (*types.Transaction, error) {
	if nonce != nil {
		signer.Nonce = big.NewInt(int64(*nonce))
		// signer.GasTipCap = big.NewInt(150)
	}
	return t.contract.Transact(signer, "approve", spender, value)
}

// Wrapper to revoke approvement
func (t *ERC20Token) Revoke(spender common.Address, signer *bind.TransactOpts) (*types.Transaction, error) {
	return t.contract.Transact(signer, "approve", spender, ZERO_BIG_INT)
}

func (t *ERC20Token) Allowance(owner, spender common.Address) (allowance *big.Int, err error) {
	var result []interface{}
	if err = t.contract.Call(nil, &result, "allowance", owner, spender); err != nil {
		return nil, err
	}
	// fmt.Println("RESULT", result[0])
	allowance = result[0].(*big.Int)
	return allowance, err
}

// Helper to retrieve token balance
func RetrieveERC20Balance(p Polygon, client *ethclient.Client, walletAddress, address string, decimals int32) (decimal.Decimal, *big.Int, *ERC20Token) {
	erc20ContractABIString, err := p.LoadABI("erc20")
	if err != nil {
		log.Printf("Failed to read ERC-20 contract ABI: %v", err)
		return decimal.NewFromInt(0), nil, nil
	}

	erc20ABI, err := abi.JSON(strings.NewReader(erc20ContractABIString))
	if err != nil {
		log.Printf("Failed to parse ERC-20 contract ABI: %v", err)
		return decimal.NewFromInt(0), nil, nil
	}

	tokenAddress := common.HexToAddress(address)
	erc20Token := NewERC20Token(tokenAddress, client, erc20ABI, &decimals)

	balance, err := erc20Token.BalanceOf(common.HexToAddress(walletAddress))
	if err != nil {
		log.Printf("Failed to get balance: %v, %s", err, address)
		return decimal.NewFromInt(0), nil, nil
	}

	// log.Printf("Bot Wallet Balance: %s, %s", balance, address)
	// fmt.Println("BALANCE BIG INT", balance[0].(*big.Int))

	if len(balance) > 0 {
		return decimal.NewFromBigInt(balance[0].(*big.Int), -decimals), balance[0].(*big.Int), erc20Token
	} else {
		return decimal.NewFromBigInt(big.NewInt(0), -decimals), big.NewInt(0), erc20Token
	}
}

func (p Polygon) PreApproveERC20TokensForDexs(client *ethclient.Client, _walletAddress string) {
	if client == nil {
		nodeSupportPool, ok := p.NodeSupportPool.([]string)
		if !ok {
			log.Fatal("Failed to assert NodeSupportPool as []string")
		} else {
			node := nodeSupportPool[rand.Intn(len(nodeSupportPool))]
			client = p.GetClient(node)
		}
	}

	for _coinName, _coinData := range GlobalSettings.Polygon.Coins {
		for _dexType, _dexAddress := range GlobalSettings.Polygon.DEXs {
			fmt.Println(_coinName, ": preapproving coin for", _dexType)
			decimalsInt32 := _coinData[1].(int32)
			// decimalsInt32 := int32(decimals.Coefficient().Int64())

			// go func(_coinData []interface{}, _dexAddress string, decimalsInt32 int32) {
			PreApproveERC20Token(p, client, _walletAddress, _coinData[0].(string), _dexAddress, decimalsInt32, nil, nil)
			// }(_coinData, _dexAddress, decimalsInt32)
		}
	}
}

func (p Polygon) PreApproveERC20ContractsForDexs(client *ethclient.Client, _walletAddress string, amount *int64) {
	if client == nil {
		nodeSupportPool, ok := p.NodeSupportPool.([]string)
		if !ok {
			log.Fatal("Failed to assert NodeSupportPool as []string")

		}

		node := nodeSupportPool[rand.Intn(len(nodeSupportPool))]
		client = p.GetClient(node)
	}

	// _nonce, err := client.NonceAt(context.Background(), common.HexToAddress(_walletAddress), nil)
	// if err != nil {
	// 	log.Printf("Failed to get nonce: %v", err)
	// 	return
	// }

	for _contractAddress, _contractData := range GlobalSettings.Polygon.Contracts.Whitelist {
		// _decimals, err := GetTokenDecimals(client, common.HexToAddress(_contractAddress))
		// if err != nil {
		// 	log.Printf("Failed to get decimals for contract: %v", err)
		// 	continue
		// }

		decimalsInt32 := _contractData.Decimals

		for _dexType, _dexAddress := range GlobalSettings.Polygon.DEXs {
			fmt.Println(_contractAddress, ": preapproving contract for", _dexType)

			// decimals := decimal.NewFromInt32(*_decimals)

			// decimalsInt32 := int32(decimals.Coefficient().Int64())

			// go func(_contractAddress, _dexAddress string) {

			PreApproveERC20Token(p, client, _walletAddress, _contractAddress, _dexAddress, *decimalsInt32, amount, nil)

			// }(_contractAddress, _dexAddress)
		}
	}
}

var balanceSimpleLock = false
var balanceMutex sync.Mutex

func WalletKnownBalances(p Polygon, client *ethclient.Client) {
	var wg sync.WaitGroup
	balanceSimpleLock = true

	for _, _walletAddress := range GlobalSettings.Polygon.Wallets.Main {
		wg.Add(1)
		go func(_walletAddress models.Wallet) {
			defer wg.Done()

			maticBalance, maticBalanceBigInt, err := RetrieveBalance(client, *_walletAddress.Address)
			if err != nil {
				log.Printf("Failed to retrieve Matic balance: %v", err)
				return
			}
			if GlobalSettings.Polygon.WalletBalance == nil {
				GlobalSettings.Polygon.WalletBalance = map[string]map[string]Balance{}
			}

			if GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address] == nil {
				GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address] = map[string]Balance{}
			}

			balanceMutex.Lock()
			GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address]["matic"] = Balance{
				Decimal:    maticBalance,
				BigInt:     maticBalanceBigInt,
				ERC20Token: nil,
			}
			balanceMutex.Unlock()

			for _, _coinData := range GlobalSettings.Polygon.Coins {
				wg.Add(1)
				go func(_coinData []interface{}) {
					defer wg.Done()

					_coinAddress := _coinData[0].(string)
					decimalsInt32 := _coinData[1].(int32)

					erc20Balance, erc20BalanceBigInt, erc20Token := RetrieveERC20Balance(p, client, *_walletAddress.Address, _coinAddress, decimalsInt32)

					balanceMutex.Lock()
					GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address][_coinAddress] = Balance{
						Decimal:    erc20Balance,
						BigInt:     erc20BalanceBigInt,
						ERC20Token: erc20Token,
					}
					balanceMutex.Unlock()
				}(_coinData)
			}
			for _, _wContracts := range GlobalSettings.Polygon.Contracts.Whitelist {
				wg.Add(1)
				go func(_wContracts Contract) {
					defer wg.Done()

					erc20Balance, erc20BalanceBigInt, erc20Token := RetrieveERC20Balance(p, client, *_walletAddress.Address, *_wContracts.Address, *_wContracts.Decimals)

					balanceMutex.Lock()
					GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address][*_wContracts.Address] = Balance{
						Decimal:    erc20Balance,
						BigInt:     erc20BalanceBigInt,
						ERC20Token: erc20Token,
					}
					balanceMutex.Unlock()
				}(_wContracts)

			}
		}(_walletAddress)
	}

	go func() {
		wg.Wait()
		balanceSimpleLock = false
	}()
}

var allowanceMutex sync.Mutex

func WalletKnownAllowances(p Polygon, client *ethclient.Client) {
	var wg sync.WaitGroup
	balanceSimpleLock = true

	for _, _walletAddress := range GlobalSettings.Polygon.Wallets.Main {
		wg.Add(1)
		go func(_walletAddress models.Wallet) {
			defer wg.Done()

			maticBalance, maticBalanceBigInt, err := RetrieveBalance(client, *_walletAddress.Address)
			if err != nil {
				log.Printf("Failed to retrieve Matic balance: %v", err)
				return
			}
			if GlobalSettings.Polygon.WalletBalance == nil {
				GlobalSettings.Polygon.WalletBalance = map[string]map[string]Balance{}
			}

			if GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address] == nil {
				GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address] = map[string]Balance{}
			}

			allowanceMutex.Lock()
			GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address]["matic"] = Balance{
				Decimal:    maticBalance,
				BigInt:     maticBalanceBigInt,
				ERC20Token: nil,
			}
			allowanceMutex.Unlock()

			for _, _coinData := range GlobalSettings.Polygon.Coins {
				wg.Add(1)
				go func(_coinData []interface{}) {
					defer wg.Done()

					_coinAddress := _coinData[0].(string)
					decimalsInt32 := _coinData[1].(int32)

					erc20Balance, erc20BalanceBigInt, erc20Token := RetrieveERC20Balance(p, client, *_walletAddress.Address, _coinAddress, decimalsInt32)

					allowanceMutex.Lock()
					GlobalSettings.Polygon.WalletBalance[*_walletAddress.Address][_coinAddress] = Balance{
						Decimal:    erc20Balance,
						BigInt:     erc20BalanceBigInt,
						ERC20Token: erc20Token,
					}
					allowanceMutex.Unlock()
				}(_coinData)
			}
		}(_walletAddress)
	}

	go func() {
		wg.Wait()
		balanceSimpleLock = false
	}()
}

func TokenBalance(p Polygon, client *ethclient.Client, walletAddress, tokenAddress string, decimals int32) (tokenBalance *Balance) {
	// fmt.Println("TOKEN BALANCE", walletAddress, tokenAddress)
	var _tokenBalance = GlobalSettings.Polygon.WalletBalance[walletAddress][tokenAddress]
	// fmt.Println("TOKEN BALANCE", _tokenBalance)

	tokenBalance = &_tokenBalance
	go func() {
		if balanceSimpleLock {
			return
		}
		balanceSimpleLock = true
		WalletKnownBalances(p, client)
	}()

	return tokenBalance
}

func PreApproveERC20Token(p Polygon, client *ethclient.Client, walletAddress, tokenAddress, dexRouter string, decimals int32, amount *int64, _nonce *uint64) {
	erc20Balance, _, erc20Token := RetrieveERC20Balance(p, client, walletAddress, tokenAddress, decimals)

	// fmt.Println("ERC20 BALANCE", erc20Balance)
	botMainWalletAddress := common.HexToAddress(walletAddress)
	dexRouterAddress := common.HexToAddress(dexRouter)

	// walletAddress, tokenAddress, dexRouter = strings.ToLower(walletAddress), strings.ToLower(tokenAddress), strings.ToLower(dexRouter)

	if GlobalSettings.Polygon.WalletAllowance == nil {
		GlobalSettings.Polygon.WalletAllowance = map[string]map[string]map[string]Allowance{}
	}
	if GlobalSettings.Polygon.WalletAllowance[walletAddress] == nil {
		GlobalSettings.Polygon.WalletAllowance[walletAddress] = map[string]map[string]Allowance{}
	}
	if GlobalSettings.Polygon.WalletAllowance[walletAddress][tokenAddress] == nil {
		GlobalSettings.Polygon.WalletAllowance[walletAddress][tokenAddress] = map[string]Allowance{}
	}

	// if _pendingNonce, err = client.PendingNonceAt(context.Background(), common.HexToAddress(*GlobalSettings.Polygon.Wallets.Main.Address)); err != nil {
	// 	log.Printf("Failed to get pending nonce: %v", err)
	// 	return
	// }

	if amount == nil {
		drawDown := GlobalSettings.Polygon.Settings.DrawDown

		if erc20Balance.Equals(decimal.Zero) {
			log.Println("balance is zero")
			return
		}

		erc20Allowance, err := erc20Token.Allowance(botMainWalletAddress, dexRouterAddress)
		if err != nil {
			log.Fatalf("Failed to get allowance: %v, %s", err, botMainWalletAddress)
		}

		// fmt.Println("ERC20 ALLOWANCE", erc20Allowance)

		allowance := decimal.NewFromBigInt(erc20Allowance, -decimals)
		allowancePercentage := allowance.Div(erc20Balance).Mul(decimal.NewFromInt(100))

		drawDownDecimal := decimal.NewFromFloat(drawDown) //.Sub(decimal.NewFromInt(10))
		if allowancePercentage.LessThan(drawDownDecimal.Sub(decimal.NewFromInt(10))) {
			fmt.Printf("Allowance percentage %s%% is less than the drawdown limit %s%%. Pre-approval required.\n", allowancePercentage, drawDownDecimal)

			// Calculate the drawdown amount from the erc20Balance
			drawdownAmount := erc20Balance.Mul(drawDownDecimal).Div(decimal.NewFromInt(100)) //.Mul(decimal.NewFromInt(int64(decimals) * 10))
			drawDownPseudoBigInt := drawdownAmount.Mul(decimal.NewFromInt(int64(math.Pow(10, float64(decimals)))))
			drawdownAmountBigInt := drawDownPseudoBigInt.BigInt()

			// Approve the drawdown amount
			fmt.Println(walletAddress)
			fmt.Println(p.Auth[walletAddress])
			tx, err := erc20Token.Approve(dexRouterAddress, p.Auth[walletAddress], drawdownAmountBigInt, nil)
			if err != nil {
				log.Fatalf("Failed to approve ERC20 token: %v", err)
			}

			// Wait for the approval transaction to be confirmed
			// ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			// defer cancel()

			reciept, err := bind.WaitMined(context.Background(), client, tx)

			fmt.Println(err)
			fmt.Println(reciept)

			erc20Allowance = drawdownAmountBigInt
			allowance = decimal.NewFromBigInt(erc20Allowance, -decimals)
			log.Printf("Approval transaction hash: %s", tx.Hash().Hex())
		} else {
			log.Printf("Allowance percentage %s%% meets or exceeds the drawdown limit %s%%. No pre-approval required.", allowancePercentage, drawDownDecimal)
		}

		allowanceMutex.Lock()
		GlobalSettings.Polygon.WalletAllowance[walletAddress][tokenAddress][dexRouter] = Allowance{
			Decimal: allowance,
			BigInt:  erc20Allowance,
			// DEX:     dexRouter,
		}

		allowanceMutex.Unlock()
	} else {
		erc20Allowance, err := erc20Token.Allowance(botMainWalletAddress, dexRouterAddress)
		if err != nil {
			log.Fatalf("Failed to get allowance: %v, %s", err, botMainWalletAddress)
		}

		// fmt.Println(erc20Allowance)

		_amountToApproveDecimal := decimal.NewFromInt(*amount).Mul(decimal.NewFromInt(int64(math.Pow(10, float64(decimals)))))
		amountToApprove := _amountToApproveDecimal.BigInt()
		// Compare erc20Allowance to amountToApprove

		halfAmountToApprove := new(big.Int).Div(amountToApprove, big.NewInt(2))

		// log.Println("HALF AMOUNT TO APPROVE", halfAmountToApprove, erc20Allowance)

		if erc20Allowance.Cmp(halfAmountToApprove) < 0 {
			// fmt.Println("NONCE", *_nonce)

			tx, err := erc20Token.Approve(dexRouterAddress, p.Auth[walletAddress], amountToApprove, _nonce)
			if err != nil {
				log.Fatalf("Failed to approve ERC20 token: %v", err)
			}

			// Wait for the approval transaction to be confirmed
			// ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			// defer cancel()
			bind.WaitMined(context.Background(), client, tx)
			log.Printf("Approval transaction hash: %s", tx.Hash().Hex())
		}

		allowanceMutex.Lock()
		GlobalSettings.Polygon.WalletAllowance[walletAddress][tokenAddress][dexRouter] = Allowance{
			Decimal: _amountToApproveDecimal,
			BigInt:  amountToApprove,
			// DEX:     dexRouter,
		}
		allowanceMutex.Unlock()
	}
}

func RetrieveBalance(client *ethclient.Client, walletAddress string) (decimal.Decimal, *big.Int, error) {
	balance, err := client.BalanceAt(context.Background(), common.HexToAddress(*GlobalSettings.Polygon.Wallets.Main[0].Address), nil)
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
		return decimal.NewFromInt(0), nil, err
	}

	return decimal.NewFromBigInt(balance, -18), balance, nil
}

type Polygon struct {
	Node            interface{}                   `json:"node"`
	NodePool        interface{}                   `json:"node_pool"`
	NodeSupportPool interface{}                   `json:"node_support_pool"`
	Client          []*ethclient.Client           `json:"client"`
	Auth            map[string]*bind.TransactOpts `json:"auth"`
	Nonce           *uint64                       `json:"nonce"`
}

type Nodes struct {
	Polygon        []string `json:"polygon"`
	PolygonTest    []string `json:"polygon_test"`
	PolygonSupport []string `json:"polygon_support"`
}

func ReadJson(filename string) (Nodes, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return Nodes{}, err
	}

	var result Nodes
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return Nodes{}, err
	}

	return result, nil
}

type txAmount struct {
	amount    *big.Int
	amountMin *big.Int
	amountMax *big.Int
}

func WalletNonceSync(p Polygon, client *ethclient.Client, walletAddress string) (err error) {
	if client == nil {
		nodeSupportPool, ok := p.NodeSupportPool.([]string)
		if !ok {
			log.Fatal("Failed to assert NodeSupportPool as []string")
		} else {
			node := nodeSupportPool[rand.Intn(len(nodeSupportPool))]
			client = p.GetClient(node)
		}
	}

	walletAddress = strings.ToLower(walletAddress)

	var _pendingNonce, _nonce uint64

	if _pendingNonce, err = client.PendingNonceAt(context.Background(), common.HexToAddress(walletAddress)); err != nil {
		return fmt.Errorf("Failed to get pending nonce: %v", err)
	}

	if _nonce, err = client.NonceAt(context.Background(), common.HexToAddress(walletAddress), nil); err != nil {
		return fmt.Errorf("Failed to get nonce: %v", err)
	}

	if GlobalSettings.Polygon.WalletNonce == nil {
		GlobalSettings.Polygon.WalletNonce = map[string]Nonce{
			walletAddress: {
				Nonce:        &_nonce,
				PendingNonce: &_pendingNonce,
			},
		}
	}
	fmt.Println("Wallet: ", walletAddress, "\n", "Nonce: ", _nonce, "\n", "Pending nonce: ", _pendingNonce)

	return nil
}

func (p *Polygon) AnalyzeTx(tx *types.Transaction, client *ethclient.Client, args ...interface{}) {
	// mutex2.Lock()
	// if _, exists := duplicateTxHashes[tx.Hash().Hex()]; exists {
	// 	return
	// }
	// mutex2.Unlock()

	// loger
	// fmt.Println(time.Now().Format("2006-01-02T15:04:05.999Z07:00"), tx.Hash().Hex())
	// loger

	if tx.To() == nil {
		// log.Println("===========================================================")
		// log.Println("Contract Creation: skipping")
		// log.Println("===========================================================")
		return
	}

	dex, _dexRouter, contains := utils.MapContains(GlobalSettings.Polygon.DEXs, tx.To().String())
	if !contains {
		// log.Printf("Tx routing swap through unknown dex %s", tx.To().String())
		// log.Println("===========================================================")
		return
	}
	// if true {
	// 	return
	// }
	// if strings.Contains(*dex, "v3") {
	// 	return
	// }

	dexRouter := common.HexToAddress(*_dexRouter)
	// logger
	log.Println("===========================================================")

	log.Printf("Using known dex: %s. Router: %s\n", *dex, dexRouter)
	// logger
	var err error

	// _nonce := args[0].(uint64)
	// walletToAttackWith := args[1].(*common.Address)
	// privateKey := args[2].(*ecdsa.PrivateKey)

	// fmt.Println("Allowance", GlobalSettings.Polygon.WalletAllowance)
	// fmt.Println("Balance", GlobalSettings.Polygon.WalletBalance)

	// preapprovement check
	// p.PreApproveERC20TokensForDexs(nil)

	// var dexContractABIString string
	// if dexContractABIString, err = p.LoadABI(*dex); err != nil {
	// 	log.Printf("Failed to read DEX contract ABI: %v", err)
	// 	log.Println("===========================================================")
	// 	return
	// }

	// var parsedABI abi.ABI
	// if parsedABI, err = abi.JSON(strings.NewReader(dexContractABIString)); err != nil {
	// 	log.Printf("Failed to parse DEX contract ABI: %v", err)
	// 	log.Println("===========================================================")
	// 	return
	// }

	var parsedABI = GlobalSettings.Polygon.ABI[*dex]

	if len(tx.Data()) < 4 {
		return
	}

	methodID := tx.Data()[:4]
	method, err := parsedABI.MethodById(methodID)
	if err != nil {
		methodIDStr := hex.EncodeToString(methodID)
		log.Println(tx.Hash().Hex())
		if polygonMethods["0x"+methodIDStr] != "" {
			log.Println("METHOD", methodIDStr, polygonMethods["0x"+methodIDStr], tx.Hash())
		}
		log.Printf("Method doesn't exsits in ABI: %v", err)
		log.Println("===========================================================")
		return
	}

	inputs := map[string]interface{}{}
	err = method.Inputs.UnpackIntoMap(inputs, tx.Data()[4:])
	if err != nil {
		log.Printf("Failed to unpack inputs: %v", err)
		log.Println("===========================================================")
		return
	}

	// if strings.Contains(method.Name, "swapExactTokensForTokens") || strings.Contains(method.Name, "swapTokensForExactTokens") || strings.Contains(method.Name, "swapExactETHForTokens") || strings.Contains(method.Name, "swapTokensForExactETH") {
	if strings.Contains(method.Name, "swap") || strings.Contains(method.Name, "exact") {
		// logger
		log.Printf("Checking tx %s\n", tx.Hash().Hex())
		log.Printf("Exact method: %s\n", method.Name)
		// logger
		type Params struct {
			TokenIn           common.Address `json:"tokenIn"`
			TokenOut          common.Address `json:"tokenOut"`
			Path              []uint8        `json:"path"`
			Fee               int            `json:"fee"`
			Recipient         common.Address `json:"recipient"`
			Deadline          *big.Int       `json:"deadline"`
			AmountIn          *big.Int       `json:"amountIn"`
			AmountInMinimum   *big.Int       `json:"amountInMinimum"`
			AmountInMaximum   *big.Int       `json:"amountInMaximum"`
			AmountOut         *big.Int       `json:"amountOut"`
			AmountOutMinimum  *big.Int       `json:"amountOutMinimum"`
			AmountOutMaximum  *big.Int       `json:"amountOutMaximum"`
			SqrtPriceLimitX96 *big.Int       `json:"sqrtPriceLimitX96,omitempty"`
			LimitSqrtPrice    *big.Int       `json:"limitSqrtPrice,omitemtpy"`
		}

		paramsBytes, err := json.Marshal(inputs["params"])
		if err != nil {
			log.Printf("Failed to marshal params: %v", err)
		}

		var params *Params
		if err := json.Unmarshal(paramsBytes, &params); err != nil {
			log.Printf("Failed to unmarshal into Params struct: %v", err)
		}

		// TODO: ability to ignore empty AmountIN or(!) empty AmountOut
		var ok bool
		var amountIn txAmount
		amountIn.amount, ok = inputs["amountIn"].(*big.Int)
		if !ok {
			amountIn.amountMax, ok = inputs["amountInMax"].(*big.Int)
			if !ok {
				amountIn.amountMin, ok = inputs["amokuntInMin"].(*big.Int)
				if !ok {
					if params != nil {
						amountIn.amount = params.AmountIn
						amountIn.amountMin = params.AmountInMinimum
						amountIn.amountMax = params.AmountInMaximum
					} else {
						// Risky txs
						// if true {
						if false {
							fmt.Println("This target tx using matic for swap", tx.Value())
							Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to assert amountIn from params.\nTxHash: %s\nInputs: %v\n", tx.Hash().Hex(), inputs), true)
							return
						}
					}
				}
			}
		}

		var amountOut txAmount
		amountOut.amount, ok = inputs["amountOut"].(*big.Int)
		if !ok {
			amountOut.amountMax, ok = inputs["amountOutMax"].(*big.Int)
			if !ok {
				amountOut.amountMin, ok = inputs["amountOutMin"].(*big.Int)
				if !ok {
					if params != nil {
						amountOut.amount = params.AmountOut
						amountOut.amountMin = params.AmountOutMinimum
						amountOut.amountMax = params.AmountOutMaximum
					} else {
						Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to assert amountOut from params.\nTxHash: %s\nInputs: %v\n", tx.Hash().Hex(), inputs), true)
						return

					}
				}
			}
		}

		path, ok := inputs["path"].([]common.Address)
		if !ok {
			if params != nil {
				if len(params.Path) > 0 {
					hexString := hex.EncodeToString(params.Path)
					tokenLength := 40
					feeTierLength := 6
					var feeTiers []string
					i := 0
					for i < len(hexString) {
						if len(hexString[i:]) >= tokenLength {
							path = append(path, common.HexToAddress("0x"+hexString[i:i+tokenLength]))
							i += tokenLength
						}
						if len(hexString[i:]) >= feeTierLength {
							feeTierHex := hexString[i : i+feeTierLength]
							feeTierDec, err := strconv.ParseInt(feeTierHex, 16, 32)
							if err != nil {
								Logger(tx, method, &GlobalSettings, fmt.Sprintf("Error parsing fee tier: %v", err), true)
								return
							}
							feeTiers = append(feeTiers, fmt.Sprintf("%d", feeTierDec))
							i += feeTierLength
						}
					}
				} else {
					path = []common.Address{params.TokenIn, params.TokenOut}
				}
			} else {
				Logger(tx, method, &GlobalSettings, "Failed to assert path as []common.Address", true)
				return
			}
		}

		if len(path) > 0 {
			// logger
			_pathLog := ""
			for _i, _p := range path {
				if _i == len(path)-1 {
					_pathLog += _p.Hex()
				} else {
					_pathLog += _p.Hex() + " -> "
				}
			}

			// logger
			log.Printf("Tx swap path %v\n", _pathLog)
			// logger
			if len(path) > 2 {
				log.Println("Probably bundled transaction")
				// return
				// Check weitd case #1
			}

			// Contract target tx is trying to buy
			contract := strings.ToLower(path[len(path)-1].Hex())

			if _, exists := GlobalSettings.Polygon.Contracts.BlackList[contract]; exists {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Tx Contract %s is in the blacklist. Skipping transaction.", contract), true)
				return
			}

			whitelistedContract, exists := GlobalSettings.Polygon.Contracts.Whitelist[contract]
			if !exists {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Tx Contract %s is not in the whitelist. Skipping transaction.", contract), true)
				return
			}

			coin, coinAddress, coinDecimals, allowed := utils.MapContainsV2(GlobalSettings.Polygon.Coins, path[0].String())
			if !allowed {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Tx trading coin that is not supported yet. Please add wallet with that coin to metamask and log this wallet data to database. Coin: %s", path[0].String()), true)
				return
			}

			// logger
			log.Printf("Tx using %s to swap\n", *coin)
			// logger

			// var wg sync.WaitGroup
			// var maticBalance, erc20CoinBalance, erc20ContractTokenBalance decimal.Decimal
			// var maticBalanceBigInt, erc20CoinBalanceBigInt, erc20ContractTokenBalanceBigInt *big.Int
			// var erc20Coin, erc20ContractToken *ERC20Token
			// var contractDecimals *int32

			// erc20ContractABIString, err := p.LoadABI("erc20")
			// if err != nil {
			// 	Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to read erc-20 ABI file: %v", err), true)
			// 	return
			// }

			// erc20ABI, err := abi.JSON(strings.NewReader(erc20ContractABIString))
			// if err != nil {
			// 	Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to parse ERC-20 contract ABI: %vv", err), true)
			// 	return
			// }

			// maticBalance = decimal.NewFromFloat32(6.386825403753865241)
			// maticBalanceBigInt = big.NewInt(6386825403753865241)
			// erc20CoinBalance = decimal.NewFromFloat(6.64273)
			// erc20CoinBalanceBigInt = big.NewInt(6642730)
			// erc20Coin = NewERC20Token(common.HexToAddress(*address), client, erc20ABI, decimals)
			// _int18 := int32(18)
			// contractDecimals = &_int18
			// contractDecimals, err = GetTokenDecimals(client, common.HexToAddress(contract))
			// if err != nil || contractDecimals == nil {
			// 	Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to retrieve decimals for contract: %s", contract), true)
			// 	return
			// }
			// erc20ContractTokenBalance = decimal.NewFromInt(0)
			// erc20ContractTokenBalanceBigInt = big.NewInt(0)
			// erc20ContractToken = NewERC20Token(common.HexToAddress(*whitelistedContract.Address), client, erc20ABI, whitelistedContract.Decimals)

			// wg.Add(3) // Set the number of goroutines we need to wait for

			// go func() {
			// 	defer wg.Done() // Ensure we signal that this goroutine is done

			// 	// maticBalance, maticBalanceBigInt, err = RetrieveBalance(client, *GlobalSettings.Polygon.Wallets.Main.Address)
			// 	// if err != nil {
			// 	// 	Logger(tx, method, &GlobalSettings, fmt.Sprint("Failed to retrieve Matic balance"), true)
			// 	// 	return
			// 	// }

			// 	maticBalance = decimal.NewFromFloat32(6.386825403753865241)
			// 	maticBalanceBigInt = big.NewInt(6386825403753865241)

			// 	if maticBalance.Equal(decimal.NewFromInt(0)) {
			// 		Logger(tx, method, &GlobalSettings, fmt.Sprintf("Not enough MATIC to cover txs: %v", maticBalance), true)
			// 		return
			// 	}
			// }()
			// go func() {
			// 	defer wg.Done() // Ensure we signal that this goroutine is done
			// 	erc20CoinBalance = decimal.NewFromFloat(6.64273)
			// 	erc20CoinBalanceBigInt = big.NewInt(6642730)
			// 	erc20Coin = NewERC20Token(common.HexToAddress(*address), client, erc20ABI)
			// 	// erc20CoinBalance, erc20CoinBalanceBigInt, erc20Coin = RetrieveERC20Balance(*p, client, *GlobalSettings.Polygon.Wallets.Main.Address, *address, *decimals)

			// 	if erc20Coin == nil {
			// 		Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to retrieve ERC20 balance for coin: %s", contract), true)
			// 		return
			// 	}
			// }()
			// go func() {
			// 	defer wg.Done() // Ensure we signal that this goroutine is done

			// 	// contractDecimals, err = GetTokenDecimals(client, common.HexToAddress(contract))
			// 	// if err != nil || contractDecimals == nil {
			// 	// 	Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to retrieve decimals for contract: %s", contract), true)
			// 	// 	return
			// 	// }

			// 	_int18 := int32(18)
			// 	contractDecimals = &_int18
			// 	// contractDecimals, err = GetTokenDecimals(client, common.HexToAddress(contract))
			// 	// if err != nil || contractDecimals == nil {
			// 	// 	Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to retrieve decimals for contract: %s", contract), true)
			// 	// 	return
			// 	// }
			// 	erc20ContractTokenBalance = decimal.NewFromInt(0)
			// 	erc20ContractTokenBalanceBigInt = big.NewInt(0)
			// 	erc20ContractToken = NewERC20Token(common.HexToAddress(contract), client, erc20ABI)

			// 	// erc20ContractTokenBalance, erc20ContractTokenBalanceBigInt, erc20ContractToken = RetrieveERC20Balance(*p, client, *GlobalSettings.Polygon.Wallets.Main.Address, contract, *contractDecimals)
			// 	if erc20ContractToken == nil {
			// 		Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to retrieve ERC20 balance for contract: %s", contract), true)
			// 		return
			// 	}
			// }()

			// log.Println("DONE, WAITING")
			// wg.Wait() // Wait for all goroutines to finish
			// log.Printf("MATIC balance big int:%v", maticBalanceBigInt)
			// log.Printf("ERC20 balance for coin: %s, Balance: %v, BigInt: %v, Address: %s", *coin, erc20CoinBalance, erc20CoinBalanceBigInt, erc20Coin.address)
			// log.Printf("ERC20 balance for contract: %s, Balance: %v, BigInt: %v, Address: %s", contract, erc20ContractTokenBalance, erc20ContractTokenBalanceBigInt, erc20ContractToken.address)
			// return

			// // Convert coin to priceInfo key
			// var priceInfoKey string
			// switch *coin {
			// case "wmatic":
			// 	priceInfoKey = "matic-network"
			// case "weth":
			// 	priceInfoKey = "ethereum"
			// default:
			// 	priceInfoKey = *coin
			// }

			// // Request price data
			// priceData, err := utils.GetPriceInfo(priceInfoKey)
			// if err != nil {
			// 	log.Printf("Failed to get price info for %s: %v", priceInfoKey, err)
			// 	log.Println("===========================================================")
			// 	return
			// }

			// // Convert amountIn to USD
			// amountInUSD := amountInDecimal.Mul(decimal.NewFromFloat(priceData.Price))

			// // logger
			// log.Printf("Amount in %s: %v\nAmount in USD: %v", *coin, amountInDecimal, amountInUSD)

			// Retrieving Matic Balance to check if we can cover gas fee
			// maticBalance = RetrieveBalance(client, *GlobalSettings.Polygon.Wallets.Main.Address)
			// Calculating wallet balance for the erc20 token we aiming to swap

			// block, err := client.BlockByNumber(context.Background(), nil)
			// if err != nil {
			// 	Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to retrieve the latest block: %v", err), true)
			// 	return
			// }
			// baseNetworkGasFee := block.BaseFee()
			// baseNetworkGasFeeGwei := new(big.Float).Quo(new(big.Float).SetInt(baseNetworkGasFee), big.NewFloat(1e9))
			// log.Printf("Base network gas fee (GWEI): %v, Block number: %v", baseNetworkGasFeeGwei, block.Number().Uint64())

			// tx GasPrice GWEI
			// txGasPriceWei := new(big.Int).Add(tx.GasTipCap(), baseNetworkGasFee)
			txGasPriceWei := tx.GasTipCap()
			txGasPriceGwei := new(big.Float).Quo(new(big.Float).SetInt(txGasPriceWei), big.NewFloat(1e9))
			txGasPriceGweiInt, _ := txGasPriceGwei.Int(nil)
			txGasPriceGweiDecimal := decimal.NewFromBigInt(txGasPriceGweiInt, 0) // Convert to decimal
			// logger
			// Calculate the base network gas fee at the moment of the block

			log.Println("TX GAS PRICE WEI", txGasPriceWei, tx.GasPrice())
			log.Println("TX GAS PRICE GWEI", txGasPriceGwei)
			// logger

			// global settings Gas Fee Max
			fmt.Println("FAST GAS PRICE", utils.GasPriceData.Result.Result.FastGasPrice)
			networkFastGasPrice, err := decimal.NewFromString(utils.GasPriceData.Result.Result.FastGasPrice)
			if err != nil {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to convert fast gas price to decimal: %v", err), true)
				return
			}

			// logger
			log.Printf("Gas fee max GWEI: %v Fast gas price GWEI decimal: %v", GlobalSettings.Polygon.Settings.GasFeeMax, networkFastGasPrice)
			// logger

			// tx GasPrice GWEI compared to network networkFastGasPrice %
			txNetworkGasPriceDifferencePercentage := txGasPriceGweiDecimal.Sub(networkFastGasPrice).Div(txGasPriceGweiDecimal).Mul(decimal.NewFromInt(100))

			// logger
			if txNetworkGasPriceDifferencePercentage.IsPositive() {
				log.Printf("Gas price is %v%% higher than the network's fast gas price", txNetworkGasPriceDifferencePercentage)
			} else {
				log.Printf("Gas price is %v%% lower than the network's fast gas price", txNetworkGasPriceDifferencePercentage.Abs())
			}
			// logger

			// compare txNetworkGasPriceDifferencePercentge to the TargetGasMrkupAllowed for targeted tx, to not try to attack tx, that's running with huge GasPrice
			if txNetworkGasPriceDifferencePercentage.GreaterThan(GlobalSettings.Polygon.Settings.TargetGasMarkupAllowed) {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Gas price difference percentage %v%% is greater than the target gas markup allowed %v%%", txNetworkGasPriceDifferencePercentage, GlobalSettings.Polygon.Settings.TargetGasMarkupAllowed), true)
				return
			} else {
				// logger
				log.Printf("Gas price difference percentage %v%% is less than or equal to the target gas markup allowed %v%%", txNetworkGasPriceDifferencePercentage, GlobalSettings.Polygon.Settings.TargetGasMarkupAllowed)
				// logger
			}

			// Price check
			var amountInDecimal, amountOutDecimal decimal.Decimal
			if amountIn.amount != nil {
				amountInDecimal = decimal.NewFromBigInt(amountIn.amount, -*coinDecimals)
			} else if amountIn.amountMax != nil {
				amountInDecimal = decimal.NewFromBigInt(amountIn.amountMax, -*coinDecimals)
			} else if amountIn.amountMin != nil {
				amountInDecimal = decimal.NewFromBigInt(amountIn.amountMin, -*coinDecimals)
			} else {
				amountInDecimal = decimal.NewFromInt(0)
			}

			if amountOut.amount != nil {
				amountOutDecimal = decimal.NewFromBigInt(amountOut.amount, -*whitelistedContract.Decimals)
			} else if amountOut.amountMax != nil {
				amountOutDecimal = decimal.NewFromBigInt(amountOut.amountMax, -*whitelistedContract.Decimals)
			} else {
				amountOutDecimal = decimal.NewFromBigInt(amountOut.amountMin, -*whitelistedContract.Decimals)
			}

			if amountInDecimal.IsZero() && amountOutDecimal.IsZero() {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Malformed tx"), true)
				return
			}
			if amountOutDecimal.IsZero() {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Error: cannot build local txs due to some parameters missing from the TTX"), true)
				return
			}

			// slippage := decimal.NewFromFloat(GlobalSettings.Polygon.Settings.Slippage)
			var newTxAmountInDecimal, newTxAmountOutDecimal decimal.Decimal
			var newTxAmountInBigInt, newTxAmountOutBigInt *big.Int

			if !amountInDecimal.IsZero() {
				newTxAmountInDecimal = amountInDecimal.Mul(GlobalSettings.Polygon.Settings.UsdPerTrade).Div(decimal.NewFromInt(100))
				newTxAmountInBigInt = newTxAmountInDecimal.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(*coinDecimals)))).BigInt()
			} else {
				newTxAmountInDecimal = decimal.NewFromInt(0)
				newTxAmountInBigInt = big.NewInt(0)
			}

			amountOutTolerance := GlobalSettings.Polygon.Settings.UsdPerTrade.Mul(decimal.NewFromFloat(0.10))
			newTxAmountOutDecimal = amountOutDecimal.Mul(GlobalSettings.Polygon.Settings.UsdPerTrade.Sub(amountOutTolerance)).Div(decimal.NewFromInt(100))
			log.Println("AMOUNT OUT DECIMAL", amountOutDecimal, newTxAmountOutDecimal)
			newTxAmountOutBigInt = newTxAmountOutDecimal.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(*whitelistedContract.Decimals)))).BigInt()

			if amountInDecimal.GreaterThan(GlobalSettings.Polygon.Settings.TargetValueMax) || amountInDecimal.LessThan(GlobalSettings.Polygon.Settings.TargetValueMin) {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("tx amountIn %v is gt target tx value MAX %v or lt target tx value MIN %v", amountInDecimal, GlobalSettings.Polygon.Settings.TargetValueMax, GlobalSettings.Polygon.Settings.TargetValueMin), true)
				return
			}

			// Building values for the upcoming swapExactTokensToTokens
			// Gas fee of target transaction + % of gas tolerance <= gasfee
			globalSettingsGasTolerance := decimal.NewFromFloat(GlobalSettings.Polygon.Settings.GasTolerance)
			log.Println("GLOBAL SETTIGNS GAS TOLERANCE", globalSettingsGasTolerance)
			toleranceAmount := txGasPriceGweiDecimal.Mul(globalSettingsGasTolerance).Div(decimal.NewFromInt(100))
			log.Println("TOLERANCE AMOUNT", toleranceAmount)
			newTxGasFee := txGasPriceGweiDecimal.Add(toleranceAmount)
			log.Println("NEW TX GAS FEE", newTxGasFee)
			log.Println("GAS TIP CAP", tx.GasTipCap())
			// logger
			log.Printf("Calculated newTxGasFee to attack is %v\n. TTX Gas Price: %v. Tolerance: %v", newTxGasFee, txGasPriceGweiDecimal, globalSettingsGasTolerance)
			// logger

			if newTxGasFee.GreaterThan(GlobalSettings.Polygon.Settings.GasFeeMax) {
				// logger
				log.Printf("NewTxGasFee %v to attack is greater than globalSettings.GasFeeMax %v\n", newTxGasFee, GlobalSettings.Polygon.Settings.GasFeeMax)
				// logger
				newTxGasFee = GlobalSettings.Polygon.Settings.GasFeeMax
			}

			var walletToAttackWith *common.Address
			var privateKey *ecdsa.PrivateKey

			for _, _wttx := range GlobalSettings.Polygon.Wallets.Main {
				if _, exists := GlobalSettings.Polygon.WalletTTX[*_wttx.Address]; !exists {

					privateKey, err = utils.HexToECDSAV2(*_wttx.PrivateKey)
					if err != nil {
						// Logger(tx, method, &GlobalSettings, fmt.Sprintf("failed to pack PK to ecdsa. %s", err), true)
						return
					}

					_walletToAttackWith := common.HexToAddress(*_wttx.Address)
					walletToAttackWith = &_walletToAttackWith

					break
				}
			}

			if walletToAttackWith == nil {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Lock on the wallet engaged, skipping tx. TxHash: %s", tx.Hash().Hex()), true)
				return
			}

			// Allowance check
			log.Println("WALLET TO ATTACK WITH", walletToAttackWith.Hex(), walletToAttackWith.String(), strings.ToLower(walletToAttackWith.Hex()))
			walletAllowances, ok := GlobalSettings.Polygon.WalletAllowance[strings.ToLower(walletToAttackWith.Hex())]
			if !ok {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("No wallet allowances found for wallet: %s", walletToAttackWith.Hex()), true)
				if !IsPreApprovementInProgress() {
					PreApprovement(*p)
				}
				return
			}

			fmt.Println("ALLOWANCE INSIDE", walletAllowances[*coinAddress][strings.ToLower(dexRouter.Hex())])
			fmt.Println("CONTRACT INSIDE", coinAddress)
			fmt.Println("DEX INSIDE", dexRouter.Hex())

			coinDexAllowance, ok := walletAllowances[*coinAddress][strings.ToLower(dexRouter.Hex())]
			if !ok {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("No token allowance found for coin: %s and dex: %s", *coinAddress, *dex), true)
				if !IsPreApprovementInProgress() {
					PreApprovement(*p)
				}
				return
			}

			if coinDexAllowance.BigInt.Cmp(newTxAmountInBigInt) < 0 {
				Logger(tx, method, &GlobalSettings, fmt.Sprintf("Not enough allowance to execute an attack. Allowance: %v. AmountOut: %v. TxHash: %s", coinDexAllowance, newTxAmountOutBigInt, tx.Hash().Hex()), true)
				if !IsPreApprovementInProgress() {
					PreApprovement(*p)
				}
				return
			}
			// Goes here

			fmt.Println(*whitelistedContract.Address, *whitelistedContract.Decimals)
			maticBalance := TokenBalance(*p, client, strings.ToLower(walletToAttackWith.Hex()), "matic", int32(18))
			erc20Coin := TokenBalance(*p, client, strings.ToLower(walletToAttackWith.Hex()), *coinAddress, *coinDecimals)
			erc20Contract := TokenBalance(*p, client, strings.ToLower(walletToAttackWith.Hex()), *whitelistedContract.Address, *whitelistedContract.Decimals)

			// erc20Balance, _ := RetrieveERC20Balance(p, client, *GlobalSettings.Polygon.Wallets.Main.Address, *address, *decimals)
			log.Println("MATIC BALANCE:", maticBalance.Decimal, "MATIC BALANCE GWEI:", maticBalance.BigInt, "ERC20 COIN BALANCE", erc20Coin.Decimal, "TARGET TOKEN BALACE", erc20Contract.Decimal)

			// log.Printf("Checking MATIC balance before exec. MATIC Balance: %v, NewTxGasFee: %v, GasPriority: %v, GasLimit: %v\n", maticBalance.BigInt, newTxGasFee, GlobalSettings.Polygon.Settings.GasPriority, GlobalSettings.Polygon.Settings.GasLimit)
			// if maticBalance.Decimal.LessThan(newTxGasFee.Add(GlobalSettings.Polygon.Settings.GasPriority).Add(decimal.NewFromInt(int64(GlobalSettings.Polygon.Settings.GasLimit)))) {
			// 	Logger(tx, method, &GlobalSettings, fmt.Sprintf("Not enough MATIC in the wallet to cover newTx gas fee. Current MATIC balance: %v.NewTxGasFee: %v. Gas Priority: %v. Gas Limit: %v", maticBalance.Decimal, newTxGasFee, GlobalSettings.Polygon.Settings.GasPriority, GlobalSettings.Polygon.Settings.GasLimit), true)
			// 	return
			// }

			// Risky tx
			if !amountInDecimal.IsZero() {
				log.Printf("Checking if erc20Balance %v is less than required newTxAmountInDecimal %v\n", erc20Coin.BigInt, newTxAmountInDecimal)
				if erc20Coin.Decimal.LessThan(newTxAmountInDecimal) {
					Logger(tx, method, &GlobalSettings, fmt.Sprintf("Insufficient ERC20 balance for the transaction. Balance: %v, Required: %v", erc20Coin.Decimal, newTxAmountInDecimal), true)
					return
				}
			}

			// logger
			log.Printf("Preparing to attack. %v %v. AmountIn: %v, amountOutMin: %v\n", *coin, erc20Coin.Decimal, newTxAmountInBigInt, newTxAmountOutBigInt)
			// logger

			log.Println("===========================================================")
			// if strings.Contains(*dex, "v3") {
			// 	Logger(tx, method, &GlobalSettings, "Temp return", true)
			// 	return
			// }
			// return

			go func() {
				_txHash := tx.Hash().Hex()

				if GlobalSettings.Polygon.WalletTTX == nil {
					GlobalSettings.Polygon.WalletTTX = make(map[string]string)
				}

				GlobalSettings.Polygon.WalletTTX[strings.ToLower(walletToAttackWith.Hex())] = _txHash

				nonces, ok := GlobalSettings.Polygon.WalletNonce[strings.ToLower(walletToAttackWith.Hex())]
				if !ok {
					WalletNonceSync(*p, nil, walletToAttackWith.Hex())
					return
				}

				_nonce := nonces.Nonce
				_pendingNonce := nonces.PendingNonce

				log.Printf("Pending nonce: %v", *_pendingNonce)
				log.Printf("Nonce: %v", *_nonce)
				log.Printf("Forced Nonce: %v", *_nonce+1)

				go func(_nonce uint64) {
					fmt.Println("NONCE FRONTRUN", _nonce)
					newTxMethodName := "swapTokensForExactTokens"
					if strings.Contains(strings.ToLower(*dex), "v3") {
						newTxMethodName = "exactOutputSingle"
					}

					p.Swap(&_nonce, newTxMethodName, *walletToAttackWith, dexRouter, erc20Contract.ERC20Token.address, client, erc20Coin.ERC20Token, parsedABI, newTxAmountInBigInt, newTxAmountOutBigInt, newTxGasFee, GlobalSettings.Polygon.Settings.GasPriority, GlobalSettings.Polygon.Settings.GasFeeMax, GlobalSettings.Polygon.Settings.GasLimit, privateKey, CHAIN_ID, &_txHash, false, false)
				}(*_nonce)
				*_nonce++

				go func(_nonce uint64) {
					fmt.Println("NONCE BACKRUN", _nonce)
					// fmt.Println("FAST GAS PRICE", utils.GasPriceData.Result.Result.FastGasPrice)
					// brTxExitGas, err := decimal.NewFromString(GlobalSettings.Polygon.Settings.ExitGas.String())
					if err != nil {
						log.Fatalf("Failed to parse fast gas price: %v", err)
						return
					}

					// brTxExitGas = brTxExitGas.Mul(decimal.NewFromFloat(1.6))
					// brTxExitGas := txGasPriceGweiDecimal.Mul(decimal.NewFromFloat(1))
					exitGasPercentage := GlobalSettings.Polygon.Settings.ExitGas.Div(decimal.NewFromInt(100))
					// brTxExitGas, _ := decimal.NewFromString(utils.GasPriceData.Result.Result.FastGasPrice)
					// brTxExitGas = brTxExitGas.Mul(exitGasPercentage)
					brTxExitGas := txGasPriceGweiDecimal.Mul(exitGasPercentage)
					// brTxExitGas := txGasPriceGweiDecimal.Mul(GlobalSettings.Polygon.Settings.ExitGas.Div(decimal.NewFromInt(100)))

					fmt.Println(txGasPriceGweiDecimal)
					// fmt.Println(tx.GasPrice())
					fmt.Println(brTxExitGas)

					// fmt.Println("BR TX EXIT GAS", brTxExitGas)
					newTxMethodName := "swapExactTokensForTokens"
					// newFRTxMethodName := "exactInputSingle"
					if strings.Contains(strings.ToLower(*dex), "v3") {
						newTxMethodName = "exactInputSingle"
					}

					p.Swap(&_nonce, newTxMethodName, *walletToAttackWith, dexRouter, erc20Coin.ERC20Token.address, client, erc20Contract.ERC20Token, parsedABI, newTxAmountOutBigInt, ZERO_BIG_INT, decimal.NewFromInt(0), brTxExitGas, GlobalSettings.Polygon.Settings.GasFeeMax, GlobalSettings.Polygon.Settings.GasLimit, privateKey, CHAIN_ID, &_txHash, false, false)
					nextNonce := _nonce + 1
					GlobalSettings.Polygon.WalletNonce[strings.ToLower(walletToAttackWith.Hex())] = Nonce{
						Nonce:        &nextNonce,
						PendingNonce: &_nonce,
					}
				}(*_nonce)

				go func() {
					Logger(tx, method, &GlobalSettings, fmt.Sprintf("Attacking tx. TxHash: %s", _txHash), false)
					txReceipt := TxReceipt(tx.Hash(), client)
					if err != nil {
						Logger(tx, method, &GlobalSettings, fmt.Sprintf("Failed to get transaction receipt for TxHash: %s, Error: %s", _txHash, err), true)
						return
					}

					if txReceipt != nil {
						txReceiptJSON, err := json.Marshal(txReceipt)
						if err != nil {
							log.Printf("Error marshalling receipt to JSON: %v", err)
							return
						}
						if err := controllers.DB.Debug().Model(&models.Order{}).Where("hash", tx.Hash().Hex()).Update("receipt", datatypes.JSON(txReceiptJSON)).Error; err != nil {
							log.Printf("Error updating order receipt in db. Hash: %s. err: %v", tx.Hash().Hex(), err)
							return
						}
					}
				}()
			}()
		}
	} else {
		Logger(tx, method, &GlobalSettings, fmt.Sprintf("Unknown method.\nTxHash: %s\nFunction: %s\nInputs: %v", tx.Hash().Hex(), method.Name, inputs), true)
	}
}

func GweiToWei(gwei decimal.Decimal) *big.Int {
	wei := gwei.Mul(decimal.NewFromInt(1e9)).BigInt()
	return wei
}

func (p *Polygon) Authenticator(walletAddress string, privateKey *ecdsa.PrivateKey, chainID *big.Int) {
	var err error

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	// walletAddress1 := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	// fmt.Println("AUTHENTICATOR", auth.Nonce, auth.From, auth, "Wallet Address:", walletAddress1)

	if p.Auth == nil {
		p.Auth = make(map[string]*bind.TransactOpts)
	}

	fmt.Println("WALLET ADDRESS", walletAddress)
	p.Auth[walletAddress] = auth
}

func buildPath(addresses []string, feeTiers []int) []byte {
	var path []byte

	for i, address := range addresses {
		// Convert address to bytes and append to path
		addressBytes := common.HexToAddress(address).Bytes()
		path = append(path, addressBytes...)

		// If there's a corresponding fee tier, convert it to bytes and append to path
		if i < len(feeTiers) {
			feeTierBytes := make([]byte, 3)
			big.NewInt(int64(feeTiers[i])).FillBytes(feeTierBytes)
			path = append(path, feeTierBytes...)
		}
	}

	return path
}

// Flags []bool{legacy, dryRun}
func (p Polygon) Swap(nonce *uint64, method string, ownerWallet, dexRouter, tokenContract common.Address, client *ethclient.Client, erc20Token *ERC20Token, dexAbi abi.ABI, amountIn, amountOut *big.Int, gasPrice, gasPriority, gasFeeMax decimal.Decimal, gasLimit uint64, privateKey *ecdsa.PrivateKey, chainID *big.Int, targetTxHash *string, flags ...bool) {

	// if i > 1 {
	// 	return
	// }
	// i++

	// decide on extra flags
	var legacy, dryRun bool
	if len(flags) > 0 {
		legacy = flags[0]
		if len(flags) > 1 {
			dryRun = flags[1]
		}
	}

	// Signer -> move to sepparate struct function
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	// Tx params
	// Path for dex to know what token to swap for what token
	// path := []common.Address{erc20Token.address, tokenContract}
	path := []common.Address{erc20Token.address, tokenContract} //common.HexToAddress("0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"), erc20Token.address}

	deadline := big.NewInt(time.Now().Add(time.Minute * time.Duration(GlobalSettings.Polygon.Settings.Deadline)).Unix()) // Transaction must be mined within 10 minutes

	// Nonce
	// Get the account's nonce for the next transaction
	// var _pendingNonce uint64
	// if nonce == nil {
	// 	var _nonce uint64
	// 	if _pendingNonce, err = client.PendingNonceAt(context.Background(), ownerWallet); err != nil {
	// 		log.Printf("Failed to get pending nonce: %v", err)
	// 		return
	// 	}

	// 	if _nonce, err = client.NonceAt(context.Background(), ownerWallet, nil); err != nil {
	// 		log.Printf("Failed to get nonce: %v", err)
	// 		return
	// 	}

	// 	nonce = &_nonce

	// }
	// var nonce uint64
	// if nonce, err = client.PendingNonceAt(context.Background(), ownerWallet); err != nil {
	// 	log.Printf("Failed to get nonce: %v", err)
	// 	return
	// }
	// fmt.Println(nonce)
	// ? Redundant
	auth.Nonce = new(big.Int).SetUint64(*nonce)

	// Underlying call of dex contract method
	var data []byte

	auth.GasPrice = GweiToWei(gasPrice)
	auth.GasTipCap = GweiToWei(gasPrice)
	// auth.GasTipCap = GweiToWei(gasPriority)
	auth.GasLimit = gasLimit
	auth.GasFeeCap = GweiToWei(gasFeeMax)

	if strings.Contains(method, "exact") {
		// path := []common.Address{erc20Token.address, tokenContract}
		dex, _, contains := utils.MapContains(GlobalSettings.Polygon.DEXs, strings.ToLower(dexRouter.Hex()))
		if !contains {
			log.Printf("Tx routing swap through unknown dex %s", dexRouter)
			return
		}

		type Params struct {
			TokenIn   common.Address `json:"tokenIn,omitempty"`
			TokenOut  common.Address `json:"tokenOut,omitempty"`
			Recipient common.Address `json:"recipient,omitempty"`
			// Path              []uint8        `json:"path,omitempty"`
			Fee               *big.Int `json:"fee,omitempty"`
			Deadline          *big.Int `json:"deadline"`
			AmountIn          *big.Int `json:"amountIn"`
			AmountOutMinimum  *big.Int `json:"amountOutMinimum"`
			SqrtPriceLimitX96 *big.Int `json:"sqrtPriceLimitX96,omitempty"`
			LimitSqrtPrice    *big.Int `json:"limitSqrtPrice,omitempty"`
			AmountOut         *big.Int `json:"amountOut,omitempty"`
			AmountInMaximum   *big.Int `json:"amountInMaximum,omitempty"`
		}

		params := &Params{
			Recipient: ownerWallet,
			// Fee:              big.NewInt(500),
			Deadline: deadline,
			// AmountIn:         amountIn,
			// AmountOutMinimum: amountOutMin,
			// SqrtPriceLimitX96: ZERO_BIG_INT,
		}

		// if method == "exactOuptut" {

		// 	addresses := []string{
		// 		"0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
		// 		"0x838C9634dE6590B96aEadC4Bc6DB5c28Fd17E3C2",
		// 	}
		// 	feeTiers := []int{500, 3000}

		// 	path := buildPath(addresses, feeTiers)
		// 	params.Path = path

		// 	params.AmountOut = amountOut
		// 	params.AmountInMaximum = amountIn

		// } else {

		params.TokenIn = path[0]
		params.TokenOut = path[1]
		if strings.Contains(method, "Input") {
			params.AmountIn = amountIn
			params.AmountOutMinimum = amountOut
		} else if strings.Contains(method, "Output") {
			params.AmountOut = amountOut
			params.AmountInMaximum = amountIn
			params.Fee = big.NewInt(500)
		}
		if !strings.Contains(*dex, "quick") {
			params.Fee = big.NewInt(500)
			params.SqrtPriceLimitX96 = ZERO_BIG_INT
		} else {
			params.LimitSqrtPrice = ZERO_BIG_INT
		}

		// }

		// routerContract := bind.NewBoundContract(dexRouter, dexAbi, client, client, client)
		// if routerContract == nil {
		// 	log.Printf("Failed to instantiate the router contract: %v", err)
		// 	return
		// }

		log.Println("TX PARAMS", params)

		if data, err = dexAbi.Pack(method, *params); err != nil {
			log.Printf("Failed to pack tx for v3 router: %v", err)
			return
		}

		// tx, err := dexAbi.ExactInputSingle(auth, &types.AccessList{}, tokenIn, tokenOut, amountIn, amountOutMin, deadline)
		// if err != nil {
		// 	log.Fatal(err)
		// }

	} else if strings.Contains(method, "swap") {
		if strings.Contains(method, "ExactTokensForTokens") {
			if data, err = dexAbi.Pack(method, amountIn, ZERO_BIG_INT, path, ownerWallet, deadline); err != nil {
				log.Printf("Failed to pack etft tx for v2 router: %v", err)
				return
			}
		} else if strings.Contains(method, "TokensForExactTokens") {
			if data, err = dexAbi.Pack(method, amountOut, amountIn, path, ownerWallet, deadline); err != nil {
				log.Printf("Failed to pack tfet tx for v2 router: %v", err)
				return
			}
		}
	}

	//

	// Legacy on/off
	var tx *types.Transaction
	var signer types.Signer
	if legacy {
		tx = types.NewTransaction(
			*nonce,
			dexRouter,
			ZERO_BIG_INT,
			auth.GasLimit,
			auth.GasPrice,
			data,
		)
		signer = types.NewEIP155Signer(chainID)
	} else {
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   CHAIN_ID,
			Nonce:     auth.Nonce.Uint64(),
			GasTipCap: auth.GasTipCap,
			GasFeeCap: auth.GasFeeCap,
			Gas:       auth.GasLimit,
			To:        &dexRouter,
			Value:     ZERO_BIG_INT,
			Data:      data,
		})
		signer = types.LatestSignerForChainID(chainID)
	}

	var signedTx *types.Transaction
	if signedTx, err = types.SignTx(tx, signer, privateKey); err != nil {
		log.Printf("Failed to sign transaction: %v", err)
		return
	}

	// Logger
	log.Printf("From: %s", auth.From.Hex())
	log.Printf("To: %s", signedTx.To().Hex())
	log.Printf("Value: %s", signedTx.Value().String())
	log.Printf("Gas limit: %d", signedTx.Gas())
	log.Printf("Gas price: %s", signedTx.GasPrice().String())
	log.Printf("Data: %x", signedTx.Data())
	log.Printf("Nonce: %d", signedTx.Nonce())
	log.Printf("Normal Nonce: %d", nonce)
	// log.Printf("Pneding Nonce: %d", _pendingNonce)
	log.Printf("Path: %v", path)

	if dryRun {
		return
	}

	if err = client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Printf("Failed to send transaction: %v", err)
		return
	}

	signedTxHash := signedTx.Hash().Hex()
	log.Printf("Transaction hash: %v", signedTx.Hash().Hex())

	go func(method string) {

		mutex2.Lock()
		duplicateTxHashes[signedTxHash] = struct{}{}
		mutex2.Unlock()

		_status := models.StatusType("pending")
		_type := models.TransactionType("outbound")
		if !_status.IsValid() || !_type.IsValid() {
			log.Printf("Error: Invalid status or type for txHash: %s", *targetTxHash)
			return
		}
		var orderID uint
		if err := controllers.DB.Debug().Select("id").Where("hash = ?", targetTxHash).Model(&models.Order{}).First(&orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("Order with hash %s not found", *targetTxHash)
				return
			} else {
				log.Printf("Error querying the database: %v", err)
				return
			}
		}

		txDataJSON, err := json.Marshal(signedTx.Data())
		if err != nil {

			delete(GlobalSettings.Polygon.WalletTTX, strings.ToLower(ownerWallet.Hex()))
			log.Printf("Error marshalling transaction data to JSON: %v", err)
			return
		}

		contract := path[1].Hex()

		_tx := models.Transaction{
			Status: models.Status{
				Status: _status,
			},
			Type:     _type,
			Hash:     &signedTxHash,
			Contract: &contract,
			OrderID:  &orderID,
			RawData:  datatypes.JSON(txDataJSON),
		}

		if err := controllers.DB.Debug().Create(&_tx).Error; err != nil {
			log.Printf("Error: failed to write tx to db. Hash: %s. err: %v", *targetTxHash, err)
			return
		}

		receipt := TxReceipt(signedTx.Hash(), client)
		if receipt != nil {
			receiptJSON, err := json.Marshal(receipt)

			if err != nil {
				log.Printf("Error marshalling receipt to JSON: %v", err)
				return
			}

			if err := controllers.DB.Debug().Model(&_tx).Update("receipt", datatypes.JSON(receiptJSON)).Error; err != nil {
				log.Printf("Error updating transaction receipt in db. Hash: %s. err: %v", *targetTxHash, err)
				return
			}

			allowanceMutex.Lock()
			var allowance = GlobalSettings.Polygon.WalletAllowance[strings.ToLower(ownerWallet.Hex())][strings.ToLower(erc20Token.address.Hex())][strings.ToLower(dexRouter.Hex())]
			allowance = Allowance{
				BigInt:  new(big.Int).Sub(allowance.BigInt, amountIn),
				Decimal: allowance.Decimal.Sub(decimal.NewFromBigInt(amountIn, -*erc20Token.decimals)),
			}
			GlobalSettings.Polygon.WalletAllowance[strings.ToLower(ownerWallet.Hex())][strings.ToLower(erc20Token.address.Hex())][strings.ToLower(dexRouter.Hex())] = allowance
			allowanceMutex.Unlock()
		}

		// WalletNonceSync(p, nil, ownerWallet.Hex())
		delete(GlobalSettings.Polygon.WalletTTX, strings.ToLower(ownerWallet.Hex()))

	}(method)
}

func TxReceipt(hash common.Hash, client *ethclient.Client) *types.Receipt {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("Transaction receipt polling timed out: %v", err)
				return nil
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if receipt != nil {
			fmt.Println("===========================================================================")
			log.Printf("Transaction has been mined. Status: %v", receipt.Status)
			log.Printf("Receipt status: %d", receipt.Status)
			log.Printf("Receipt Tx Hash: %s", receipt.TxHash.Hex())
			log.Printf("Receipt Block Hash: %s", receipt.BlockHash.Hex())
			log.Printf("Receipt Block Number: %d", receipt.BlockNumber)
			log.Printf("Receipt Gas Used: %d", receipt.GasUsed)
			log.Printf("Receipt Cumulative Gas Used: %d", receipt.CumulativeGasUsed)
			log.Printf("Receipt Contract Address: %s", receipt.ContractAddress.Hex())
			log.Printf("Receipt Logs: %v", receipt.Logs)
			fmt.Println("===========================================================================")
			return receipt
		}
	}
}

func Queue(pool bool, args ...interface{}) (interface{}, error) {
	nodes := args[0].([]string)
	if len(nodes) > 0 {
		if pool {
			return nodes, nil
		}
		return nodes[0], nil
	}
	return nil, fmt.Errorf("no arguments provided")
}

// func QueueV2(args ...interface{}) ([]interface{}, error) {
// 	nodes := args[0].([]string)
// 	if len(args) > 0 {
// 		return nodes, nil
// 	}
// 	return nil, fmt.Errorf("no arguments provided")
// }

func (p Polygon) ClientPool(nodes interface{}) (clients []*ethclient.Client) {
	// fmt.Println(p.NodePool)
	nodePool, ok := p.NodePool.([]string)
	if !ok {
		log.Fatalf("Failed to assert NodePool as []string")
	}

	for _, _node := range nodePool {
		clients = append(clients, p.GetClient(_node))
	}

	return clients
}

var uniqueTxHashes = make(map[string]struct{})
var duplicateTxHashes = make(map[string]struct{})
var mutex = &sync.Mutex{}
var mutex2 = &sync.Mutex{}
var globalLock = &sync.Mutex{}
var preApprovementInProgress = false

func PreApprovement(p Polygon) {
	// globalLock.Lock()
	// preApprovementInProgress = true
	// globalLock.Unlock()

	// defer func() {
	// 	globalLock.Lock()
	// 	preApprovementInProgress = false
	// 	globalLock.Unlock()
	// }()

	var wg sync.WaitGroup
	for _, _wallet := range GlobalSettings.Polygon.Wallets.Main {
		wg.Add(1)
		// fmt.Println("WALLETS", _wallet)
		go func(_wallet models.Wallet) {
			defer wg.Done()
			if _wallet.Address != nil && _wallet.PrivateKey != nil {
				privateKey, err := utils.HexToECDSAV2(*_wallet.PrivateKey)
				if err != nil {
					log.Printf("failed to pack PK to ecdsa. %s", err)
					return
				}
				p.Authenticator(*_wallet.Address, privateKey, CHAIN_ID)
				// // // preapprovement check
				p.PreApproveERC20TokensForDexs(nil, *_wallet.Address)
				amount := int64(500000000)
				p.PreApproveERC20ContractsForDexs(nil, *_wallet.Address, &amount)

				WalletNonceSync(p, nil, *_wallet.Address)
			}
		}(_wallet)
	}
	wg.Wait()
}

func IsPreApprovementInProgress() bool {
	globalLock.Lock()
	defer globalLock.Unlock()
	return preApprovementInProgress
}

func (p Polygon) ScanMempoolV2(callbacks ...interface{}) {
	if GlobalSettings.KillSwitch.IsOn != nil && *GlobalSettings.KillSwitch.IsOn {
		log.Print("KillSwitch is on")
		return
	}

	p.GetNode(true)
	clientPool := p.ClientPool(p.NodePool)

	if len(GlobalSettings.Polygon.Wallets.Main) == 0 {
		log.Printf("Error: Main wallets are not setup")
		return
	}

	GlobalSettings.Polygon.WalletTTX = make(map[string]string)
	GlobalSettings.Polygon.WalletBalance = make(map[string]map[string]Balance)
	GlobalSettings.Polygon.WalletAllowance = make(map[string]map[string]map[string]Allowance)

	// var wg sync.WaitGroup
	// for _, _wallet := range GlobalSettings.Polygon.Wallets.Main {
	// 	wg.Add(1)
	// 	go func(_wallet models.Wallet) {
	// 		defer wg.Done()
	// 		if _wallet.Address != nil && _wallet.PrivateKey != nil {

	// 			privateKey, err := utils.HexToECDSAV2(*_wallet.PrivateKey)
	// 			if err != nil {
	// 				log.Printf("failed to pack PK to ecdsa. %s", err)
	// 				return
	// 			}

	// 			p.Authenticator(*_wallet.Address, privateKey, CHAIN_ID)
	// 			// // // preapprovement check
	// 			p.PreApproveERC20TokensForDexs(nil, *_wallet.Address)
	// 			amount := int64(5000)
	// 			p.PreApproveERC20ContractsForDexs(nil, *_wallet.Address, &amount)

	// 			WalletNonceSync(p, nil, *_wallet.Address)
	// 		}
	// 	}(_wallet)
	// }
	// wg.Wait()

	PreApprovement(p)

	log.Printf("\nScanning mempool:\n --nodes:\n %v", p.NodePool)

	initialTime := time.Now()
	fmt.Printf("Initial time: %s\n", initialTime.Format(time.RFC3339))

	for _i, _client := range clientPool {
		nodePool, ok := p.NodePool.([]string)
		if !ok {
			log.Fatalf("Failed to assert NodePool as []string")
		}
		__node := nodePool[_i]
		go func(_node string, _c *ethclient.Client) {
			p.ScanMempool(_node, _c, callbacks...)
		}(__node, _client)
	}

	time.Sleep(time.Minute)
	fmt.Printf("Unique transactions after 1 minute: %v\nDuplicate transactions after 1 minute: %v\n", len(uniqueTxHashes), len(duplicateTxHashes))
	fmt.Printf("Ended at: %s\n", time.Now().Format(time.RFC3339))
}

func MockAttack(p Polygon, client *ethclient.Client) {
	log.Println("httpsSTART")

	var newTxAmountIn = big.NewInt(493100) // USDT
	var newTxAmountOut = big.NewInt(29855960702416041)
	// var newTxAmountOutMin = big.NewInt(100000000000000000) // DAI
	// var newTxAmountOutMin = big.NewInt(409440656214998200) // DAI

	privateKey, err := utils.HexToECDSAV2(*GlobalSettings.Polygon.Wallets.Main[0].PrivateKey) // Replace with the actual private key
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
		return
	}

	// _, _, erc20Token := RetrieveERC20Balance(p, client, *GlobalSettings.Polygon.Wallets.Main.Address, "0x8328e6fceC9477C28298c9f02d740Dd87a1683e5", 18)
	_, _, erc20Token := RetrieveERC20Balance(p, client, *GlobalSettings.Polygon.Wallets.Main[0].Address, "0xc2132D05D31c914a87C6611C10748AEb04B58e8F", 6)
	newTxGasFee := decimal.NewFromInt(100)

	botWallet := common.HexToAddress(*GlobalSettings.Polygon.Wallets.Main[0].Address)
	router := common.HexToAddress("0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff")
	tokenContract := common.HexToAddress("0xE06Bd4F5aAc8D0aA337D13eC88dB6defC6eAEefE")
	router = common.HexToAddress("0xe592427a0aece92de3edee1f18e0157c05861564")
	tokenContract = common.HexToAddress("0xBbba073C31bF03b8ACf7c28EF0738DeCF3695683")
	tokenContract = common.HexToAddress("0x838C9634dE6590B96aEadC4Bc6DB5c28Fd17E3C2")
	// tokenContract = common.HexToAddress("0x765Af38A6e8FDcB1EFEF8a3dd2213EFD3090B00F")

	// router = common.HexToAddress("0xf5b509bB0909a69B1c207E495f687a596C168E12")
	// erc20Token.Revoke(router, p.Auth)
	// tokenContract = common.HexToAddress("0xaA3717090CDDc9B227e49d0D84A28aC0a996e6Ff")
	// tokenContract := common.HexToAddress("0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063")
	// go func() {
	erc20ContractABIString, err := p.LoadABI("erc20")
	if err != nil {
		log.Fatalf("Failed to read ERC-20 contract ABI: %v", err)
	}

	erc20ABI, err := abi.JSON(strings.NewReader(erc20ContractABIString))
	if err != nil {
		log.Fatalf("Failed to parse ERC-20 contract ABI: %v", err)
	}

	tokenDecimals := int32(18)
	erc20TokenToSell := NewERC20Token(tokenContract, client, erc20ABI, &tokenDecimals)
	// newTxAmountToApproveBigInt := .Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(*decimals)))).BigInt()
	// erc20TokenToSell.Approve(router, p.Auth, big.NewInt(6000000000000000000))
	// erc20Token.Approve(router, p.Auth, big.NewInt(6000000))
	// }()

	// amount := new(big.Int)
	// amount.SetString("5000000000000000000000", 10)
	// _tx, _err := erc20TokenToSell.Approve(router, p.Auth, amount, nil)
	// if _err != nil {
	// 	log.Fatalf("Failed to read ERC-20 contract ABI: %v", err)
	// }
	// receipt, err := bind.WaitMined(context.Background(), client, _tx)
	// if err != nil {
	// 	log.Fatalf("Failed to wait for transaction to be mined: %v", err)
	// }

	// if receipt.Status != types.ReceiptStatusSuccessful {
	// 	log.Fatalf("Transaction failed with status: %v", receipt.Status)
	// }

	// log.Printf("Transaction mined successfully with hash: %s", receipt.TxHash.Hex())

	var dexContractABIString string
	// if dexContractABIString, err = p.LoadABI("uniswapv3"); err != nil {
	// 	log.Printf("Failed to read DEX contract ABI: %v", err)
	// 	return
	// }
	if dexContractABIString, err = p.LoadABI("uniswapv3"); err != nil {
		log.Printf("Failed to read DEX contract ABI: %v", err)
		return
	}

	var parsedABI abi.ABI
	if parsedABI, err = abi.JSON(strings.NewReader(dexContractABIString)); err != nil {
		log.Printf("Failed to parse DEX contract ABI: %v", err)
		return
	}

	var _pendingNonce, _nonce uint64

	if _pendingNonce, err = client.PendingNonceAt(context.Background(), common.HexToAddress(*GlobalSettings.Polygon.Wallets.Main[0].Address)); err != nil {
		log.Printf("Failed to get pending nonce: %v", err)
		return
	}

	log.Println("CONTINUE")
	if _nonce, err = client.NonceAt(context.Background(), common.HexToAddress(*GlobalSettings.Polygon.Wallets.Main[0].Address), nil); err != nil {
		log.Printf("Failed to get nonce: %v", err)
		return
	}

	__nonce := _nonce + 1

	log.Printf("Pending nonce: %v", _pendingNonce)
	log.Printf("Nonce: %v", _nonce)
	log.Printf("Forced Nonce: %v", __nonce)

	// go func() {
	// 	erc20Token.Approve(router, p.Auth, newTxAmountIn, &_nonce)
	// }()
	// return

	go func(_nonce uint64) {
		log.Println("AND FRONTRUN")
		legacy := false
		dryRun := false
		legacy = true
		// dryRun = true
		mockTxHash := "asfjghalsdkjalsdkfhaldjfhaslkdjlaskdfhlajklaskjdk"
		p.Swap(&_nonce, "exactOutputSingle", botWallet, router, tokenContract, client, erc20Token, parsedABI, newTxAmountIn, newTxAmountOut, newTxGasFee, GlobalSettings.Polygon.Settings.GasPriority, GlobalSettings.Polygon.Settings.GasFeeMax, GlobalSettings.Polygon.Settings.GasLimit, privateKey, CHAIN_ID, &mockTxHash, legacy, dryRun)
	}(_nonce)
	_nonce++

	go func() {
		log.Println("AND BACKRUN")
		legacy := false
		dryRun := false
		// legacy = true
		// dryRun = true
		networkFastGasPrice, err := decimal.NewFromString(utils.GasPriceData.Result.Result.FastGasPrice)
		networkFastGasPrice = networkFastGasPrice.Mul(decimal.NewFromFloat(1.5))
		if err != nil {
			log.Printf("Failed to convert fast gas price to decimal: %v", err)
			return
		}

		mockTxHash := "asfjghalsdkjalsdkfhaldjfhaslkdjlaskdfhlajklaskjdk"
		p.Swap(&__nonce, "exactInputSingle", botWallet, router, erc20Token.address, client, erc20TokenToSell, parsedABI, newTxAmountOut, ZERO_BIG_INT, newTxGasFee, networkFastGasPrice, GlobalSettings.Polygon.Settings.GasFeeMax, GlobalSettings.Polygon.Settings.GasLimit, privateKey, CHAIN_ID, &mockTxHash, legacy, dryRun)
	}()
}

func (p Polygon) ScanMempool(node string, client *ethclient.Client, callbacks ...interface{}) {
	if client == nil {
		p.GetNode(false)

		client = p.GetClient(p.Node)

		defer client.Close()
	}

	// _client := gethclient.New(client.Client())
	WalletKnownBalances(p, client)
	for balanceSimpleLock {
		time.Sleep(100 * time.Millisecond)
	}

	txs := make(chan common.Hash)
	var sub ethereum.Subscription
	var err error

	for {
		sub, err = client.Client().EthSubscribe(context.Background(), txs, "newPendingTransactions")
		if err != nil {
			log.Printf("Error: subscription to new pending transactions failed: %v. Retrying...", err)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Println("Successfully subscribed to new pending transactions")
		break
	}

	for {
		select {
		case err := <-sub.Err():
			log.Printf("Error: failed to connect to rpc node: %v. Trying to reconnect...", err)
			sub.Unsubscribe()
			time.Sleep(2 * time.Second)
			sub, err = client.Client().EthSubscribe(context.Background(), txs, "newPendingTransactions")
			if err != nil {
				log.Printf("Error: subscription to new pending transactions failed: %v. Retrying...", err)
			}
			log.Fatalf("Error: failed to connect to rpc node: %v. Trying to reconnect...", err)
		case txHash := <-txs:
			// log.Printf("Received new pending transaction hash: %s", txHash.Hex())
			if GlobalSettings.KillSwitch.IsOn != nil && *GlobalSettings.KillSwitch.IsOn {
				log.Print("KillSwitch is on")
				return
			}
			if IsPreApprovementInProgress() {
				log.Print("PreApprovement is in progress")
				return
			}

			go func(p Polygon, txH common.Hash) {
				// fmt.Println("Current time before mutex:", time.Now().Format("2006-01-02T15:04:05.000Z07:00"))
				txHash := txH.Hex()
				mutex.Lock()
				if _, exists := uniqueTxHashes[txHash]; exists {
					mutex2.Lock()
					duplicateTxHashes[txHash] = struct{}{}
					mutex2.Unlock()
					mutex.Unlock()
					return
				}
				uniqueTxHashes[txHash] = struct{}{}
				mutex.Unlock()
				// fmt.Println("Current time after mutex:", time.Now().Format("2006-01-02T15:04:05.000Z07:00"))

				go func(p Polygon, txH common.Hash) {
					// startTime := time.Now()
					tx, pending, _ := p.GetTransactionByHash(nil, txH)
					// elapsedTime := time.Since(startTime)
					// log.Printf("GetTransactionByHash execution time: %s", elapsedTime)
					if pending && tx != nil {
						p.AnalyzeTx(tx, client)
					}
				}(p, txH)
			}(p, txHash)
		}
	}
}

func (p Polygon) GetTransactionByHash(client *ethclient.Client, txHash common.Hash) (*types.Transaction, bool, error) {
	if client == nil {
		nodeSupportPool, ok := p.NodeSupportPool.([]string)
		if !ok {
			log.Fatal("Failed to assert NodeSupportPool as []string")

		}

		node := nodeSupportPool[rand.Intn(len(nodeSupportPool))]
		client = p.GetClient(node)
	}

	// startTime := time.Now()
	var ctx context.Context
	var cancel context.CancelFunc
	if GlobalSettings.Polygon.Settings.TTXMaxLatency != 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(GlobalSettings.Polygon.Settings.TTXMaxLatency)*time.Millisecond)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 280*time.Millisecond)
	}
	defer cancel()

	tx, isPending, err := client.TransactionByHash(ctx, txHash)
	// elapsedTime := time.Since(startTime)
	// log.Printf("TransactionByHash execution time: %s", elapsedTime)
	if err != nil {
		return nil, false, err
	}
	return tx, isPending, nil
}

func (p *Polygon) GetNode(pooling bool) {
	var nodes Nodes

	_nodes, err := ReadJson("nodes.json")
	if err != nil {
		log.Fatal(err.Error())
	}

	nodes = _nodes

	var rpcUrls []string
	if os.Getenv("GIN_MODE") == "release" {
		if len(nodes.Polygon) > 0 {
			rpcUrls = nodes.Polygon
		} else {
			log.Fatal("No Polygon RPC Url found for production")
		}
	} else {
		if len(nodes.PolygonTest) > 0 {
			rpcUrls = nodes.PolygonTest
		} else {
			log.Fatal("No Polygon RPC Url found for testing")
		}
	}
	if len(nodes.PolygonSupport) > 0 {
		p.NodeSupportPool = nodes.PolygonSupport
	} else {
		if p.NodeSupportPool == nil {
			p.NodeSupportPool = []string{}
		}
		for _, url := range rpcUrls {
			if strings.HasPrefix(url, "wss://") {
				p.NodeSupportPool = append(p.NodeSupportPool.([]string), strings.Replace(url, "wss://", "https://", 1))
			}
		}
	}

	p.Node, err = Queue(pooling, rpcUrls)
	if err != nil {
		log.Fatalf("Error accessing node queue: %v", err)
	}

	p.NodePool, err = Queue(pooling, rpcUrls)
	if err != nil {
		log.Fatalf("Error accessing nodepool queue: %v", err)
	}
}

func (p Polygon) GetState() interface{} {
	return p
}

// TODO: Extend to accept http
func (p *Polygon) GetClient(nodeUrl interface{}) (client *ethclient.Client) {
	if nodeUrl == nil {
		p.GetNode(false)
		nodeUrl = p.Node
	}

	nodeStr, ok := nodeUrl.(string)
	if !ok {
		log.Fatalf("Failed to assert node as string")
	}

	var err error

	client, err = ethclient.Dial(nodeStr)
	if err != nil {
		log.Fatalf("Failed to connect to the Polygon network via WebSocket: %v", err)
	}

	return client
}

// Cache it
func (p Polygon) LoadABI(provider string) (string, error) {
	bytes, err := os.ReadFile("abi/" + provider + "ABI.json")
	if err != nil {
		log.Fatalf("Error reading ABI json: %v", err)
		return "", err
	}

	return string(bytes), nil
}
