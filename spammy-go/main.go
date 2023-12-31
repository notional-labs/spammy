package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	BatchSize  = 100
	MaxWorkers = 1000
)

func main() {
	// tracking vars
	var successfulTxns int
	var failedTxns int
	var mu sync.Mutex
	// Declare a map to hold response codes and their counts
	responseCodes := make(map[uint32]int)

	// keyring
	// read seed phrase
	mnemonic, _ := os.ReadFile("seedphrase")
	privkey, pubKey, acctaddress := getPrivKey(mnemonic)
	// Create an in-mempory keyring

	// get initial sequence
	address := acctaddress
	sequence, accNum := getInitialSequence(address)

	// get correct chain-id
	chainID, err := getChainID("https://rest.sentry-01.theta-testnet.polypore.xyz/")
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	successfulNodes := loadNodes()
	fmt.Printf("Number of nodes: %d\n", len(successfulNodes))

	var wg sync.WaitGroup

	// Compile the regex outside the loop
	reMismatch := regexp.MustCompile("account sequence mismatch")
	reExpected := regexp.MustCompile(`expected (\d+)`)

	for _, nodeURL := range successfulNodes {
		wg.Add(1)
		go func(nodeURL string) {
			defer wg.Done()

			currentMempoolSize := mempoolSize(nodeURL)
			fmt.Printf("Node: %s, Mempool size: %s bytes, Number of transactions: %s\n", nodeURL, currentMempoolSize.TotalBytes, currentMempoolSize.NTxs)

			startBlock := currentBlock(nodeURL)
			fmt.Printf("Script starting at block height: %s\n", startBlock)

			for {
				lastBlock := startBlock
				lastBlockSize := blockSize(lastBlock, nodeURL)
				currentMempoolSize := mempoolSize(nodeURL)

				fmt.Println("Last block height: ", lastBlock)
				fmt.Println("Last Block Size: ", lastBlockSize)
				fmt.Println(nodeURL, "Current mempool txns: "+currentMempoolSize.NTxs+" transactions")
				fmt.Println(nodeURL, "mempool byte size:", currentMempoolSize.TotalBytes)

				var wgBatch sync.WaitGroup
				wgBatch.Add(BatchSize)

				for i := 0; i < BatchSize; i++ {
					go func() {
						defer wgBatch.Done()
						currentSequence := sequence// Use atomic to ensure thread-safety

						resp, _, err := sendIBCTransferViaRPC(nodeURL, chainID, uint64(currentSequence), uint64(accNum), privkey, pubKey, address)
						if err != nil {
							mu.Lock()
							failedTxns++
							mu.Unlock()
						} else {
							mu.Lock()
							successfulTxns++
							mu.Unlock()
							if resp != nil {
								// Increment the count for this response code
								mu.Lock()
								responseCodes[resp.Code]++
								mu.Unlock()
							}

							match := reMismatch.MatchString(resp.Log)
							if match {
								matches := reExpected.FindStringSubmatch(resp.Log)
								if len(matches) > 1 {
									newSequence, err := strconv.Atoi(matches[1])
									if err != nil {
										log.Fatalf("Failed to convert sequence to integer: %v", err)
									}
									// Safely update the global sequence if needed
									atomic.SwapInt64(&sequence, int64(newSequence))
									fmt.Printf("we had an account sequence mismatch, adjusting to %d\n", newSequence)
								}
							}
						}
					}() // Pass the currentSequence variable to the goroutine
				}

				wgBatch.Wait()
				fmt.Println("successful transactions: ", successfulTxns)
				fmt.Println("failed transactions: ", failedTxns)
				totalTxns := successfulTxns + failedTxns
				fmt.Println("Response code breakdown:")
				for code, count := range responseCodes {
					percentage := float64(count) / float64(totalTxns) * 100
					fmt.Printf("Code %d: %d (%.2f%%)\n", code, count, percentage)
				}

				for {
					if currentBlock(nodeURL) > lastBlock {
						break
					}
				}
			}
		}(nodeURL)
	}

	wg.Wait()
}
