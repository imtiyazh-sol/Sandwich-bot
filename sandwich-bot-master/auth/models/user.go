package models

type User struct {
	Model
	Active
	Verified
	Telegram []Telegram `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"telegram"`
	Mnemonic Mnemonic   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"mnemonic"`
	Role     []Role     `gorm:"many2many:auth_user_role_connection" json:"role"`
	Access   []Access   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"access"`
}

func (User) TableName() string {
	return "auth_users"
}

type Role struct {
	Model
	Active
	Title  string `gorm:"uniqueIndex;not null" json:"title"`
	Weight int    `gorm:"not null" json:"weight"`
}

func (Role) TableName() string {
	return "auth_user_roles"
}

type Telegram struct {
	Model
	TgID      *int   `gorm:"uniqueIndex:idx_user_tg;not null" json:"telegram_id"`
	UserID    *uint  `gorm:"not null" json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

func (Telegram) TableName() string {
	return "auth_user_telegram"
}

type Mnemonic struct {
	Model
	UserID *uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	Phrase *string `gorm:"not null" json:"phrase"`
}

func (Mnemonic) TableName() string {
	return "auth_user_mnemonics"
}

type Access struct {
	Model
	UserID *uint `gorm:"not null" json:"user_id"`
}

func (Access) TableName() string {
	return "auth_user_access"
}
