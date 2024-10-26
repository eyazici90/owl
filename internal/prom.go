package internal

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

var (
	validMetricNameExpr          = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	variableRangeQueryRangeRegex = regexp.MustCompile(`\[\$?\w+?]`)
	variableSubqueryRangeRegex   = regexp.MustCompile(`\[\$?\w+:\$?\w+?]`)
	variableReplacer             = strings.NewReplacer(
		"$__interval", "5m",
		"$interval", "5m",
		"$resolution", "5s",
		"$__rate_interval", "15s",
		"$rate_interval", "15s",
		"$__range", "1d",
		"${__range_s:glob}", "30",
		"${__range_s}", "30",
	)
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
	expr, err := parser.ParseExpr(replaceVariables(query))
	if err != nil {
		return nil, err
	}

	var res MetricNames
	parser.Inspect(expr, func(node parser.Node, _ []parser.Node) error {
		if n, ok := node.(*parser.VectorSelector); ok {
			if n.Name != "" {
				res = append(res, MetricName(n.Name))
				return nil
			}
			for _, m := range n.LabelMatchers {
				if m.Name == labels.MetricName && validMetricNameExpr.MatchString(m.Value) {
					res = append(res, MetricName(n.Name))
					return nil
				}
			}
		}
		return nil
	})
	return res, nil
}

func replaceVariables(query string) string {
	query = variableReplacer.Replace(query)
	query = variableRangeQueryRangeRegex.ReplaceAllLiteralString(query, `[5m]`)
	query = variableSubqueryRangeRegex.ReplaceAllLiteralString(query, `[5m:1m]`)
	return query
}

func humanizeLabelSet(labels model.LabelSet) string {
	arr := make([]string, 0, len(labels))
	for name, val := range labels {
		arr = append(arr, fmt.Sprintf("%s=%s", name, val))
	}
	return strings.Join(arr, ",")
}
