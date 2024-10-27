package internal

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/sync/errgroup"
)

type IdlerConfig struct {
	RulesFile, MetricsFile string
	Limit                  uint64
}

type RuleMissingMetrics struct {
	Rule    Rule
	Metrics MetricNames
}

type PromRulesIdler struct {
	cfg *IdlerConfig
}

func NewPromRulesIdler(cfg *IdlerConfig) *PromRulesIdler {
	return &PromRulesIdler{
		cfg: cfg,
	}
}

func (pri *PromRulesIdler) List(ctx context.Context) ([]RuleMissingMetrics, error) {
	metrics, err := realAllMetricsCSV(ctx, pri.cfg.MetricsFile)
	if err != nil {
		return nil, err
	}
	rf, err := os.Open(pri.cfg.RulesFile)
	if err != nil {
		return nil, fmt.Errorf("open rules: %w", err)
	}
	defer func() {
		_ = rf.Close()
	}()

	rr := csv.NewReader(rf)
	if _, err = rr.Read(); err != nil { // reading header
		return nil, fmt.Errorf("read header: %w", err)
	}

	var results []RuleMissingMetrics
EXIT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if pri.isOffLimit(len(results)) {
				break EXIT
			}
			rec, err := rr.Read()
			if err == io.EOF {
				break EXIT
			}
			if err != nil {
				return nil, fmt.Errorf("read rule: %w", err)
			}
			rule := Rule{
				Group: rec[0],
				Type:  rec[1],
				Name:  rec[2],
				Query: rec[3],
			}
			ms, err := parsePromQuery(rule.Query)
			if err != nil {
				return nil, fmt.Errorf("parse prom expr: %w", err)
			}
			missing, found := missingValues(metrics, ms...)
			if !found {
				continue
			}
			results = append(results, RuleMissingMetrics{
				Rule:    rule,
				Metrics: missing,
			})
		}
	}
	return results, nil
}

func (pri *PromRulesIdler) isOffLimit(n int) bool {
	return uint64(n) >= pri.cfg.Limit
}

type IdleDashboardsResult struct {
	IdleDashboards []struct {
		Board    Board
		Missings map[MetricName]struct{}
	}
	ParseErrs []error
}

type DashboardsIdlerConfig struct {
	*IdlerConfig
	DashboardsFile string
}

type DashboardsIdler struct {
	cfg *DashboardsIdlerConfig
}

func NewDashboardsIdler(cfg *DashboardsIdlerConfig) *DashboardsIdler {
	return &DashboardsIdler{
		cfg: cfg,
	}
}

func (dsi *DashboardsIdler) List(ctx context.Context) (*IdleDashboardsResult, error) {
	var (
		metrics    map[MetricName]struct{}
		rules      map[RuleName]struct{}
		silentErrs []error
	)
	eg, egctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		res, err := realAllMetricsCSV(egctx, dsi.cfg.MetricsFile)
		if err != nil {
			return err
		}
		metrics = res
		return nil
	})
	eg.Go(func() error {
		res, se, err := realAllRulesCSV(egctx, dsi.cfg.RulesFile)
		if err != nil {
			return err
		}
		rules = distinctRuleNames(res)
		silentErrs = se
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("wait eg: %w", err)
	}
	f, err := os.Open(dsi.cfg.DashboardsFile)
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

	var idleDashboards []struct {
		Board    Board
		Missings map[MetricName]struct{}
	}
OUT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if dsi.isOffLimit(len(idleDashboards)) {
				break OUT
			}
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

			missings := make(map[MetricName]struct{})
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
						if _, ok := rules[RuleName(m)]; ok {
							continue
						}
						if _, ok := metrics[m]; ok {
							continue
						}
						missings[m] = struct{}{}
					}
				}
			}
			if len(missings) > 0 {
				idleDashboards = append(idleDashboards, struct {
					Board    Board
					Missings map[MetricName]struct{}
				}{
					Board: Board{
						UID:   board[0],
						Title: board[1],
					},
					Missings: missings,
				})
			}
		}
	}
	return &IdleDashboardsResult{
		IdleDashboards: idleDashboards,
		ParseErrs:      silentErrs,
	}, nil
}

func (dsi *DashboardsIdler) isOffLimit(n int) bool {
	return uint64(n) >= dsi.cfg.Limit
}

func missingValues[T comparable](search map[T]struct{}, vals ...T) ([]T, bool) {
	var res []T
	for _, v := range vals {
		if _, ok := search[v]; !ok {
			res = append(res, v)
		}
	}
	return res, len(res) > 0
}
