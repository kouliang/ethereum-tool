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

func SuggestGasPrice() (*big.Int, error) {
	return client.SuggestGasPrice(context.Background())
}

// callData, err := nftpool.Pack("dividingTime")
// resultData, err := client.Call(nftPoolAddress, callData)
// result, err := nftpool.Unpack("dividingTime", resultData)
func Call(to string, abi abi.ABI, name string, args ...interface{}) ([]interface{}, error) {
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

func SendTransaction(to string, amount *big.Int, gasPrice *big.Int, callData []byte, log logger) error {
	contractAddress := common.HexToAddress(to)
	log.Println("contract address:", contractAddress.Hex())

	nonce, err := client.PendingNonceAt(context.Background(), FromAddress)
	if err != nil {
		return err
	}

	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: FromAddress,
		To:   &contractAddress,
		Data: callData,
	})
	if err != nil {
		return err
	}

	tx := types.NewTransaction(nonce, contractAddress, amount, gasLimit, gasPrice, callData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ChainID), privateKey)
	if err != nil {
		return err
	}
	log.Printf("nonce:%d gasPrice:%v gasLimit:%d\n", nonce, gasPrice, gasLimit)

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}
	log.Println("tx broadcast:", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("receipted - status:%d, blockNumber:%s\n", receipt.Status, receipt.BlockNumber.String())
	}

	return nil
}
