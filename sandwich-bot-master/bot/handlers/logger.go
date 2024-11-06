package handlers

import (
	"bot/controllers"
	"bot/models"
	"bot/utils"
	"encoding/json"
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"gorm.io/datatypes"
)

func Logger(tx *types.Transaction, method *abi.Method, settings *GlobalSettingsStruct, reason string, rejected bool) {
	hash := tx.Hash().Hex()

	inputs := map[string]interface{}{}
	if err := method.Inputs.UnpackIntoMap(inputs, tx.Data()[4:]); err != nil {
		log.Printf("Failed to unpack inputs: %v", err)
		return
	}

	transaction := map[string]interface{}{
		"hash":          hash,
		"inputs":        inputs,
		"to":            tx.To().Hex(),
		"gas_price_wei": tx.GasPrice(),
		"gas":           tx.Gas(),
		"gas_fee_cap":   tx.GasFeeCap(),
		"gas_tip_cap":   tx.GasTipCap(),
		"value":         tx.Value(),
	}

	transactionJSON, _ := json.Marshal(transaction)
	transactionJSONData := datatypes.JSON(transactionJSON)

	if rejected {

		// reject := models.Reject{
		// 	Hash:       &hash,
		// 	Transction: &transactionJSONData, // Use the address of the new variable
		// 	Method:     method.Name,
		// 	Reason:     &reason,
		// }

		// if err := controllers.DB.Clauses(clause.OnConflict{
		// 	Columns:   []clause.Column{{Name: "hash"}},
		// 	DoNothing: true,
		// }).Create(&reject).Error; err != nil {
		// 	log.Fatalf("Error: saving reject to db: %v", err)
		// }

	} else {
		status := models.StatusType("pending")
		settingsJSON, err := json.Marshal(settings.Polygon.Settings)
		if err != nil {
			log.Fatalf("Error marshaling settings: %v", err)
		}
		settingsJSONData := datatypes.JSON(settingsJSON)

		order := models.Order{
			Status: models.Status{
				Status: status,
			},
			BlockchainID: models.BlockchainID{
				BlockchainID: utils.IntToUint(1),
			},
			Hash:     &hash,
			Method:   method.Name,
			Data:     &transactionJSONData,
			Settings: &settingsJSONData,
		}
		if err := controllers.DB.Create(&order).Error; err != nil {
			log.Fatalf("Error: saving order to db: %v", err)
		}
	}

	log.Print(reason)
	log.Println("===========================================================")
}
