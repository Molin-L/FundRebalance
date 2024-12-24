package account

import (
	"context"
	"math/big"

	"github.com/Molin-L/FundRebalance/pkg/golib/pkg/log"
	"github.com/Molin-L/FundRebalance/pkg/idl/pb/blockchains"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

type InfraAccount struct {
}

func (a *InfraAccount) GetBalance(address Address, coin blockchains.Coin) (float64, error) {
	if coin == blockchains.Coin_ETH {
		return a._get_eth_balance(address)
	} else if coin == blockchains.Coin_MANT {
		return a._get_mant_balance(address)
	}

	return 0, nil
}

func (a *InfraAccount) _get_eth_balance(address Address) (float64, error) {
	ctx := context.Background()
	client, err := ethclient.Dial(address.RpcUrl)
	if err != nil {
		log.Error(ctx, "failed to dial to rpc url", zap.Error(err))
		return 0, err
	}

	infraAddress := common.HexToAddress(address.Address)

	balance, err := client.BalanceAt(ctx, infraAddress, nil)
	if err != nil {
		log.Error(ctx, "failed to get balance", zap.Error(err))
		return 0, err
	}
	etherValue := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))

	value, _ := etherValue.Float64()
	log.Info(ctx, "balance", zap.Float64("balance", value))

	return value, nil
}

func (a *InfraAccount) _get_mant_balance(address Address) (float64, error) {
	return 0, nil
}
