package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"unsafe"

	"github.com/BurntSushi/toml"
	cometrpc "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

func getStatus(nodeURL string) *coretypes.ResultStatus {
	cmtCli, err := cometrpc.New(nodeURL, "/websocket")
	if err != nil {
		log.Fatal(err) //nolint:gocritic
	}

	ctx := context.Background()

	status, err := cmtCli.Status(ctx)

	return status
}

func mempoolSize(nodeURL string) *coretypes.ResultUnconfirmedTxs {

	cmtCli, err := cometrpc.New(nodeURL, "/websocket")
	if err != nil {
		log.Fatal(err) //nolint:gocritic
	}
	ctx := context.Background()
	unconfirmed, err := cmtCli.NumUnconfirmedTxs(ctx)
	if err != nil {
		panic(err)
	}

	return unconfirmed
}

func blockSize(height int64, nodeURL string) uintptr {
	resp, err := httpGet(fmt.Sprintf("%s/block?height=%s", nodeURL, height))
	if err != nil {
		log.Printf("Failed to get block size: %v", err)
	}
	var blockRes BlockResult
	err = json.Unmarshal(resp, &blockRes)
	if err != nil {
		log.Printf("Failed to unmarshal block result: %v", err)
	}
	size := unsafe.Sizeof(blockRes)

	return size
}

func getInitialSequence(address string) (int64, int64) {
	resp, err := httpGet("https://rest.sentry-01.theta-testnet.polypore.xyz/cosmos/auth/v1beta1/accounts/" + address)
	if err != nil {
		log.Printf("Failed to get initial sequence: %v", err)
		return 0, 0
	}

	var accountRes AccountResult
	err = json.Unmarshal(resp, &accountRes)
	if err != nil {
		log.Printf("Failed to unmarshal account result: %v", err)
		return 0, 0
	}

	seqint, err := strconv.ParseInt(accountRes.Account.Sequence, 10, 64)
	if err != nil {
		log.Printf("Failed to convert sequence to int: %v", err)
		return 0, 0
	}

	accnum, err := strconv.ParseInt(accountRes.Account.AccountNumber, 10, 64)
	if err != nil {
		log.Printf("Failed to convert account number to int: %v", err)
		return 0, 0
	}

	return seqint, accnum
}

func httpGet(url string) ([]byte, error) {
	resp, err := client.Get(url) //nolint:gosec // this is what it thinks it is
	if err != nil {
		netErr, ok := err.(net.Error)
		if ok && netErr.Timeout() {
			log.Printf("Request to %s timed out, continuing...", url)
			return nil, nil
		}
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// This function will load our nodes from nodes.toml
func loadNodes() []string {
	var config Config
	if _, err := toml.DecodeFile("nodes.toml", &config); err != nil {
		log.Fatalf("Failed to load nodes.toml: %v", err)
	}
	return config.SuccessfulNodes
}
