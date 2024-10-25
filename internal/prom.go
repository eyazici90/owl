package internal

import (
	"fmt"

	"github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

func mustNewPromAPIV1(addr string) promapiv1.API {
	v1api, err := newPromAPIV1(addr)
	if err != nil {
		panic(err)
	}
	return v1api
}

func newPromAPIV1(addr string) (promapiv1.API, error) {
	cl, err := api.NewClient(api.Config{
		Address: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("new prom client: %w", err)
	}
	return promapiv1.NewAPI(cl), nil
}

func parsePromQuery(query string) (MetricNames, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, fmt.Errorf("parse expr: %w", err)
	}

	var res MetricNames
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
