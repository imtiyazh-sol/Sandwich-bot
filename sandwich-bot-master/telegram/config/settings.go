package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
)

var Telegram TelegramConfig

type TelegramConfig struct {
	BotToken     string `json:"bot_token"`
	ChannelID    int64  `json:"channel_id"`
	APIEndpoint  string `json:"api_endpoint"`
	AppSecret    string `json:"app_secret"`
	LogDirectory string `json:"log_directory"`
	MaxLogSize   int64  `json:"max_log_size"`
	Debug        bool   `json:"debug"`
}

func init() {
	file, err := os.ReadFile(".env.json")
	if err != nil {
		log.Fatalf("Error loading .env.json file: %v", err)
	}

	if err := json.Unmarshal(file, &Telegram); err != nil {
		log.Fatalf("Error unmarshalling .env.json: %v", err)
	}
}

var InternalEndpoint = func(_service, path string, args ...interface{}) (*url.URL, error) {

	services := map[string]map[string]interface{}{
		"auth": {
			"host": "auth",
			"port": 30084,
		},
		"bot": {
			"host": "bot",
			"port": 30083,
		},
	}

	service, ok := services[_service]
	if !ok {
		return nil, errors.New("service not found")
	}

	endpoint := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%v", service["host"], service["port"]),
		Path:   fmt.Sprintf("%s/api/v1/%s", service["host"], path),
	}

	if args != nil || len(args) > 0 {
		q := endpoint.Query()
		qParams := args[0].(map[string]interface{})
		for key, value := range qParams {
			q.Set(key, fmt.Sprintf("%v", value))
		}
		endpoint.RawQuery = q.Encode()
	}

	fmt.Println("URL", endpoint.String())

	return &endpoint, nil
}
