package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/Multisend-ETH/go-multisend/multisendvy"
)

const (
	url = "http://127.0.0.1:8545"
)

func main() {

	ctx := context.Background()

	callData := multisendvy.MultsendEtherData(ctx, [100]string{"0x00B6845c6F47C770cE630B96df9BD4A6dA91C65d"}, [100]float64{0.32})

	fmt.Println(hexutil.Encode(callData.Data))
	params := &multisendvy.RPCSendETHTransactionCallData{
		From:     "0x19bDc405cb5C673e30D56F8d9CEfB4b2009E36D6",
		To:       "0x2267Df87E5A2e3e6B1065c5549cDf1D78B516337",
		Value:    callData.Value.String(),
		Data:     hexutil.Encode(callData.Data),
		Gas:      "3000000",
		GasPrice: "2000000000",
	}
	r, err := multisendvy.RPCSendETHTransaction(url, params)
	if err != nil {
		fmt.Println("err", err)
	}
	fmt.Println(r)

}
