package controllers

import (
	"bot/models"
	"bot/utils"
	"encoding/json"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Seed() {
	_true := true
	_false := false
	var _one uint = 1
	var _default uint = 0

	var name = "polygon"
	var currency = "matic"
	var chainID = 137

	blockchain := []models.Blockchain{
		{
			ModelExtended: models.ModelExtended{
				ID:        1,
				CreatedBy: &_default,
				UpdatedBy: &_default,
			},
			Uid:      utils.StringToUlid("01HVRF4N9NJNVNT4B34R2Y7VKD"),
			Name:     &name,
			ChainID:  &chainID,
			Currency: &currency,
		},
	}

	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "chain_id", "currency"}),
	}).Create(&blockchain)

	_settings, _ := json.Marshal(map[string]interface{}{
		"gas_fee_max":               500,    // GWEI -- Max Gas Fee we are comfortable paying
		"gas_limit":                 300000, // Units
		"gas_priority":              getDynamicGasPrice(),     // GWEI dynamically fetched
		"ttx_max_latency":           275,    // ms
		"dex_fee":                   300,
		"exit_gas":                  100,   // percentage
		"slippage":                  25,    // percentage of token bot's buying
		"target_value_min":          40,    // USD
		"target_value_max":          1000,  // USD
		"target_gas_markup_allowed": 70,    // percentage of target tx GasPrice compared to network fast gas price
		"usd_per_trade":             10.0,  // percentage of target tx amountIn
		"deadline":                  5,     // minutes
		"draw_down":                 200,   // percentage preapproved
		"gas_tolerance":             20,    // how much higher over target tx gas price
		"withdrawal_threshold":      500.0, // USD
	})

	settings := []models.Settings{
		{
			ModelExtended: models.ModelExtended{
				ID:        10000,
				CreatedBy: &_default,
				UpdatedBy: &_default,
			},
			BlockchainID: &_one,
			Settings:     _settings,
			Active:       &_true,
		},
	}

	DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"settings"}),
	}).Create(&settings)

	killSwitch := models.KillSwitch{
		ModelExtended: models.ModelExtended{
			ID:        1000,
			CreatedBy: &_default,
			UpdatedBy: &_default,
		},
		IsOn: &_false,
	}

	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_on"}),
	}).Create(&killSwitch)

	// DEX listings
	dexs := []models.DEX{
		{
			ModelExtended: models.ModelExtended{
				CreatedBy: &_default,
				UpdatedBy: &_default,
			},
			BlockchainID: models.BlockchainID{
				BlockchainID: &_one,
			},
			Type:    utils.StringToPointer("quickswap"),
			Address: utils.StringToPointer("0xa5e0829caced8ffdd4de3c43696c57f7d7a678ff"),
		},
		{
			ModelExtended: models.ModelExtended{
				CreatedBy: &_default,
				UpdatedBy: &_default,
			},
			BlockchainID: models.BlockchainID{
				BlockchainID: &_one,
			},
			Type:    utils.StringToPointer("sushiswap"),
			Address: utils.StringToPointer("0x1b02da8cb0d097eb8d57a175b88c7d8b47997506"),
		},
	}

	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		DoNothing: true,
	}).Create(&dexs)

	// Token Listings
	coins := []models.Coin{
		{
			ModelExtended: models.ModelExtended{
				ID:        1000,
				CreatedBy: &_default,
				UpdatedBy: &_default,
			},
			BlockchainID: models.BlockchainID{
				BlockchainID: &_one,
			},
			Name:     utils.StringToPointer("usdt"),
			Decimals: utils.StringToInt32("6"),
			Address:  utils.StringToPointer("0xc2132d05d31c914a87c6611c10748aeb04b58e8f"),
		},
		{
			ModelExtended: models.ModelExtended{
				ID:        1001,
				CreatedBy: &_default,
				UpdatedBy: &_default,
				DeletedAt: gorm.DeletedAt{Time: time.Now().UTC(), Valid: true},
				DeletedBy: &_default,
			},
			BlockchainID: models.BlockchainID{
				BlockchainID: &_one,
			},
			Name:     utils.StringToPointer("dai"),
			Decimals: utils.StringToInt32("18"),
			Address:  utils.StringToPointer("0x8f3cf7ad23cd3cadbd9735aff958023239c6a063"),
		},
	}

	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "decimals", "address", "deleted_at", "deleted_by"}),
	}).Create(&coins)

	// Dynamic Contract Listings
	_contractsListings := []models.Contract{
		{
			ModelExtended: models.ModelExtended{
				ID:        1000,
				CreatedBy: &_default,
				UpdatedBy: &_default,
			},
			BlockchainID: models.BlockchainID{
				BlockchainID: &_one,
			},
			Address:   utils.StringToPointer("0xe06bd4f5aac8d0aa337d13ec88db6defc6eaeefe"),
			Blacklist: &_false,
			Name:      "planetix",
			Decimals:  utils.IntToInt32(18),
		},
		{
			ModelExtended: models.ModelExtended{
				ID:        1001,
				CreatedBy: &_default,
				UpdatedBy: &_default,
			},
			BlockchainID: models.BlockchainID{
				BlockchainID: &_one,
			},
			Address:   utils.StringToPointer("0x61299774020da444af134c82fa83e3810b309991"),
			Blacklist: &_true,
			Name:      "rndr",
			Decimals:  utils.IntToInt32(18),
		},
	}

	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"blacklist", "address", "name"}),
	}).Create(&_contractsListings)
}

// Helper function to get dynamic gas price
func getDynamicGasPrice() int {
	client, err := rpc.Dial("https://polygon-rpc.com")
	if err != nil {
		log.Fatalf("Failed to connect to Polygon RPC: %v", err)
	}
	defer client.Close()

	var result string
	err = client.Call(&result, "eth_gasPrice")
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}

	gasPrice := utils.HexToInt(result) / 1e9 // Convert from Wei to GWEI
	return gasPrice
}
