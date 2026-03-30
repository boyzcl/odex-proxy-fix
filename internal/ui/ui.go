package ui

import (
	"encoding/json"
	"fmt"
	"io"
)

func PrintJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func Line(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format+"\n", args...)
}
