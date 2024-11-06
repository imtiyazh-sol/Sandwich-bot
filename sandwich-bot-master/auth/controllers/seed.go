package controllers

import (
	"auth/models"

	"gorm.io/gorm/clause"
)

func Seed() {
	_true := true
	// _false := false

	roles := []models.Role{
		{Title: "guest", Weight: 1, Active: models.Active{Active: &_true}},
		{Title: "user", Weight: 10, Active: models.Active{Active: &_true}},
		{Title: "admin", Weight: 100, Active: models.Active{Active: &_true}},
		{Title: "owner", Weight: 1000, Active: models.Active{Active: &_true}},
	}

	DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "title"}},
		DoUpdates: clause.AssignmentColumns([]string{"weight", "active"}),
	}).Create(&roles)
}
