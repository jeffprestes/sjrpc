package model

import (
	"encoding/json"
	"strings"

	"golang.org/x/crypto/blake2b"
)

type RPCRequest struct {
	JsonRpcVersion string   `json:"jsonrpc"`
	Method         string   `json:"method"`
	Params         []string `json:"params"`
	ID             int      `json:"id"`
}

func (rpc *RPCRequest) ToByte() (data []byte) {
	data, _ = json.Marshal(rpc)
	return
}

func (rpc *RPCRequest) Hash() (hash []byte) {
	data := rpc.ToByte()
	tmp := blake2b.Sum512(data)
	hash = tmp[:] // blake2b.Sum2
	return
}

func (rpc *RPCRequest) IsCacheable() (resp bool) {
	switch rpc.Method {
	case "eth_getTransactionReceipt",
		"eth_getTransactionCount",
		"eth_getTransactionByHash",
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
	case "eth_getLogs", "eth_getCode", "eth_feeHistory", "eth_getStorageAt", "eth_getBalance":
		thereIsLatest := false
		for _, param := range rpc.Params {
			if strings.ToLower(param) == "latest" {
				thereIsLatest = true
				break
			}
		}
		resp = !thereIsLatest
	}
	return
}
