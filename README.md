# ollama-inspector

A CLI tool for inspecting locally stored [Ollama](https://ollama.com) model blobs. It reads directly from the on-disk model store.

---

## What it does

Ollama stores models as a set of content-addressed blobs under a path like `/usr/share/ollama/.ollama/models/blobs/sha256-<hex>`. Each blob is either a GGUF weights file, a JSON config, a plain-text template/system prompt/license, or raw parameters. The manifest file ties them together under a human-readable name like `llama3:latest`.

`ollama-inspector` gives you three commands:

| Command   | What it does |
|-----------|-------------|
| `inspect` | Decode a model's manifest and config blob, print architecture, quantization, context length, system prompt, license, and all layer digests |
| `blobs`   | Walk the entire blobs directory, cross-reference every blob against all manifests, show size and which models reference each one |
---

## Installation

```bash
git clone https://github.com/MohamedAliBouhaouala/ollama-inspector
cd ollama-inspector
go build -o ollama-inspector .
```

Requires Go 1.25.

---

## Model store resolution

The tool searches for models in this order:

1. `$OLLAMA_MODELS` environment variable (if set)
2. `~/.ollama/models` (standard per-user install) — **only if it contains manifests**
3. `/usr/share/ollama/.ollama/models` (Linux system-wide / package install)

If your models are in a non-standard location (e.g. you ran `ollama serve` as root, or used a Docker volume), set `OLLAMA_MODELS` explicitly:

```bash
export OLLAMA_MODELS=/usr/share/ollama/.ollama/models
ollama-inspector inspect llama3
```

---

## Commands

### `inspect`

```
ollama-inspector inspect [flags] <model>
```

Resolves the model name, reads its manifest and config blob, and prints it as indented JSON.

**Flags:**

| Flag | Description |
|------|-------------|
| `-format <tmpl>` | Go template for each row, e.g. `'{{.Layers}}\t{{.ModelPath}}'` |
| `-format json` | Default argument. Prints indented JSON to stdout. |

**Examples:**

```bash
# Prints the model's manifest + config as indented JSON
ollama-inspector inspect llama3

# Pipe to jq to pull out a field
ollama-inspector inspect llama3 | jq .config.architecture

./ollama-inspector inspect llama3 --format {{.Layers}}\ {{.ModelPath}}

```

**Sample output:**

```json
{ "name": "...", "short_name": "llama3:latest", "config": {...}, "layers": [...], "total_size_bytes": 4712345678 }
```
---

### `blobs`

```
ollama-inspector blobs [flags]
```

Lists every blob in the model store, sorted by size descending, with cross-references to the models that use each blob.

**Flags:**

| Flag | Description |
|------|-------------|
| `-orphans` | Show only blobs not referenced by any manifest (safe to delete) |
| `-quiet`, `-q` | Print only each blob's absolute path, one per line — for piping into `xargs`, `rm`, `du`, etc. |
| `-format <tmpl>` | Go template for each row, e.g. `'{{.Digest}}\t{{.Size}}'`; prefix with `table ` for aligned columns (the default). Pass `-format json` for JSON output. |

**Examples:**

```bash
# List all blobs (default table output)
ollama-inspector blobs

# Find unreferenced blobs you can clean up
ollama-inspector blobs -orphans

# Delete all orphaned blobs
ollama-inspector blobs -orphans -q | xargs rm

# JSON output for scripting
ollama-inspector blobs -format json

# Custom template — just digest and size, tab-separated
ollama-inspector blobs -format '{{.Digest}}\t{{.Size}}'

# Aligned custom table with a header
ollama-inspector blobs -format 'table {{.Digest}}\t{{.RefBy}}'
```

**Sample output:**

```
Blobs directory: /usr/share/ollama/.ollama/models/blobs

DIGEST                SIZE    TYPE      REFERENCED BY            FILE
sha256:6a0746a1…      4.7 GB  model     llama3:latest            sha256-6a0746a1…
sha256:dde5aa3c…      4.1 GB  model     mistral:7b               sha256-dde5aa3c…
sha256:8ab4849b…      1.4 KB  template  llama3:latest, mistral:7b  sha256-8ab4849b…
sha256:fa304d67…      7.0 KB  license   llama3:latest            sha256-fa304d67…

4 blob(s) — total 8.8 GB
```

---

## Model name formats

All commands accept model names in the same formats `ollama` itself uses:

```
llama3                                          # short: defaults applied
llama3:latest                                   # with tag
mistral:7b
myns/mymodel:v1                                 # custom namespace
registry.ollama.ai/library/llama3:latest        # fully qualified
```

---

## Known limitations

- **No daemon required, but also no daemon features** — this tool reads the raw store; it doesn't pull, push, or run models.

---

## Acknowledgments

The model name parsing/validation logic in `ollama/name.go`,
`ollama/string_utils.go`, and parts of `ollama/types.go` and
`ollama/constants.go` is adapted from Ollama's own `types/model` package
(https://github.com/ollama/ollama), MIT licensed. This was a deliberate
choice — this tool only works if it parses names exactly the way Ollama
itself does, so reusing the upstream grammar directly is more correct than
reimplementing it independently. See [NOTICE.md](./NOTICE.md) for the
file-by-file breakdown and [LICENSE](./LICENSE) for the full license texts.

---
