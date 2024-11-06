package models

import "gorm.io/datatypes"


type Logger struct {
	Model
	UserID      int
	UserRole    int
	RequestID   int
	RequestPath string
	Method      string
	Payload     datatypes.JSON
	UserAgent   string
	ClientIP    string
}

func (Logger) TableName() string {
	return "bot_request_logger"
}
