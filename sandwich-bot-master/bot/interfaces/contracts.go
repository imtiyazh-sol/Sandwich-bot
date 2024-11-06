package interfaces

import (
	"bot/controllers"
	"bot/handlers"
	"bot/models"
	"bot/types"
	"bot/utils"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func WhiteBlacklistContract(_data []byte) (int, interface{}, string, error) {
	var payload types.WhiteBlacklistContractsReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadRequest, nil, "", err
	}

	message := "📝"
	contracts := []models.Contract{}
	var wg sync.WaitGroup
	for _, _c := range payload.Address {
		wg.Add(1)
		go func(_c *string) {
			defer wg.Done()

			_address := strings.ToLower(*_c)
			_c = &_address
			// log.Println("Address", *_c)
			contractDecimals, err := handlers.GetTokenDecimals(nil, common.HexToAddress(*_c))
			if err != nil {
				fmt.Println(err)
				return
			}

			contract := models.Contract{
				ModelExtended: models.ModelExtended{
					UpdatedBy: payload.UserID,
					CreatedBy: payload.UserID,
				},
				BlockchainID: models.BlockchainID{
					BlockchainID: utils.IntToUint(1),
				},
				Address:  _c,
				Decimals: contractDecimals,
			}

			if payload.Blacklist != nil && *payload.Blacklist {
				contract.Blacklist = payload.Blacklist
			}

			message += fmt.Sprintf(" **`%s`**", *_c)
			contracts = append(contracts, contract)
		}(_c)
	}
	wg.Wait()

	if len(contracts) > 1 {
		message += " are"
	} else {
		message += " is"
	}

	if payload.Blacklist != nil && *payload.Blacklist {
		message += " blacklisted to swap to."
	} else {
		message += " whitelisted to swap to."
	}

	if len(contracts) > 0 {
		if err := controllers.DB.Debug().Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "address"}},
			DoUpdates: clause.AssignmentColumns([]string{"blacklist", "updated_at", "decimals"}),
		}).Create(&contracts).Error; err != nil {
			return http.StatusInternalServerError, nil, "", err
		}
		handlers.UpdateGlobalSettings(1)
	} else {
		message = "No valid contracts were provided."
	}

	return http.StatusOK, nil, message, nil
}

func RetrieveContract(_data []byte) (int, interface{}, string, error) {
	var payload types.RetrieveContractReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	var contracts []types.RetrieveContractRespType

	queryPartial := controllers.DB.Debug().Model(&models.Contract{}).Select("DISTINCT ON (address) *").Order("address, created_at DESC")

	fmt.Println(payload.AddressPartial)
	if payload.AddressPartial != nil {
		addressPartial := utils.FormatLikeClause(payload.AddressPartial)
		if err := queryPartial.Find(&contracts, "address LIKE ?", addressPartial).Error; err != nil {
			return http.StatusInternalServerError, nil, "", err
		}
	} else {
		if payload.Blacklisted != nil && *payload.Blacklisted > 0 {
			if err := queryPartial.Find(&contracts, "blacklist = true").Error; err != nil {
				return http.StatusInternalServerError, nil, "", err
			}
		} else {
			if err := queryPartial.Find(&contracts, "blacklist is null or blacklist = false").Error; err != nil {
				return http.StatusInternalServerError, nil, "", err
			}
		}
	}

	if len(contracts) == 0 {
		return http.StatusNotFound, nil, "No contracts found", errors.New("")
	}

	message := ""
	for _, _c := range contracts {
		fmt.Println(*_c.Blacklist, *_c.Address)
		message += fmt.Sprintf("📝 `%s`:\n**%s**", *_c.Address, _c.CreatedAt.Format("2006-01-02"))
		if _c.Blacklist != nil && *_c.Blacklist {
			message += " BL\n\n"
		} else {
			message += " WL\n\n"
		}
	}

	return http.StatusOK, contracts, message, nil
}

