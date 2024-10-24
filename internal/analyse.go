package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

type AnalyserConfig struct {
	RulesFile, MetricsFile string
	Limit                  uint64
}

type RuleMissingMetrics struct {
	Rule     string
	RuleType string
	Metrics  []string
}

type PromRulesAnalyser struct {
	cfg *AnalyserConfig
}

func NewPromRulesAnalyser(cfg *AnalyserConfig) (*PromRulesAnalyser, error) {
	return &PromRulesAnalyser{
		cfg: cfg,
	}, nil
}

func (pra *PromRulesAnalyser) FindRulesMissingMetrics(ctx context.Context) ([]RuleMissingMetrics, error) {
	mf, err := os.Open(pra.cfg.MetricsFile)
	if err != nil {
		return nil, fmt.Errorf("open metrics: %w", err)
	}
	defer func() {
		_ = mf.Close()
	}()

	metrics := make(map[string]struct{})
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
			metrics[rec[0]] = struct{}{}
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

			typ, name, query, _, _ := rec[0], rec[1], rec[2], rec[3], rec[4]
			ms := parsePromQuery(query)
			missing, found := missingValues(metrics, ms...)
			if !found {
				continue
			}
			result = append(result, RuleMissingMetrics{
				Rule:     name,
				RuleType: typ,
				Metrics:  missing,
			})
		}
	}
	return result, nil
}

func (pra *PromRulesAnalyser) isOffLimit(n int) bool {
	return uint64(n) > pra.cfg.Limit
}

var validMetricNameExp = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

func parsePromQuery(query string) []string {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		log.Printf("%v", err)
		return nil
	}

	var res []string
	parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
		if n, ok := node.(*parser.VectorSelector); ok {
			if n.Name != "" {
				res = append(res, n.Name)
				return nil
			}
			for _, m := range n.LabelMatchers {
				if m.Name == labels.MetricName && validMetricNameExp.MatchString(m.Value) {
					res = append(res, n.Name)
					return nil
				}
			}
		}
		return nil
	})
	return res
}

func missingValues(search map[string]struct{}, vals ...string) ([]string, bool) {
	var res []string
	for _, v := range vals {
		if _, ok := search[v]; !ok {
			res = append(res, v)
		}
	}
	return res, len(res) > 0
}
