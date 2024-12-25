package internal

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type (
	// Board represents Grafana dashboard.
	Board struct {
		ID     uint     `mapstructure:"id,omitempty"`
		UID    string   `mapstructure:"uid,omitempty"`
		Title  string   `mapstructure:"title"`
		Tags   []string `mapstructure:"tags"`
		Panels []*Panel `mapstructure:"panels"`
	}
	Panel struct {
		ID      uint      `mapstructure:"id"`
		OfType  panelType `mapstructure:"-"`     // it required for defining type of the panel
		Title   string    `mapstructure:"title"` // general
		Type    string    `mapstructure:"type"`
		Targets []Target  `mapstructure:"targets,omitempty"`
	}
	Target struct {
		Datasource any    `mapstructure:"datasource,omitempty"`
		Expr       string `mapstructure:"expr,omitempty"`
	}
	panelType int8
)

type colBoard uint8

const (
	colBoardUID colBoard = iota
	colBoardTitle
	colBoardPanels
	colBoardNum
)

func writeAllBoardsCSV(ctx context.Context, file string, boards []*Board) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	const batchSize, numCol = 100, colBoardNum
	wr := &csvBatchWriter{
		size: batchSize,
		buf:  make([]string, numCol),
		w:    csv.NewWriter(f),
	}
	err = wr.Write(ctx, func(buf []string) {
		buf[colBoardUID], buf[colBoardTitle], buf[colBoardPanels] = "uid", "title", "panels"
	})
	if err != nil {
		return fmt.Errorf("write headers: %w", err)
	}
	for _, board := range boards {
		jsn, err := json.Marshal(board.Panels)
		if err != nil {
			return fmt.Errorf("marshal panels: %w", err)
		}
		err = wr.Write(ctx, func(buf []string) {
			buf[colBoardUID] = board.UID
			buf[colBoardTitle] = board.Title
			buf[colBoardPanels] = string(jsn)
		})
	}
	wr.Flush()
	return nil
}

func readAllBoardsCSV(ctx context.Context, file string) ([]*Board, []error, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, fmt.Errorf("open boards: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	r := csv.NewReader(f)
	if _, err = r.Read(); err != nil { // reading header
		return nil, nil, fmt.Errorf("read header: %w", err)
	}

	var (
		boards     []*Board
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

			var panels []*Panel
			if err := json.Unmarshal([]byte(rec[2]), &panels); err != nil {
				return nil, nil, fmt.Errorf("marshal panels: %w", err)
			}
			boards = append(boards, &Board{
				UID:    rec[colBoardUID],
				Title:  rec[colBoardTitle],
				Panels: panels,
			})
		}
	}
	return boards, silentErrs, nil
}
