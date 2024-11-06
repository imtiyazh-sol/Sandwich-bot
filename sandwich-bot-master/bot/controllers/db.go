package controllers

import (
	"bot/models"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	var err error

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")

	log.Println("Connecting to ", host, port)

	dbuser := os.Getenv("POSTGRES_USER")
	dbpassword := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai", host, dbuser, dbpassword, dbname, port)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("Failed to connect to database!")
	}

	DB.AutoMigrate(
		&models.Blockchain{},
		&models.Order{},
		&models.Reject{},
		&models.Transaction{},
		&models.Settings{},
		&models.Wallet{},
		&models.KillSwitch{},
		&models.Contract{},
		&models.DEX{},
		&models.Coin{},
	)
}
