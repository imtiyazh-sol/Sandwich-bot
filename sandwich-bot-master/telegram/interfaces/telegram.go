package interfaces

import (
	"fmt"
	"net/http"
	"telegram/config"
	"telegram/types"
	"telegram/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func Ping(_data []byte) (int, interface{}, string, error) {
	return http.StatusOK, nil, "", nil
}

func SendMessageToChannel(_data []byte) (int, interface{}, string, error) {
	var payload types.SendMessageType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadRequest, nil, "", err
	}

	botToken := config.Telegram.BotToken
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return http.StatusInternalServerError, nil, "", fmt.Errorf("failed to create bot: %v", err)
	}

	msg := tgbotapi.NewMessage(payload.ChannelID, payload.Message)
	_, err = bot.Send(msg)
	if err != nil {
		return http.StatusInternalServerError, nil, "", fmt.Errorf("failed to send message: %v", err)
	}

	return http.StatusOK, nil, "Message sent successfully", nil
}

func RetrieveBotSettings(_data []byte) (int, interface{}, string, error) {
	var payload types.RetrieveSettingsType

	if err := utils.Parse(_data, &payload); err != nil {
		return http.StatusBadRequest, nil, "", err
	}

	// TODO Filters
	// var settings *models.BotSettings
	// if err := controllers.DB.Debug().First(&settings).Error; err != nil {
	// 	if errors.Is(err, gorm.ErrRecordNotFound) {
	// 		return http.StatusNotFound, nil, "", fmt.Errorf("settings not exists")
	// 	}

	// 	return http.StatusInternalServerError, nil, "", err
	// }

	// return http.StatusOK, settings, "", nil
	return http.StatusOK, nil, "", nil
}
