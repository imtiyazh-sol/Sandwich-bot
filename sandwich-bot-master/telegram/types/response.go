package types

type CreateUpdateBotSettingsRespType struct {
	ID uint `json:"id,omitempty"`
}

type APIResponseUserCreateType struct {
	ID       uint   `json:"id"`
	Mnemonic string `json:"mnenominc"`
}

type APIResponseUserHasAccessType struct {
	Access bool `json:"access"`
}
