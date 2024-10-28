package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type RuleName string

type Rule struct {
	Group, Type, Name, Query, Labels string
	EvalDuration                     float64
}

func writeAllRulesCSV(ctx context.Context, file string, rules promapiv1.RulesResult) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	const batchSize, numCol = 100, 7
	wr := &csvBatchWriter{
		size: batchSize,
		buf:  make([]string, numCol),
		w:    csv.NewWriter(f),
	}
	err = wr.Write(ctx, func(buf []string) {
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5], buf[6] = "group", "type", "name", "query", "labels", "evalTime", "lastEval"
	})
	if err != nil {
		return fmt.Errorf("write headers: %w", err)
	}
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			switch r := rule.(type) {
			case promapiv1.RecordingRule:
				err = wr.Write(ctx, func(buf []string) {
					buf[0] = group.Name
					buf[1], buf[2], buf[3] = "record", r.Name, r.Query
					buf[4], buf[5], buf[6] = humanizeLabelSet(r.Labels), strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
				})
				if err != nil {
					return fmt.Errorf("write rule: %w", err)
				}
			case promapiv1.AlertingRule:
				err = wr.Write(ctx, func(buf []string) {
					buf[0] = group.Name
					buf[1], buf[2], buf[3] = "alert", r.Name, r.Query
					buf[4], buf[5], buf[6] = humanizeLabelSet(r.Labels), strconv.FormatFloat(r.EvaluationTime, 'g', -1, 64), r.LastEvaluation.String()
				})
				if err != nil {
					return fmt.Errorf("write rule: %w", err)
				}
			default:
			}
		}
	}
	wr.Flush()
	return nil
}

func readAllRulesCSV(ctx context.Context, file string) ([]Rule, []error, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, fmt.Errorf("open rules: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	r := csv.NewReader(f)
	if _, err = r.Read(); err != nil { // reading header
		return nil, nil, fmt.Errorf("read header: %w", err)
	}

	var (
		rules      []Rule
		silentErrs []error
	)
OUT:
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
			rec, err := r.Read()
			if err == io.EOF {
				break OUT
			}
			if err != nil {
				silentErrs = append(silentErrs, fmt.Errorf("read rule: %w", err))
				continue
			}
			dur, err := strconv.ParseFloat(rec[5], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("parse eval-duration: %w", err)
			}
			rules = append(rules, Rule{
				Group:        rec[0],
				Type:         rec[1],
				Name:         rec[2],
				Query:        rec[3],
				Labels:       rec[4],
				EvalDuration: dur,
			})
		}
	}
	return rules, silentErrs, nil
}

func distinctRuleNames(rules []Rule) map[RuleName]struct{} {
	m := make(map[RuleName]struct{}, len(rules))
	for _, rule := range rules {
		m[RuleName(rule.Name)] = struct{}{}
	}
	return m
}
