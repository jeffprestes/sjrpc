package model

type BlockNumberResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  string `json:"result"`
}

type BlockByNumberResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		BaseFeePerGas   string `json:"baseFeePerGas"`
		Difficulty      string `json:"difficulty"`
		ExtraData       string `json:"extraData"`
		GasLimit        string `json:"gasLimit"`
		GasUsed         string `json:"gasUsed"`
		Hash            string `json:"hash"`
		LogsBloom       string `json:"logsBloom"`
		Miner           string `json:"miner"`
		MixHash         string `json:"mixHash"`
		Nonce           string `json:"nonce"`
		Number          string `json:"number"`
		ParentHash      string `json:"parentHash"`
		ReceiptsRoot    string `json:"receiptsRoot"`
		Sha3Uncles      string `json:"sha3Uncles"`
		Size            string `json:"size"`
		StateRoot       string `json:"stateRoot"`
		Timestamp       string `json:"timestamp"`
		TotalDifficulty string `json:"totalDifficulty"`
		Transactions    []struct {
			AccessList           []interface{} `json:"accessList,omitempty"`
			BlockHash            string        `json:"blockHash"`
			BlockNumber          string        `json:"blockNumber"`
			ChainID              string        `json:"chainId"`
			From                 string        `json:"from"`
			Gas                  string        `json:"gas"`
			GasPrice             string        `json:"gasPrice"`
			Hash                 string        `json:"hash"`
			Input                string        `json:"input"`
			MaxFeePerGas         string        `json:"maxFeePerGas,omitempty"`
			MaxPriorityFeePerGas string        `json:"maxPriorityFeePerGas,omitempty"`
			Nonce                string        `json:"nonce"`
			R                    string        `json:"r"`
			S                    string        `json:"s"`
			To                   string        `json:"to"`
			TransactionIndex     string        `json:"transactionIndex"`
			Type                 string        `json:"type"`
			V                    string        `json:"v"`
			Value                string        `json:"value"`
		} `json:"transactions"`
		TransactionsRoot string        `json:"transactionsRoot"`
		Uncles           []interface{} `json:"uncles"`
		Withdrawals      []struct {
			Address        string `json:"address"`
			Amount         string `json:"amount"`
			Index          string `json:"index"`
			ValidatorIndex string `json:"validatorIndex"`
		} `json:"withdrawals"`
		WithdrawalsRoot string `json:"withdrawalsRoot"`
	} `json:"result"`
}
