package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
)

func sendIBCTransferViaRPC(senderKeyName, rpcEndpoint string, sequence uint64, kr keyring.Keyring) (broadcastlog string, err error) {
	encodingConfig := simapp.MakeTestEncodingConfig()

	// Create a new TxBuilder.
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	info, err := kr.Key(senderKeyName)
	if err != nil {
		return "", err
	}

	address := info.GetAddress()
	receiver, err := generateRandomString(30)
	if err != nil {
		return "", err
	}
	token := sdk.NewCoin("uatom", sdk.NewInt(1))
	msg := types.NewMsgTransfer(
		"transfer",
		"channel-51",
		token,
		address.String(),
		receiver,
		clienttypes.NewHeight(0, 10000),
		types.DefaultRelativePacketTimeoutTimestamp,
	)

	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return "", err
	}

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2
	sigV2 := signing.SignatureV2{
		PubKey: info.GetPubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  encodingConfig.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: sequence,
	}

	sigsV2 = append(sigsV2, sigV2)
	err = txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return "", err
	}

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}

	// Now, marshal the signDoc to bytes so it can be signed.
	signBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", err
	}

	// Now, sign the signBytes.
	signed, _, err := kr.Sign(info.GetName(), signBytes)
	if err != nil {
		return "", err
	}

	// If you need to use the pubkey, you can retrieve it from the info struct.
	pubkey := info.GetPubKey()

	// Now, create a new SignatureV2 struct with the signed bytes and public key.
	sigV2 = signing.SignatureV2{
		PubKey: pubkey,
		Data: &signing.SingleSignatureData{
			SignMode:  encodingConfig.TxConfig.SignModeHandler().DefaultMode(),
			Signature: signed,
		},
		Sequence: sequence, // Adjust the sequence number as necessary.
	}

	fee := sdk.NewCoin("uatom", sdk.NewInt(5000))
	fees := sdk.NewCoins(fee)

	txBuilder.SetGasLimit(300000)
	txBuilder.SetFeeAmount(fees)
	txBuilder.SetMemo("testing 1 2 3")

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return "", err
	}

	// Encode the signed transaction to broadcast it
	bz, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", err
	}

	broadcastReq := map[string]interface{}{
		"tx": hex.EncodeToString(bz),
	}
	reqBytes, err := json.Marshal(broadcastReq)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(rpcEndpoint+"/broadcast_tx_sync", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("Response:", string(body))
	return broadcastlog, nil
}

func generateRandomString(sizeKB int) (string, error) {
	// Calculate the number of bytes to generate (2 characters per byte in hex encoding)
	nBytes := sizeKB * 1024 / 2

	randomBytes := make([]byte, nBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}
