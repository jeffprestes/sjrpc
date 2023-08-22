package model

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
)

type RPCRequest struct {
	JsonRpcVersion string `json:"jsonrpc"`
	Method         string `json:"method"`
	Params         []any  `json:"params"`
	ID             int    `json:"id"`
}

func (rpc *RPCRequest) ToByte() (data []byte) {
	// TODO: optimize the response removing ID from hash generation
	// but update it back to the original ID of the call
	// to allow it to be compatible with Ethers
	// tmpId := rpc.ID
	// rpc.ID = 1
	// data, _ = json.Marshal(rpc)
	// rpc.ID = tmpId
	data, _ = json.Marshal(rpc)
	return
}

func (rpc *RPCRequest) Hash() (hash []byte) {
	data := rpc.ToByte()
	tmp := blake2b.Sum512(data)
	hash = tmp[:] // blake2b.Sum2
	return
}

func (rpc *RPCRequest) Base64Hash() (hash string) {
	byteHash := rpc.Hash()
	hash = base64.StdEncoding.EncodeToString(byteHash)
	return
}

func (rpc *RPCRequest) IsCacheable() (resp bool) {
	switch rpc.Method {
	case
		"eth_getTransactionByBlockNumberAndIndex",
		"eth_getTransactionByBlockHashAndIndex",
		"web3_clientVersion",
		"web3_sha3",
		"net_version",
		"eth_chainId",
		"eth_getBlockByHash",
		"eth_getBlockByNumber",
		"eth_getBlockTransactionCountByHash",
		"eth_getBlockTransactionCountByNumber":
		resp = true
	}
	return
}

func (rpc *RPCRequest) IsAfterFinalCacheable() (resp bool) {
	switch rpc.Method {
	case "eth_getTransactionReceipt", "eth_getTransactionByHash":
		resp = true
	}
	return
}

func (rpc *RPCRequest) IsTimelyCacheable() (resp bool) {
	switch rpc.Method {
	case "eth_getLogs", "eth_getCode", "eth_getTransactionCount", "eth_feeHistory", "eth_getStorageAt", "eth_getBalance":
		resp = true
	}
	return
}

func (rpc *RPCRequest) IsEnvCacheable() (resp bool) {
	switch rpc.Method {
	case "eth_accounts":
		resp = true
	}
	return
}

func (rpc *RPCRequest) IsResultFinal(resp string) (final bool) {
	final = true
	if strings.Contains(resp, `"result":null`) ||
		strings.Contains(resp, `"result": null`) ||
		strings.Contains(resp, `"blockNumber": null,`) ||
		strings.Contains(resp, `"blockNumber":null,`) {
		final = false
	}
	return
}

type EphemeralRequest struct {
	Base64Hash  []byte
	Request     RPCRequest
	Response    string
	BlockNumber uint64
	When        int64
}

func (erpc *EphemeralRequest) IsStillValid() (ok bool) {
	now := time.Now().UTC().Unix()
	max := erpc.When + 12
	if now <= max {
		ok = true
	}
	return
}
