package types

type QuickAccessUserDataType struct {
	ID        uint     `json:"id"`
	Mnemonic  string   `json:"mnemonic"`
	TGiD      []int    `json:"tg_id"`
	HasAccess bool     `json:"has_access"`
	Role      []string `json:"role"`
	IsAdmin   bool     `json:"is_admin"`
	IsOwner   bool     `json:"is_owner"`
	Multisig  bool     `json:"multisig"`
}
