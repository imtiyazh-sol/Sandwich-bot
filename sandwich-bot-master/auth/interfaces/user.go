package interfaces

import (
	"auth/controllers"
	"auth/models"
	"auth/types"
	"auth/utils"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func CreateUser(_data []byte) (int, interface{}, string, error) {
	var payload types.CreateUpdateUserType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	var user models.User
	if err := controllers.DB.Transaction(func(tx *gorm.DB) error {
		var role models.Role
		if err := controllers.DB.First(&role, "title = ?", "guest").Error; err != nil {
			return err
		}

		_true := true
		mnemonic := utils.GenerateMnemonic(24)
		user = models.User{
			Verified: models.Verified{
				Verified: &_true,
			},
			Telegram: []models.Telegram{
				{TgID: payload.TgID, FirstName: payload.FirstName, LastName: payload.LastName, Username: payload.Username},
			},
			Mnemonic: models.Mnemonic{
				Phrase: &mnemonic,
			},
			Role: []models.Role{
				role,
			},
		}

		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return http.StatusConflict, nil, "", errors.New("user already exists")
			}
		}
		return http.StatusForbidden, nil, "", errors.New("action not allowed")
	}

	var response = types.APIResponseUserCreateType{
		ID:       user.ID,
		Mnemonic: *user.Mnemonic.Phrase,
	}

	return http.StatusCreated, response, "Please carefully save your mnemonic phrase. It will be displayed only once and is essential for future access or recovery of your account", nil
}

func RetrieveAccess(_data []byte) (int, interface{}, string, error) {
	var payload types.HasAccessType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadGateway, nil, "", err
	}

	var user models.User
	var accessExists bool
	err := controllers.DB.Debug().Model(&models.User{}).Joins("JOIN auth_user_telegram telegram ON telegram.user_id = auth_users.id").Joins("JOIN auth_user_access access ON access.user_id = auth_users.id").Where("telegram.tg_id = ? AND access.created_at > NOW() - INTERVAL '15 minutes'", *payload.TgID).First(&user).Error
	if err == nil {
		accessExists = true
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		accessExists = false
	} else {
		return http.StatusInternalServerError, nil, "", err
	}

	response := types.APIResponseUserHasAccessType{
		Access: accessExists,
	}

	return http.StatusOK, response, "", nil
}

func RetrieveUser(_data []byte) (int, interface{}, string, error) {
	var payload types.RetrieveUserType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadRequest, nil, "", err
	}

	var user *models.User
	var err error
	if payload.TgID != nil {
		err = controllers.DB.Debug().Preload("Mnemonic").Preload("Role").Preload("Telegram").Preload("Access", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("created_at > NOW() - INTERVAL '15 minutes'").Limit(1)
		}).
			Model(&models.User{}).
			Joins("JOIN auth_user_telegram telegram ON telegram.user_id = auth_users.id").
			Where("telegram.tg_id = ?", *payload.TgID).
			First(&user).Error
	} else if payload.ID != nil {
		err = controllers.DB.Debug().Preload("Mnemonic").Preload("Role").Preload("Telegram").Preload("Access", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("created_at > NOW() - INTERVAL '15 minutes'").Limit(1)
		}).
			Model(&models.User{}).
			Where("id = ?", *payload.ID).
			First(&user).Error
	} else {
		return http.StatusBadRequest, nil, "", errors.New("No valid identifier provided")
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, nil, "", errors.New("User not found")
		}
		return http.StatusInternalServerError, nil, "", err
	}

	return http.StatusOK, user, "User retrieved successfully.", nil
}

func CreateAccess(_data []byte) (int, interface{}, string, error) {
	var payload types.UserRequiredAssociationType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadRequest, nil, "", err
	}

	var user models.User
	err := controllers.DB.Debug().Model(&models.User{}).Where("id = ?", *payload.UserID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, nil, "", errors.New("User not found")
		}
		return http.StatusInternalServerError, nil, "", err
	}

	access := models.Access{
		UserID: payload.UserID,
	}

	if err := controllers.DB.Debug().Create(&access).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	return http.StatusCreated, nil, "Access created successfully.", nil
}

func Multisig(_data []byte) (int, interface{}, string, error) {

	var users []models.User
	if err := controllers.DB.Preload("Role").Preload("Access", func(tx *gorm.DB) *gorm.DB {
		return tx.Where("created_at > NOW() - INTERVAL '15 minutes'")
	}).Joins("JOIN auth_user_role_connection on auth_user_role_connection.user_id = auth_users.id").
		Joins("JOIN auth_user_roles on auth_user_roles.id = auth_user_role_connection.role_id AND auth_user_roles.title = ?", "owner").
		Where("EXISTS (SELECT 1 FROM auth_user_access WHERE auth_user_access.user_id = auth_users.id AND auth_user_access.created_at > NOW() - INTERVAL '15 minutes')").
		Find(&users).Error; err != nil {
		return http.StatusInternalServerError, nil, "", err
	}

	response := map[string]interface{}{
		"multisig": false,
	}

	// TODO:
	if len(users) >= 1 {
		response["multisig"] = true
	}

	fmt.Println(len(users))

	return http.StatusOK, response, "", nil
}
