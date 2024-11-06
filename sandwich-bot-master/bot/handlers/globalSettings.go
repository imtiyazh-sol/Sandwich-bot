package handlers

import (
	"bot/controllers"
	"bot/models"
	"encoding/json"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/shopspring/decimal"
)

type GlobalSettingsStruct struct {
	KillSwitch models.KillSwitch `json:"killswitch"`
	Polygon    struct {
		Settings struct {
			GasFeeMax              decimal.Decimal `json:"gas_fee_max"`
			GasLimit               uint64          `json:"gas_limit"`
			GasPriority            decimal.Decimal `json:"gas_priority"`
			TTXMaxLatency          uint64          `json:"ttx_max_latency"`
			ExitGas                decimal.Decimal `json:"exit_gas"`
			Slippage               float64         `json:"slippage"`
			TargetValueMin         decimal.Decimal `json:"target_value_min"`
			TargetValueMax         decimal.Decimal `json:"target_value_max"`
			TargetGasMarkupAllowed decimal.Decimal `json:"target_gas_markup_allowed"`
			UsdPerTrade            decimal.Decimal `json:"usd_per_trade"`
			Deadline               int             `json:"deadline"`
			DrawDown               float64         `json:"draw_down"`
			GasTolerance           float64         `json:"gas_tolerance"`
			WithdrawalThreshold    decimal.Decimal `json:"withdrawal_threshold"`
		} `json:"settings"`
		Wallets struct {
			Main       []models.Wallet `json:"main"`
			Withdrawal []models.Wallet `json:"withdrawal"`
		} `json:"wallets"`
		DEXs      map[string]string `json:"dexs"`
		Contracts struct {
			BlackList map[string][]interface{} `json:"blacklist"`
			Whitelist map[string]Contract      `json:"whitelist"`
		} `json:"contracts"`
		Coins           map[string][]interface{}                   `json:"coins"`
		WalletNonce     map[string]Nonce                           `json:"wallet_nonce"`
		WalletTTX       map[string]string                          `json:"wallet_ttx"`
		WalletBalance   map[string]map[string]Balance              `json:"wallet_balance"`
		WalletAllowance map[string]map[string]map[string]Allowance `json:"wallet_allowance"`
		ABI             map[string]abi.ABI                         `json:"abi"`
	} `json:"polygon"`
}

type Nonce struct {
	Nonce        *uint64 `json:"nonce"`
	PendingNonce *uint64 `json:"pendingNonce"`
}

type Contract struct {
	Address    *string     `json:"address"`
	Decimals   *int32      `json:"decimals"`
	Name       *string     `json:"name"`
	ERC20Token *ERC20Token `json:"erc20Token"`
}

type Balance struct {
	Decimal    decimal.Decimal `json:"decimal"`
	BigInt     *big.Int        `json:"bigInt"`
	ERC20Token *ERC20Token     `json:"erc20Token"`
}

type Allowance struct {
	Decimal decimal.Decimal `json:"decimal"`
	BigInt  *big.Int        `json:"bigInt"`
	// DEX     string          `json:"dex"`
	// ERC20Token *ERC20Token     `json:"erc20Token"`
}

var GlobalSettings GlobalSettingsStruct

// type

