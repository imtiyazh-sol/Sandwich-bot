package models

// White/Blacklisted cotracts
type Contract struct {
	ModelExtended
	BlockchainID
	Address   *string `gorm:"uniqueIndex;not null" json:"address"`
	Blacklist *bool   `gorm:"default:false" json:"blacklist"`
	Name      string  `json:"name"`
	Decimals  *int32  `gorm:"not null;default:18" json:"decimals"`
}

func (Contract) TableName() string {
	return "bot_contracts"
}

// Allowed Dexes
type DEX struct {
	ModelExtended
	BlockchainID
	Address *string `gorm:"uniqueIndex;not null" json:"address"`
	Type    *string `gorm:"not null" json:"type"`
}

func (DEX) TableName() string {
	return "bot_dexs"
}

// Tradable coins
type Coin struct {
	ModelExtended
	BlockchainID
	Name     *string `gorm:"index;not null" json:"name"`
	Decimals *int32  `gorm:"not null" json:"decimals"`
	Address  *string `gorm:"uniqueIndex;not null" json:"address"`
}

func (Coin) TableName() string {
	return "bot_coins"
}

type Allowance struct {
}
