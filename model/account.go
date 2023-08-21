package model

import "encoding/json"

type AccountResponse struct {
	Jsonrpc string   `json:"jsonrpc"`
	ID      int      `json:"id"`
	Result  []string `json:"result"`
}

func (ar *AccountResponse) ToString() string {
	tmp, _ := json.Marshal(ar)
	return string(tmp)
}
