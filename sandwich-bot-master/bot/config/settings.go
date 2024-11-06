package config

import (
	"bot/models"
	"encoding/json"
	"log"

	"gorm.io/datatypes"
)

var Settings models.Settings

func init() {

	settingsMap := map[string]interface{}{
		"bot":                  "online",
		"gas_limit":            300000,
		"gas_priority":         2,
		"gas_fee":              50,
		"slippage":             5.0,
		"transaction_size_usd": 1000,
		"transaction_size_eth": 0.0,

		"profitThreshold": 100,
	}

	settingsJSON, err := json.Marshal(settingsMap)
	if err != nil {
		log.Fatal(err) // Handle error appropriately in production code
	}

	Settings = models.Settings{
		Settings: datatypes.JSON(settingsJSON),
	}
}
