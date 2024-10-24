package internal

import (
	"context"
	"fmt"

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

func (e *RuleExporter) Export(ctx context.Context) error {
	return nil
}
