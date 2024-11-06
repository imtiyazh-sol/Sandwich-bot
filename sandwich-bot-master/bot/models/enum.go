package models

type ENUM interface {
	IsValid() bool
}

type TransactionType string
type StatusType string
type WalletType string

const (
	// TransactionType
	Outbound TransactionType = "outbound"
	Inbound  TransactionType = "inbound"
	// StatusType
	Pending   StatusType = "pending"
	Indexing  StatusType = "indexing"
	Confirmed StatusType = "confirmed"
	Reverted  StatusType = "reverted"
	Fail      StatusType = "fail"
	// WalletType
	Withdrawal WalletType = "withdrawal"
	Main       WalletType = "main"
)

var ValidWalletTypes = []WalletType{Withdrawal, Main}

func (wt WalletType) IsValid() bool {
	for _, _vwt := range ValidWalletTypes {
		if wt == _vwt {
			return true
		}
	}
	return false
}

var ValidTransactionTypes = []TransactionType{Outbound, Inbound}

func (tt TransactionType) IsValid() bool {
	for _, _vtt := range ValidTransactionTypes {
		if tt == _vtt {
			return true
		}
	}
	return false
}

var ValidStatusTypes = []StatusType{Pending, Indexing, Confirmed, Reverted, Fail}

func (st StatusType) IsValid() bool {
	for _, _vst := range ValidStatusTypes {
		if st == _vst {
			return true
		}
	}
	return false
}

func Validate(e ENUM) bool {
	return e.IsValid()
}
