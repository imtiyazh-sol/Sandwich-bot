package models

import (
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type ModelExtended struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	CreatedBy *uint          `gorm:"index;not null" json:"created_by"`
	UpdatedBy *uint          `gorm:"index;not null" json:"updated_by"`
	DeletedBy *uint          `gorm:"index" json:"deleted_by"`
}

type Active struct {
	Active *bool `gorm:"default:true" json:"active"`
}

type Public struct {
	Public *bool `gorm:"default:false" json:"public"`
}

type Verified struct {
	Verified *bool `gorm:"default:false" json:"verified"`
}

type Status struct {
	Status StatusType `gorm:"default:indexing" json:"status"`
}

type BlockchainID struct {
	BlockchainID *uint `gorm:"not null" json:"blockchain_id"`
}
