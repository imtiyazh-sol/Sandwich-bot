package types

type SendMessageType struct {
	ChannelID int64  `json:"channel_id" validate:"required,omitempty"`
	Message   string `json:"message" validate:"required,omitempty"`
}

type RetrieveSettingsType struct {
	
}
