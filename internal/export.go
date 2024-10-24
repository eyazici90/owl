package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
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
	return &RulesExporter{
		cfg:   cfg,
		v1api: mustNewPromAPIV1(cfg.Addr),
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

	const batchSize, numCol = 100, 7
	w := csv.NewWriter(f)
	buf := make([]string, numCol)
	if err = re.writeHeaders(w, buf); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}

	var n uint16
	for _, group := range rules.Groups {
		buf[0] = group.Name
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

func (*RulesExporter) writeHeaders(w *csv.Writer, buf []string) error {
	buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6] = "group", "type", "name", "query", "labels", "evalTime", "lastEval"
	return w.Write(buf)
}

func (*RulesExporter) writeRecordingRule(w *csv.Writer, buf []string, r promapiv1.RecordingRule) error {
	buf[1], buf[2], buf[3] = "record", r.Name, r.Query
	buf[4], buf[5], buf[6] = humanizeLabelSet(r.Labels), strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	return w.Write(buf)
}

func (*RulesExporter) writeAlertingRule(w *csv.Writer, buf []string, r promapiv1.AlertingRule) error {
	buf[1], buf[2], buf[3] = "alert", r.Name, r.Query
	buf[4], buf[5], buf[6] = humanizeLabelSet(r.Labels), strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	return w.Write(buf)
}

func humanizeLabelSet(labels model.LabelSet) string {
	arr := make([]string, 0, len(labels))
	for name, val := range labels {
		arr = append(arr, fmt.Sprintf("%s=%s", name, val))
	}
	return strings.Join(arr, ",")
}

type MetricsExporter struct {
	cfg   *ExportConfig
	v1api promapiv1.API
}

func NewMetricsExporter(cfg *ExportConfig) (*MetricsExporter, error) {
	return &MetricsExporter{
		cfg:   cfg,
		v1api: mustNewPromAPIV1(cfg.Addr),
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
