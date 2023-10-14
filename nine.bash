#!/bin/bash

NODE_URL="http://127.0.0.1:6000"
BATCH_SIZE=30
ACCOUNT="72559"
CHANNEL="channel-1"
KEY_NAME="test"
UDENOM="utia"
APPNAME="celestia-appd"
CHAINID="mocha-4"
IBCMEMO=50000  # 50kb in bytes
IBCTIMEOUTS="--packet-timeout-timestamp 0 --packet-timeout-height 0-100000"
FEES="391405"
GAS="3114054"
ADDRESS="celestia1x7jn3tafxdhk844vgle5ga4plyqqxk39z4zsnk"

# Calculate bytes for RECEIVEADDR
RECIEVEADDR=$(($IBCMEMO * 2))

# Get the current block number
current_block() {
  curl -s $NODE_URL/block | jq -r .result.block.header.height
}

# Get the size of the mempool
mempool_size() {
  curl -s $NODE_URL/unconfirmed_txs?limit=1 | jq -r .result.n_txs
}

# Get the size of the latest block
block_size() {
  curl -s $NODE_URL/block?height=$1 | jq -r .result.block.data.txs | jq length
}

start_block=$(current_block)
echo "Script starting at block height: $start_block"

# Query sequence number initially
SEQUENCE=$(curl http://127.0.0.1:5003/cosmos/auth/v1beta1/accounts/$ADDRESS | jq --raw-output ' .account.sequence ')



# Main loop
while true; do
  last_block=$(current_block)
  last_block_size=$(block_size $last_block)
  current_mempool_size=$(mempool_size)

  echo "Last block height: $last_block, size: $last_block_size transactions"
  echo "Current mempool size: $current_mempool_size transactions"

  # Send transactions in a batch
  for ((i=0; i<$BATCH_SIZE; i++)); do
    # Transaction body generation
    $APPNAME tx ibc-transfer transfer transfer $CHANNEL $ADDRESS 1$UDENOM --account-number $ACCOUNT --keyring-backend test --memo $(openssl rand -hex $IBCMEMO) --chain-id $CHAINID --yes $IBCTIMEOUTS --generate-only --fees $FEES$UDENOM --gas $GAS --from $ADDRESS &> bareibctx.json
    echo "transaction body generated with $((IBCMEMO*2)) byte ibc memo field"

    openssl rand -hex $RECIEVEADDR > tmp.txt
    jq --rawfile random_str tmp.txt '.body.messages[0].receiver = $random_str' bareibctx.json > autobanana.json
    rm tmp.txt
    echo "$(($RECIEVEADDR*2)) byte random string inserted"

    # Sign the transaction
    $APPNAME tx sign autobanana.json --account-number $ACCOUNT --from $ADDRESS --yes --sequence $SEQUENCE --chain-id $CHAINID --keyring-backend test --offline &> ban.json
    echo "transaction signed"

    # Broadcast the signed transaction
    $APPNAME tx broadcast ban.json --node $NODE_URL --chain-id $CHAINID &> broadcast.log
    echo "transaction broadcasted"

    # If there's an account sequence mismatch, parse the expected value and use it
if cat broadcast.log | grep -q "account sequence mismatch"; then
  SEQUENCE=$(cat broadcast.log | grep -oP 'expected \K\d+')
else
  ((SEQUENCE++))
fi

  done

  # Check for a new block before looping again
  while [[ $(current_block) -le $last_block ]]; do
    continue
  done
done

