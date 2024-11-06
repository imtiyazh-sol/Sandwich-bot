package interfaces

import (
	"bot/controllers"
	"bot/models"
	"bot/utils"
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

// var status int
// var message string
// var err error
// var walletType models.WalletType
// var wallet, walletPK *string

// if payload.BotWallet != nil || payload.BotWalletPK != nil {
// 	wallet = payload.BotWallet
// 	walletPK = payload.BotWalletPK
// 	walletType = models.WalletType("trade")
// }

// if payload.WithdrawalWallet != nil || payload.WithdrawalWalletPK != nil {
// 	wallet = payload.WithdrawalWallet
// 	walletPK = payload.WithdrawalWalletPK
// 	walletType = models.WalletType("withdrawal")
// }

// status, _, message, err = handleWallet(wallet, walletPK, walletType, payload.UserID)
// if err != nil {
// 	return status, nil, message, err
// }

// Deprecated
func handleWallet(payloadWallet, payloadWalletPK *string, walletType models.WalletType, userID *uint) (int, interface{}, string, error) {
	var botWallet *models.Wallet
	if err := controllers.DB.Debug().Order("created_at DESC").First(&botWallet, "").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if !walletType.IsValid() {
				return http.StatusBadRequest, nil, "", fmt.Errorf("Error: invalid wallet type: %s", walletType)
			}

			createWallet := models.Wallet{
				BlockchainID: models.BlockchainID{
					BlockchainID: utils.IntToUint(1),
				},
				ModelExtended: models.ModelExtended{
					UpdatedBy: userID,
					CreatedBy: userID,
				},
				Type: &walletType,
			}
			if payloadWallet != nil {
				createWallet.Address = payloadWallet
			}
			if payloadWalletPK != nil {
				createWallet.PrivateKey = payloadWalletPK
			}
			if err = controllers.DB.Debug().Create(&createWallet).Error; err != nil {
				return http.StatusInternalServerError, nil, "", err
			}
		} else {
			return http.StatusInternalServerError, nil, "", err
		}
	} else {
		if payloadWallet != nil {
			botWallet.Address = payloadWallet
		}
		if payloadWalletPK != nil {
			botWallet.PrivateKey = payloadWalletPK
		}
		botWallet.ModelExtended.UpdatedBy = userID
		botWallet.ModelExtended.CreatedBy = userID

		botWallet.ID = 0
		if err = controllers.DB.Debug().Create(&botWallet).Error; err != nil {
			return http.StatusInternalServerError, nil, "", err
		}
	}

	return http.StatusAccepted, nil, "Wallet Updated Successfully", nil
}

// Then you can use this function in your original code like this:
