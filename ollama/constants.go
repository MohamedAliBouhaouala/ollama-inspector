package ollama

import (
	"errors"
	"regexp"
)

var digestPattern = regexp.MustCompile(`^sha256[:-][0-9a-fA-F]{64}$`)

var ErrModelsNotFound = errors.New("models directory not found")

var ErrInvalidDigestFormat = errors.New("invalid digest format")

var ErrUnqualifiedName = errors.New("unqualified name")

const (
	MediaTypeImageTensor = "application/vnd.ollama.image.tensor"
	MediaTypeImageModel  = "application/vnd.ollama.image.model"
	MediaTypeImageParams = "application/vnd.ollama.image.params"
	MediaTypeImageSystem = "application/vnd.ollama.image.system"
	MediaTypeImageConfig = "application/vnd.ollama.image.config"
	MediaTypeImageJSON   = "application/vnd.ollama.image.json"
	MediaTypeTemplate    = "application/vnd.ollama.image.template"
	MediaTypeLicense     = "application/vnd.ollama.image.license"
	MediaTypeAdapter     = "application/vnd.ollama.image.adapter"
	MediaTypeProjector   = "application/vnd.ollama.image.projector"
)

const MissingPart = "!MISSING!"

const (
	defaultHost           = "registry.ollama.ai"
	defaultNamespace      = "library"
	defaultTag            = "latest"
	defaultProtocolScheme = "https"
)

const (
	kindHost partKind = iota
	kindNamespace
	kindModel
	kindTag
	kindDigest
)
