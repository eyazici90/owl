package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/prometheus/common/model"
)

type (
	MetricNames []MetricName
	MetricName  string
)

func csvMetricsWriteAll(ctx context.Context, file string, metrics model.LabelValues) error {
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
