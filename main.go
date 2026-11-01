package main

import (
	"fmt"

	ch "github.com/plusev-terminal/go-plugin-common/v2/datasrc/chain"
	"github.com/plusev-terminal/go-plugin-common/v2/logging"
	m "github.com/plusev-terminal/go-plugin-common/v2/meta"
	"github.com/plusev-terminal/go-plugin-common/v2/plugin"
	"github.com/plusev-terminal/go-plugin-common/v2/requester"

	"github.com/trading-peter/plusev_datasource_helius_plugin/helius"
	"github.com/trading-peter/plusev_datasource_helius_plugin/ratelimits"
)

var log = logging.NewLogger("helius-solana-datasource")

type HeliusChainPlugin struct {
	config *plugin.ConfigStore
	client *helius.Client
}

func (p *HeliusChainPlugin) GetMeta() m.Meta {
	return m.Meta{
		PluginID:    "helius-solana-datasource",
		Name:        "Helius Solana RPC",
		AppID:       "datasrc",
		Category:    "chain",
		Chain:       "solana",
		Description: "Reliable Solana RPC access via Helius API",
		Author:      "trading_peter",
		Version: "v1.1.0",
		Repository:  "https://github.com/trading-peter/plusev_datasource_helius_plugin",
		Tags:        []string{"solana", "helius", "rpc", "chain"},
		Contacts: []m.AuthorContact{
			{Kind: "x.com", Value: "https://x.com/trading_peter"},
		},
		Resources: m.ResourceAccess{
			AllowedNetworkTargets: []m.NetworkTargetRule{
				// The only host the client calls; carries Helius's published
				// Free-plan budgets (see the ratelimits package for sources).
				{Pattern: "https://api-mainnet.helius-rpc.com/*", Limits: ratelimits.HeliusRPC()},
				{Pattern: "https://mainnet.helius-rpc.com/*"},
			},
		},
		Features: []string{},
	}
}

func (p *HeliusChainPlugin) GetConfigFields() []plugin.ConfigField {
	return helius.GetConfigFields()
}

func (p *HeliusChainPlugin) OnInit(config *plugin.ConfigStore) error {
	p.config = config

	apiKey := config.GetString("apiKey")
	log.Debug("OnInit called")

	if apiKey == "" {
		log.Error("OnInit failed: apiKey is empty")
		return fmt.Errorf("apiKey is required")
	}

	log.DebugWithData("OnInit: creating Helius client", map[string]any{"apiKeyLength": len(apiKey)})
	p.client = helius.NewClient(requester.NewRequester(), apiKey)
	log.Debug("OnInit completed successfully")

	return nil
}

func (p *HeliusChainPlugin) OnShutdown() error {
	return nil
}

// GetRateLimits intentionally returns an empty array: the plugin declares
// Helius's real budgets as manifest Limits on its network targets (see the
// ratelimits package), so the legacy per-command limiter must stay off to
// avoid double-throttling.
func (p *HeliusChainPlugin) GetRateLimits() []plugin.RateLimit {
	return []plugin.RateLimit{}
}

func (p *HeliusChainPlugin) RegisterCommands(router *plugin.CommandRouter) {
	router.Register(ch.CMD_GET_INFLATION_REWARD, p.handleGetInflationReward)
	router.Register(ch.CMD_GET_EPOCH_INFO, p.handleGetEpochInfo)
	router.Register(ch.CMD_GET_PROGRAM_ACCOUNTS, p.handleGetProgramAccounts)
	router.Register(ch.CMD_GET_ACCOUNT_INFO, p.handleGetAccountInfo)
	router.Register(ch.CMD_GET_STAKE_ACTIVATION, p.handleGetStakeActivation)
	router.Register(ch.CMD_GET_BLOCK_TIME, p.handleGetBlockTime)
}

func (p *HeliusChainPlugin) handleGetInflationReward(params map[string]any) plugin.Response {
	log.Debug("handleGetInflationReward called")

	accountsRaw, ok := params["accounts"].([]any)
	if !ok || len(accountsRaw) == 0 {
		return plugin.ErrorResponseMsg("accounts parameter is required ([]string)")
	}
	accounts := make([]string, len(accountsRaw))
	for i, a := range accountsRaw {
		s, ok := a.(string)
		if !ok {
			return plugin.ErrorResponseMsg("accounts must be an array of strings")
		}
		accounts[i] = s
	}

	var epochs []uint64
	if e, ok := params["epoch"].(float64); ok {
		epochs = []uint64{uint64(e)}
	}

	rewards, err := p.client.GetInflationReward(accounts, epochs)
	if err != nil {
		log.ErrorWithData("handleGetInflationReward failed", map[string]any{"error": err.Error(), "epoch": epochs})
		return plugin.ErrorResponse(err)
	}
	log.DebugWithData("handleGetInflationReward success", map[string]any{"count": len(rewards)})
	return plugin.SuccessResponse(rewards)
}

func (p *HeliusChainPlugin) handleGetEpochInfo(params map[string]any) plugin.Response {
	log.Debug("handleGetEpochInfo called")
	info, err := p.client.GetEpochInfo()
	if err != nil {
		return plugin.ErrorResponse(err)
	}
	return plugin.SuccessResponse(info)
}

func (p *HeliusChainPlugin) handleGetProgramAccounts(params map[string]any) plugin.Response {
	log.Debug("handleGetProgramAccounts called")
	authority, ok := params["authority"].(string)
	if !ok || authority == "" {
		return plugin.ErrorResponseMsg("authority parameter is required (staker wallet address)")
	}

	accounts, err := p.client.DiscoverStakeAccounts(authority)
	if err != nil {
		return plugin.ErrorResponse(err)
	}
	return plugin.SuccessResponse(accounts)
}

func (p *HeliusChainPlugin) handleGetAccountInfo(params map[string]any) plugin.Response {
	address, ok := params["address"].(string)
	if !ok || address == "" {
		return plugin.ErrorResponseMsg("address parameter is required")
	}

	info, err := p.client.GetAccountInfo(address)
	if err != nil {
		log.ErrorWithData("handleGetAccountInfo failed", map[string]any{"error": err.Error()})
		return plugin.ErrorResponse(err)
	}
	log.Debug("handleGetAccountInfo success")
	return plugin.SuccessResponse(info)
}

func (p *HeliusChainPlugin) handleGetStakeActivation(params map[string]any) plugin.Response {
	address, ok := params["address"].(string)
	if !ok || address == "" {
		return plugin.ErrorResponseMsg("address parameter is required")
	}

	activation, err := p.client.GetStakeActivation(address)
	if err != nil {
		return plugin.ErrorResponse(err)
	}
	return plugin.SuccessResponse(activation)
}

func (p *HeliusChainPlugin) handleGetBlockTime(params map[string]any) plugin.Response {
	slotRaw, ok := params["slot"].(float64)
	if !ok {
		return plugin.ErrorResponseMsg("slot parameter is required (uint64)")
	}

	ts, err := p.client.GetBlockTime(uint64(slotRaw))
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	return plugin.SuccessResponse(ts)
}

func init() {
	plugin.RegisterPlugin(&HeliusChainPlugin{})
}

func main() {}
