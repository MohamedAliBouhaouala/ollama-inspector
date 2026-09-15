package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"ollama-inspector/ollama"
)

type Report struct {
	Name   string          `json:"name"`
	Short  string          `json:"short_name"`
	Digest string          `json:"manifest_digest"`
	Config ollama.ConfigV2 `json:"config"`
	// Params is embedded as raw JSON (not re-escaped into a string) so
	// consumers piping -json output through jq can query it directly,
	// e.g. `ollama-inspector inspect llama3 -json | jq .params.temperature`.
	Params    json.RawMessage `json:"params,omitempty"`
	System    string          `json:"system_prompt,omitempty"`
	License   []string        `json:"license,omitempty"`
	ModelPath string          `json:"model_path,omitempty"`
	Layers    []ollama.Layer  `json:"layers"`
	TotalSize int64           `json:"total_size_bytes"`
}

// JSON output
func FormatJSON(model *ollama.Model, m *ollama.Manifest) error {
	report := Report{
		Name:      model.Name,
		Short:     model.ShortName,
		Digest:    model.Digest,
		Config:    model.Config,
		System:    model.System,
		License:   model.License,
		ModelPath: model.ModelPath,
		Layers:    m.Layers,
		TotalSize: m.Size(),
	}

	if model.Params != "" {
		// Only embed as raw JSON if it's actually valid; otherwise fall
		// back to a quoted string so a malformed blob doesn't produce
		// invalid JSON output for the whole report.
		if json.Valid([]byte(model.Params)) {
			report.Params = json.RawMessage(model.Params)
		} else {
			raw, err := json.Marshal(model.Params)
			if err != nil {
				return fmt.Errorf("encoding params: %w", err)
			}
			report.Params = raw
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)

}
