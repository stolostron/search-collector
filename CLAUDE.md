# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Purpose

`search-collector` runs as an addon on each managed cluster. It watches all Kubernetes resources via dynamic informers, transforms them into a flat property map (a `Node`), computes relationships between resources (edges), and syncs the resulting diff to the `search-indexer` over HTTPS.

## Commands

```bash
make run        # go run -tags development main.go --v=2
make test       # go test ./... -failfast  (sets DEPLOYED_IN_HUB=true)
make coverage   # test + open HTML coverage report
make lint       # golangci-lint + gosec
make build      # CGO_ENABLED=1 go build -o output/search-collector
```

Run a single test package:
```bash
go test ./pkg/transforms/... -failfast -run TestFoo
```

## Architecture

The pipeline is linear: **Informer → Transformer → Reconciler → Sender**

```
K8s API watch → informer.GenericInformer
                    ↓ Event{Type, Node}
               transforms.Transformer   (fan-out to numCPU goroutines)
                    ↓ NodeEvent
               reconciler.Reconciler    (in-memory state: all nodes + edges)
                    ↓ Diff{add/update/delete nodes+edges}
               send.Sender              (POST JSON to search-indexer /aggregator/clusters/<name>/sync)
```

### Key packages

- **`pkg/informer`** — `GenericInformer` wraps the dynamic client with its own watch/re-list loop (does not use `client-go` SharedInformerFactory). `RunInformers` discovers all GVRs via the discovery API, starts one informer per resource type, and watches for new CRDs to start additional informers dynamically.

- **`pkg/transforms`** — One file per resource kind (e.g. `pod.go`, `deployment.go`). Each implements `BuildNode()` to extract searchable properties and `BuildEdges()` to compute relationships. `common.go` handles properties and edges that apply to every resource (owner references, hosting annotations). Properties prefixed with `_` are internal and not user-searchable. `configurableCollection.go` merges the `search-collector-config` ConfigMap to allow/deny-list specific resources.

- **`pkg/reconciler`** — Holds the full in-memory state of all nodes and edges seen so far. On each cycle it computes the diff (add/update/delete) against the previous state. Uses an LRU cache to handle out-of-order delete/add sequences.

- **`pkg/send`** — Serializes the `Diff` into the `Payload` struct and POSTs it to the indexer. Implements exponential backoff on failure. On the first successful sync after a failure or startup, sends `ClearAll: true` to let the indexer reset state for this cluster.

### Data model

A `Node` is `map[string]interface{}` with a stable `UID`. Every resource gets these common properties: `kind`, `name`, `namespace`, `created`, `apigroup`, `apiversion`, `label`. Type-specific transforms add additional properties. The `NodeStore` passed to `BuildEdges()` provides lookup by UID and by (kind, namespace, name) triple.

### Dev config

For local development, set values in `config.json` (committed with safe defaults); env vars override the file. `DEPLOYED_IN_HUB=true` skips the lease reconciler (required for `make test`). The `-tags development` build tag is not currently used but is kept for compatibility with other search repos.

## Fleet Engineering Skills

All skills are available as slash commands. See the [Fleet Engineering skills catalog](https://github.com/OpenShift-Fleet/agentic-sdlc/blob/main/skills/README.md) for the full list with when-to-use guidance.

## Personal configuration

Read personal config at the start of any task that needs an assignee, email, or project key.
Use the tool-aware fallback chain: `~/.config/opencode/user.local.md` (OpenCode),
`.claude/user.local.md` (Claude Code), or `.cursor/rules/user.local.mdc` (Cursor, already in context).
If none exist, fall back to agent memory (`user-config`), then placeholders.
