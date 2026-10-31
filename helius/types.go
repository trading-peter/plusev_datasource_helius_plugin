package helius

import (
	"encoding/json"
	"fmt"
)

const (
	baseURL    = "https://api-mainnet.helius-rpc.com"
	publicRPC  = "https://api.mainnet-beta.solana.com"
)

// Solana JSON-RPC request/response types

type rpcRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetInflationReward returns the staking rewards for a list of stake accounts
// for a given set of epochs. This wraps the Solana getInflationReward JSON-RPC method.
type GetInflationRewardParams struct {
	Accounts []string `json:"accounts"`
	Epochs   []uint64 `json:"epochs"`
}

type InflationReward struct {
	Epoch           uint64 `json:"epoch"`
	EffectiveSlot   uint64 `json:"effectiveSlot"`
	Amount          uint64 `json:"amount"`
	PostBalance     uint64 `json:"postBalance"`
	Commission      *uint8 `json:"commission"`
}

// EpochInfo is the result of getEpochInfo
type EpochInfo struct {
	Epoch              uint64 `json:"epoch"`
	SlotIndex          uint64 `json:"slotIndex"`
	SlotsInEpoch       uint64 `json:"slotsInEpoch"`
	AbsoluteSlot       uint64 `json:"absoluteSlot"`
	BlockHeight        uint64 `json:"blockHeight"`
	TransactionCount   uint64 `json:"transactionCount"`
}

// StakeActivation is the parsed account data for a stake account
type StakeActivation struct {
	State  string  `json:"state"`
	Active uint64  `json:"active"`
	Inactive uint64 `json:"inactive"`
}

// AccountInfo is the result of getAccountInfo with jsonParsed encoding
type AccountInfo struct {
	Lamports   uint64             `json:"lamports"`
	Owner      string             `json:"owner"`
	Executable bool               `json:"executable"`
	RentEpoch  uint64             `json:"rentEpoch"`
	Data       json.RawMessage    `json:"data"`
}

// StakeAccountInfo holds parsed info from getAccountInfo for a stake account
type StakeAccountInfo struct {
	Address    string `json:"address"`
	Lamports   uint64 `json:"lamports"`
	Owner      string `json:"owner"`
	Staker     string `json:"staker"`
	Withdrawer string `json:"withdrawer"`
	Voter      string `json:"voter"`
}

// DiscoveredAccount is returned by getProgramAccounts for the Stake Program
type DiscoveredAccount struct {
	Address         string `json:"address"`
	Lamports        uint64 `json:"lamports"`
	Staker          string `json:"staker"`
	Voter           string `json:"voter"`
	Active          bool   `json:"active"`
	ActivationEpoch string `json:"activationEpoch"`
}

// BuildRPCRequest creates a JSON-RPC request body
func BuildRPCRequest(method string, params []any) ([]byte, error) {
	return json.Marshal(rpcRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
}

// ParseRPCResponse parses a JSON-RPC response, returning the raw result or error
func ParseRPCResponse(body []byte) (json.RawMessage, error) {
	var resp rpcResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse RPC response: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}
