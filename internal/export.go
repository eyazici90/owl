package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type RuleExportConfig struct {
	Addr   string
	Output string
}

type RuleExporter struct {
	cfg   *RuleExportConfig
	v1api promapiv1.API
}

func NewRuleExporter(cfg *RuleExportConfig) (*RuleExporter, error) {
	cl, err := api.NewClient(api.Config{
		Address: cfg.Addr,
	})
	if err != nil {
		return nil, fmt.Errorf("new prom client: %w", err)
	}
	return &RuleExporter{
		cfg:   cfg,
		v1api: promapiv1.NewAPI(cl),
	}, nil
}

func (re *RuleExporter) Export(ctx context.Context) error {
	rules, err := re.v1api.Rules(ctx)
	if err != nil {
		return fmt.Errorf("get rules: %w", err)
	}
	f, err := os.Create(re.cfg.Output)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	const batchSize, numCol = 100, 5
	w := csv.NewWriter(f)
	buf := make([]string, numCol)
	if err = re.writeHeaders(w, buf); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}

	var n uint16
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if n >= batchSize {
					w.Flush()
					if err = w.Error(); err != nil {
						return fmt.Errorf("flush csv: %w", err)
					}
					n = 0
				}
				n++
				switch r := rule.(type) {
				case promapiv1.RecordingRule:
					if err = re.writeRecordingRule(w, buf, r); err != nil {
						return fmt.Errorf("write recording rule: %w", err)
					}
				case promapiv1.AlertingRule:
					if err = re.writeAlertingRule(w, buf, r); err != nil {
						return fmt.Errorf("write alerting rule: %w", err)
					}
				default:
				}
			}
		}
	}
	w.Flush()
	return nil
}

func (re *RuleExporter) writeHeaders(w *csv.Writer, buf []string) error {
	buf[0], buf[1], buf[2], buf[3], buf[4] = "type", "name", "query", "evalTime", "lastEval"
	return w.Write(buf)
}

func (re *RuleExporter) writeRecordingRule(w *csv.Writer, buf []string, r promapiv1.RecordingRule) error {
	buf[0], buf[1], buf[2] = "record", r.Name, r.Query
	buf[3], buf[4] = strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	return w.Write(buf)
}

func (re *RuleExporter) writeAlertingRule(w *csv.Writer, buf []string, r promapiv1.AlertingRule) error {
	buf[0], buf[1], buf[2] = "alert", r.Name, r.Query
	buf[3], buf[4] = strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	return w.Write(buf)
}
