package internal

import (
	"fmt"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

type MetricName string

func parsePromQuery(query string) ([]MetricName, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, fmt.Errorf("parse expr: %w", err)
	}

	var res []MetricName
	parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
		if n, ok := node.(*parser.VectorSelector); ok {
			if n.Name != "" {
				res = append(res, MetricName(n.Name))
				return nil
			}
			for _, m := range n.LabelMatchers {
				if m.Name == labels.MetricName && validMetricNameExp.MatchString(m.Value) {
					res = append(res, MetricName(n.Name))
					return nil
				}
			}
		}
		return nil
	})
	return res, nil
}
