package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/cosmos/ibc-go/v4/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v4/testing/simapp"
)

var client = &http.Client{
	Timeout: time.Millisecond * 500,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 1,
		IdleConnTimeout:     500 * time.Millisecond,
		TLSHandshakeTimeout: 500 * time.Millisecond,
	},
}

func sendIBCTransferViaRPC(senderKeyName, rpcEndpoint string, sequence uint64) (response *BroadcastResponse, txbody string, err error) {
	encodingConfig := simapp.MakeTestEncodingConfig()

	// Register IBC and other necessary types
	transferModule := transfer.AppModuleBasic{}
	transferModule.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	// Create a new TxBuilder.
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	// Create a new keyring to access keys
	kr, err := keyring.New("cosmos", keyring.BackendTest, "/root/.gaia-rs", nil)
	if err != nil {
		return nil, "", err
	}

	info, err := kr.Key(senderKeyName)
	if err != nil {
		return nil, "", err
	}

	address := info.GetAddress()
	receiver, _ := generateRandomString()
	token := sdk.NewCoin("uatom", sdk.NewInt(1))
	msg := types.NewMsgTransfer(
		"transfer",
		"channel-58",
		token,
		address.String(),
		receiver,
		clienttypes.NewHeight(0, 10000),
		types.DefaultRelativePacketTimeoutTimestamp,
	)

	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, "", err
	}
	fmt.Println("messages set")

	if len(txBuilder.GetTx().GetMsgs()) == 0 {
		return nil, "", fmt.Errorf("transaction has no messages set")
	}

	sigBytes, _, err := kr.SignByAddress(address, msg.GetSignBytes())
	if err != nil {
		fmt.Println("coulnd't sign")
		return nil, "", err
	}
	//	fmt.Println("signed")

	sig := signing.SignatureV2{
		PubKey:   info.GetPubKey(),
		Data:     &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT, Signature: sigBytes},
		Sequence: sequence,
	}

	err = txBuilder.SetSignatures(sig)
	if err != nil {
		fmt.Println("cannot set signatures")
		panic(err)
	}
	//	fmt.Println("set signatures")

	// Generate a JSON string.
	txJSONBytes, err := encodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	if err != nil {
		fmt.Println("some issue with the string")
		fmt.Println(err)
		return nil, "", err
	}
	//	fmt.Println(string(txJSONBytes))

	resp, err := BroadcastTransaction(txJSONBytes, rpcEndpoint)
	if err != nil {
		// handle error, for example:
		return nil, "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return resp, string(txJSONBytes), nil
}

func BroadcastTransaction(txBytes []byte, rpcEndpoint string) (*BroadcastResponse, error) {
	encodedTx := hex.EncodeToString(txBytes)

	broadcastReq := BroadcastRequest{
		Jsonrpc: "2.0",
		ID:      "",
		Method:  "broadcast_tx_sync",
		BroadcastRequestParams: BroadcastRequestParams{
			Tx: encodedTx,
		},
	}
	fmt.Println("sigsv2 assigned")

	reqBytes, err := json.Marshal(broadcastReq)
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(rpcEndpoint, "application/json", bytes.NewBuffer(reqBytes)) //nolint:gosec // not worth fixing
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var broadcastResp BroadcastResponse
	err = json.Unmarshal(body, &broadcastResp)
	if err != nil {
		return nil, err
	}

	if broadcastResp.BroadcastResult.Code != 0 {
		return nil, fmt.Errorf("transaction failed with code: %d", broadcastResp.BroadcastResult.Code)
	}

	return &broadcastResp, nil
}

func generateRandomString() (string, error) {
	rand.Seed(time.Now().UnixNano())
	sizeB := rand.Intn(175000-100+1) + 100 // Generate random size between 100 and 175000 bytes

	// Calculate the number of bytes to generate (2 characters per byte in hex encoding)
	nBytes := sizeB / 2

	randomBytes := make([]byte, nBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}
