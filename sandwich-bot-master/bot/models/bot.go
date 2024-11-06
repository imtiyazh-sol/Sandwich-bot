package models

import "gorm.io/datatypes"

type Order struct {
	Model
	Active
	Status
	BlockchainID

	Hash     *string         `gorm:"not null" json:"hash"`
	Method   string          `method:"method"`
	Data     *datatypes.JSON `gorm:"not null" json:"data"`
	Reason   string          `json:"reason"`
	Settings *datatypes.JSON `gorm:"not null" json:"settings"`
	Receipt  *datatypes.JSON `json:"receipt"`

	Transaction []Transaction `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"transactions"`
}

func (Order) TableName() string {
	return "bot_orders"
}

type Reject struct {
	Model
	Hash       *string         `gorm:"uniqueIndex;not null" json:"hash"`
	Method     string          `json:"method"`
	Transction *datatypes.JSON `gorm:"not null" json:"transaction"`
	Reason     *string         `gorm:"not null" json:"reason"`
}

func (Reject) TableName() string {
	return "bot_rejects"
}

type Transaction struct {
	Model
	Verified
	Status
	Hash     *string         `gorm:"index;not null" json:"hash"`
	Contract *string         `gorm:"index;not null" json:"contract"`
	RawData  datatypes.JSON  `gorm:"not null" json:"raw_data"`
	Type     TransactionType `gorm:"index;not null" json:"type"`
	Receipt  *datatypes.JSON `json:"receipt"`
	OrderID  *uint           `json:"order_id"`
}

func (Transaction) TableName() string {
	return "bot_external_transactions"
}
