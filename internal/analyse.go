package internal

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

type Config struct {
	Addr string
}

type PromRuleAnalyser struct {
	v1api promapiv1.API
}

func NewPromRuleAnalyser(cfg *Config) (*PromRuleAnalyser, error) {
	cl, err := api.NewClient(api.Config{
		Address: cfg.Addr,
	})
	if err != nil {
		return nil, fmt.Errorf("new prom client: %w", err)
	}
	return &PromRuleAnalyser{
		v1api: promapiv1.NewAPI(cl),
	}, nil
}

type RuleMissingMetrics struct {
	Rule     string
	RuleType string
	Metrics  []string
}

func (pra *PromRuleAnalyser) FindRulesMissingMetrics(ctx context.Context) ([]RuleMissingMetrics, error) {
	var t time.Time
	metrics, _, err := pra.v1api.LabelValues(ctx, labels.MetricName, nil, t, t)
	if err != nil {
		return nil, fmt.Errorf("get metrics: %w", err)
	}

	metricSearch := make(map[string]struct{}, len(metrics))
	for _, metric := range metrics {
		metricSearch[string(metric)] = struct{}{}
	}
	rules, err := pra.v1api.Rules(ctx)
	if err != nil {
		return nil, fmt.Errorf("get rules: %w", err)
	}

	var result []RuleMissingMetrics
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			switch v := rule.(type) {
			case promapiv1.RecordingRule:
				ms := parseQuery(v.Query)
				missing, found := missingValues(metricSearch, ms...)
				if !found {
					continue
				}
				result = append(result, RuleMissingMetrics{
					Rule:     v.Name,
					RuleType: "recording",
					Metrics:  missing,
				})
			case promapiv1.AlertingRule:
				ms := parseQuery(v.Query)
				missing, found := missingValues(metricSearch, ms...)
				if !found {
					continue
				}
				result = append(result, RuleMissingMetrics{
					Rule:     v.Name,
					RuleType: "alerting",
					Metrics:  missing,
				})
			default:
			}
		}
	}
	return result, nil
}

var validMetricNameExp = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

func parseQuery(query string) []string {
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
