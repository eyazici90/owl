package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
)

type TopListerConfig struct {
	DashboardFile string
	Limit         uint64
}

type MetricsUsageInBoard struct {
	Metric MetricName
	Used   uint32
}

type TopMetricsLister struct {
	cfg *TopListerConfig
}

func NewTopMetricsLister(cfg *TopListerConfig) *TopMetricsLister {
	return &TopMetricsLister{cfg: cfg}
}

func (tl *TopMetricsLister) List(ctx context.Context) ([]MetricsUsageInBoard, error) {
	f, err := os.Open(tl.cfg.DashboardFile)
	if err != nil {
		return nil, fmt.Errorf("open dashboards: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	r := csv.NewReader(f)
	if _, err = r.Read(); err != nil { // reading header
		return nil, fmt.Errorf("read header: %w", err)
	}

	return nil, nil
}
