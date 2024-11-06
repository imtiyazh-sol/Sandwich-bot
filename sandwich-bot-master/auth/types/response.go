package types

type CreateUpdateBotSettingsRespType struct {
	ID uint `json:"id,omitempty"`
}

type APIResponseUserCreateType struct {
	ID       uint   `json:"id"`
	Mnemonic string `json:"mnemonic"`
}

type APIResponseUserHasAccessType struct {
	Access bool `json:"access"`
}
