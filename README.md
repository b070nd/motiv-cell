# motiv-cell

An air-gapped, deterministic network of cells, written in Go, for detecting timelines, narrative, actors and their motivations in text. The first target is stock-exchange news; the same core is meant to serve storyline detection in literature (novels, fiction, essays, news reports).

**Status:** scaffold only. Packages compile but contain no logic yet.

## Principles

- **Air-gapped:** no network access at build or run time; data enters only as files from `data/`.
- **Deterministic:** the same input and config give byte-identical output. Fixed-point integers in the cell rule, no randomness in the core, no wall clock, canonical JSONL output.
- **No global state:** no `init()` and no package-level mutable state; modules are built from explicit config and injected dependencies.
- **Generic core:** a cell is a pure `(state, inputs) -> state` function; topology (graph, lattice, layered) and tick source are pluggable. The Motivation Cell (owner, void, evidence, peak) is one domain plug-in.
- **Plug-in resources:** lexicons and taxonomies load from `data/` and are not bundled, because several finance lexicons carry non-commercial licences.

## Layout

```
app/
  go.mod                 module github.com/b070nd/motiv-cell
  config/                default configuration files
  src/
    model/               shared value types
    cell/ lattice/ engine/            cell rule, topologies, stepper
    clock/ config/                    injected time, typed config
    ingest/ newsfeed/ market/         file readers, news items, exchange vocabulary
    segment/ lexicon/                 text segmentation, lexicon loading
    actors/ voids/ evidence/ peak/    motivation detection
    timeline/ narrative/              ordering and assembly
    annotate/ agreement/              annotation records, agreement metrics
    store/ report/ tui/               persistence and output
    golden/ testkit/ version/         test support, build info
    cmd/motivation/                   CLI
    cmd/motivation-tui/               terminal UI
data/                    inputs and outputs, gitignored
research/                background research notes
```

## Build

Requires Go 1.27.1.

```bash
cd app && go build ./... && go vet ./...
```

## Research

Design background and sources: [research/core-cell-network.md](research/core-cell-network.md).

## Security

See [SECURITY.md](SECURITY.md).

## License

MIT © 2026 Botond Biro. See [LICENSE](LICENSE).
