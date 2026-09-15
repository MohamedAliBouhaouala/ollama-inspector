package ollama

import (
	"fmt"
	"io"
)

// NewModelFromManifest assembles a Model from a Manifest, reading the config
// blob and each text/system/template layer from disk.
//
// Blob read errors for individual layers are non-fatal: the relevant field is
// left empty and the error is collected in the returned slice so callers can
// decide whether to surface them.
func NewModelFromManifest(n Name, m *Manifest) (*Model, []error) {
	cfg, err := m.ReadConfig()
	var errs []error
	if err != nil {
		errs = append(errs, fmt.Errorf("reading config: %w", err))
		cfg = &ConfigV2{} // proceed with zero config rather than aborting
	}

	model := &Model{
		Name:      n.String(),
		ShortName: n.DisplayShortest(),
		Digest:    m.Digest(),
		Config:    *cfg,
	}

	for _, l := range m.Layers {
		switch l.MediaType {
		case MediaTypeImageModel:
			model.ModelPath, _ = l.BlobPath()
		case MediaTypeAdapter:
			if p, e := l.BlobPath(); e == nil {
				model.AdapterPaths = append(model.AdapterPaths, p)
			}
		case MediaTypeProjector:
			if p, e := l.BlobPath(); e == nil {
				model.ProjectorPaths = append(model.ProjectorPaths, p)
			}
		case MediaTypeImageSystem, MediaTypeLicense, MediaTypeImageParams:
			rc, err := l.Open()
			if err != nil {
				errs = append(errs, fmt.Errorf("opening layer %s (%s): %w", l.Digest, l.MediaType, err))
				continue
			}
			data, readErr := io.ReadAll(rc)
			if closeErr := rc.Close(); closeErr != nil {
				errs = append(errs,
					fmt.Errorf("closing layer %s: %w", l.Digest, closeErr))
			}

			if readErr != nil {
				errs = append(errs, fmt.Errorf("reading layer %s: %w", l.Digest, readErr))
				continue
			}

			switch l.MediaType {
			case MediaTypeImageSystem:
				model.System = string(data)
			case MediaTypeLicense:
				model.License = append(model.License, string(data))

			case MediaTypeImageParams:
				model.Params = string(data)
			}
		}
	}

	return model, errs
}
