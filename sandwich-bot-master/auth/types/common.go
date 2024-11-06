package types

type UserRequiredAssociationType struct {
	UserID *uint `json:"user_id" validate:"required,omitempty"`
}