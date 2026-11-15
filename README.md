# motiv-cell

A deterministic network of cells, written in Go, for detecting timelines, narrative, actors and their motivations in text. The first target is stock-exchange news; the same core is meant to serve storyline detection in literature (novels, fiction, essays, news reports).

**Status:** the generic core, the Motivation Cell rule, the news reader and the laya provider are implemented and tested. The `motivation` command runs the whole chain. Owners are not detected yet.

## Principles

- **Network-free core:** the core packages never touch the network. Model providers such as a local laya server live in separate, explicitly listed packages and can either be called during a run or write annotation files the core reads.
- **Deterministic:** the same input and config give byte-identical output. Fixed-point integers in the cell rule, no randomness in the core, no wall clock, canonical JSONL output.
- **No global state:** no `init()` and no package-level variables; modules are built from explicit config.
- **Generic core:** a cell is a pure `(state, inputs) -> state` function; topology (graph, lattice, layered) and tick source are pluggable. The Motivation Cell (owner, void, evidence, peak) is one domain plug-in.
- **Plug-in resources:** lexicons and taxonomies load from `data/` and are not bundled, because several finance lexicons carry non-commercial licences.

## Layout

```
app/
  go.mod                 module github.com/b070nd/motiv-cell
  config/questions.json  the questions put to each sentence
  src/
    cell/ lattice/ engine/   generic core: rule contract, edge builders, stepper
    canon/                   canonical JSONL writer
    model/ motive/           Motivation Cell data and rule
    newsfeed/                news export reader
    segment/ reading/        sentence splitter; numbered, fingerprinted sentences
    annotate/                answer format, file provider, recorder, comparison
    laya/                    client for a local laya-serve (network)
    evidence/                questions file and what an answer means
    narrative/               documents + answers -> cells, evidence, links
    report/                  result -> JSONL records
    guard/                   structural rules enforced as a test
    cmd/motivation/          command line: flags and files only
data/                    inputs and outputs, gitignored
research/                background research notes
GLOSSARY.md              the project's terms
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

How a run works: documents are ordered by timestamp, the headline is sentence 0 and the body is split into sentences. Each sentence gets a fingerprint of its text. Each sentence is put to the questions in `config/questions.json`. The file must contain a choice question named `void` and a score question named `intensity`; a yes/no question named `pursuit` is optional. A misnamed or mistyped question is an error, not an empty result. A sentence becomes evidence when its void is not the `none` label and its yes-probability for `pursuit` is at least `pursuit_min` (per-mille). The first evidence for a void in a document creates a cell; cells in different documents with the same void are linked. The result is JSONL with one `run` line, then `cell`, `evidence` and `link` lines.

Recorded answers carry the sentence fingerprint, and a replay matches answers to sentences by that fingerprint, not by position. Sentences that only moved (for example because one was inserted above them) keep their answers. A sentence whose text has no recorded answers, in a document that was recorded, stops the replay with an error; re-record after editing the text or changing the splitter.

The answers depend on the laya checkpoint and on the exact question wording, and are uncalibrated. Set `LAYA_API_KEY` if the server requires a key.

## Modules

- `cell`: the contract. A `Rule[S, E]` maps a cell's state, its neighbours' states and the events delivered this tick to the next state.
- `engine`: steps every cell once per tick, synchronously or in ascending cell order. Structure arrives one way: as cells and links, at construction and on any later tick. A failed step changes nothing. Ticks come from a source: one per text position, or one per distinct timestamp.
- `lattice`: builders that produce links for common shapes (`Graph`, `Line`, `Layers`); `lattice.Edges` turns one into the links the engine takes.
- `model`, `motive`: a Motivation Cell holds owners, a void, evidence and a peak. The peak keeps a supplied value next to a computed field: the sentence with the highest summed intensity, the sentence of the strongest single item, the weighted centre, and the slope and bend at the latest evidence.
- `reading`: turns a document into numbered sentences with fingerprints. The only place that decides sentence numbers.
- `evidence`: parses and validates the questions file and decides whether a sentence's answers amount to evidence. The only place that knows question roles, yes/no polarity and the threshold.
- `annotate`, `laya`: the answer format with a file provider, a recorder and a comparison; and the live client.
- `narrative`: `Build(docs, provider, spec)` returns the cells, their evidence and the links as values.
- `report`: writes that result as canonical JSONL.
- `guard`: a test that fails on unregistered packages, forbidden imports, `init()`, package-level variables and wall-clock calls, and in the core packages on floats, maps, channels and goroutines.

## Research

Design background and sources: [research/core-cell-network.md](research/core-cell-network.md).

## Security

See [SECURITY.md](SECURITY.md).

## License

MIT © 2026 Botond Biro. See [LICENSE](LICENSE).
