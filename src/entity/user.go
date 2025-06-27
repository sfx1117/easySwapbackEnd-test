package entity

type LoginReq struct {
	ChainId   int    `json:"chain_id"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Address   string `json:"address"`
}

type UserLoginInfo struct {
	Token     string `json:"token"`
	IsAllowed bool   `json:"isAllowed"`
}

type UserLoginRes struct {
	Result interface{} `json:"result"`
}

type UserLoginMessageRes struct {
	Address string `json:"address"`
	Message string `json:"message"`
}

type UserSignStatusRes struct {
	IsSigned bool `json:"is_signed"`
}
