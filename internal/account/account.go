package account

import (
	"encoding/json"
	"fmt"
)

type Account struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	RpcUrl     string `json:"rpc_url"`
}

type AccountConfig struct {
	Accounts []Account `json:"accounts"`
}

func GetAccount(config string) ([]Account, error) {
	var accountConfig AccountConfig
	err := json.Unmarshal([]byte(config), &accountConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	if len(accountConfig.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts found in config")
	}

	// Return the first account from the config
	return accountConfig.Accounts, nil
}
