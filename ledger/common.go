// Copyright 2023 Blink Labs, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ledger

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/internal/base58"
	"github.com/blinklabs-io/gouroboros/internal/bech32"

	"golang.org/x/crypto/blake2b"
)

type Blake2b256 [32]byte

func NewBlake2b256(data []byte) Blake2b256 {
	b := Blake2b256{}
	copy(b[:], data)
	return b
}

func (b Blake2b256) String() string {
	return hex.EncodeToString([]byte(b[:]))
}

func (b Blake2b256) Bytes() []byte {
	return b[:]
}

type Blake2b224 [28]byte

func NewBlake2b224(data []byte) Blake2b224 {
	b := Blake2b224{}
	copy(b[:], data)
	return b
}

func (b Blake2b224) String() string {
	return hex.EncodeToString([]byte(b[:]))
}

func (b Blake2b224) Bytes() []byte {
	return b[:]
}

type Blake2b160 [20]byte

func NewBlake2b160(data []byte) Blake2b160 {
	b := Blake2b160{}
	copy(b[:], data)
	return b
}

func (b Blake2b160) String() string {
	return hex.EncodeToString([]byte(b[:]))
}

func (b Blake2b160) Bytes() []byte {
	return b[:]
}

type MultiAssetTypeOutput = uint64
type MultiAssetTypeMint = int64

// MultiAsset represents a collection of policies, assets, and quantities. It's used for
// TX outputs (uint64) and TX asset minting (int64 to allow for negative values for burning)
type MultiAsset[T MultiAssetTypeOutput | MultiAssetTypeMint] struct {
	data map[Blake2b224]map[cbor.ByteString]T
}

type multiAssetJson[T MultiAssetTypeOutput | MultiAssetTypeMint] struct {
	Name        string `json:"name"`
	NameHex     string `json:"nameHex"`
	PolicyId    string `json:"policyId"`
	Fingerprint string `json:"fingerprint"`
	Amount      T      `json:"amount"`
}

func (m *MultiAsset[T]) UnmarshalCBOR(data []byte) error {
	_, err := cbor.Decode(data, &(m.data))
	return err
}

func (m *MultiAsset[T]) MarshalCBOR() ([]byte, error) {
	return cbor.Encode(&(m.data))
}

func (m MultiAsset[T]) MarshalJSON() ([]byte, error) {
	tmpAssets := []multiAssetJson[T]{}
	for policyId, policyData := range m.data {
		for assetName, amount := range policyData {
			tmpObj := multiAssetJson[T]{
				Name:        string(assetName),
				NameHex:     hex.EncodeToString(assetName.Bytes()),
				Amount:      amount,
				PolicyId:    policyId.String(),
				Fingerprint: NewAssetFingerprint(policyId.Bytes(), assetName.Bytes()).String(),
			}
			tmpAssets = append(tmpAssets, tmpObj)
		}
	}
	return json.Marshal(&tmpAssets)
}

func (m *MultiAsset[T]) Policies() []Blake2b224 {
	var ret []Blake2b224
	for policyId := range m.data {
		ret = append(ret, policyId)
	}
	return ret
}

func (m *MultiAsset[T]) Assets(policyId Blake2b224) [][]byte {
	assets, ok := m.data[policyId]
	if !ok {
		return nil
	}
	var ret [][]byte
	for assetName := range assets {
		ret = append(ret, assetName.Bytes())
	}
	return ret
}

func (m *MultiAsset[T]) Asset(policyId Blake2b224, assetName []byte) T {
	policy, ok := m.data[policyId]
	if !ok {
		return 0
	}
	return policy[cbor.ByteString(assetName)]
}

type AssetFingerprint struct {
	policyId  []byte
	assetName []byte
}

func NewAssetFingerprint(policyId []byte, assetName []byte) AssetFingerprint {
	return AssetFingerprint{
		policyId:  policyId,
		assetName: assetName,
	}
}

func (a AssetFingerprint) Hash() Blake2b160 {
	// We can ignore the error return here because our fixed size/key arguments will
	// never trigger an error
	tmpHash, _ := blake2b.New(20, nil)
	tmpHash.Write(a.policyId)
	tmpHash.Write(a.assetName)
	return NewBlake2b160(tmpHash.Sum(nil))
}