func CreateUpdateDEX(_data []byte) (int, interface{}, string, error) {
	var payload types.CreateUpdateDEXReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	p := handlers.Polygon{}
	if _, err := p.LoadABI(*payload.Type); err != nil {
		return http.StatusNotFound, nil, "", err
	}

	_address := strings.ToLower(*payload.Address)
	payload.Address = &_address

	var dex models.DEX
	if err := controllers.DB.Debug().Unscoped().First(&dex, "address = ?", *payload.Address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dex = models.DEX{
				ModelExtended: models.ModelExtended{
					UpdatedBy: payload.UserID,
					CreatedBy: payload.UserID,
				},
				BlockchainID: models.BlockchainID{
					BlockchainID: utils.IntToUint(1),
				},
				Address: payload.Address,
				Type:    payload.Type,
			}

			if err := controllers.DB.Debug().Create(&dex).Error; err != nil {
				return http.StatusInternalServerError, nil, "", err
			}
		} else {
			return http.StatusInternalServerError, nil, "", err
		}
	} else {
		if err := controllers.DB.Debug().Model(&dex).Clauses(clause.Returning{}).Unscoped().Where("address = ?", *payload.Address).Updates(map[string]interface{}{
			"type":       *payload.Type,
			"deleted_at": nil,
			"deleted_by": nil,
			"updated_by": *payload.UserID,
			// "created_by": *payload.UserID,
		}).Error; err != nil {
			return http.StatusInternalServerError, nil, "", err
		}
	}

	message := fmt.Sprintf("Dex with router address **`%s`** has been added to database", *dex.Address)
	handlers.UpdateGlobalSettings(1)
	return http.StatusAccepted, nil, message, nil
}

func DeleteDEX(_data []byte) (int, interface{}, string, error) {
	var payload types.DeleteDEXReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	_address := strings.ToLower(*payload.Address)
	payload.Address = &_address

	var dex models.DEX
	if err := controllers.DB.Debug().First(&dex, "address = ?", *payload.Address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, nil, "", err
		}
		return http.StatusInternalServerError, nil, "", err
	}

	dex.ModelExtended.DeletedAt = gorm.DeletedAt{Time: time.Now().UTC()}
	dex.ModelExtended.DeletedBy = payload.UserID

	if err := controllers.DB.Debug().Model(&dex).Update("deleted_by", *payload.UserID).Delete(&dex).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	message := fmt.Sprintf("Dex with router address **`%s`** has been deleted from database", *dex.Address)

	handlers.UpdateGlobalSettings(1)
	return http.StatusAccepted, nil, message, nil
}

func RetrieveDEX(_data []byte) (int, interface{}, string, error) {
	var payload types.RetrieveDEXReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	var dexs []*types.RetrieveDEXRespType
	if err := controllers.DB.Debug().Model(&models.DEX{}).Find(&dexs, "blockchain_id = ?", *payload.BlockchainID).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	message := ""
	for _, _d := range dexs {
		message += fmt.Sprintf("**%s** **`%s`**\n\n", *_d.Type, *_d.Address)
	}

	return http.StatusOK, dexs, message, nil
}

func CreteUpdateCoin(_data []byte) (int, interface{}, string, error) {
	var payload types.CreateUpdateCoinReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	_address := strings.ToLower(*payload.Address)
	payload.Address = &_address

	_name := strings.ToLower(*payload.Name)
	payload.Name = &_name
	// TODO: non-checksummed address
	coin := models.Coin{
		ModelExtended: models.ModelExtended{
			CreatedBy: payload.UserID,
			UpdatedBy: payload.UserID,
			DeletedBy: nil,
			DeletedAt: gorm.DeletedAt{},
		},
		BlockchainID: models.BlockchainID{
			BlockchainID: payload.BlockchainID,
		},
		Name:     payload.Name,
		Address:  payload.Address,
		Decimals: payload.Decimals,
	}

	if err := controllers.DB.Unscoped().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "deleted_at", "deleted_by", "updated_by", "decimals"}),
	}).Create(&coin).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	message := fmt.Sprintf("Coin **%s** with contract address **`%s`** has been added to database", *coin.Name, *coin.Address)

	handlers.UpdateGlobalSettings(1)
	return http.StatusAccepted, nil, message, nil
}

func DeleteCoin(_data []byte) (int, interface{}, string, error) {
	var payload types.DeleteCoinReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	_address := strings.ToLower(*payload.Address)
	payload.Address = &_address

	coin := models.Coin{
		// Address: &toLowerAddress,
		ModelExtended: models.ModelExtended{
			DeletedBy: payload.UserID,
			DeletedAt: gorm.DeletedAt{Time: time.Now().UTC()},
		},
	}

	if err := controllers.DB.Debug().Model(&models.Coin{}).Where("address = ?", *payload.Address).Updates(&coin).Delete(&coin).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	message := fmt.Sprintf("Coin with contract address **`%s`** has been deleted from database", *payload.Address)

	handlers.UpdateGlobalSettings(1)
	return http.StatusAccepted, nil, message, nil
}

func RetrieveCoin(_data []byte) (int, interface{}, string, error) {
	var payload types.RetrieveCoinReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	// TODO: pagination
	var coins []*models.Coin
	if err := controllers.DB.Debug().Find(&coins, "blockchain_id = ?", *payload.BlockchainID).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	message := ""
	for _, _c := range coins {
		message += fmt.Sprintf("**%s** **`%s`**\n\n", *_c.Name, *_c.Address)
	}

	return http.StatusAccepted, coins, message, nil
}
