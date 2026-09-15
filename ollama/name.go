package ollama

import (
	"cmp"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

func (n Name) String() string {
	var b strings.Builder
	if n.Host != "" {
		b.WriteString(n.Host)
		b.WriteByte('/')
	}
	if n.Namespace != "" {
		b.WriteString(n.Namespace)
		b.WriteByte('/')
	}
	b.WriteString(n.Model)
	if n.Tag != "" {
		b.WriteByte(':')
		b.WriteString(n.Tag)
	}
	return b.String()
}

func (n Name) Filepath() string {
	if !n.IsFullyQualified() {
		panic("illegal attempt to get filepath of invalid name")
	}
	return filepath.Join(n.Host, n.Namespace, n.Model, n.Tag)
}

func (n Name) LogValue() slog.Value {
	return slog.StringValue(n.String())
}

func (n Name) EqualFold(o Name) bool {
	return strings.EqualFold(n.Host, o.Host) &&
		strings.EqualFold(n.Namespace, o.Namespace) &&
		strings.EqualFold(n.Model, o.Model) &&
		strings.EqualFold(n.Tag, o.Tag)
}

// DisplayShortest returns the shortest unambiguous string representation.
func (n Name) DisplayShortest() string {
	var sb strings.Builder

	if !strings.EqualFold(n.Host, defaultHost) {
		sb.WriteString(n.Host)
		sb.WriteByte('/')
		sb.WriteString(n.Namespace)
		sb.WriteByte('/')
	} else if !strings.EqualFold(n.Namespace, defaultNamespace) {
		sb.WriteString(n.Namespace)
		sb.WriteByte('/')
	}

	sb.WriteString(n.Model)
	sb.WriteString(":")
	sb.WriteString(n.Tag)
	return sb.String()
}

func (n Name) IsValid() bool {
	return n.IsFullyQualified()
}

func (n Name) IsFullyQualified() bool {
	parts := []string{n.Host, n.Namespace, n.Model, n.Tag}
	for i, part := range parts {
		if !isValidPart(partKind(i), part) {
			return false
		}
	}
	return true
}

func Unqualified(n Name) error {
	return fmt.Errorf("%w: %s", ErrUnqualifiedName, n)
}

func DefaultName() Name {
	return Name{
		Host:           defaultHost,
		Namespace:      defaultNamespace,
		Tag:            defaultTag,
		ProtocolScheme: defaultProtocolScheme,
	}
}

// ParseName parses a name string and fills missing parts from DefaultName.
func ParseName(s string) Name {
	return Merge(ParseNameBare(s), DefaultName())
}

// ParseNameBare parses s as a name string with no default filling.
func ParseNameBare(s string) Name {
	var n Name
	var promised bool

	if strings.LastIndex(s, ":") > strings.LastIndex(s, "/") {
		s, n.Tag, _ = cutPromised(s, ":")
	}

	s, n.Model, promised = cutPromised(s, "/")
	if !promised {
		n.Model = s
		return n
	}

	s, n.Namespace, promised = cutPromised(s, "/")
	if !promised {
		n.Namespace = s
		return n
	}

	scheme, host, ok := strings.Cut(s, "://")
	if ok {
		n.ProtocolScheme = scheme
	} else {
		host = scheme
	}
	n.Host = host

	return n
}

// ParseNameFromFilepath parses a 4-part filepath as a Name.
func ParseNameFromFilepath(s string) (n Name) {
	parts := strings.Split(s, string(filepath.Separator))
	if len(parts) != 4 {
		return Name{}
	}

	n.Host = parts[0]
	n.Namespace = parts[1]
	n.Model = parts[2]
	n.Tag = parts[3]
	if !n.IsFullyQualified() {
		return Name{}
	}

	return n
}

// Merge merges two Names, preferring non-empty fields from a.
func Merge(a, b Name) Name {
	a.Host = cmp.Or(a.Host, b.Host)
	a.Namespace = cmp.Or(a.Namespace, b.Namespace)
	a.Tag = cmp.Or(a.Tag, b.Tag)
	a.ProtocolScheme = cmp.Or(a.ProtocolScheme, b.ProtocolScheme)
	return a
}

func (k partKind) String() string {
	switch k {
	case kindHost:
		return "host"
	case kindNamespace:
		return "namespace"
	case kindModel:
		return "model"
	case kindTag:
		return "tag"
	case kindDigest:
		return "digest"
	default:
		return "unknown"
	}
}
