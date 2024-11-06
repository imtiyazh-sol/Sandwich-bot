package interfaces

import (
	"bot/controllers"
	"bot/handlers"
	"bot/models"
	"bot/types"
	"bot/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpdateSettings(_data []byte) (int, interface{}, string, error) {
	var payload types.UpdateSettingsReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}
	var err error

	// Fetch the latest record from the database

	message := ""
	if err = controllers.DB.Transaction(func(tx *gorm.DB) error {
		var settingsObj models.Settings
		if err = tx.First(&settingsObj, "active = ?", true).Error; err != nil {
			return err
		}

		if err = controllers.DB.Model(&models.Settings{}).Where("active = ?", true).Update("active", false).Error; err != nil {
			return err
		}

		settingsMap := map[string]interface{}{}
		if err = json.Unmarshal(settingsObj.Settings, &settingsMap); err != nil {
			return err
		}

		// fmt.Println("FIRST", settingsMap)

		delete(settingsMap, "user_id")

		var updatedFieldSlice []string
		// var updatedFieldNewValue interface{}
		plType := reflect.TypeOf(payload)
		plValue := reflect.ValueOf(payload)
		for i := 0; i < plType.NumField(); i++ {
			field := plType.Field(i)
			jsonTag := field.Tag.Get("json")
			jsonTag = strings.TrimSuffix(jsonTag, ",omitempty") // Remove 'omitempty' from the tag
			fmt.Println(jsonTag)
			if jsonTag != "user_id" {
				fieldValue := plValue.FieldByName(field.Name)

				// if jsonTag == "exit_gas" {

				// 	log.Println(fieldValue.IsValid())
				// 	log.Println(fieldValue.Kind())
				// 	log.Println(fieldValue.Kind() == reflect.Ptr)
				// 	log.Println(fieldValue.Kind() == reflect.Slice)
				// 	log.Println(fieldValue.Kind() == reflect.Map)
				// 	log.Println(fieldValue.Kind() == reflect.Interface)
				// 	log.Println(fieldValue.Kind() == reflect.Chan)
				// 	log.Println(fieldValue.Kind() == reflect.Func)
				// 	log.Println(!fieldValue.IsNil())
				// 	log.Println(jsonTag)
				// 	log.Println("AAA", fieldValue)
				// }

				if fieldValue.IsValid() && ((fieldValue.Kind() == reflect.Ptr || fieldValue.Kind() == reflect.Slice || fieldValue.Kind() == reflect.Map || fieldValue.Kind() == reflect.Interface || fieldValue.Kind() == reflect.Chan || fieldValue.Kind() == reflect.Func) && !fieldValue.IsNil()) {
					settingsMap[jsonTag] = fieldValue.Interface()
					updatedFieldSlice = strings.Split(jsonTag, "_")
					// updatedFieldNewValue = fieldValue.Interface()
				}

			}
		}

		message = fmt.Sprintf("%s has been updated.", strings.Title(strings.Join(updatedFieldSlice, " ")))

		var settingsByte []byte
		if settingsByte, err = json.Marshal(&settingsMap); err != nil {
			return err
		}

		_true := true
		newSettings := models.Settings{
			Active:   &_true,
			Settings: settingsByte,
			ModelExtended: models.ModelExtended{
				UpdatedBy: payload.UserID,
				CreatedBy: payload.UserID,
			},
			BlockchainID: settingsObj.BlockchainID,
		}

		// fmt.Println("SECOND", settingsMap)

		if err = controllers.DB.Debug().Create(&newSettings).Error; err != nil {
			return err
		}

		handlers.UpdateGlobalSettings(int(*settingsObj.BlockchainID))

		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, nil, "", err
		}
		return http.StatusInternalServerError, nil, "", err
	}

	return http.StatusCreated, nil, message, nil
}

func RetrieveSettings(_data []byte) (int, interface{}, string, error) {

	var settingsObj models.Settings
	if err := controllers.DB.Debug().First(&settingsObj, "active = ?", true).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, nil, "No settings found", err
		}
		return http.StatusInternalServerError, nil, "", err
	}

	return http.StatusOK, settingsObj, "", nil
}

