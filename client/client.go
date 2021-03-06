package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var client *ethclient.Client
var privateKey *ecdsa.PrivateKey

var ChainID *big.Int
var FromAddress common.Address

func InitClient(rpc string, privateStr string) (err error) {
	client, err = ethclient.Dial(rpc)
	if err != nil {
		return
	}
	ChainID, err = client.NetworkID(context.Background())
	if err != nil {
		return
	}

	privateKey, err = crypto.HexToECDSA(privateStr)
	if err != nil {
		return
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		err = fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return
	}
	FromAddress = crypto.PubkeyToAddress(*publicKeyECDSA)
	return
}

func Nonce() (uint64, error) {
	return client.PendingNonceAt(context.Background(), FromAddress)
}

func SuggestGasPrice() (*big.Int, error) {
	return client.SuggestGasPrice(context.Background())
}

func EstimateGas(to string, callData []byte) (uint64, error) {
	contractAddress := common.HexToAddress(to)
	return client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: FromAddress,
		To:   &contractAddress,
		Data: callData,
	})
}

// callData, err := nftpool.Pack("dividingTime")
// resultData, err := client.Call(nftPoolAddress, callData)
// result, err := nftpool.Unpack("dividingTime", resultData)
func Call(to string, abi *abi.ABI, name string, args ...interface{}) ([]interface{}, error) {
	contractAddress := common.HexToAddress(to)

	callData, err := abi.Pack(name, args...)
	if err != nil {
		return nil, err
	}

	resultData, err := client.CallContract(context.Background(), ethereum.CallMsg{
		From: FromAddress,
		To:   &contractAddress,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, err
	}

	return abi.Unpack(name, resultData)
}

func SendTransactionTo(to string, callData []byte) (string, error) {

	contractAddress := common.HexToAddress(to)
	record := "contract address:" + contractAddress.Hex()

	nonce, err := client.PendingNonceAt(context.Background(), FromAddress)
	if err != nil {
		return record, fmt.Errorf("get nonce error. %s", err.Error())
	}

	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: FromAddress,
		To:   &contractAddress,
		Data: callData,
	})
	if err != nil {
		return record, fmt.Errorf("get gaslimit error. %s", err.Error())
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return record, fmt.Errorf("get gasPrice error. %s", err.Error())
	}

	tx := types.NewTransaction(nonce, contractAddress, big.NewInt(0), gasLimit, gasPrice, callData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ChainID), privateKey)
	if err != nil {
		return record, fmt.Errorf("sign tx error. %s", err.Error())
	}
	record = fmt.Sprintf("%s\nnonce:%d gasPrice:%v gasLimit:%d", record, nonce, gasPrice, gasLimit)

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return record, fmt.Errorf("send transaction error. %s", err.Error())
	}
	record = fmt.Sprintf("%s\ntx broadcast:%s", record, signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		record = fmt.Sprintf("%s\nwait mined error:%s", record, err.Error())
	} else {
		record = fmt.Sprintf("%s\nreceipted - status:%d, blockNumber:%s", record, receipt.Status, receipt.BlockNumber.String())
	}

	return record, nil
}

func SendTransaction(tx *types.Transaction, log logger) error {

	log.Println("contract address:", tx.To().Hex())
	log.Printf("nonce:%d gasPrice:%v gasLimit:%d\n", tx.Nonce(), tx.GasPrice(), tx.Gas())

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ChainID), privateKey)
	if err != nil {
		return fmt.Errorf("sign tx error. %s", err.Error())
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("send transaction error. %s", err.Error())
	}
	log.Println("tx broadcast:", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		log.Println("wait mined error.", err)
	} else {
		log.Printf("receipted - status:%d, blockNumber:%s\n", receipt.Status, receipt.BlockNumber.String())
	}

	return nil
}
