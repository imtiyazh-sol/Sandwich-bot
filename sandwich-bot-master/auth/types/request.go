package types

type CreateUpdateUserType struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	TgID      *int   `json:"tg_id" validate:"required,omitempty"`
}

type HasAccessType struct {
	TgID *int `json:"tg_id" validate:"required,omitempty"`
}

type RetrieveUserType struct {
	TgID *int `json:"tg_id,omitempty"`
	ID   *int `json:"id,omitempty"`
}
