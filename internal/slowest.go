package internal

import (
	"context"
	"sort"
	"time"
)

type SlowestConfig struct {
	RulesFile string
	Limit     uint64
}

type (
	SlowestRulesResult struct {
		Rules     []SlowRule
		ParseErrs []error
	}
	SlowRule struct {
		Rule     Rule
		EvalTime time.Duration
	}
)

type PromRulesSlowest struct {
	cfg *SlowestConfig
}

func NewPromRulesSlowest(cfg *SlowestConfig) *PromRulesSlowest {
	return &PromRulesSlowest{cfg: cfg}
}

func (prs *PromRulesSlowest) Get(ctx context.Context) (*SlowestRulesResult, error) {
	rules, silentErrs, err := readAllRulesCSV(ctx, prs.cfg.RulesFile)
	if err != nil {
		return nil, err
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].EvalDuration > rules[j].EvalDuration
	})

	results := make([]SlowRule, prs.cfg.Limit)
	topk := rules[:prs.cfg.Limit]
	for i, rule := range topk {
		results[i] = SlowRule{
			Rule:     rule,
			EvalTime: time.Duration(rule.EvalDuration * float64(time.Second)),
		}
	}
	return &SlowestRulesResult{
		Rules:     results,
		ParseErrs: silentErrs,
	}, nil
}
