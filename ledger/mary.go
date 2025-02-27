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
	"encoding/json"
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
)

const (
	ERA_ID_MARY = 3

	BLOCK_TYPE_MARY = 4

	BLOCK_HEADER_TYPE_MARY = 3

	TX_TYPE_MARY = 3
)

type MaryBlock struct {
	cbor.StructAsArray
	cbor.DecodeStoreCbor
	Header                 *MaryBlockHeader
	TransactionBodies      []MaryTransactionBody
	TransactionWitnessSets []ShelleyTransactionWitnessSet
	TransactionMetadataSet map[uint]cbor.Value
}

func (b *MaryBlock) UnmarshalCBOR(cborData []byte) error {
	return b.UnmarshalCbor(cborData, b)
}

func (b *MaryBlock) Hash() string {
	return b.Header.Hash()
}

func (b *MaryBlock) BlockNumber() uint64 {
	return b.Header.BlockNumber()
}

func (b *MaryBlock) SlotNumber() uint64 {
	return b.Header.SlotNumber()
}

func (b *MaryBlock) Era() Era {
	return eras[ERA_ID_MARY]
}

func (b *MaryBlock) Transactions() []Transaction {
	ret := []Transaction{}
	for idx := range b.TransactionBodies {
		tmpTransaction := MaryTransaction{
			Body:       b.TransactionBodies[idx],
			WitnessSet: b.TransactionWitnessSets[idx],
			TxMetadata: b.TransactionMetadataSet[uint(idx)],
		}
		ret = append(ret, &tmpTransaction)
	}
	return ret
}

type MaryBlockHeader struct {
	ShelleyBlockHeader
}

func (h *MaryBlockHeader) Era() Era {
	return eras[ERA_ID_MARY]
}

type MaryTransactionBody struct {
	AllegraTransactionBody
	TxOutputs []MaryTransactionOutput        `cbor:"1,keyasint,omitempty"`
	Mint      MultiAsset[MultiAssetTypeMint] `cbor:"9,keyasint,omitempty"`
}

func (b *MaryTransactionBody) UnmarshalCBOR(cborData []byte) error {
	return b.UnmarshalCbor(cborData, b)
}

func (b *MaryTransactionBody) Outputs() []TransactionOutput {
	ret := []TransactionOutput{}
	for _, output := range b.TxOutputs {
		ret = append(ret, output)
	}
	return ret
}

type MaryTransaction struct {
	cbor.StructAsArray
	cbor.DecodeStoreCbor
	Body       MaryTransactionBody
	WitnessSet ShelleyTransactionWitnessSet
	TxMetadata cbor.Value
}

func (t MaryTransaction) Hash() string {
	return t.Body.Hash()
}

func (t MaryTransaction) Inputs() []TransactionInput {
	return t.Body.Inputs()
}

func (t MaryTransaction) Outputs() []TransactionOutput {
	return t.Body.Outputs()
}

func (t MaryTransaction) Metadata() cbor.Value {
	return t.TxMetadata
}

type MaryTransactionOutput struct {
	cbor.StructAsArray
	OutputAddress Address
	OutputAmount  MaryTransactionOutputValue
}

func (o MaryTransactionOutput) MarshalJSON() ([]byte, error) {
	tmpObj := struct {
		Address Address                           `json:"address"`
		Amount  uint64                            `json:"amount"`
		Assets  *MultiAsset[MultiAssetTypeOutput] `json:"assets,omitempty"`
	}{
		Address: o.OutputAddress,
		Amount:  o.OutputAmount.Amount,
		Assets:  o.OutputAmount.Assets,
	}
	return json.Marshal(&tmpObj)
}

func (o MaryTransactionOutput) Address() Address {
	return o.OutputAddress
}

func (o MaryTransactionOutput) Amount() uint64 {
	return o.OutputAmount.Amount
}

func (o MaryTransactionOutput) Assets() *MultiAsset[MultiAssetTypeOutput] {
	return o.OutputAmount.Assets
}

func (o MaryTransactionOutput) DatumHash() *Blake2b256 {
	return nil
}

func (o MaryTransactionOutput) Datum() *cbor.LazyValue {
	return nil
}

type MaryTransactionOutputValue struct {
	cbor.StructAsArray
	Amount uint64
	// We use a pointer here to allow it to be nil
	Assets *MultiAsset[MultiAssetTypeOutput]
}

func (v *MaryTransactionOutputValue) UnmarshalCBOR(data []byte) error {
	if _, err := cbor.Decode(data, &(v.Amount)); err == nil {
		return nil
	}
	if err := cbor.DecodeGeneric(data, v); err != nil {
		return err
	}
	return nil
}

func (v *MaryTransactionOutputValue) MarshalCBOR() ([]byte, error) {
	if v.Assets == nil {
		return cbor.Encode(v.Amount)
	} else {
		return cbor.EncodeGeneric(v)
	}
}

func NewMaryBlockFromCbor(data []byte) (*MaryBlock, error) {
	var maryBlock MaryBlock
	if _, err := cbor.Decode(data, &maryBlock); err != nil {
		return nil, fmt.Errorf("Mary block decode error: %s", err)
	}
	return &maryBlock, nil
}

func NewMaryTransactionBodyFromCbor(data []byte) (*MaryTransactionBody, error) {
	var maryTx MaryTransactionBody
	if _, err := cbor.Decode(data, &maryTx); err != nil {
		return nil, fmt.Errorf("Mary transaction body decode error: %s", err)
	}
	return &maryTx, nil
}

func NewMaryTransactionFromCbor(data []byte) (*MaryTransaction, error) {
	var maryTx MaryTransaction
	if _, err := cbor.Decode(data, &maryTx); err != nil {
		return nil, fmt.Errorf("Mary transaction decode error: %s", err)
	}
	return &maryTx, nil
}
