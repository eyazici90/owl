package internal

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
)

type TopListerConfig struct {
	DashboardsFile string
	Limit          uint64
}

type (
	TopUsedResult struct {
		Usages    []MetricUsageInBoard
		ParseErrs []error
	}
	MetricUsageInBoard struct {
		Metric MetricName
		Used   uint32
	}
)

type TopUsedListerInGrafana struct {
	cfg *TopListerConfig
}

func NewTopUsedListerInGrafana(cfg *TopListerConfig) *TopUsedListerInGrafana {
	return &TopUsedListerInGrafana{cfg: cfg}
}

func (tl *TopUsedListerInGrafana) List(ctx context.Context) (*TopUsedResult, error) {
	f, err := os.Open(tl.cfg.DashboardsFile)
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

	metrics := make(map[MetricName]uint32)
	var silentErrs []error
OUT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			board, err := r.Read()
			if err == io.EOF {
				break OUT
			}
			if err != nil {
				return nil, fmt.Errorf("read dashboard: %w", err)
			}

			var panels []*Panel
			if err := json.Unmarshal([]byte(board[2]), &panels); err != nil {
				return nil, fmt.Errorf("unmarshal panel: %w", err)
			}
			for _, panel := range panels {
				for _, target := range panel.Targets {
					if target.Expr == "" {
						continue
					}
					ms, err := parsePromQuery(target.Expr)
					if err != nil {
						silentErrs = append(silentErrs, fmt.Errorf("parse expr: %w", err))
						continue
					}
					for _, m := range ms {
						metrics[m]++
					}
				}
			}
		}
	}

	usage := make([]MetricUsageInBoard, 0, len(metrics))
	for m, u := range metrics {
		usage = append(usage, MetricUsageInBoard{
			Metric: m,
			Used:   u,
		})
	}
	sort.Slice(usage, func(i, j int) bool {
		return usage[i].Used > usage[j].Used
	})
	return &TopUsedResult{
		Usages:    usage[:tl.cfg.Limit],
		ParseErrs: silentErrs,
	}, nil
}
