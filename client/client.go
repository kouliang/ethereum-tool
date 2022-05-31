package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var Client *ethclient.Client
var ChainID *big.Int

var PrivateKey *ecdsa.PrivateKey
var FromAddress common.Address

func InitClient(rpc string, privateStr string) (err error) {
	Client, err = ethclient.Dial(rpc)
	if err != nil {
		return
	}
	ChainID, err = Client.NetworkID(context.Background())
	if err != nil {
		return
	}

	PrivateKey, err = crypto.HexToECDSA(privateStr)
	if err != nil {
		return
	}
	publicKey := PrivateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		err = fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return
	}
	FromAddress = crypto.PubkeyToAddress(*publicKeyECDSA)
	return
}

func SendTransaction(addressStr string, callData []byte, amount *big.Int, defaultGasPrice int64) (string, error) {
	var logInfo string

	contractAddress := common.HexToAddress(addressStr)
	logInfo = logInfo + fmt.Sprintf("\ncontract address:%s", contractAddress.Hex())

	nonce, err := Client.PendingNonceAt(context.Background(), FromAddress)
	if err != nil {
		return logInfo, err
	}

	gasPrice, err := Client.SuggestGasPrice(context.Background())
	if err != nil {
		gasPrice = big.NewInt(defaultGasPrice)
		logInfo = logInfo + fmt.Sprintf("\ngsaPrice error:%s", err.Error())
	}

	gasLimit, err := Client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: FromAddress,
		To:   &contractAddress,
		Data: callData,
	})
	if err != nil {
		return logInfo, err
	}

	tx := types.NewTransaction(nonce, contractAddress, amount, gasLimit, gasPrice, callData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ChainID), PrivateKey)
	if err != nil {
		return logInfo, err
	}
	logInfo = logInfo + fmt.Sprintf("\nnonce:%d gasPrice:%v gasLimit:%d", nonce, gasPrice, gasLimit)

	err = Client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return logInfo, err
	}
	logInfo = logInfo + fmt.Sprintf("\ntx broadcast:%s", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), Client, signedTx)
	if err != nil {
		return logInfo, err
	}
	logInfo = logInfo + fmt.Sprintf("\nreceipted - status:%d, blockNumber:%s", receipt.Status, receipt.BlockNumber.String())

	return logInfo, nil
}
