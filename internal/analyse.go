package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
)

type AnalyserConfig struct {
	RulesFile, MetricsFile string
	Limit                  uint64
}

type RuleMissingMetrics struct {
	Group, Rule, RuleType string
	Metrics               []MetricName
}

type PromRulesAnalyser struct {
	cfg *AnalyserConfig
}

func NewPromRulesAnalyser(cfg *AnalyserConfig) *PromRulesAnalyser {
	return &PromRulesAnalyser{
		cfg: cfg,
	}
}

func (pra *PromRulesAnalyser) FindRulesMissingMetrics(ctx context.Context) ([]RuleMissingMetrics, error) {
	mf, err := os.Open(pra.cfg.MetricsFile)
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

	rf, err := os.Open(pra.cfg.RulesFile)
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
			if pra.isOffLimit(len(result)) {
				break EXIT
			}
			rec, err := rr.Read()
			if err == io.EOF {
				break EXIT
			}
			if err != nil {
				return nil, fmt.Errorf("read rule: %w", err)
			}

			grp, typ, name, query := rec[0], rec[1], rec[2], rec[3]
			ms, err := parsePromQuery(query)
			if err != nil {
				return nil, fmt.Errorf("parse prom expr: %w", err)
			}
			missing, found := missingValues(metrics, ms...)
			if !found {
				continue
			}
			result = append(result, RuleMissingMetrics{
				Group:    grp,
				Rule:     name,
				RuleType: typ,
				Metrics:  missing,
			})
		}
	}
	return result, nil
}

func (pra *PromRulesAnalyser) isOffLimit(n int) bool {
	return uint64(n) >= pra.cfg.Limit
}

var validMetricNameExp = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

func missingValues[T comparable](search map[T]struct{}, vals ...T) ([]T, bool) {
	var res []T
	for _, v := range vals {
		if _, ok := search[v]; !ok {
			res = append(res, v)
		}
	}
	return res, len(res) > 0
}
