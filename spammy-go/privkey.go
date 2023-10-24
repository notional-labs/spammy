package main

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	bip39 "github.com/tyler-smith/go-bip39"
)

func getPrivKey(mnemonic []byte) (secp256k1.PrivKey, secp256k1.PubKey, string) {

	// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
	seed := bip39.NewSeed(string(mnemonic), "")

	// Create master private key from seed
	masterPriv, chainCode := hd.ComputeMastersFromSeed(seed)

	// Path for Celestia
	// This is just an example path. Adjust based on the actual requirements.
	path := hd.CreateHDPath(118, 0, 0).String()

	derivedPriv, _ := hd.DerivePrivateKeyForPath(masterPriv, chainCode, path)

	privKey := secp256k1.GenPrivKeySecp256k1(derivedPriv)

	pubKey := privKey.PubKey()

	secpPubKey, ok := pubKey.(secp256k1.PubKey)
	if !ok {
		panic(ok)
	}

	// Convert the public key to Bech32 with custom HRP
	bech32PubKey, err := bech32ifyPubKeyWithCustomHRP("celestia", secpPubKey)
	if err != nil {
		panic(err)
	}

	fmt.Println("Address Ought to be", bech32PubKey)

	return privKey, secpPubKey, bech32PubKey
}

func bech32ifyPubKeyWithCustomHRP(hrp string, pubKey secp256k1.PubKey) (string, error) {
	return types.Bech32ifyAddressBytes(hrp, pubKey.Bytes())
}