var (
	UpdateGlobalSettings = func(blockchain_id int) error {
		log.Println("Populating globalsettings")
		dbObj := controllers.DB
		var err error
		// killswitch
		if err = dbObj.Order("created_at desc").First(&GlobalSettings.KillSwitch).Error; err != nil {
			log.Fatalf("Error retrieving settings from database: %v", err)
		}
		// current settings
		var _settings models.Settings
		if err = dbObj.First(&_settings, "active = true and blockchain_id = ?", blockchain_id).Error; err != nil {
			log.Fatalf("Error retrieving settings from database: %v", err)
		} else {
			err = json.Unmarshal([]byte(_settings.Settings), &GlobalSettings.Polygon.Settings)
			if err != nil {
				log.Fatalf("Error unmarshalling settings into global polygon settings: %v", err)
			}
		}
		// current wallets
		if err = dbObj.Find(&GlobalSettings.Polygon.Wallets.Main, "type = 'main' and active = true and blockchain_id = ?", blockchain_id).Error; err != nil {
			log.Printf("Error retrieving main wallet from database: %v", err)
			// return errors.New("No main wallet is setup")
		}
		if err = dbObj.First(&GlobalSettings.Polygon.Wallets.Withdrawal, "type = 'withdrawal' and active = true and blockchain_id = ?", blockchain_id).Error; err != nil {
			log.Printf("Error retrieving withdrawal wallet from database: %v", err)
			// return errors.New("No withdrawal wallet is setup")
		}
		// active dexes
		var _dexs []models.DEX
		if err = dbObj.Find(&_dexs, "blockchain_id = ?", blockchain_id).Error; err != nil {
			log.Fatalf("Error retrieving dexs from database: %v", err)
		} else {
			GlobalSettings.Polygon.DEXs = map[string]string{}
			GlobalSettings.Polygon.ABI = map[string]abi.ABI{}
			for _, _d := range _dexs {
				GlobalSettings.Polygon.DEXs[*_d.Type] = *_d.Address
				var dexContractABIString string
				p := Polygon{}
				if dexContractABIString, err = p.LoadABI(*_d.Type); err != nil {
					log.Fatalf("Failed to read DEX contract ABI: %v", err)
					// log.Println("===========================================================")
					// return
				}

				var parsedABI abi.ABI
				if parsedABI, err = abi.JSON(strings.NewReader(dexContractABIString)); err != nil {
					log.Fatalf("Failed to parse DEX contract ABI: %v", err)
					// log.Println("===========================================================")
					// return
				}

				GlobalSettings.Polygon.ABI[*_d.Type] = parsedABI
			}
		}

		// blacklisted contracts
		var _blacklisted []models.Contract
		if err = dbObj.Find(&_blacklisted, "blacklist = true and blockchain_id = ?", blockchain_id).Error; err != nil {
			log.Fatalf("Error retrieving blacklisted contracts from database: %v", err)
		} else {
			GlobalSettings.Polygon.Contracts.BlackList = map[string][]interface{}{}
			for _, _b := range _blacklisted {
				GlobalSettings.Polygon.Contracts.BlackList[*_b.Address] = []interface{}{*_b.Decimals, _b.Name}
			}
		}

		// whitelisted contracts
		var _whitelisted []models.Contract
		if err = dbObj.Find(&_whitelisted, "(blacklist is null or blacklist = false) and blockchain_id = ?", blockchain_id).Error; err != nil {
			log.Fatalf("Error retrieving whitelisted contracts from database: %v", err)
		} else {
			GlobalSettings.Polygon.Contracts.Whitelist = map[string]Contract{}
			for _, _w := range _whitelisted {
				GlobalSettings.Polygon.Contracts.Whitelist[*_w.Address] = Contract{
					Address:  _w.Address,
					Name:     &_w.Name,
					Decimals: _w.Decimals,
				}
			}
		}

		// tradable coins
		var _coins []models.Coin
		if err = dbObj.Find(&_coins, "blockchain_id = ?", blockchain_id).Error; err != nil {
			log.Fatalf("Error retrieving blacklisted contracts from database: %v", err)
		} else {
			GlobalSettings.Polygon.Coins = map[string][]interface{}{}
			for _, _c := range _coins {
				GlobalSettings.Polygon.Coins[*_c.Name] = []interface{}{*_c.Address, *_c.Decimals}
				// TODO: ?
				GlobalSettings.Polygon.Contracts.BlackList[*_c.Address] = []interface{}{*_c.Decimals, *_c.Name}
			}
		}

		return nil
	}
)
