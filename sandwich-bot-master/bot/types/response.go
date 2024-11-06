package types

import (
	"time"
)

type CreateUpdateBotSettingsRespType struct {
	ID uint `json:"id,omitempty"`
}

type RetrieveDEXRespType struct {
	Address   *string   `json:"address"`
	Type      *string   `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *uint     `json:"created_by"`
	UpdatedBy *uint     `json:"updated_by"`
}

type RetrieveContractRespType struct {
	ID        uint      `json:"id"`
	Address   *string   `json:"address"`
	Blacklist *bool     `json:"blacklist"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *uint     `json:"created_by"`
	UpdatedBy *uint     `json:"updated_by"`
}

type RetrieveWalletRespType struct {
	Address *string `json:"address"`
	Name    *string `json:"name"`
	// Type    *string `json:"type"`
}
