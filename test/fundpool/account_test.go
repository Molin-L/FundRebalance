package fundpool

import (
	"testing"

	"github.com/Molin-L/FundRebalance/internal/fundpool/account"
	"github.com/Molin-L/FundRebalance/pkg/idl/pb/blockchains"
)

func TestInfraAccount(t *testing.T) {
	test_account := account.InfraAccount{}

	address := account.Address{
		RpcUrl:  "https://sepolia.infura.io/v3/fdb752e0f4c340278c6082540125c18e",
		Address: "0x30C7FefEAd3A512111cC40880306bedC205832dd",
	}

	balance, err := test_account.GetBalance(address, blockchains.Coin_ETH)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("balance: %f", balance)
}
