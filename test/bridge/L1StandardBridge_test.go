package bridge_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/Molin-L/FundRebalance/internal/account"
	L1StandardBridge "github.com/Molin-L/FundRebalance/internal/bridge/abi/L1/L1StandardBridge"
	"github.com/Molin-L/FundRebalance/internal/utils"
)

func TestL1StandardBridge(t *testing.T) {
	// Connect to Ethereum client
	accounts, err := account.GetAccount("config.json")
	if err != nil {
		t.Fatalf("Failed to get accounts: %v", err)
	}
	client, err := ethclient.Dial(accounts[0].RpcUrl)
	if err != nil {
		t.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// Create bridge instance
	bridgeAddr := common.HexToAddress("0x21F308067241B2028503c07bd7cB3751FFab0Fb2")
	erc20Bridge, err := L1StandardBridge.NewL1StandardBridge(bridgeAddr, client)
	if err != nil {
		t.Fatalf("Failed to instantiate L1StandardBridge contract: %v", err)
	}

	// Setup private key and auth
	privateKey, err := crypto.HexToECDSA(accounts[0].PrivateKey)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(11155111))
	if err != nil {
		t.Fatalf("Failed to create authorized transactor: %v", err)
	}

	// Test deposit parameters
	l1Token := common.HexToAddress("0x54fa517f05e11ffa87f4b22ae87d91cec0c2d7e1")
	l2Token := common.HexToAddress("0x54fa517f05e11ffa87f4b22ae87d91cec0c2d7e1")
	amount := utils.ConvertEthToWei(0.001) // 1 token
	l2Gas := uint32(200000)
	data := []byte{}

	// Attempt deposit
	tx, err := erc20Bridge.DepositERC20(auth, l1Token, l2Token, amount, l2Gas, data)
	if err != nil {
		t.Fatalf("Failed to deposit ERC20: %v", err)
	}

	// Verify transaction hash exists
	if tx.Hash() == (common.Hash{}) {
		t.Error("Transaction hash is empty")
	}

	// Test event subscription
	depositChan := make(chan *L1StandardBridge.L1StandardBridgeERC20DepositInitiated)
	sub, err := erc20Bridge.WatchERC20DepositInitiated(nil, depositChan, []common.Address{l1Token}, []common.Address{l2Token}, []common.Address{})
	if err != nil {
		t.Fatalf("Failed to create event subscription: %v", err)
	}
	defer sub.Unsubscribe()

	// Wait for event or error
	select {
	case event := <-depositChan:
		// Verify event parameters
		if event.L1Token != l1Token {
			t.Errorf("Expected L1Token %s, got %s", l1Token.Hex(), event.L1Token.Hex())
		}
		if event.L2Token != l2Token {
			t.Errorf("Expected L2Token %s, got %s", l2Token.Hex(), event.L2Token.Hex())
		}
		if event.Amount.Cmp(amount) != 0 {
			t.Errorf("Expected amount %s, got %s", amount.String(), event.Amount.String())
		}
	case err := <-sub.Err():
		t.Fatalf("Event subscription error: %v", err)
	}
}
