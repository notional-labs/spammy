package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

const (
	// TODO: replace with appropriate node URL
	NODE_URL = "http://127.0.0.1:26657"
	// TODO: replace with appropriate mnemonic
	mnemonic   = "..."
	BATCH_SIZE = 50
)

func main() {
	startBlock := currentBlock()
	fmt.Printf("Script starting at block height: %s\n", startBlock)

	// Create a new keyring for managing keys.
	kr, err := keyring.New("my-keyring", keyring.BackendTest, "", nil) // replace "" with your desired keyring directory
	if err != nil {
		log.Fatalf("Failed to create keyring: %v", err)
	}

	// Derive a new account from the mnemonic.
	info, err := kr.NewAccount("my-account", mnemonic, "", hd.CreateHDPath(118, 0, 0).String(), hd.Secp256k1)
	if err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}

	address := info.GetAddress().String()

	sequence := getInitialSequence(address)

	for {
		lastBlock := currentBlock()
		lastBlockSize := blockSize(lastBlock)
		currentMempoolSize := mempoolSize()

		fmt.Printf("Last block height: %s, size: %d transactions\n", lastBlock, len(lastBlockSize))
		fmt.Printf("Current mempool size: %s transactions\n", currentMempoolSize)
		
		// Convert sequence string to uint64
		seqNum, err := strconv.ParseUint(sequence, 10, 64)
		if err != nil {
			log.Fatalf("Failed to convert sequence to uint64: %v", err)
		}

		for i := 0; i < BATCH_SIZE; i++ {

			// Call sendIBCTransferViaRPC with appropriate arguments
			broadcastLog, err := sendIBCTransferViaRPC(info.GetAddress(), NODE_URL, seqNum, kr) // Assuming "test" is your senderKeyName
			if err != nil {
				log.Fatalf("Failed to broadcast transaction: %v", err)
			}

			fmt.Println(string(sequence))

			if strings.Contains(string(broadcastLog), "code: 20") {
				fmt.Println("\033[31mMEMPOOL FULL!!!!!!!!!\033[0m")
				time.Sleep(60 * time.Second)
				break
			}

			match, _ := regexp.MatchString("account sequence mismatch", string(broadcastLog))
			if match {
				re := regexp.MustCompile(`expected (\d+)`)
				matches := re.FindStringSubmatch(string(broadcastLog))
				if len(matches) > 1 {
					sequence = matches[1]
					fmt.Printf("we had an account sequence mismatch, adjusting to %s\n", sequence)
				}
			} else {
				seqNum, err := strconv.Atoi(sequence)
				if err != nil {
					log.Fatalf("Failed to convert sequence to integer: %v", err)
				}
				sequence = strconv.Itoa(seqNum + 1)
			}
		}

		for {
			if currentBlock() > lastBlock {
				break
			}
		}
	}
}

func currentBlock() string {
	resp, err := httpGet(fmt.Sprintf("%s/block", NODE_URL))
	if err != nil {
		log.Fatalf("Failed to get current block: %v", err)
	}
	var blockRes BlockResult
	json.Unmarshal(resp, &blockRes)
	return blockRes.Result.Block.Header.Height
}

func mempoolSize() string {
	resp, err := httpGet(fmt.Sprintf("%s/numunconfirmed_txs", NODE_URL))
	if err != nil {
		log.Fatalf("Failed to get mempool size: %v", err)
	}
	var mempoolRes MempoolResult
	json.Unmarshal(resp, &mempoolRes)
	return mempoolRes.Result.NTxs
}

func blockSize(height string) []string {
	resp, err := httpGet(fmt.Sprintf("%s/block?height=%s", NODE_URL, height))
	if err != nil {
		log.Fatalf("Failed to get block size: %v", err)
	}
	var blockRes BlockResult
	json.Unmarshal(resp, &blockRes)
	return blockRes.Result.Block.Data.Txs
}

func getInitialSequence(address string) string {
	theUrl := fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", NODE_URL, address)
	resp, err := httpGet(theUrl)
	if err != nil {
		log.Fatalf("Failed to get initial sequence: %v", err)
	}
	var accountRes AccountResult
	json.Unmarshal(resp, &accountRes)
	return accountRes.Account.Sequence
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
