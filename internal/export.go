package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
)

type ExportConfig struct {
	Addr   string
	Output string
}

type RulesExporter struct {
	cfg   *ExportConfig
	v1api promapiv1.API
}

func NewRulesExporter(cfg *ExportConfig) (*RulesExporter, error) {
	cl, err := api.NewClient(api.Config{
		Address: cfg.Addr,
	})
	if err != nil {
		return nil, fmt.Errorf("new prom client: %w", err)
	}
	return &RulesExporter{
		cfg:   cfg,
		v1api: promapiv1.NewAPI(cl),
	}, nil
}

func (re *RulesExporter) Export(ctx context.Context) error {
	rules, err := re.v1api.Rules(ctx)
	if err != nil {
		return fmt.Errorf("get rules: %w", err)
	}
	f, err := os.Create(re.cfg.Output)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

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

func (re *RulesExporter) writeHeaders(w *csv.Writer, buf []string) error {
	buf[0], buf[1], buf[2], buf[3], buf[4] = "type", "name", "query", "evalTime", "lastEval"
	return w.Write(buf)
}

func (re *RulesExporter) writeRecordingRule(w *csv.Writer, buf []string, r promapiv1.RecordingRule) error {
	buf[0], buf[1], buf[2] = "record", r.Name, r.Query
	buf[3], buf[4] = strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	return w.Write(buf)
}

func (re *RulesExporter) writeAlertingRule(w *csv.Writer, buf []string, r promapiv1.AlertingRule) error {
	buf[0], buf[1], buf[2] = "alert", r.Name, r.Query
	buf[3], buf[4] = strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	return w.Write(buf)
}

type MetricsExporter struct {
	cfg   *ExportConfig
	v1api promapiv1.API
}

func NewMetricsExporter(cfg *ExportConfig) (*MetricsExporter, error) {
	cl, err := api.NewClient(api.Config{
		Address: cfg.Addr,
	})
	if err != nil {
		return nil, fmt.Errorf("new prom client: %w", err)
	}
	return &MetricsExporter{
		cfg:   cfg,
		v1api: promapiv1.NewAPI(cl),
	}, nil
}

func (mex *MetricsExporter) Export(ctx context.Context) error {
	var t time.Time
	metrics, _, err := mex.v1api.LabelValues(ctx, labels.MetricName, nil, t, t)
	if err != nil {
		return fmt.Errorf("get metrics: %w", err)
	}
	f, err := os.Create(mex.cfg.Output)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	const batchSize, numCol = 100, 1
	w := csv.NewWriter(f)
	buf := make([]string, numCol)
	if err = mex.writeHeaders(w, buf); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}

	var n uint16
	for _, metric := range metrics {
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
			buf[0] = string(metric)
			if err = w.Write(buf); err != nil {
				return fmt.Errorf("write: %w", err)
			}
		}
	}
	w.Flush()
	return nil
}

func (mex *MetricsExporter) writeHeaders(w *csv.Writer, buf []string) error {
	buf[0] = "name"
	return w.Write(buf)
}
