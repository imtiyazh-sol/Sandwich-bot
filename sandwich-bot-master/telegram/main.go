package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"telegram/config"
	"telegram/controllers"
	"telegram/handlers"
	"telegram/types"
	"telegram/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

func sendStartMenu(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Register", "register"),
			tgbotapi.NewInlineKeyboardButtonData("Get Access", "getAccess"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Current Settings", "currentSettings"),
			tgbotapi.NewInlineKeyboardButtonData("Set Settings", "setSettings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("System Wallets", "systemWallets"),
		),
		tgbotapi.NewInlineKeyboardRow(
			// tgbotapi.NewInlineKeyboardButtonData("DEX", "DEX"),
			tgbotapi.NewInlineKeyboardButtonData("Coin", "coin"),
			tgbotapi.NewInlineKeyboardButtonData("Contract", "contract"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Kill Switch", "killSwitch"),
		),
	)

	// Send a message with the inline keyboard
	msg := tgbotapi.NewMessage(message.Chat.ID, "Choose an option:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)

}

func main() {

	log.Printf("ChannelID: %d\n", config.Telegram.ChannelID)
	log.Printf("APIEndpoint: %s\n", config.Telegram.APIEndpoint)
	log.Printf("AppSecret: %s\n", config.Telegram.AppSecret)
	log.Printf("logDirectory: %s\n", config.Telegram.LogDirectory)
	log.Printf("maxLogSize: %d\n", config.Telegram.MaxLogSize)
	log.Printf("Debug: %v\n", config.Telegram.Debug)

	controllers.ConnectDatabase()
	controllers.Seed()

	// logFile, err := utils.GetLogFile()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer logFile.Close()

	// log.SetOutput(logFile)

	// log.Println("Bot starting")

	bot, err := tgbotapi.NewBotAPI(config.Telegram.BotToken)

	if err != nil {
		log.Println("Failed to initialize Telegram BOT API:", err)
		main()
	}

	bot.Debug = config.Telegram.Debug

	var latestUpdateID int

	for {
		u := tgbotapi.NewUpdate(latestUpdateID)
		u.Timeout = 10

		updates, err := bot.GetUpdates(u)
		if err != nil {
			log.Println("Failed to get updates from telegram", err)
			log.Println("Restarting bot...")
			main()
		}

		for _, update := range updates {
			if update.UpdateID >= latestUpdateID {
				latestUpdateID = update.UpdateID + 1
				var tgID int
				fmt.Println("UPDATE\n",
					"Channel Post", update.ChannelPost != nil, "\n",
					"Edited Message", update.EditedMessage != nil, "\n",
					"InlineQuery", update.InlineQuery != nil, "\n",
					"CallbackQuery", update.CallbackQuery != nil, "\n",
					"Message", update.Message != nil, "\n",
					"ChosenInlineResult", update.ChosenInlineResult != nil, "\n",
					"ShippingQuery", update.ShippingQuery != nil, "\n",
					"PreCheckoutQuery", update.PreCheckoutQuery != nil)
				if update.CallbackQuery != nil {
					tgID = update.CallbackQuery.From.ID
				} else if update.Message != nil {
					tgID = update.Message.From.ID
				} else {
					log.Println("Could not determine the type of message or its sender.")
					continue
				}
				//  && update.Message.IsCommand() {
				// } else if update.Message != nil && update.Message.ReplyToMessage != nil {
				// 	tgID = update.Message.From.ID
				// } else {
				// 	log.Println("Could not determine the type of message or its sender.")
				// }

				var userData map[string]interface{}
				var quickAccessUserData *types.QuickAccessUserDataType
				if tgID != 0 {
					_qParams := map[string]interface{}{
						"tg_id": tgID,
					}

					_user, err := handlers.RetrieveUser(_qParams)
					if err == nil {
						user, ok := _user.(*utils.Response)
						if ok {
							userData, ok = user.Data.(map[string]interface{})
							if !ok {
								log.Printf("Failed to cast user.Data to map[string]inerface{} tg_id: %v", tgID)
							}
						} else {
							log.Printf("Failed to cast user to utils.Response tg_id: %v", tgID)
						}
						quickAccessUserData = &types.QuickAccessUserDataType{}
						var __resp *utils.Response
						if __resp, err = handlers.GenericRequest("GET", "auth", "retrieve_multisig", map[string]interface{}{}); err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						dataMap, ok := __resp.Data.(map[string]interface{})
						if !ok {
							fmt.Println("Error: Data is not a map[string]interface{}")
							return // or handle the error appropriately
						}

						// Extract the "multisig" field
						multisig, ok := dataMap["multisig"].(bool)
						if !ok {
							fmt.Println("Error: 'multisig' field is either missing or not a string")
							return // or handle the error appropriately
						}
						quickAccessUserData.Multisig = multisig

						mnemonicData, ok := userData["mnemonic"].(map[string]interface{})
						if ok {
							quickAccessUserData.Mnemonic, ok = mnemonicData["phrase"].(string)
							if !ok {
								log.Printf("Mnemonic Phrase not found for user with tg_id: %v", tgID)
							}
						} else {
							log.Printf("Mnemonic not found for user with tg_id: %v", tgID)
						}

						userID, ok := userData["id"]
						if ok {
							userIDFloat, ok := userID.(float64)
							if ok {
								quickAccessUserData.ID = uint(userIDFloat)
							} else {
								log.Printf("ID not found or casting to float64 failed for user with tg_id: %v", tgID)
							}
						} else {
							log.Printf("ID not found for user with tg_id: %v", tgID)
						}

						tgData, ok := userData["telegram"].([]interface{})
						if ok {
							for _, _tg := range tgData {
								if tgID, ok := _tg.(map[string]interface{})["telegram_id"].(float64); ok {
									quickAccessUserData.TGiD = append(quickAccessUserData.TGiD, int(tgID))
								} else {
									log.Printf("Telegram not found for user with tg_id: %v", tgID)
								}
							}
						} else {
							log.Printf("Telegram not found for user with tg_id: %v", tgID)
						}

						accessData, ok := userData["access"].([]interface{})
						if ok {
							if len(accessData) > 0 {
								quickAccessUserData.HasAccess = true
							}
						} else {
							log.Printf("Access not found for user with tg_id: %v", tgID)
						}

						roleData, ok := userData["role"].([]interface{})
						if ok {
							for _, _rd := range roleData {
								if roleTitle, ok := _rd.(map[string]interface{})["title"].(string); ok {
									quickAccessUserData.Role = append(quickAccessUserData.Role, roleTitle)
									switch roleTitle {
									case "admin":
										quickAccessUserData.IsAdmin = true
									case "owner":
										quickAccessUserData.IsOwner = true
									}
								} else {
									log.Printf("Error parsing user roles for user with tg_id: %v", tgID)
								}
							}
						} else {
							// some error

						}
					}
				}

				fmt.Println("Message", update.Message != nil, "\n", "Callback", update.CallbackQuery != nil)
				// log.Println(update.Message.IsCommand(), update.Message.Command())
				if quickAccessUserData == nil {

					if update.CallbackQuery != nil && update.CallbackQuery.Data == "register" {

					} else if update.Message != nil {

					} else if update.Message != nil && update.Message.IsCommand() && update.Message.Command() == "start" {

					} else {
						var message *tgbotapi.Message
						if update.Message != nil {
							message = update.Message
						} else if update.CallbackQuery != nil {
							message = update.CallbackQuery.Message
						}

						msg := tgbotapi.NewMessage(message.Chat.ID, handlers.HandleError(handlers.ErrUserNotFound))
						bot.Send(msg)
						continue
					}
				}

				if update.Message != nil && update.Message.ReplyToMessage != nil {
					// response = "Please enter contract adddress:"
					if strings.Contains(update.Message.ReplyToMessage.Text, "Please enter contract address:") {
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}
						_payload := map[string]interface{}{
							"user_id":         quickAccessUserData.ID,
							"address_partial": update.Message.Text,
						}
						var _response *utils.Response
						if _response, err = handlers.GenericRequest("GET", "bot", "retrieve_contract", _payload); err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, _response.Message)
						msg.ParseMode = "Markdown"
						bot.Send(msg)
						continue
					}
					if strings.Contains(update.Message.ReplyToMessage.Text, "contract data with comma as delimiter to") || strings.Contains(update.Message.ReplyToMessage.Text, "contract address to") {
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}

						response := strings.Split(update.Message.Text, ",")

						endpoint := ""
						method := ""
						_payload := map[string]interface{}{
							"user_id":       quickAccessUserData.ID,
							"blockchain_id": 1,
						}

						if strings.Contains(update.Message.ReplyToMessage.Text, " add ") {
							endpoint += "connect"
							method = "PUT"
						} else if strings.Contains(update.Message.ReplyToMessage.Text, " delete ") {
							endpoint += "delete"
							method = "DELETE"
						}

						if strings.Contains(update.Message.ReplyToMessage.Text, "DEX") {
							endpoint += "_dex"
							if method == "PUT" {
								fmt.Println(response)
								if len(response) < 2 {
									msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Inccorect data provided. Expected dex router address and name.")
									bot.Send(msg)
									continue
								}
								_payload["type"] = strings.TrimSpace(response[1])
								_payload["address"] = strings.TrimSpace(response[0])
							} else if method == "DELETE" {
								if len(response) < 1 {
									msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Inccorect data provided. Expected dex router address.")
									bot.Send(msg)
									continue
								}
								_payload["address"] = strings.TrimSpace(response[0])
							}
						} else if strings.Contains(update.Message.ReplyToMessage.Text, "coin") {
							endpoint += "_coin"
							fmt.Println(method, endpoint)
							if method == "PUT" {
								fmt.Println(response)
								if len(response) < 3 {
									msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Inccorect data provided. Expected coin contract address, decimals and name.")
									bot.Send(msg)
									continue
								}
								_payload["address"] = strings.TrimSpace(response[0])
								_payload["name"] = strings.TrimSpace(response[2])
								decimals, err := strconv.ParseInt(strings.TrimSpace(response[1]), 10, 32)
								if err == nil {
									_payload["decimals"] = int32(decimals)
								}
								if err != nil {
									msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
									bot.Send(msg)
									continue
								}
							} else if method == "DELETE" {
								if len(response) < 1 {
									msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Inccorect data provided. Expected dex router address.")
									bot.Send(msg)
									continue
								}
								_payload["address"] = strings.TrimSpace(response[0])
							}
						}

						var _response *utils.Response
						if _response, err = handlers.GenericRequest(method, "bot", endpoint, _payload); err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						fmt.Println(_response.Message)

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, _response.Message)
						msg.ParseMode = "Markdown"
						bot.Send(msg)
					}
					// Updte Settings Flow
					if strings.Contains(update.Message.ReplyToMessage.Text, "Please enter contracts to") {
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}

						response := strings.Replace(update.Message.ReplyToMessage.Text, "Please enter contracts to", "", -1)
						response = strings.TrimSpace(strings.ToLower(strings.Replace(response, ":", "", -1)))
						contractsRaw := strings.Split(update.Message.Text, ",")
						var _contracts []string
						for _, contract := range contractsRaw {
							_contracts = append(_contracts, strings.TrimSpace(contract))
						}

						if len(_contracts) == 0 {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "No contracts were provided.")
							bot.Send(msg)
							continue
						}

						_payload := map[string]interface{}{
							"user_id": quickAccessUserData.ID,
							"address": _contracts,
						}

						if strings.Contains(response, "black") {
							_payload["blacklist"] = true
						}

						log.Println("PAYLOAD", _payload)

						var _response *utils.Response
						if _response, err = handlers.GenericRequest("PUT", "bot", "create_contract", _payload); err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						fmt.Println(_response.Message)

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, _response.Message)
						msg.ParseMode = "Markdown"
						bot.Send(msg)
					}
					if strings.Contains(update.Message.ReplyToMessage.Text, "wallet credentials") {
						if !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoOwnerAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.Multisig {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoMultisig))
							bot.Send(msg)
							continue
						}

						response := strings.Split(update.Message.ReplyToMessage.Text, " ")
						wallet := strings.Split(update.Message.Text, ",")
						if len(wallet) < 2 {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error: wallet address or private key is missing, please provide wallet credentials with comma as a delimeter. Address, private key, name (optional, ex: metamask)")
							bot.Send(msg)
							continue
						}

						_payload := map[string]interface{}{
							"blockchain_id": 1,
							"address":       strings.TrimSpace(wallet[0]),
							"pk":            strings.TrimSpace(wallet[1]),
							"wallet_type":   "main",
							"user_id":       quickAccessUserData.ID,
						}
						if len(wallet) > 2 {
							_payload["name"] = strings.TrimSpace(wallet[2])
						}
						if response[2] == "withdrawal" {
							_payload["wallet_type"] = "withdrawal"
						}

						var _response *utils.Response
						if _response, err = handlers.GenericRequest("PUT", "bot", "create_wallet", _payload); err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, _response.Message)
						bot.Send(msg)
						continue
					}
					if strings.Contains(update.Message.ReplyToMessage.Text, "Please enter the new value for") {
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}

						response := strings.Replace(update.Message.ReplyToMessage.Text, "Please enter the new value for", "", -1)
						response = strings.TrimSpace(strings.ToLower(strings.Replace(response, ":", "", -1)))
						responseSlice := strings.Split(response, " ")
						responseDB := strings.Join(responseSlice, "_")

						_payload := map[string]interface{}{
							"user_id": quickAccessUserData.ID,
						}
						var err error

						switch responseDB {
						case "gas_fee_max", "exit_gas", "gas_priority", "target_value_min", "target_value_max", "target_gas_markup_allowed", "usd_per_trade", "withdrawal_threshold":
							_payload[responseDB], err = decimal.NewFromString(update.Message.Text)
							fmt.Println("PAYLOAD", _payload[responseDB])
							if err != nil {
								log.Printf("Failed to convert to decimal.Decimal: %v", err)
							}
						case "gas_limit", "ttx_max_latency":
							_payload[responseDB], err = strconv.ParseUint(update.Message.Text, 10, 64)
							if err != nil {
								msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Incorrect value provided. Value should be unsigned integer: %v", err))
								bot.Send(msg)
								continue
							}
						case "slippage", "draw_down", "gas_tolerance":
							_payload[responseDB], err = strconv.ParseFloat(update.Message.Text, 64)
							if err != nil {
								msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Incorrect value provided. Value should be a float number: %v", err))
								bot.Send(msg)
								continue
							}
						case "deadline":
							_payload[responseDB], err = strconv.Atoi(update.Message.Text)
							if err != nil {
								msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Incorrect value provided. Value should be an integer: %v", err))
								bot.Send(msg)
								continue
							}
						}

						var _response *utils.Response
						if _response, err = handlers.GenericRequest("PATCH", "bot", "update_settings", _payload); err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						fmt.Println("RESP MESSAGE", _response.Message)

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, _response.Message)
						bot.Send(msg)
					}

					// CREATE ACCESS Flow
					if strings.Contains(update.Message.ReplyToMessage.Text, "Please provide words from mnemonic phrase at indexes") {
						// Handle password input

						indexesText := regexp.MustCompile(`\d+`).FindAllString(update.Message.ReplyToMessage.Text, -1)
						var indexes []int
						for _, indexText := range indexesText {
							index, err := strconv.Atoi(indexText)
							if err != nil {
								log.Printf("Error converting index to integer: %v\n", err)
								continue
							}
							indexes = append(indexes, index)
						}

						_mnemonicPartials := strings.Split(strings.TrimSpace(update.Message.Text), " ")
						if len(_mnemonicPartials) != 2 {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Provided details are incorrect.")
							bot.Send(msg)
							continue
						}

						mnemonicWords := strings.Split(fmt.Sprintf("%v", quickAccessUserData.Mnemonic), " ")

						if mnemonicWords[indexes[0]-1] == _mnemonicPartials[0] && mnemonicWords[indexes[1]-1] == _mnemonicPartials[1] {
							// Print the extracted values
							_payload := map[string]interface{}{
								"user_id": quickAccessUserData.ID,
							}
							_, _err := handlers.CreateAccess(_payload)
							if _err != nil {
								msg := tgbotapi.NewMessage(update.Message.Chat.ID, _err.Error())
								bot.Send(msg)
								continue
							}

							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Thank you, now you have access to create/update bot settings for 15 minutes.")
							bot.Send(msg)
						} else {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Provided details are incorrect.")
							bot.Send(msg)
						}
					}
				}

				if update.Message != nil && update.Message.IsCommand() {

					switch update.Message.Command() {
					case "start":
						sendStartMenu(bot, update.Message)
					default:
						// Process user input here
						response := "Hi Command👋"
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
						bot.Send(msg)
					}
				}
				if update.ChannelPost != nil && update.ChannelPost.Chat.ID == config.Telegram.ChannelID {
					switch update.ChannelPost.Text {
					case "/start":
					default:
						response := "Hi Channel 👋"
						msg := tgbotapi.NewMessage(update.ChannelPost.Chat.ID, response)
						bot.Send(msg)
					}
				}
				if update.CallbackQuery != nil {
					callbackData := update.CallbackQuery.Data
					callbackConfig := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
					bot.AnswerCallbackQuery(callbackConfig)

					switch callbackData {
					case "set_main_wallet", "set_withdrawal_wallet":
						if !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoOwnerAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.Multisig {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoMultisig))
							bot.Send(msg)
							continue
						}
						callbackDataParts := strings.Split(callbackData, "_")

						response := fmt.Sprintf("Please enter %s wallet credentials in set order sepparated by comma. E.g.: address, private key, name (optional, e.g.: metamask)", callbackDataParts[1])

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, response)
						msg.ReplyMarkup = tgbotapi.ForceReply{
							ForceReply: true,
							Selective:  true,
						}
						bot.Send(msg)

					case "set_gas_fee_max", "set_gas_limit", "set_ttx_max_latency", "set_gas_priority", "set_exit_gas", "set_target_value_min", "set_target_value_max", "set_target_gas_markup_allowed", "set_usd_per_trade", "set_draw_down", "set_gas_tolerance", "set_withdrawal_threshold", "set_deadline":
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}

						callbackDataParts := strings.Split(callbackData, "_")
						callbackDataParts = callbackDataParts[1:]

						for i, part := range callbackDataParts {
							callbackDataParts[i] = strings.Title(part)
						}

						// var fieldValueMap map[string]interface{}

						// currentValue := ""
						// if len(fieldValueMap) > 0 {
						// 	for _, _v := range fieldValueMap {
						// 		currentValue = fmt.Sprintf("%v", _v)
						// 	}
						// }

						// interim := fmt.Sprintf("**_⚙️ Current value for %v: ", strings.Join(callbackDataParts, " ")) + currentValue + "_**"
						// msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, interim)
						// msg.ParseMode = "Markdown"
						// bot.Send(msg)

						response := "Please enter the new value for "
						response += strings.Join(callbackDataParts, " ") + ":"

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, response)
						msg.ReplyMarkup = tgbotapi.ForceReply{
							ForceReply: true,
							Selective:  true,
						}
						bot.Send(msg)
					case "currentSettings":

						_payload := map[string]interface{}{
							"user_id": quickAccessUserData.ID,
						}

						var _response *utils.Response
						if _response, err = handlers.GenericRequest("GET", "bot", "retrieve_settings", _payload); err != nil {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						fmt.Println(_response.Data)

						// select user

						settingsMap, ok := _response.Data.(map[string]interface{})
						if !ok {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Error: failed to bind bot response to map")
							bot.Send(msg)
							continue
							// handle the error
						}
						// if err := json.Unmarshal(currentSettings.Settings, &settingsMap); err != nil {
						// 	log.Printf("Error unmarshalling settings: %v\n", err)
						// 	continue
						// }

						message := "```\n"
						message += fmt.Sprintf("%-25s | %s\n", "Key", "Value")
						message += strings.Repeat("-", 50) + "\n"

						for _k, _v := range settingsMap["settings"].(map[string]interface{}) {
							keyParts := strings.Split(_k, "_")
							formattedKey := strings.Title(strings.Join(keyParts, " "))
							valueStr := fmt.Sprintf("%v", _v)
							// if strings.Contains(strings.ToLower(_k), "wallet") && len(valueStr) > 7 {
							// 	valueStr = valueStr[:3] + "..." + valueStr[len(valueStr)-4:]
							// } else if len(valueStr) > 20 {
							// 	valueStr = valueStr[:17] + "..."
							// }
							message += fmt.Sprintf("%-25s | %s\n", formattedKey, valueStr)
						}

						_qParams := map[string]interface{}{
							"id": settingsMap["created_by"],
						}
						var createdByUsername interface{}
						_user, err := handlers.RetrieveUser(_qParams)
						if err == nil {
							user, ok := _user.(*utils.Response)
							if !ok {
								msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, errors.New("cannot convert user interface{}").Error())
								bot.Send(msg)
								continue
							}

							for _, _v := range user.Data.(map[string]interface{})["telegram"].([]interface{}) {
								_vMap, ok := _v.(map[string]interface{}) // Type assertion to map
								if !ok {
									continue // Handle the case where the assertion fails
								}
								createdByUsername = "@"
								username, ok := _vMap["username"].(string)
								if ok && username != "" && username != "0" {
									createdByUsername = "@" + username
									break
								}
							}
						} else {
							createdByUsername = "default"
						}

						message += "```"

						message += fmt.Sprintf("\nCreated By: %v", createdByUsername)

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, message)
						msg.ParseMode = "Markdown"
						bot.Send(msg)
					case "whiteListContract", "blackListContract":
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}
						response := ""
						action := ""
						if strings.Contains(callbackData, "white") {
							response = "Please enter contracts to whitelist: "
							action = "create"
						} else if strings.Contains(callbackData, "black") {
							response = "Please enter contracts to blacklist: "
							action = "delete"
						} else {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "internal error")
							bot.Send(msg)
							continue
						}

						tip := "**_👋 Tip: multiple contracts can be entered at the same time using comma as a delimeter_**"
						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, tip)
						msg.ParseMode = "Markdown"
						bot.Send(msg)

						fmt.Println(action)

						msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, response)
						msg.ReplyMarkup = tgbotapi.ForceReply{
							ForceReply: true,
							Selective:  true,
						}
						bot.Send(msg)
					case "listContracts", "findContracts", "listBlacklisted", "listWhitelisted":
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if strings.Contains(callbackData, "find") {
							response := "Please enter contract address:"

							tip := "**_👋 Tip: partial contract address can be used to locate contract in the system _**"
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, tip)
							msg.ParseMode = "Markdown"
							bot.Send(msg)

							msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, response)
							msg.ReplyMarkup = tgbotapi.ForceReply{
								ForceReply: true,
								Selective:  true,
							}
							bot.Send(msg)
						} else if strings.Contains(callbackData, "list") {
							_payload := map[string]interface{}{
								"user_id": quickAccessUserData.ID,
							}
							switch {
							case strings.Contains(callbackData, "Contract"):
								keyboard := tgbotapi.NewInlineKeyboardMarkup(
									tgbotapi.NewInlineKeyboardRow(
										tgbotapi.NewInlineKeyboardButtonData("Blacklisted", "listBlacklisted"),
										tgbotapi.NewInlineKeyboardButtonData("Whitelisted", "listWhitelisted"),
									),
									tgbotapi.NewInlineKeyboardRow(
										tgbotapi.NewInlineKeyboardButtonData("Back", "go_back"),
									),
								)

								msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Choose an option to list:")
								msg.ReplyMarkup = keyboard
								bot.Send(msg)
								continue
							case strings.Contains(callbackData, "Black"):
								_payload["blacklisted"] = 1
							}

							var _response *utils.Response
							if _response, err = handlers.GenericRequest("GET", "bot", "retrieve_contract", _payload); err != nil {
								msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
								bot.Send(msg)
								continue
							}

							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, _response.Message)
							msg.ParseMode = "Markdown"
							bot.Send(msg)
						} else {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "internal error")
							bot.Send(msg)
						}

					case "getAccess":
						// msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hello! Please enter some information:")
						_one, _two := utils.GenerateTwoUniqueRandomNumbers(1, 24)
						_one = 1
						_two = 2
						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Please provide words from mnemonic phrase at indexes %v and %v, sepparated by space:", _one, _two))
						msg.ReplyMarkup = tgbotapi.ForceReply{
							ForceReply: true,
							Selective:  true,
						}
						bot.Send(msg)

					case "setSettings":
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Gas Fee Max", "set_gas_fee_max"),
								tgbotapi.NewInlineKeyboardButtonData("Exit Gas (%)", "set_exit_gas"),
								// tgbotapi.NewInlineKeyboardButtonData("Gas Limit", "set_gas_limit"),
							),
							// tgbotapi.NewInlineKeyboardRow(
							// tgbotapi.NewInlineKeyboardButtonData("Gas Priority", "set_gas_priority"),
							// ),
							// tgbotapi.NewInlineKeyboardRow(
							// 	tgbotapi.NewInlineKeyboardButtonData("Deadline (mm)", "set_deadline"),
							// ),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Target Value Min", "set_target_value_min"),
								tgbotapi.NewInlineKeyboardButtonData("Target Value Max", "set_target_value_max"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Target Gas Markup Allowed (%)", "set_target_gas_markup_allowed"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("TTX Max Latency (ms)", "set_ttx_max_latency"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("USD Per Trade (%)", "set_usd_per_trade"),
								tgbotapi.NewInlineKeyboardButtonData("Draw Down (%)", "set_draw_down"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Gas Tolerance (%)", "set_gas_tolerance"),
								tgbotapi.NewInlineKeyboardButtonData("Withdrawal Threshold", "set_withdrawal_threshold"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Main Wallet", "set_main_wallet"),
								tgbotapi.NewInlineKeyboardButtonData("Withdrawal Wallet", "set_withdrawal_wallet"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Back", "go_back"),
							),
						)

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Choose an option:")
						msg.ReplyMarkup = keyboard
						bot.Send(msg)
					case "go_back":
						sendStartMenu(bot, update.CallbackQuery.Message)
					case "set_kill_switch_on", "set_kill_switch_off":
						if !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoOwnerAccess))
							bot.Send(msg)
							continue
						}

						ks := strings.Split(callbackData, "_")
						state := ks[len(ks)-1]

						_payload := map[string]interface{}{
							"user_id": quickAccessUserData.ID,
						}
						switch state {
						case "on":
							_payload["is_on"] = true
						case "off":
							_payload["is_on"] = false
						}

						var _response *utils.Response
						if _response, err = handlers.GenericRequest("PATCH", "bot", "toggle_killswitch", _payload); err != nil {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, _response.Message)
						//  fmt.Sprintf("Kill Switch is %v", state))
						bot.Send(msg)

					case "killSwitch":
						// if !quickAccessUserData.HasAccess {
						// 	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
						// 	bot.Send(msg)
						// 	continue
						// }
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("On", "set_kill_switch_on"),
								tgbotapi.NewInlineKeyboardButtonData("Off", "set_kill_switch_off"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Back", "go_back"),
							),
						)

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Please, confirm that you want to activate kill switch, which will stop all bot operations:")
						msg.ReplyMarkup = keyboard
						bot.Send(msg)
					case "register":
						userMnemonic, err := handlers.CreateUser(map[string]interface{}{
							"tg_id":      update.CallbackQuery.From.ID,
							"first_name": update.CallbackQuery.From.FirstName,
							"last_name":  update.CallbackQuery.From.LastName,
							"username":   update.CallbackQuery.From.UserName,
						})
						if err != nil {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "⚠️ Warning: failed to register, "+err.Error())
							bot.Send(msg)
							continue
						}

						tip := "**_👋 Tip: Please, save the mnemonic phrase and then delete the message afterwards. _**"
						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, tip)
						msg.ParseMode = "Markdown"
						bot.Send(msg)

						message := fmt.Sprintf("`%s`", userMnemonic)
						msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, message)
						msg.ParseMode = "Markdown"
						bot.Send(msg)

						msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Registration successful")
						msg.ParseMode = "Markdown"
						bot.Send(msg)
					case "systemWallets":
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Main", "main_wallet"),
								tgbotapi.NewInlineKeyboardButtonData("Withdrawal", "withdrawal_wallet"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Back", "go_back"),
							),
						)

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Please select the type of wallet for which you would like to view information:")
						msg.ReplyMarkup = keyboard
						bot.Send(msg)
					case "main_wallet", "withdrawal_wallet":
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}

						_message := strings.Split(callbackData, "_")

						_payload := map[string]interface{}{
							"user_id":     quickAccessUserData.ID,
							"wallet_type": _message[0],
						}

						var _response *utils.Response
						if _response, err = handlers.GenericRequest("GET", "bot", "retrieve_wallet", _payload); err != nil {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, _response.Message)
						msg.ParseMode = "Markdown"
						bot.Send(msg)
					case "DEX", "coin":
						capitalizedCallbckData := strings.Title(callbackData)
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("List %ss", capitalizedCallbckData), fmt.Sprintf("list_%ss", callbackData)),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Add %s", capitalizedCallbckData), fmt.Sprintf("add_%ss", callbackData)),
								tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Delete %s", capitalizedCallbckData), fmt.Sprintf("delete_%ss", callbackData)),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Back", "go_back"),
							),
						)
						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Please select action")
						msg.ReplyMarkup = keyboard
						bot.Send(msg)
					case "add_DEXs", "add_coins", "delete_DEXs", "delete_coins":
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}
						fmt.Println("DELTE COIN", quickAccessUserData.IsAdmin, quickAccessUserData.IsOwner)
						if !quickAccessUserData.IsAdmin && !quickAccessUserData.IsOwner {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAdminOrOwnerAccess))
							bot.Send(msg)
							continue
						}
						commandSlice := strings.Split(callbackData, "_")
						message := "Please provide"
						if commandSlice[0] == "add" {
							message += fmt.Sprintf(" %s contract data with comma as delimiter to %s", commandSlice[1][:len(commandSlice[1])-1], commandSlice[0])
							message += " into the system."
						} else {
							message += fmt.Sprintf(" %s contract address to %s", commandSlice[1][:len(commandSlice[1])-1], commandSlice[0])
							message += " from the system."
						}

						if !strings.Contains(callbackData, "delete") {
							if strings.Contains(callbackData, "DEX") {
								message += " E.g.: address, name (uniswapv3 or quickswap)"
							} else {
								message += " E.g.: address, decimals (6 or 18), name (usdt or dai)"
							}
						}

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, message)
						msg.ReplyMarkup = tgbotapi.ForceReply{
							ForceReply: true,
							Selective:  true,
						}
						bot.Send(msg)
					case "list_DEXs", "list_coins":
						if !quickAccessUserData.HasAccess {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, handlers.HandleError(handlers.ErrNoAccess))
							bot.Send(msg)
							continue
						}

						resource := ""
						if strings.Contains(callbackData, "DEX") {
							resource = "dex"
						} else if strings.Contains(callbackData, "coin") {
							resource = "coin"
						}

						_payload := map[string]interface{}{
							"user_id":       quickAccessUserData.ID,
							"blockchain_id": 1,
						}

						var _response *utils.Response
						if _response, err = handlers.GenericRequest("GET", "bot", "retrieve_"+resource, _payload); err != nil {
							msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Error: %v", err))
							bot.Send(msg)
							continue
						}

						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, _response.Message)
						msg.ParseMode = "Markdown"
						bot.Send(msg)
					case "contract":
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("List Contracts", "listContracts"),
								tgbotapi.NewInlineKeyboardButtonData("Find Contract", "findContracts"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Whitelist Contract", "whiteListContract"),
								tgbotapi.NewInlineKeyboardButtonData("Blacklist Contract", "blackListContract"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Back", "go_back"),
							),
						)
						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Please select action")
						msg.ReplyMarkup = keyboard
						bot.Send(msg)
					default:
						response := "Hi Callback 👋" + callbackData
						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, response)
						bot.Send(msg)
					}

				}
			}
		}
	}
}
