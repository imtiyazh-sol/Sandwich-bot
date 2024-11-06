package models

import "github.com/oklog/ulid/v2"

// Chains available in the system
type Blockchain struct {
	ModelExtended
	Uid      ulid.ULID `gorm:"uniqueIndex; not null" json:"uid"`
	Name     *string   `gorm:"not null" json:"name"`
	ChainID  *int      `gorm:"not null" json:"chain_id"`
	Currency *string   `json:"currency"`
	Order    []Order   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"order"`
}

func (Blockchain) TableName() string {
	return "bot_blockchains"
}
