package internal

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/sync/errgroup"
)

type IdlerConfig struct {
	RulesFile, MetricsFile, DashboardsFile string
	Limit                                  uint64
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
	metrics, err := readAllMetricsCSV(ctx, pri.cfg.MetricsFile)
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

type (
	IdleDashboardsResult struct {
		IdleDashboards []IdleDashboard
		ParseErrs      []error
	}
	IdleDashboard struct {
		Board    Board
		Missings map[MetricName]struct{}
	}
)

type DashboardsIdler struct {
	cfg *IdlerConfig
}

func NewDashboardsIdler(cfg *IdlerConfig) *DashboardsIdler {
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
		res, err := readAllMetricsCSV(egctx, dsi.cfg.MetricsFile)
		if err != nil {
			return err
		}
		metrics = res
		return nil
	})
	eg.Go(func() error {
		res, se, err := readAllRulesCSV(egctx, dsi.cfg.RulesFile)
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

	var idles []IdleDashboard
OUT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if dsi.isOffLimit(len(idles)) {
				break OUT
			}
			board, err := r.Read()
			if err == io.EOF {
				break OUT
			}
			if err != nil {
				return nil, fmt.Errorf("read dashboard: %w", err)
			}
			missings, se, err := dsi.scanDashboard(board, rules, metrics)
			if err != nil {
				return nil, fmt.Errorf("scan dashboard: %w", err)
			}
			silentErrs = append(silentErrs, se...)
			if len(missings) > 0 {
				idles = append(idles, IdleDashboard{
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
		IdleDashboards: idles,
		ParseErrs:      silentErrs,
	}, nil
}

func (dsi *DashboardsIdler) scanDashboard(
	board []string,
	rules map[RuleName]struct{},
	metrics map[MetricName]struct{},
) (map[MetricName]struct{}, []error, error) {
	var panels []*Panel
	if err := json.Unmarshal([]byte(board[2]), &panels); err != nil {
		return nil, nil, fmt.Errorf("unmarshal panel: %w", err)
	}

	var silentErrs []error
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
	return missings, silentErrs, nil
}

func (dsi *DashboardsIdler) isOffLimit(n int) bool {
	return uint64(n) >= dsi.cfg.Limit
}

type IdleMetricsResult struct {
	IdleMetrics []MetricName
	ParseErrs   []error
}

type MetricsIdler struct {
	cfg *IdlerConfig
}

func NewMetricsIdler(cfg *IdlerConfig) *MetricsIdler {
	return &MetricsIdler{
		cfg: cfg,
	}
}

func (mi *MetricsIdler) List(ctx context.Context) (*IdleMetricsResult, error) {
	var (
		metrics map[MetricName]struct{}
		rules   []Rule
		boards  []*Board

		mu         sync.RWMutex
		silentErrs []error
	)
	eg, egctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		res, se, err := readAllBoardsCSV(egctx, mi.cfg.DashboardsFile)
		if err != nil {
			return err
		}
		boards = res
		mu.Lock()
		silentErrs = se
		mu.Unlock()
		return nil
	})
	eg.Go(func() error {
		res, se, err := readAllRulesCSV(egctx, mi.cfg.RulesFile)
		if err != nil {
			return err
		}
		rules = res
		mu.Lock()
		silentErrs = se
		mu.Unlock()
		return nil
	})
	eg.Go(func() error {
		res, err := readAllMetricsCSV(egctx, mi.cfg.MetricsFile)
		if err != nil {
			return err
		}
		metrics = res
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("wait eg: %w", err)
	}

	used, se := mi.usedMetricsFrom(boards, rules)
	if len(se) > 0 {
		silentErrs = append(silentErrs, se...)
	}

	var idles []MetricName
	for m, _ := range metrics {
		if mi.isOffLimit(len(idles)) {
			break
		}
		if _, ok := used[m]; !ok {
			idles = append(idles, m)
		}
	}
	return &IdleMetricsResult{
		IdleMetrics: idles,
		ParseErrs:   silentErrs,
	}, nil
}

func (mi *MetricsIdler) usedMetricsFrom(boards []*Board, rules []Rule) (map[MetricName]struct{}, []error) {
	metrics := make(map[MetricName]struct{})
	var silentErrs []error
	for _, rule := range rules {
		ms, err := parsePromQuery(rule.Query)
		if err != nil {
			silentErrs = append(silentErrs, fmt.Errorf("parse expr: %w", err))
			continue
		}
		for _, m := range ms {
			metrics[m] = struct{}{}
		}
	}
	for _, board := range boards {
		for _, panel := range board.Panels {
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
					metrics[m] = struct{}{}
				}
			}
		}
	}
	return metrics, silentErrs
}

func (mi *MetricsIdler) isOffLimit(n int) bool {
	return uint64(n) >= mi.cfg.Limit
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