func (a AssetFingerprint) String() string {
	// Convert data to base32 and encode as bech32
	convData, err := bech32.ConvertBits(a.Hash().Bytes(), 8, 5, true)
	if err != nil {
		panic(fmt.Sprintf("unexpected error converting data to base32: %s", err))
	}
	encoded, err := bech32.Encode("asset", convData)
	if err != nil {
		panic(fmt.Sprintf("unexpected error encoding data as bech32: %s", err))
	}
	return encoded
}

const (
	addressHeaderTypeMask    = 0xF0
	addressHeaderNetworkMask = 0x0F
	addressHashSize          = 28

	addressTypeKeyKey        = 0b0000
	addressTypeScriptKey     = 0b0001
	addressTypeKeyScript     = 0b0010
	addressTypeScriptScript  = 0b0011
	addressTypeKeyPointer    = 0b0100
	addressTypeScriptPointer = 0b0101
	addressTypeKeyNone       = 0b0110
	addressTypeScriptNone    = 0b0111
	addressTypeByron         = 0b1000
	addressTypeNoneKey       = 0b1110
	addressTypeNoneScript    = 0b1111
)

type Address struct {
	addressType    uint8
	networkId      uint8
	paymentAddress []byte
	stakingAddress []byte
}

// NewAddress returns an Address based on the provided bech32 address string
func NewAddress(addr string) (Address, error) {
	_, data, err := bech32.DecodeNoLimit(addr)
	if err != nil {
		return Address{}, err
	}
	decoded, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return Address{}, err
	}
	a := Address{}
	a.populateFromBytes(decoded)
	return a, nil
}

func (a *Address) populateFromBytes(data []byte) {
	// Extract header info
	header := data[0]
	a.addressType = (header & addressHeaderTypeMask) >> 4
	a.networkId = header & addressHeaderNetworkMask
	// Extract payload
	// NOTE: this is probably incorrect for Byron
	payload := data[1:]
	a.paymentAddress = payload[:addressHashSize]
	a.stakingAddress = payload[addressHashSize:]
	// Adjust stake addresses
	if a.addressType == addressTypeNoneKey || a.addressType == addressTypeNoneScript {
		a.stakingAddress = a.paymentAddress[:]
		a.paymentAddress = make([]byte, 0)
	}
}

func (a *Address) UnmarshalCBOR(data []byte) error {
	// Decode bytes from CBOR
	tmpData := []byte{}
	if _, err := cbor.Decode(data, &tmpData); err != nil {
		return err
	}
	a.populateFromBytes(tmpData)
	return nil
}

func (a *Address) MarshalCBOR() ([]byte, error) {
	return cbor.Encode(a.Bytes())
}

// StakeAddress returns a new Address with only the stake key portion. This will return nil if the address is not a payment/staking key pair
func (a Address) StakeAddress() *Address {
	if a.addressType != addressTypeKeyKey && a.addressType != addressTypeScriptKey {
		return nil
	}
	newAddr := &Address{
		addressType:    addressTypeNoneKey,
		networkId:      a.networkId,
		stakingAddress: a.stakingAddress[:],
	}
	return newAddr
}

func (a Address) generateHRP() string {
	var ret string
	if a.addressType == addressTypeNoneKey || a.addressType == addressTypeNoneScript {
		ret = "stake"
	} else {
		ret = "addr"
	}
	// Add test_ suffix if not mainnet
	if a.networkId != 1 {
		ret += "_test"
	}
	return ret
}

// Bytes returns the underlying bytes for the address
func (a Address) Bytes() []byte {
	ret := []byte{}
	ret = append(ret, (byte(a.addressType)<<4)|(byte(a.networkId)&addressHeaderNetworkMask))
	ret = append(ret, a.paymentAddress...)
	ret = append(ret, a.stakingAddress...)
	return ret
}

// String returns the bech32-encoded version of the address
func (a Address) String() string {
	data := a.Bytes()
	if a.addressType == addressTypeByron {
		// Encode data to base58
		encoded := base58.Encode(data)
		return encoded
	} else {
		// Convert data to base32 and encode as bech32
		convData, err := bech32.ConvertBits(data, 8, 5, true)
		if err != nil {
			panic(fmt.Sprintf("unexpected error converting data to base32: %s", err))
		}
		// Generate human readable part of address for output
		hrp := a.generateHRP()
		encoded, err := bech32.Encode(hrp, convData)
		if err != nil {
			panic(fmt.Sprintf("unexpected error encoding data as bech32: %s", err))
		}
		return encoded
	}
}

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(`"` + a.String() + `"`), nil
}
