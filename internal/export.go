package internal

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
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
	wr := &csvBatchWriter{
		size: batchSize,
		buf:  make([]string, numCol),
		w:    csv.NewWriter(f),
	}
	if err = re.writeHeaders(ctx, wr); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			switch r := rule.(type) {
			case promapiv1.RecordingRule:
				if err = re.writeRecordingRule(ctx, wr, r, group.Name); err != nil {
					return fmt.Errorf("write recording rule: %w", err)
				}
			case promapiv1.AlertingRule:
				if err = re.writeAlertingRule(ctx, wr, r, group.Name); err != nil {
					return fmt.Errorf("write alerting rule: %w", err)
				}
			default:
			}
		}
	}
	w.Flush()
	return nil
}

func (*RulesExporter) writeHeaders(ctx context.Context, wr *csvBatchWriter) error {
	return wr.Write(ctx, func(buf []string) {
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6] = "group", "type", "name", "query", "labels", "evalTime", "lastEval"
	})
}

func (*RulesExporter) writeRecordingRule(ctx context.Context, wr *csvBatchWriter, r promapiv1.RecordingRule, grp string) error {
	return wr.Write(ctx, func(buf []string) {
		buf[0] = grp
		buf[1], buf[2], buf[3] = "record", r.Name, r.Query
		buf[4], buf[5], buf[6] = humanizeLabelSet(r.Labels), strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	})
}

func (*RulesExporter) writeAlertingRule(ctx context.Context, wr *csvBatchWriter, r promapiv1.AlertingRule, grp string) error {
	return wr.Write(ctx, func(buf []string) {
		buf[0] = grp
		buf[1], buf[2], buf[3] = "alert", r.Name, r.Query
		buf[4], buf[5], buf[6] = humanizeLabelSet(r.Labels), strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
	})
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
	wr := &csvBatchWriter{
		size: batchSize,
		buf:  make([]string, numCol),
		w:    csv.NewWriter(f),
	}
	if err = mex.writeHeaders(ctx, wr); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}
	for _, metric := range metrics {
		err = wr.Write(ctx, func(buf []string) {
			buf[0] = string(metric)
		})
		if err != nil {
			return err
		}
	}
	wr.Flush()
	return nil
}

func (mex *MetricsExporter) writeHeaders(ctx context.Context, wr *csvBatchWriter) error {
	return wr.Write(ctx, func(buf []string) {
		buf[0] = "name"
	})
}

type DashboardsExportConfig struct {
	*ExportConfig
	SvcToken string
}

type DashboardsExporter struct {
	cfg     *DashboardsExportConfig
	grafana *goapi.GrafanaHTTPAPI
}

func NewDashboardsExporter(cfg *DashboardsExportConfig) (*DashboardsExporter, error) {
	return &DashboardsExporter{
		cfg: cfg,
		grafana: newGrafanaOAPI(&GrafanaConfig{
			URL:    cfg.Addr,
			Scheme: "https",
			APIKey: cfg.SvcToken,
		}),
	}, nil
}

func (dex *DashboardsExporter) Export(ctx context.Context) error {
	boardIDs, err := getAllDashboards(ctx, dex.grafana)
	if err != nil {
		return fmt.Errorf("get all dashboards: %w", err)
	}

	c := len(boardIDs)
	slog.Info("Found dashboards", slog.Int("count", c))
	boards := make([]*Board, 0, c)
	for _, uid := range boardIDs {
		slog.Debug("Fetching board", slog.String("uid", uid))
		db, err := getDashboardByUID(ctx, dex.grafana, uid)
		if err != nil {
			return fmt.Errorf("get board: %w", err)
		}
		boards = append(boards, db)
	}

	f, err := os.Create(dex.cfg.Output)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	const batchSize, numCol = 100, 3
	wr := &csvBatchWriter{
		size: batchSize,
		buf:  make([]string, numCol),
		w:    csv.NewWriter(f),
	}
	if err = dex.writeHeaders(ctx, wr); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}
	for _, board := range boards {
		jsn, err := json.Marshal(board.Panels)
		if err != nil {
			return fmt.Errorf("marshal panels: %w", err)
		}
		err = wr.Write(ctx, func(buf []string) {
			buf[0] = board.UID
			buf[1] = board.Title
			buf[2] = string(jsn)
		})
	}
	wr.Flush()
	return nil
}

func (dex *DashboardsExporter) writeHeaders(ctx context.Context, wr *csvBatchWriter) error {
	return wr.Write(ctx, func(buf []string) {
		buf[0], buf[1], buf[2] = "uid", "title", "panels"
	})
}
