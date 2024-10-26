package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/prometheus/common/model"
)

type (
	MetricNames []MetricName
	MetricName  string
)

func writeAllMetricsCSV(ctx context.Context, file string, metrics model.LabelValues) error {
	f, err := os.Create(file)
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
	err = wr.Write(ctx, func(buf []string) {
		buf[0] = "name"
	})
	if err != nil {
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

func realAllMetricsCSV(ctx context.Context, file string) (map[MetricName]struct{}, error) {
	mf, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open metrics: %w", err)
	}
	defer func() {
		_ = mf.Close()
	}()

	metrics := make(map[MetricName]struct{})
	mr := csv.NewReader(mf)
	if _, err = mr.Read(); err != nil { // reading header
		return nil, fmt.Errorf("read header: %w", err)
	}

OUT:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			rec, err := mr.Read()
			if err == io.EOF {
				break OUT
			}
			if err != nil {
				return nil, fmt.Errorf("read metric: %w", err)
			}
			metrics[MetricName(rec[0])] = struct{}{}
		}
	}
	return metrics, nil
}
