package main

import (
	// _ "bot/docs"
	_ "auth/config"
	"auth/controllers"
	"auth/interfaces"
	"auth/middleware"
	"errors"
	"fmt"
	"net/http"
	"os"

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

	auth := r.Group(fmt.Sprintf("auth/api/%s", apiVersion))
	auth.Use()
	{
		user := auth.Group("/")
		user.Use()
		{
			user.PUT("/create_user", middleware.Wrapper(interfaces.CreateUser))
			user.GET("/retrieve_user", middleware.Wrapper(interfaces.RetrieveUser))
			user.GET("/retrieve_access", middleware.Wrapper(interfaces.RetrieveAccess))
			user.PUT("/create_access", middleware.Wrapper(interfaces.CreateAccess))
			user.GET("/retrieve_multisig", middleware.Wrapper(interfaces.Multisig))
		}
	}

	r.NoRoute(func(c *gin.Context) {
		c.AbortWithError(http.StatusNotFound, errors.New("Endpoint not found."))
	})

	r.Run(appPort)

}
