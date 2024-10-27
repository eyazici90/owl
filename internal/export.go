package internal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
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
	return writeAllRulesCSV(ctx, re.cfg.Output, rules)
}

type MetricsExporterConfig struct {
	*ExportConfig
	Since string
}

type MetricsExporter struct {
	cfg   *MetricsExporterConfig
	v1api promapiv1.API
}

func NewMetricsExporter(cfg *MetricsExporterConfig) (*MetricsExporter, error) {
	return &MetricsExporter{
		cfg:   cfg,
		v1api: mustNewPromAPIV1(cfg.Addr),
	}, nil
}

func (mex *MetricsExporter) Export(ctx context.Context) error {
	since, err := time.ParseDuration(mex.cfg.Since)
	if err != nil {
		return fmt.Errorf("parse dur: %w", err)
	}

	start, end := time.Now().Add(-1*since), time.Now()
	metrics, _, err := mex.v1api.LabelValues(ctx, labels.MetricName, nil, start, end)
	if err != nil {
		return fmt.Errorf("get metrics: %w", err)
	}
	return writeAllMetricsCSV(ctx, mex.cfg.Output, metrics)
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
	return writeAllBoardsCSV(ctx, dex.cfg.Output, boards)
}
