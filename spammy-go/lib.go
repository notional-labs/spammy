package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

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

// This function will load our nodes from nodes.toml
func loadNodes() []string {
	var config Config
	if _, err := toml.DecodeFile("nodes.toml", &config); err != nil {
		log.Fatalf("Failed to load nodes.toml: %v", err)
	}
	return config.SuccessfulNodes
}

func monitorMempool(SuccessfulNodes []string, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var totalMempoolSize int64
		for _, nodeURL := range SuccessfulNodes {
			currentMempoolSize := mempoolSize(nodeURL)
			totalMempoolSize += currentMempoolSize.TotalBytes
		}
		averageMempoolSize := totalMempoolSize / int64(len(SuccessfulNodes))
		fmt.Printf("Average global mempool size: %d bytes\n", averageMempoolSize)
	}
}
