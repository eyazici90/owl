package internal

import (
	"context"
	"encoding/csv"
	"fmt"
)

type csvBatchWriter struct {
	size uint64
	c    uint64
	buf  []string
	w    *csv.Writer
}

func (wr *csvBatchWriter) Write(ctx context.Context, fn func(buf []string)) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if wr.c >= wr.size {
			wr.Flush()
			if err := wr.w.Error(); err != nil {
				return fmt.Errorf("flush csv: %w", err)
			}
			wr.c = 0
		}
		fn(wr.buf)
		if err := wr.w.Write(wr.buf); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		wr.c++
	}
	return nil
}

func (wr *csvBatchWriter) Flush() {
	wr.w.Flush()
}
