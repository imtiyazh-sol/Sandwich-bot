package handlers

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockchainClient interface {
	// GetNode()
	GetState() interface{}
	GetClient(interface{}) *ethclient.Client
	ScanMempool(string, *ethclient.Client, ...interface{})
	ScanMempoolV2(...interface{})
	// BuyToken(walletToBuyWithAddress, dexContractAddress, tokenToBuyAddress string, amountIn, amountOutMin *big.Int, privateKey *ecdsa.PrivateKey, chainId *big.Int) *types.Transaction
	// SellToken(walletAddress, dexContractAddress, tokenToSellAddress, tokenToReceiveAddress string, amountIn, amountOutMin *big.Int, privateKey *ecdsa.PrivateKey, chainId *big.Int) *types.Transaction
}

func NewBlockchainClient(clientType string) BlockchainClient {
	switch clientType {
	case "polygon":
		return &Polygon{}
	default:
		return nil
	}
}

func Run(clientType string, args ...interface{}) {
	bc := NewBlockchainClient(clientType)
	if bc == nil {
		log.Fatal("Client not found. Please try another client type.")
	}

	bc.ScanMempoolV2(ScenarioEvent)
}

func ScenarioEvent(tx *types.Transaction, client interface{}, args ...interface{}) func() {
	// return func() {
	_client := client.(BlockchainClient)
	fmt.Println("CLIENT", _client)
	fmt.Println(_client)

	// _client.BuyToken(tx, tx, tx)
	// fmt.Println(_client)
	// }
	return func() {}
}
