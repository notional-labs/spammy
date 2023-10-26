package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	BatchSize = 100
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

	// get initial sequence
	address := acctaddress
	globalSequence, globalAccNum := getInitialSequence(address)

	successfulNodes := loadNodes()
	fmt.Printf("Number of nodes: %d\n", len(successfulNodes))

	// Compile the regex outside the loop
	reMismatch := regexp.MustCompile("account sequence mismatch")
	reExpected := regexp.MustCompile(`expected (\d+)`)

	var wg sync.WaitGroup

	// Use only one node
	nodeURL := successfulNodes[0]
	wg.Add(1)

	go func(nodeURL string) {
		defer wg.Done()

		sequence := globalSequence
		accNum := globalAccNum

		// Create a ticker for rate limiting
		ticker := time.NewTicker(time.Second / BatchSize)
		defer ticker.Stop()

		for {
			currentMempoolSize := mempoolSize(nodeURL)
			fmt.Printf("Node: %s, Mempool size: %d bytes, Number of transactions: %d\n", nodeURL, currentMempoolSize.TotalBytes, currentMempoolSize.Count)

			startBlock := getStatus(nodeURL)
			fmt.Printf("Script starting at block height: %d\n", startBlock)

			for i := 0; i < BatchSize; i++ {
				<-ticker.C // Wait for the ticker

				resp, _, err := sendIBCTransferViaRPC(nodeURL, "composable-testnet-4", uint64(sequence), uint64(accNum), privkey, pubKey, address)
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
							sequence = int64(newSequence)
							fmt.Printf("we had an account sequence mismatch for node %s, adjusting to %d\n", nodeURL, newSequence)
						}
					}
				}
			}

			mu.Lock()
			fmt.Println("successful transactions: ", successfulTxns)
			fmt.Println("failed transactions: ", failedTxns)
			totalTxns := successfulTxns + failedTxns
			fmt.Println("Response code breakdown:")
			for code, count := range responseCodes {
				percentage := float64(count) / float64(totalTxns) * 100
				fmt.Printf("Code %d: %d (%.2f%%)\n", code, count, percentage)
			}
			mu.Unlock()
		}
	}(nodeURL)

	// Monitor mempool size on all nodes
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			var totalMempoolSize int
			for _, nodeURL := range successfulNodes {
				currentMempoolSize := mempoolSize(nodeURL)
				totalMempoolSize += currentMempoolSize.TotalBytes
			}
			averageMempoolSize := totalMempoolSize / len(successfulNodes)
			fmt.Printf("Average global mempool size: %d bytes\n", averageMempoolSize)
		}
	}()

	wg.Wait()
}
