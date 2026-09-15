# Third-party code notice

This project reads and interprets Ollama's on-disk model store format, and
some of the name-parsing and manifest-resolution logic needed to do that
correctly is adapted directly from Ollama's own source code rather than
reimplemented from scratch.

## What was adapted

Source: https://github.com/ollama/ollama, package `types/model`
(MIT License, Copyright (c) Ollama — full text in [LICENSE](./LICENSE)).

The following files contain logic ported with little or no modification
from `types/model` (primarily `name.go`):

- `ollama/name.go` — `Name.String`, `Filepath`, `LogValue`, `EqualFold`,
  `DisplayShortest`, `IsValid`, `IsFullyQualified`, `ParseName`,
  `ParseNameBare`, `ParseNameFromFilepath`, `Merge`, and the `partKind`
  stringer — adapted from the corresponding functions in Ollama's
  `types/model/name.go`.
- `ollama/string_utils.go` — `isValidLen`, `isValidPart`,
  `isAlphanumericOrUnderscore`, `cutLast`, `cutPromised` — adapted from
  Ollama's `types/model/name.go`.
- `ollama/types.go` — the `Name` struct and `partKind` type — adapted from
  Ollama's `types/model/name.go`.
- `ollama/constants.go` — the `defaultHost`/`defaultNamespace`/`defaultTag`/
  `defaultProtocolScheme` values, the `MissingPart` sentinel, the `kindHost`
  / `kindNamespace` / `kindModel` / `kindTag` / `kindDigest` constants, and
  the `application/vnd.ollama.image.*` media type strings — adapted from
  Ollama's `types/model/name.go` and its manifest-handling code.

Reusing this logic verbatim (rather than reinventing Ollama's own name
grammar) was a deliberate choice: this tool only works if it parses model
names exactly the way Ollama itself does, including edge cases around
hosts, namespaces, and the `MissingPart` sentinel for malformed promised
segments. Re-deriving that independently would have meant maintaining a
second, unofficial copy of the same grammar with the constant risk of
silent drift from upstream — worse for correctness than a clearly marked
adaptation.

## License compatibility

Ollama is MIT licensed, which permits this kind of reuse provided the
original copyright notice and permission notice are preserved. Both are
included in [LICENSE](./LICENSE).
