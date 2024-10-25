package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
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
	f, err := os.Open(prs.cfg.RulesFile)
	if err != nil {
		return nil, fmt.Errorf("open rules: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	r := csv.NewReader(f)
	if _, err = r.Read(); err != nil { // reading header
		return nil, fmt.Errorf("read header: %w", err)
	}

	var (
		rules      []Rule
		silentErrs []error
	)
OUT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			rec, err := r.Read()
			if err == io.EOF {
				break OUT
			}
			if err != nil {
				silentErrs = append(silentErrs, fmt.Errorf("read rule: %w", err))
				continue
			}
			dur, err := strconv.ParseFloat(rec[5], 64)
			if err != nil {
				return nil, fmt.Errorf("parse eval-duration: %w", err)
			}
			rules = append(rules, Rule{
				Group:        rec[0],
				Type:         rec[1],
				Name:         rec[2],
				Query:        rec[3],
				Labels:       rec[4],
				EvalDuration: dur,
			})
		}
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
