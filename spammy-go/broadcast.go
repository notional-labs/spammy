package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	cometrpc "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/testing/simapp"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

var client = &http.Client{
	Timeout: 10 * time.Second, // Adjusted timeout to 10 seconds
	Transport: &http.Transport{
		MaxIdleConns:        100,              // Increased maximum idle connections
		MaxIdleConnsPerHost: 10,               // Increased maximum idle connections per host
		IdleConnTimeout:     90 * time.Second, // Increased idle connection timeout
		TLSHandshakeTimeout: 10 * time.Second, // Increased TLS handshake timeout
	},
}

var (
	cdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

func init() {
	types.RegisterInterfaces(cdc.InterfaceRegistry())
}

func sendIBCTransferViaRPC(status *coretypes.ResultStatus, successfulNodes []string, rpcEndpoint string, sequence uint64, privKey cryptotypes.PrivKey, pubKey cryptotypes.PubKey, address string) (response []*coretypes.ResultBroadcastTx, txbody string, err error) {
	encodingConfig := simapp.MakeTestEncodingConfig()
	encodingConfig.Marshaler = cdc

	// Register IBC and other necessary types
	transferModule := transfer.AppModuleBasic{}
	ibcModule := ibc.AppModuleBasic{}

	ibcModule.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	transferModule.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	// Create a new TxBuilder.
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	cmtCli, err := cometrpc.New(rpcEndpoint, "/websocket")
	if err != nil {
		log.Fatal(err) //nolint:gocritic
	}
	ctx := context.Background()
	netinfo, err := cmtCli.NetInfo(ctx)

	// moniker, height, number of peers
	nodeinfo := fmt.Sprint("Moniker: ", status.NodeInfo.Moniker, " Height: ", status.SyncInfo.LatestBlockHeight, " indexing? ", status.TxIndexEnabled(), " Peers: ", netinfo.NPeers)

	receiver, numBytes, err := generateRandomString()
	token := sdk.NewCoin("adv4tnt", sdk.NewInt(1))
	msg := types.NewMsgTransfer(
		"transfer",
		"channel-0",
		token,
		address,
		receiver,
		clienttypes.NewHeight(1, 3812921),
		uint64(0),
		nodeinfo,
	)

	// set messages
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, "", err
	}

	gas := uint64(950000 + numBytes*17)
	//	feeWithGas := int64(float64(gas) * 0.00269)
	feecoin := sdk.NewCoin("adv4tnt", sdk.NewInt(39052500000000000))
	fee := sdk.NewCoins(feecoin)

	txBuilder.SetGasLimit(gas)
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetMemo(nodeinfo)
	txBuilder.SetTimeoutHeight(0)

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	sigV2 := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  encodingConfig.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: sequence,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		fmt.Println("error setting signatures")
		return nil, "", err
	}

	signerData := authsigning.SignerData{
		ChainID:       "dydx-testnet-4",
		AccountNumber: 4130, // set actual account number
		Sequence:      4575, // set actual sequence number
	}

	signed, err := tx.SignWithPrivKey(
		encodingConfig.TxConfig.SignModeHandler().DefaultMode(), signerData,
		txBuilder, privKey, encodingConfig.TxConfig, sequence)
	if err != nil {
		fmt.Println("coulnd't sign")
		return nil, "", err
	}

	if err != nil {
		return nil, "", err
	}

	err = txBuilder.SetSignatures(signed)
	if err != nil {
		return nil, "", err
	}

	// Generate a JSON string.
	txJSONBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}

	resp, err := BroadcastTransaction(txJSONBytes, successfulNodes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return resp, string(txJSONBytes), nil
}

func BroadcastTransaction(txBytes []byte, successfulNodes []string) ([]*coretypes.ResultBroadcastTx, error) {
	var results []*coretypes.ResultBroadcastTx

	for _, rpcEndpoint := range successfulNodes {
		cmtCli, err := cometrpc.New(rpcEndpoint, "/websocket")
		if err != nil {
			log.Fatal(err) //nolint:gocritic
		}

		t := tmtypes.Tx(txBytes)

		ctx := context.Background()
		res, err := cmtCli.BroadcastTxSync(ctx, t)
		if err != nil {
			return nil, err
		}

		//	fmt.Println("log: ", res.Log)
		//	fmt.Println("code: ", res.Code)

		results = append(results, res)
	}

	return results, nil
}

func generateRandomString() (string, int, error) {
	rand.Seed(time.Now().UnixNano())
	//	sizeB := rand.Intn(600000-100000+1) + 100 // Generate random size between 100 and 175000 bytes

	// Calculate the number of bytes to generate (2 characters per byte in hex encoding)
	nBytes := 10000 / 2

	randomBytes := make([]byte, nBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(randomBytes), nBytes, nil
}
