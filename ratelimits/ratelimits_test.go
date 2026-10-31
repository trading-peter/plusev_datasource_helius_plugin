package ratelimits

import (
	"testing"
	"time"

	"github.com/plusev-terminal/go-plugin-common/meta"
)

// Pins the declared Helius budgets so they cannot silently drift.
// Sources (2026-10-31): https://www.helius.dev/docs/billing/rate-limits.md
// Free plan: RPC 10 req/s per key; getProgramAccounts 5/sec per key.
func TestHeliusRPCRules(t *testing.T) {
	rules := HeliusRPC()
	want := []meta.RateLimitRule{
		{
			ID:       "rpc",
			Label:    "RPC requests (Free plan)",
			Limit:    10,
			WindowMs: (1 * time.Second).Milliseconds(),
			Scope:    meta.RateLimitScopeCredential,
		},
		{
			ID:            "per-method",
			Label:         "Per-method cap (getProgramAccounts, Free plan)",
			Limit:         5,
			WindowMs:      (1 * time.Second).Milliseconds(),
			Scope:         meta.RateLimitScopeCredential,
			PerKeyEnabled: true,
		},
	}
	if len(rules) != len(want) {
		t.Fatalf("got %d rules, want %d: %+v", len(rules), len(want), rules)
	}
	for i, r := range rules {
		w := want[i]
		if r.ID != w.ID || r.Label != w.Label || r.Limit != w.Limit ||
			r.WindowMs != w.WindowMs || r.Scope != w.Scope || r.PerKeyEnabled != w.PerKeyEnabled {
			t.Errorf("rule %d = %+v, want %+v", i, r, w)
		}
		if r.Burst != 0 {
			t.Errorf("rule %d Burst = %d, want 0 (defaults to Limit)", i, r.Burst)
		}
		if len(r.Paths) != 0 {
			t.Errorf("rule %d Paths = %v, want none (applies to the whole host)", i, r.Paths)
		}
		if r.CostWeighted {
			t.Errorf("rule %d must not be weighted", i)
		}
	}
}
