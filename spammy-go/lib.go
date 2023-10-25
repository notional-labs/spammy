package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"unsafe"

	"github.com/BurntSushi/toml"
)

func currentBlock(nodeURL string) string {
	resp, err := httpGet(fmt.Sprintf("%s/block", nodeURL))
	if err != nil {
		log.Printf("Failed to get current block: %v", err)
	}
	var blockRes BlockResult
	err = json.Unmarshal(resp, &blockRes)
	if err != nil {
		log.Printf("Failed to unmarshal block result: %v", err)
	}
	return blockRes.Result.Block.Header.Height
}

func mempoolSize(nodeURL string) Result {
	resp, err := httpGet(fmt.Sprintf("%s/num_unconfirmed_txs", nodeURL))
	if err != nil {
		log.Printf("Failed to get mempool size: %v", err)
		return Result{} // Return an empty Result on error
	}

	var mempoolRes MempoolResult
	err = json.Unmarshal(resp, &mempoolRes)
	if err != nil {
		log.Printf("Failed to unmarshal mempool result: %v", err)
		return Result{} // Return an empty Result on error
	}

	return mempoolRes.Result
}

func blockSize(height, nodeURL string) uintptr {
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

	//	seqint, err := strconv.ParseInt(accountRes.Account.Sequence, 10, 64)
	//	if err != nil {
	//		log.Printf("Failed to convert sequence to int: %v", err)
	//		return 0, 0
	//	}

	//	accnum, err := strconv.ParseInt(accountRes.Account.AccountNumber, 10, 64)
	//	if err != nil {
	//		log.Printf("Failed to convert account number to int: %v", err)
	//		return 0, 0
	//	}

	return 0, 0
}

func getChainID(nodeURL string) (string, error) {
	resp, err := httpGet(fmt.Sprintf("%s/status", nodeURL))
	if err != nil {
		log.Printf("Failed to get node status: %v", err)
		return "", err
	}

	var statusRes NodeStatusResponse
	err = json.Unmarshal(resp, &statusRes)
	if err != nil {
		log.Printf("Failed to unmarshal node status result: %v", err)
		return "", err
	}

	return statusRes.Result.NodeInfo.Network, nil
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
