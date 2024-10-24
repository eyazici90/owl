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

type SlowRule struct {
	Group, Typ, Name, Query, Labels string
	EvalTime                        time.Duration
}

type PromRulesSlowest struct {
	cfg *SlowestConfig
}

func NewPromRulesSlowest(cfg *SlowestConfig) *PromRulesSlowest {
	return &PromRulesSlowest{cfg: cfg}
}

func (prs *PromRulesSlowest) Get(ctx context.Context) ([]SlowRule, error) {
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

	var rules []struct {
		grp, typ, name, query, labels string
		evalTime                      float64
	}
EXIT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			rec, err := r.Read()
			if err == io.EOF {
				break EXIT
			}
			if err != nil {
				return nil, fmt.Errorf("read rule: %w", err)
			}
			dur, err := strconv.ParseFloat(rec[5], 64)
			if err != nil {
				return nil, fmt.Errorf("parse eval-duration: %w", err)
			}
			rules = append(rules, struct {
				grp, typ, name, query, labels string
				evalTime                      float64
			}{
				grp:      rec[0],
				typ:      rec[1],
				name:     rec[2],
				query:    rec[3],
				labels:   rec[4],
				evalTime: dur,
			})
		}
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].evalTime > rules[j].evalTime
	})

	result := make([]SlowRule, prs.cfg.Limit)
	top := rules[:prs.cfg.Limit]
	for i, rule := range top {
		result[i] = SlowRule{
			Group:    rule.grp,
			Typ:      rule.typ,
			Name:     rule.name,
			Query:    rule.query,
			Labels:   rule.labels,
			EvalTime: time.Duration(rule.evalTime * float64(time.Second)),
		}
	}
	return result, nil
}
