package multisendvy

import (
	"context"

	"github.com/ethereum/go-ethereum"
)

// MultsendEtherData send ether to multiple addresses
func MultsendEtherData(ctx context.Context, addresses [100]string, amounts [100]float64) *ethereum.CallMsg {
	methodID := getMethodID(multiSendEtherABI)

	byteAmount, etherValue := getEncodedAmounts(amounts)
	byteAddr := getEncodedAddresses(addresses)

	data := ethereum.CallMsg{
		Data:  []byte(``),
		Value: etherValue,
	}

	data.Data = append(data.Data, methodID...)
	data.Data = append(data.Data, byteAddr...)
	data.Data = append(data.Data, byteAmount...)
	return &data
}
