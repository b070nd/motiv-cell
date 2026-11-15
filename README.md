# motiv-cell

A deterministic network of cells, written in Go, for detecting timelines, narrative, actors and their motivations in text. The first target is stock-exchange news; the same core is meant to serve storyline detection in literature (novels, fiction, essays, news reports).

**Status:** the generic core, the Motivation Cell rule, the news reader and the laya provider are implemented and tested. The `motivation` command runs the whole chain. Owners are not detected yet, and the remaining domain packages (actors, voids, evidence and the rest) are empty stubs.

## Principles

- **Network-free core:** the core packages never touch the network. Model providers such as a local laya server live in separate, explicitly listed packages and can either be called during a run or write annotation files the core reads.
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
    canon/                            canonical JSONL writer
    segment/                          rule-based sentence splitter
    motive/                           Motivation Cell rule and peak field
    laya/                             client for a local laya-serve (network)
    guard/                            structural rules enforced as a test
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
cd app && go build ./... && go vet ./... && go test ./...
```

## Run

Start a local laya server bound to localhost (it binds to all interfaces by default):

```bash
LAYA_HOST=127.0.0.1 LAYA_PORT=8765 LAYA_MODELS=english laya-serve
```

Live run, recording laya's answers:

```bash
cd app && go run ./src/cmd/motivation run -news ../data/news/sample.jsonl -questions config/questions.json -laya http://127.0.0.1:8765 -record ../data/annotations/live.jsonl -out ../data/runs/live.jsonl
```

Replay from the recorded answers, without laya:

```bash
cd app && go run ./src/cmd/motivation run -news ../data/news/sample.jsonl -questions config/questions.json -answers ../data/annotations/live.jsonl -out ../data/runs/replay.jsonl
```

Compare two answer files:

```bash
cd app && go run ./src/cmd/motivation compare -a ../data/annotations/live.jsonl -b ../data/annotations/other.jsonl
```

How a run works: documents are ordered by timestamp, the headline is sentence 0 and the body is split into sentences. Each sentence is put to the questions in `config/questions.json`, which must be named `void` (choice), `intensity` (score) and optionally `pursuit` (yes/no). A sentence becomes evidence when its void is not the `none` label and its yes-probability for `pursuit` is at least `pursuit_min` (per-mille). The first evidence for a void in a document creates a cell; cells in different documents with the same void are linked. The result is JSONL with one `run` line, then `cell`, `evidence` and `link` lines.

The answers depend on the laya checkpoint and on the exact question wording, and are uncalibrated. Set `LAYA_API_KEY` if the server requires a key.

## Core

- `cell`: the contract. A `Rule[S, E]` maps a cell's state, its neighbours' states and the events delivered this tick to the next state. `S` and `E` are chosen by the domain.
- `lattice`: topologies behind one `Topology` interface. `Graph` (explicit directed edges with a link kind), `Line` (one-dimensional text positions with a radius) and `Layers` (several topologies stacked into one ID space, plus links between them). Neighbour order is fixed.
- `engine`: steps every cell once per tick. `Synchronous` reads the previous tick's states; `Sequential` updates in ascending cell order. A tick can add cells and links before it steps. A failed step changes nothing, including any growth. Ticks come from a `Source`: `NewDiscourse` gives one tick per text position, `NewTimestamps` one tick per distinct timestamp.
- `clock`: injected time, fixed or replayed. Nothing reads the wall clock.
- `canon`: writes flat records (integers and strings) as JSONL with sorted keys and fixed escaping, without `encoding/json`.
- `model` and `motive`: a Motivation Cell holds owners, a void, evidence (sentence index and integer intensity) and a peak. The peak keeps a supplied value next to a computed field: the sentence with the highest summed intensity, the sentence of the strongest single item, the intensity-weighted centre, and the slope and bend of the intensity at the latest evidence.
- `newsfeed`: reads the news export, JSONL with `id`, `ts` (RFC 3339), `source`, `headline`, `body`.
- `annotate`: the shared answer format (document, sentence, question, label, score in per-mille), with a file provider, a recorder and a comparison of two answer sets.
- `laya`: calls a local `laya-serve` and returns answers in that format. Wrapped in the recorder, a live run also writes the file that replays it, so live and file modes can be compared.
- `guard`: a test that fails on forbidden imports, `init()`, package-level variables, wall-clock calls, and, in the core packages, floats, maps, channels and goroutines. New packages must be registered in its table.

## Research

Design background and sources: [research/core-cell-network.md](research/core-cell-network.md).

## Security

See [SECURITY.md](SECURITY.md).

## License

MIT © 2026 Botond Biro. See [LICENSE](LICENSE).
