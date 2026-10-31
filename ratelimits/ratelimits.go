// Package ratelimits declares the Helius request budgets this plugin paces
// itself against. Numbers are the Free plan tier (the published default);
// users on Developer/Business/Professional plans raise them via Terminal's
// Rate Limits UI override.
//
// Sources (retrieved 2026-10-31):
//   - https://www.helius.dev/docs/billing/rate-limits.md
//     ("Standard Rate Limits": Free = 10 requests/s RPC;
//     "Complex RPC Calls": getProgramAccounts = 5/sec on Free)
//   - https://www.helius.dev/docs/billing/plans.md
//     (Free plan tier table, confirms the same numbers)
package ratelimits

import (
	"time"

	"github.com/plusev-terminal/go-plugin-common/meta"
)

// HeliusRPC returns the manifest rate-limit rules for Helius's keyed mainnet
// JSON-RPC host (api-mainnet.helius-rpc.com). Helius enforces plan limits per
// API key, so both rules are PerCredential.
func HeliusRPC() []meta.RateLimitRule {
	return []meta.RateLimitRule{
		// Free plan "RPC Rate Limit": 10 requests/s, enforced per API key
		// (the endpoint is keyed: /?api-key=...).
		// https://www.helius.dev/docs/billing/rate-limits.md
		meta.NewRateLimitRule("rpc", 10, time.Second).
			WithLabel("RPC requests (Free plan)").
			PerCredential(),
		// Free plan "Complex RPC Calls": getProgramAccounts 5/sec per key.
		// The limiter keys JSON-RPC buckets by the method the host parses
		// from the request body (the plugin sends JSON-RPC bodies, so no
		// per-request RateKey is needed), which gives every method a 5 rps
		// bucket per credential — exact for getProgramAccounts, a touch
		// conservative for the read-only methods, which are anyway capped
		// at 10 rps total by "rpc".
		// https://www.helius.dev/docs/billing/rate-limits.md
		meta.NewRateLimitRule("per-method", 5, time.Second).
			WithLabel("Per-method cap (getProgramAccounts, Free plan)").
			PerCredential().
			PerKey(),
	}
}
