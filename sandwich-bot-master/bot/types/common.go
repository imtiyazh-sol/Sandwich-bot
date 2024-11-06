package types

type UserRequiredType struct {
	UserID *uint `json:"user_id" validate:"required"`
}

type BlockchainType struct {
	BlockchainID *uint `json:"blockchain_id" validate:"required"`
}
