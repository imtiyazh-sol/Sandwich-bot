package main

import (
	// _ "bot/docs"
	_ "bot/config"
	"bot/controllers"
	"bot/handlers"
	"bot/interfaces"
	"bot/middleware"
	"bot/utils"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const apiVersion = "v1"

func main() {
	r := gin.Default()

	appPort := os.Getenv("APP_PORT")

	if appPort != "" {
		appPort = ":" + appPort
	}

	controllers.ConnectDatabase()
	controllers.Seed()
	r.Use(middleware.ErrorHandler())

	health := r.Group("/health")
	health.Use()
	{
		health.GET("/check", func(c *gin.Context) {
			middleware.Response(c, http.StatusOK, nil, "", "success")
		})
	}

	bot := r.Group(fmt.Sprintf("bot/api/%s", apiVersion))
	bot.Use()
	{
		settings := bot.Group("/")
		settings.Use()
		{
			settings.GET("/retrieve_settings", middleware.Wrapper(interfaces.RetrieveSettings))
			settings.PATCH("/update_settings", middleware.Wrapper(interfaces.UpdateSettings))

			settings.GET("/retrieve_wallet", middleware.Wrapper(interfaces.RetrieveWallet))
			settings.PUT("/create_wallet", middleware.Wrapper(interfaces.CreateWallet))

			settings.GET("/retrieve_killswitch", middleware.Wrapper(interfaces.RetrieveKillSwitch))
			settings.PATCH("/toggle_killswitch", middleware.Wrapper(interfaces.ToggleKillSwitch))
		}
		contracts := bot.Group("/")
		contracts.Use()
		{
			contracts.GET("/retrieve_contract", middleware.Wrapper(interfaces.RetrieveContract))
			contracts.PUT("/create_contract", middleware.Wrapper(interfaces.WhiteBlacklistContract))

			contracts.GET("/retrieve_dex", middleware.Wrapper(interfaces.RetrieveDEX))
			contracts.PUT("/connect_dex", middleware.Wrapper(interfaces.CreateUpdateDEX))
			contracts.DELETE("/delete_dex", middleware.Wrapper(interfaces.DeleteDEX))

			contracts.GET("/retrieve_coin", middleware.Wrapper(interfaces.RetrieveCoin))
			contracts.PUT("/connect_coin", middleware.Wrapper(interfaces.CreteUpdateCoin))
			contracts.DELETE("/delete_coin", middleware.Wrapper(interfaces.DeleteCoin))
		}
	}

	// Background tasks
	// TODO -- possible memory leak
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered in f: %v", r)
					}
				}()
				ticker := time.NewTicker(5 * time.Second)
				for ; true; <-ticker.C {
					utils.GetPairPrice([]string{"matic-network", "ethereum"}, []string{"usd"})
					if _, err := utils.GetGasPrice(); err != nil {
						log.Print("Failed to retrieve gas price", err)
					}
				}
			}()
		}
	}()

	go func() {
		// for {
		// 	func() {
		// 		defer func() {
		// 			if r := recover(); r != nil {
		// 				log.Printf("Recovered in f: %v", r)
		// 				return // Exit the goroutine if a panic occurs
		// 			}
		// 		}()
		// Currently for polygon
		handlers.UpdateGlobalSettings(1)
		handlers.Run("polygon")
		// 	}()
		// }
	}()

	r.NoRoute(func(c *gin.Context) {
		c.AbortWithError(http.StatusNotFound, errors.New("Endpoint not found."))
	})

	r.Run(appPort)

}
