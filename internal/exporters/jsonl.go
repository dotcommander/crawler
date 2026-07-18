package exporters

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type JSONLExporter struct {
	mu  sync.Mutex
	enc *json.Encoder
}

func NewJSONLExporter(w io.Writer) *JSONLExporter {
	return &JSONLExporter{enc: json.NewEncoder(w)}
}

func (e *JSONLExporter) WriteRecord(record PageRecord) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.enc.Encode(record); err != nil {
		return fmt.Errorf("encode jsonl record: %w", err)
	}
	return nil
}

func (e *JSONLExporter) Close() error {
	return nil
}
