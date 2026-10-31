package helius

import (
	"encoding/json"
	"fmt"

	"github.com/plusev-terminal/go-plugin-common/logging"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
)

var rpcLog = logging.NewLogger("helius-rpc")

// Client wraps the Helius RPC endpoint for chain-level queries.
type Client struct {
	requester rt.RequestDoer
	apiKey    string
	url       string
}

func NewClient(req rt.RequestDoer, apiKey string) *Client {
	return &Client{
		requester: req,
		apiKey:    apiKey,
		url:       fmt.Sprintf("%s/?api-key=%s", baseURL, apiKey),
	}
}

// GetConfigFields returns the configuration fields for this plugin.
func GetConfigFields() []plugin.ConfigField {
	return []plugin.ConfigField{
		{
			Name:        "apiKey",
			Label:       "Helius API Key",
			Type:        "text",
			Required:    true,
			Encrypt:     true,
			Mask:        true,
			Description: "Your Helius API key. Get one at https://www.helius.dev",
		},
	}
}

func (c *Client) rpcCall(method string, params []any) (json.RawMessage, error) {
	body, err := BuildRPCRequest(method, params)
	if err != nil {
		return nil, err
	}

	rpcLog.DebugWithData("RPC request", map[string]any{"method": method, "url": c.url})

	req := rt.Request{
		Method: "POST",
		URL:    c.url,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}

	resp, err := c.requester.Send(&req, nil)
	if err != nil {
		rpcLog.ErrorWithData("RPC request failed", map[string]any{"method": method, "error": err.Error()})
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.Status >= 400 {
		rpcLog.ErrorWithData("RPC HTTP error", map[string]any{"method": method, "status": resp.Status, "body": string(resp.Body)})
		return nil, fmt.Errorf("HTTP %d: %s", resp.Status, string(resp.Body))
	}

	rpcLog.DebugWithData("RPC response", map[string]any{"method": method, "status": resp.Status, "bodyLen": len(resp.Body)})

	return ParseRPCResponse(resp.Body)
}

// GetEpochInfo returns the current Solana epoch.
func (c *Client) GetEpochInfo() (*EpochInfo, error) {
	result, err := c.rpcCall("getEpochInfo", []any{})
	if err != nil {
		return nil, err
	}
	var info EpochInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("failed to parse epoch info: %w", err)
	}
	return &info, nil
}

// GetInflationReward returns staking rewards for the given accounts and epochs.
func (c *Client) GetInflationReward(accounts []string, epochs []uint64) ([]InflationReward, error) {
	params := []any{accounts}
	if epochs != nil {
		paramsObj := map[string]any{"epoch": epochs[0]}
		params = append(params, paramsObj)
	}
	result, err := c.rpcCall("getInflationReward", params)
	if err != nil {
		return nil, err
	}
	var rewards []InflationReward
	if err := json.Unmarshal(result, &rewards); err != nil {
		return nil, fmt.Errorf("failed to parse rewards: %w", err)
	}
	return rewards, nil
}

// GetAccountInfo returns parsed account info for a given address.
func (c *Client) GetAccountInfo(address string) (*AccountInfo, error) {
	params := []any{
		address,
		map[string]any{"encoding": "jsonParsed"},
	}
	result, err := c.rpcCall("getAccountInfo", params)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Value *AccountInfo `json:"value"`
	}
	if err := json.Unmarshal(result, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse account info: %w", err)
	}
	if wrapper.Value == nil {
		return nil, fmt.Errorf("account not found: %s", address)
	}
	return wrapper.Value, nil
}

// GetStakeActivation returns the stake activation state for an account.
func (c *Client) GetStakeActivation(address string) (*StakeActivation, error) {
	params := []any{
		address,
		map[string]any{"commitment": "confirmed"},
	}
	result, err := c.rpcCall("getStakeActivation", params)
	if err != nil {
		return nil, err
	}
	var activation StakeActivation
	if err := json.Unmarshal(result, &activation); err != nil {
		return nil, fmt.Errorf("failed to parse stake activation: %w", err)
	}
	return &activation, nil
}

// GetBlockTime returns the estimated production time of a block (unix seconds).
func (c *Client) GetBlockTime(slot uint64) (int64, error) {
	result, err := c.rpcCall("getBlockTime", []any{slot})
	if err != nil {
		return 0, err
	}

	var ts int64
	if err := json.Unmarshal(result, &ts); err != nil {
		return 0, fmt.Errorf("failed to parse block time: %w", err)
	}

	return ts, nil
}

// DiscoverStakeAccounts uses getProgramAccounts to find all stake accounts
// delegated to a given vote account, or all stake accounts owned by
// the Stake Program (Stake11111111111111111111111111111111111111).
func (c *Client) DiscoverStakeAccounts(authority string) ([]DiscoveredAccount, error) {
	params := []any{
		"Stake11111111111111111111111111111111111111",
		map[string]any{
			"encoding": "jsonParsed",
			"filters": []map[string]any{
				{
					"memcmp": map[string]any{
						"offset": 12,
						"bytes":  authority,
					},
				},
				{
					"dataSize": 200,
				},
			},
		},
	}
	result, err := c.rpcCall("getProgramAccounts", params)
	if err != nil {
		return nil, err
	}

	var entries []struct {
		Pubkey  string          `json:"pubkey"`
		Account json.RawMessage `json:"account"`
	}
	if err := json.Unmarshal(result, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse program accounts: %w", err)
	}

	var accounts []DiscoveredAccount
	for _, entry := range entries {
		var acc struct {
			Lamports uint64 `json:"lamports"`
			Data     struct {
				Parsed struct {
					Info struct {
						Stake struct {
							Delegation struct {
								Voter             string `json:"voter"`
								Stake             string `json:"stake"`
								ActivationEpoch   string `json:"activationEpoch"`
								DeactivationEpoch string `json:"deactivationEpoch"`
							} `json:"delegation"`
							Authorized struct {
								Staker     string `json:"staker"`
								Withdrawer string `json:"withdrawer"`
							} `json:"authorized"`
						} `json:"stake"`
					} `json:"info"`
				} `json:"parsed"`
			} `json:"data"`
		}
		if err := json.Unmarshal(entry.Account, &acc); err != nil {
			continue
		}

		di := acc.Data.Parsed.Info.Stake
		active := di.Delegation.DeactivationEpoch == "0" ||
			di.Delegation.DeactivationEpoch == "18446744073709551615"

		accounts = append(accounts, DiscoveredAccount{
			Address:         entry.Pubkey,
			Lamports:        acc.Lamports,
			Staker:          di.Authorized.Staker,
			Voter:           di.Delegation.Voter,
			Active:          active,
			ActivationEpoch: di.Delegation.ActivationEpoch,
		})
	}
	return accounts, nil
}

