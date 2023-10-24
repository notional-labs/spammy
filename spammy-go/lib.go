package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
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
	}
	var mempoolRes MempoolResult
	err = json.Unmarshal(resp, &mempoolRes)
	if err != nil {
		log.Printf("Failed to unmarshal mempool result: %v", err)
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

func getInitialSequence() int {
	resp, err := httpGet("http://127.0.0.1:1317/cosmos/auth/v1beta1/accounts/cosmos140rptve4cr0mxgknzprl86868nfslydfyem3nq")
	if err != nil {
		log.Printf("Failed to get initial sequence: %v", err)
	}
	var accountRes AccountResult
	err = json.Unmarshal(resp, &accountRes)
	if err != nil {
		log.Printf("Failed to unmarshal account result: %v", err)
	}
	fmt.Println("sequence is", accountRes.Account.Sequence)
	seqint, err := strconv.Atoi(accountRes.Account.Sequence)
	if err != nil {
		log.Printf("Failed to convert to string: %v", err)
	}
	return seqint
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
