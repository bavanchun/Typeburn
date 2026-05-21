package output

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// RenderTable writes a plain tab-aligned table with no ANSI styling.
func RenderTable(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err := writeRow(tw, headers); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeRow(tw, row); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeRow(w io.Writer, row []string) error {
	for i, cell := range row {
		if i > 0 {
			if _, err := fmt.Fprint(w, "\t"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, cell); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}
