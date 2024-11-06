package types

import (
	"bot/models"

	"github.com/shopspring/decimal"
)

type UpdateSettingsReqType struct {
	GasFeeMax              *decimal.Decimal `json:"gas_fee_max,omitempty"`
	GasLimit               *uint64          `json:"gas_limit,omitempty"`
	GasPriority            *decimal.Decimal `json:"gas_priority,omitempty"`
	TTXMaxLatency          *uint64          `json:"ttx_max_latency,omitempty"`
	ExitGas                *decimal.Decimal `json:"exit_gas,omitempty"`
	Slippage               *float64         `json:"slippage,omitempty"`
	TargetValueMin         *decimal.Decimal `json:"target_value_min,omitempty"`
	TargetValueMax         *decimal.Decimal `json:"target_value_max,omitempty"`
	TargetGasMarkupAllowed *decimal.Decimal `json:"target_gas_markup_allowed,omitempty"`
	UsdPerTrade            *decimal.Decimal `json:"usd_per_trade,omitempty"`
	Deadline               *int             `json:"deadline,omitempty"`
	DrawDown               *float64         `json:"draw_down,omitempty"`
	GasTolerance           *float64         `json:"gas_tolerance,omitempty"`
	WithdrawalThreshold    *decimal.Decimal `json:"withdrawal_threshold,omitempty"`
	UserID                 *uint            `json:"user_id" validate:"required"`
}

type CreateWalletReqType struct {
	Address    *string            `json:"address" validate:"required"`
	Name       string             `json:"name,omitempty"`
	PrivateKey *string            `json:"pk" validate:"required"`
	WalletType *models.WalletType `json:"wallet_type" validate:"required"`
	UserID     *uint              `json:"user_id" validate:"required"`
}

type ToggleKillSwitchReqType struct {
	UserID *uint `json:"user_id" validate:"required"`
	IsOn   *bool `json:"is_on" validate:"required"`
}

type WhiteBlacklistContractsReqType struct {
	UserID    *uint     `json:"user_id" validate:"required"`
	Blacklist *bool     `json:"blacklist"`
	Address   []*string `json:"address" validate:"required"`
	// AddressV2 []map[string]interface{} `json:"address_v2" validate:"required"`
}

type WhiteBlackListContractsV2ReqType struct {
	Address  *string          `json:"address" validate:"required"`
	Decimals *decimal.Decimal `json:"decimals"`
}

type RetrieveContractReqType struct {
	UserID         *uint       `json:"user_id" validate:"required"`
	AddressPartial interface{} `json:"address_partial"`
	Blacklisted    *int        `json:"blacklisted"`
}

type CreateUpdateDEXReqType struct {
	UserRequiredType
	Address *string `json:"address" validate:"required"`
	Type    *string `json:"type" validate:"required"`
}

type DeleteDEXReqType struct {
	UserRequiredType
	Address *string `json:"address" validate:"required"`
}

type RetrieveDEXReqType struct {
	UserRequiredType
	BlockchainType
}

type RetrieveCoinReqType struct {
	UserRequiredType
	BlockchainType
}

//		ModelExtended
//		BlockchainID
//		Name     *string          `gorm:"index;not null" json:"name"`
//		Decimals *decimal.Decimal `gorm:"not null" json:"decimals"`
//		Address  *string          `gorm:"uniqueIndex;not null" json:"address"`
//	}

type CreateUpdateCoinReqType struct {
	UserRequiredType
	BlockchainType
	Name     *string `json:"name" validate:"required"`
	Decimals *int32  `json:"decimals" validate:"required"`
	Address  *string `json:"address" validate:"required"`
}

type DeleteCoinReqType struct {
	UserRequiredType
	Address *string `json:"address" validate:"required"`
}

type RetrieveWalletReqType struct {
	UserRequiredType
	WalletType *models.WalletType `json:"wallet_type" validate:"required"`
}
