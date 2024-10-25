package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

type CheckerConfig struct {
	RulesFile, MetricsFile string
	Limit                  uint64
}

type RuleMissingMetrics struct {
	Rule    Rule
	Metrics MetricNames
}

type PromRulesChecker struct {
	cfg *CheckerConfig
}

func NewPromRulesChecker(cfg *CheckerConfig) *PromRulesChecker {
	return &PromRulesChecker{
		cfg: cfg,
	}
}

func (prc *PromRulesChecker) CheckStaleRules(ctx context.Context) ([]RuleMissingMetrics, error) {
	mf, err := os.Open(prc.cfg.MetricsFile)
	if err != nil {
		return nil, fmt.Errorf("open metrics: %w", err)
	}
	defer func() {
		_ = mf.Close()
	}()

	metrics := make(map[MetricName]struct{})
	mr := csv.NewReader(mf)
	if _, err = mr.Read(); err != nil { // reading header
		return nil, fmt.Errorf("read header: %w", err)
	}

OUT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			rec, err := mr.Read()
			if err == io.EOF {
				break OUT
			}
			if err != nil {
				return nil, fmt.Errorf("read metric: %w", err)
			}
			metrics[MetricName(rec[0])] = struct{}{}
		}
	}

	rf, err := os.Open(prc.cfg.RulesFile)
	if err != nil {
		return nil, fmt.Errorf("open rules: %w", err)
	}
	defer func() {
		_ = rf.Close()
	}()

	rr := csv.NewReader(rf)
	if _, err = rr.Read(); err != nil { // reading header
		return nil, fmt.Errorf("read header: %w", err)
	}

	var result []RuleMissingMetrics
EXIT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if prc.isOffLimit(len(result)) {
				break EXIT
			}
			rec, err := rr.Read()
			if err == io.EOF {
				break EXIT
			}
			if err != nil {
				return nil, fmt.Errorf("read rule: %w", err)
			}
			rule := Rule{
				Group: rec[0],
				Type:  rec[1],
				Name:  rec[2],
				Query: rec[3],
			}
			ms, err := parsePromQuery(rule.Query)
			if err != nil {
				return nil, fmt.Errorf("parse prom expr: %w", err)
			}
			missing, found := missingValues(metrics, ms...)
			if !found {
				continue
			}
			result = append(result, RuleMissingMetrics{
				Rule:    rule,
				Metrics: missing,
			})
		}
	}
	return result, nil
}

func (prc *PromRulesChecker) isOffLimit(n int) bool {
	return uint64(n) >= prc.cfg.Limit
}

func missingValues[T comparable](search map[T]struct{}, vals ...T) ([]T, bool) {
	var res []T
	for _, v := range vals {
		if _, ok := search[v]; !ok {
			res = append(res, v)
		}
	}
	return res, len(res) > 0
}
