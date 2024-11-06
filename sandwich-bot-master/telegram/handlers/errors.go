package handlers

import (
	"fmt"
	"strings"
)

var Test = func(t string) {
	fmt.Println(t)
}

type ErrorType struct {
	Message string `json:"message"`
	Prefix  string `json:"prefix"`
	Type    string `json:"type"`
	Ico     string `json:"ico"`
}

var (
	ErrNoAccess = ErrorType{
		Message: "Currently, you do not have access. Please create access for your account.",
		Prefix:  "",
		Ico:     "⚠️",
		Type:    "warning",
	}
	ErrNoOwnerAccess = ErrorType{
		Message: "Owner role and multisig access are required. ",
		Prefix:  "",
		Ico:     "⚠️",
		Type:    "warning",
	}
	ErrNoAdminOrOwnerAccess = ErrorType{
		Message: "Admin role or owner role is required.",
		Prefix:  "",
		Ico:     "⚠️",
		Type:    "warning",
	}
	ErrNoMultisig = ErrorType{
		Message: "Multisig access is required",
		Prefix:  "",
		Ico:     "⚠️",
		Type:    "warning",
	}
	ErrUserExists = ErrorType{
		Message: "User already exists",
		Prefix:  "",
		Ico:     "⚠️",
		Type:    "notice",
	}
	ErrUserNotFound = ErrorType{
		Message: "User not found. Use Register button to create a user",
		Prefix:  "",
		Ico:     "⚠️",
		Type:    "notice",
	}
	ErrInternalError = ErrorType{
		Message: "Internal error occured",
		Prefix:  "",
		Ico:     "❌",
		Type:    "error",
	}
)

var (
	HandleError = func(err ErrorType, args ...interface{}) string {
		return err.Ico + " " + err.Prefix + " " + strings.ToUpper(err.Type) + ": " + err.Message
	}
)