func CreateWallet(_data []byte) (int, interface{}, string, error) {
	var payload types.CreateWalletReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	if !payload.WalletType.IsValid() {
		return http.StatusBadGateway, nil, "", fmt.Errorf("Unsupported wallet type: %v", *payload.WalletType)
	}

	if _, err := utils.HexToECDSAV2(*payload.PrivateKey); err != nil {
		return http.StatusBadRequest, nil, "Incorrect or malformed private key provided", err
	}
	// else {
	// 	fmt.Println(_privateKey)
	// }
	// non checksummed address to normalize it
	_address := strings.ToLower(*payload.Address)
	payload.Address = &_address

	_true := true
	wallet := models.Wallet{
		ModelExtended: models.ModelExtended{
			UpdatedBy: payload.UserID,
			CreatedBy: payload.UserID,
		},
		BlockchainID: models.BlockchainID{
			BlockchainID: utils.IntToUint(1),
		},
		Active: models.Active{
			Active: &_true,
		},
		Type:       payload.WalletType,
		Name:       payload.Name,
		Address:    payload.Address,
		PrivateKey: payload.PrivateKey,
	}

	var err error
	message := ""
	if err = controllers.DB.Transaction(func(tx *gorm.DB) error {
		if err = controllers.DB.Model(&models.Wallet{}).Where("type = ?", payload.WalletType).Update("active", false).Error; err != nil {
			return err
		}

		if err := controllers.DB.Debug().Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "address"}},
				DoUpdates: clause.AssignmentColumns([]string{"private_key", "name", "active"}),
			}).Create(&wallet).Error; err != nil {
			return err
		}

		message = fmt.Sprintf("Wallet %s has been set up as a %s wallet", *payload.Address, *payload.WalletType)

		handlers.UpdateGlobalSettings(1)
		return nil
	}); err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	return http.StatusCreated, nil, message, nil
}

func RetrieveWallet(_data []byte) (int, interface{}, string, error) {
	var payload types.RetrieveWalletReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadRequest, nil, "", err
	}

	if !payload.WalletType.IsValid() {
		return http.StatusBadRequest, nil, "", fmt.Errorf("Unsupported wallet type: %v", *payload.WalletType)
	}

	var wallet models.Wallet
	if err := controllers.DB.Debug().First(&wallet, "active = ? and type = ?", true, *payload.WalletType).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	address := *wallet.Address // Assuming 'wallets.Address' contains the full wallet address
	if len(address) > 9 {
		address = address[:5] + "..." + address[len(address)-4:]
	}

	var response = types.RetrieveWalletRespType{
		Address: &address,
		Name:    &wallet.Name,
	}

	message := fmt.Sprintf("📝 **`%s`**", *wallet.Address)
	if response.Name != nil {
		message += "\n\n " + *response.Name
	}

	return http.StatusOK, response, message, nil
}

func ToggleKillSwitch(_data []byte) (int, interface{}, string, error) {
	var payload types.ToggleKillSwitchReqType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	toggleKillSwitch := models.KillSwitch{
		ModelExtended: models.ModelExtended{
			UpdatedBy: payload.UserID,
			CreatedBy: payload.UserID,
		},
		IsOn: payload.IsOn,
	}

	if err := controllers.DB.Create(&toggleKillSwitch).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	message := "KillSwitch is"
	if *payload.IsOn {
		message += " on"
	} else {
		message += " off"
	}

	handlers.UpdateGlobalSettings(1)
	return http.StatusOK, nil, message, nil
}

func RetrieveKillSwitch(_data []byte) (int, interface{}, string, error) {
	var payload types.UserRequiredType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	var killSwitch models.KillSwitch
	if err := controllers.DB.Debug().Order("created_at DESC").First(&killSwitch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, nil, "no killswitch data found", err
		}
		return http.StatusInternalServerError, nil, "", err
	}

	return http.StatusOK, nil, "", nil
}
