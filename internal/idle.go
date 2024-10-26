package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

type IdlerConfig struct {
	RulesFile, MetricsFile string
	Limit                  uint64
}

type RuleMissingMetrics struct {
	Rule    Rule
	Metrics MetricNames
}

type PromRulesIdler struct {
	cfg *IdlerConfig
}

func NewPromRulesIdler(cfg *IdlerConfig) *PromRulesIdler {
	return &PromRulesIdler{
		cfg: cfg,
	}
}

func (pri *PromRulesIdler) List(ctx context.Context) ([]RuleMissingMetrics, error) {
	metrics, err := realAllMetricsCSV(ctx, pri.cfg.MetricsFile)
	if err != nil {
		return nil, err
	}
	rf, err := os.Open(pri.cfg.RulesFile)
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
			if pri.isOffLimit(len(result)) {
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

func (pri *PromRulesIdler) isOffLimit(n int) bool {
	return uint64(n) >= pri.cfg.Limit
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
