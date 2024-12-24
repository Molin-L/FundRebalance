package utils

import "math/big"

func ConvertEthToWei(eth float64) *big.Int {
	return big.NewInt(int64(eth * 1e18))
}

func ConvertWeiToEth(wei *big.Int) float64 {
	return float64(wei.Int64()) / 1e18
}

func ConvertMantleToWei(mantle float64) *big.Int {
	return big.NewInt(int64(mantle * 1e18))
}

func ConvertWeiToMantle(wei *big.Int) float64 {
	return float64(wei.Int64()) / 1e18
}
