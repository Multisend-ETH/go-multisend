package multisendvy

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"regexp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

type network string

func (n network) ToString() string {
	return string(n)
}

// constants ...
var (
	Networks = struct {
		Ropsten   string
		Homestead string
	}{ropsten.ToString(), homestead.ToString()}
	MultsendAddresses = struct {
		Ropsten   common.Address
		Homestead common.Address
	}{
		Ropsten:   common.HexToAddress("0x19054018704Bf85101eE221937dfc3632b532870"),
		Homestead: common.HexToAddress("0x941F40C2955EE09ba638409F67ef27C531fc055C"),
	}
)

var (
	ropsten   network = "ropsten"
	homestead network = "homestead"

	zeroHash = regexp.MustCompile("^0?x?0+$")

	multiSendEtherABI = []byte("multiSendEther(address[100],uint256[100])")
	multisendTokenABI = []byte("multiSendToken(address,address[100],uint256[100])")

	zeroAddress       = common.HexToAddress("0x0000000000000000000000000000000000000000")
	paddedZeroAddress = common.LeftPadBytes(zeroAddress.Bytes(), 32)
)

// GetTxParams ...
func GetTxParams(client *ethclient.Client, key string) (*ecdsa.PrivateKey, common.Address, uint64, error) {
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		fmt.Println(err)
		return nil, common.Address{}, 0, err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Println("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return nil, common.Address{}, 0, errors.New("Privkey error")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Println("Getting nonce")
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	fmt.Println("Got nonce", nonce)
	if err != nil {
		return nil, common.Address{}, 0, errors.New("get nonce err")
	}
	return privateKey, fromAddress, nonce, nil
}

func getEncodedAddresses(addresses [100]string) []byte {
	paddedAddresses := []byte{}
	for _, addr := range addresses {
		if addr == "" {
			paddedAddresses = append(paddedAddresses, paddedZeroAddress...)
		} else {
			address := common.HexToAddress(addr)
			paddedAddress := common.LeftPadBytes(address.Bytes(), 32)
			paddedAddresses = append(paddedAddresses, paddedAddress...)
		}
	}
	return paddedAddresses
}

func getEncodedAmounts(amounts [100]float64) ([]byte, *big.Int) {
	paddedAmounts := []byte{}
	etherValue := new(big.Int)
	for _, amount := range amounts {
		_value := new(big.Int)
		weiValue := amount * math.Pow10(18)
		_value.SetString(fmt.Sprintf("%.0f", weiValue), 10)
		etherValue.Add(etherValue, _value)
		paddedAmount := common.LeftPadBytes(_value.Bytes(), 32)
		paddedAmounts = append(paddedAmounts, paddedAmount...)
	}
	return paddedAmounts, etherValue
}

func getEncodedWeiAmounts(amounts [100]string) ([]byte, *big.Int) {
	paddedAmounts := []byte{}
	etherValue := new(big.Int)
	for _, weiValue := range amounts {
		_value := new(big.Int)
		_value.SetString(weiValue, 10)
		etherValue.Add(etherValue, _value)
		paddedAmount := common.LeftPadBytes(_value.Bytes(), 32)
		paddedAmounts = append(paddedAmounts, paddedAmount...)
	}
	return paddedAmounts, etherValue
}

func getMethodID(methodABI []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(methodABI)
	methodID := hash.Sum(nil)[:4]
	return methodID
}

// DoPost ...
func DoPost(url string, method string, params interface{}) (*JSONRpcResp, error) {
	jsonReq := map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params, "id": 0}
	data, _ := json.Marshal(jsonReq)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Length", (string)(len(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	r := &http.Client{}
	resp, err := r.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp *JSONRpcResp
	err = json.NewDecoder(resp.Body).Decode(&rpcResp)
	if err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, errors.New(rpcResp.Error["message"].(string))
	}
	return rpcResp, err
}

// RPCSendETHTransactionCallData ...
type RPCSendETHTransactionCallData struct {
	From     string
	To       string
	Value    string
	Data     string
	Gas      string
	GasPrice string
}

// RPCSendETHTransaction ...
func RPCSendETHTransaction(url string, callData *RPCSendETHTransactionCallData) (string, error) {
	params := map[string]string{
		"from":     callData.From,
		"to":       callData.To,
		"value":    callData.Value,
		"data":     callData.Data,
		"gas":      callData.Gas,
		"gasPrice": callData.GasPrice,
	}
	resp, err := DoPost(url, "eth_sendTransaction", []interface{}{params})
	var reply string
	if err != nil {
		return reply, err
	}
	err = json.Unmarshal(*resp.Result, &reply)
	if err != nil {
		return reply, err
	}
	/* There is an inconsistence in a "standard". Geth returns error if it can't unlock signer account,
	 * but Parity returns zero hash 0x000... if it can't send tx, so we must handle this case.
	 * https://github.com/ethereum/wiki/wiki/JSON-RPC#returns-22
	 */
	if zeroHash.MatchString(reply) {
		err = errors.New("transaction is not yet available")
	}
	return reply, err
}

// JSONRpcResp ...
type JSONRpcResp struct {
	ID     *json.RawMessage       `json:"id"`
	Result *json.RawMessage       `json:"result"`
	Error  map[string]interface{} `json:"error"`
}
