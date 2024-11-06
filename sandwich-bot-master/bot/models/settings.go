package models

import "gorm.io/datatypes"

type Settings struct {
	ModelExtended
	Active       *bool          `gorm:"default:true" json:"active"`
	BlockchainID *uint          `gorm:"not null" json:"blockchain_id"`
	Settings     datatypes.JSON `gorm:"not null" json:"settings"`
}

func (Settings) TableName() string {
	return "bot_settings"
}

type Wallet struct {
	ModelExtended
	BlockchainID
	Active
	Name       string      `gorm:"index" json:"name"`
	PrivateKey *string     `gorm:"not null" json:"pk"`
	Address    *string     `gorm:"uniqueIndex;not null" json:"address"`
	Type       *WalletType `gorm:"not null" json:"type"`
}

func (Wallet) TableName() string {
	return "bot_wallets"
}

type KillSwitch struct {
	ModelExtended
	IsOn *bool `gorm:"default:false" json:"is_on"`
}

func (KillSwitch) TableName() string {
	return "bot_killswitch"
}
