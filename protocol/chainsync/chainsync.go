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

// Package chainsync implements the Ouroboros chain-sync protocol
package chainsync

import (
	"time"

	"github.com/blinklabs-io/gouroboros/protocol"
	"github.com/blinklabs-io/gouroboros/protocol/common"
)

// Protocol identifiers
const (
	ProtocolName         = "chain-sync"
	ProtocolIdNtN uint16 = 2
	ProtocolIdNtC uint16 = 5
)

var (
	stateIdle      = protocol.NewState(1, "Idle")
	stateCanAwait  = protocol.NewState(2, "CanAwait")
	stateMustReply = protocol.NewState(3, "MustReply")
	stateIntersect = protocol.NewState(4, "Intersect")
	stateDone      = protocol.NewState(5, "Done")
)

// ChainSync protocol state machine
var StateMap = protocol.StateMap{
	stateIdle: protocol.StateMapEntry{
		Agency: protocol.AgencyClient,
		Transitions: []protocol.StateTransition{
			{
				MsgType:  MessageTypeRequestNext,
				NewState: stateCanAwait,
			},
			{
				MsgType:  MessageTypeFindIntersect,
				NewState: stateIntersect,
			},
			{
				MsgType:  MessageTypeDone,
				NewState: stateDone,
			},
		},
	},
	stateCanAwait: protocol.StateMapEntry{
		Agency: protocol.AgencyServer,
		Transitions: []protocol.StateTransition{
			{
				MsgType:  MessageTypeAwaitReply,
				NewState: stateMustReply,
			},
			{
				MsgType:  MessageTypeRollForward,
				NewState: stateIdle,
			},
			{
				MsgType:  MessageTypeRollBackward,
				NewState: stateIdle,
			},
		},
	},
	stateIntersect: protocol.StateMapEntry{
		Agency: protocol.AgencyServer,
		Transitions: []protocol.StateTransition{
			{
				MsgType:  MessageTypeIntersectFound,
				NewState: stateIdle,
			},
			{
				MsgType:  MessageTypeIntersectNotFound,
				NewState: stateIdle,
			},
		},
	},
	stateMustReply: protocol.StateMapEntry{
		Agency: protocol.AgencyServer,
		Transitions: []protocol.StateTransition{
			{
				MsgType:  MessageTypeRollForward,
				NewState: stateIdle,
			},
			{
				MsgType:  MessageTypeRollBackward,
				NewState: stateIdle,
			},
		},
	},
	stateDone: protocol.StateMapEntry{
		Agency: protocol.AgencyNone,
	},
}

// ChainSync is a wrapper object that holds the client and server instances
type ChainSync struct {
	Client *Client
	Server *Server
}

// Config is used to configure the ChainSync protocol instance
type Config struct {
	RollBackwardFunc RollBackwardFunc
	RollForwardFunc  RollForwardFunc
	IntersectTimeout time.Duration
	BlockTimeout     time.Duration
	PipelineLimit    int
}

// Callback function types
type RollBackwardFunc func(common.Point, Tip) error
type RollForwardFunc func(uint, interface{}, Tip) error

// New returns a new ChainSync object
func New(protoOptions protocol.ProtocolOptions, cfg *Config) *ChainSync {
	c := &ChainSync{
		Client: NewClient(protoOptions, cfg),
		Server: NewServer(protoOptions, cfg),
	}
	return c
}

// ChainSyncOptionFunc represents a function used to modify the ChainSync protocol config
type ChainSyncOptionFunc func(*Config)

// NewConfig returns a new ChainSync config object with the provided options
func NewConfig(options ...ChainSyncOptionFunc) Config {
	c := Config{
		PipelineLimit:    0,
		IntersectTimeout: 5 * time.Second,
		// We should really use something more useful like 30-60s, but we've seen 55s between blocks
		// in the preview network
		// https://preview.cexplorer.io/block/cb08a386363a946d2606e912fcd81ffed2bf326cdbc4058297b14471af4f67e9
		// https://preview.cexplorer.io/block/86806dca4ba735b233cbeee6da713bdece36fd41fb5c568f9ef5a3f5cbf572a3
		BlockTimeout: 180 * time.Second,
	}
	// Apply provided options functions
	for _, option := range options {
		option(&c)
	}
	return c
}

// WithRollBackwardFunc specifies the RollBackward callback function
func WithRollBackwardFunc(rollBackwardFunc RollBackwardFunc) ChainSyncOptionFunc {
	return func(c *Config) {
		c.RollBackwardFunc = rollBackwardFunc
	}
}

// WithRollForwardFunc specifies the RollForward callback function
func WithRollForwardFunc(rollForwardFunc RollForwardFunc) ChainSyncOptionFunc {
	return func(c *Config) {
		c.RollForwardFunc = rollForwardFunc
	}
}

// WithIntersectTimeout specifies the timeout for intersect operations
func WithIntersectTimeout(timeout time.Duration) ChainSyncOptionFunc {
	return func(c *Config) {
		c.IntersectTimeout = timeout
	}
}

// WithBlockTimeout specifies the timeout for block fetch operations
func WithBlockTimeout(timeout time.Duration) ChainSyncOptionFunc {
	return func(c *Config) {
		c.BlockTimeout = timeout
	}
}

// WithPipelineLimit specifies the maximum number of block requests to pipeline
func WithPipelineLimit(limit int) ChainSyncOptionFunc {
	return func(c *Config) {
		c.PipelineLimit = limit
	}
}
