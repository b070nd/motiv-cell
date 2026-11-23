# motiv-cell

A deterministic network of cells, written in Go, for detecting timelines, narrative, actors and their motivations in text. The first target is stock-exchange news; the same core is meant to serve storyline detection in literature (novels, fiction, essays, news reports).

**Status:** the generic core, the Motivation Cell rule, the news reader and the laya provider are implemented and tested. The `motivation` command runs the whole chain. Owners come from a gazetteer of names you supply.

## Claimed and implemented

What this README and the project description promise, against what the code does today, so the repository never overstates itself.

| Claim | Implemented |
|---|---|
| Deterministic, byte-identical output | Yes. Live run and replay, and repeated replays, give identical files |
| Network-free core, providers in separate packages | Yes, enforced by the guard test. The laya client uses the network, so the project as a whole is not air-gapped |
| Actors | Partly. Owners come from a gazetteer you supply: the first actor named in a sentence, else the most recent one in the document. No coreference |
| Motivations (owner, void, evidence, peak) | Partly. Cells, evidence and the computed peak field are produced. The answers behind them are uncalibrated and nothing has been compared with human judgement |
| A network of cells | Partly. Two levels, coarse over fine: each thread cell reads its document cells through part links and recomputes from them on every tick. Same-void and same-owner links are still only written to the output |
| Narrative | Not yet. The result is cells and links; nothing reads a storyline out of them |
| Timelines | Partly. Each thread places one owner's motivation on the time-ordered documents, with its peak, centre, slope and bend over time; thread evidence carries the document and its timestamp. No event timeline beyond that |
| Pluggable topology (graph, line, layers) | Built and tested in `lattice`; the pipeline builds its links directly and uses none of the line or layer builders |
| Pluggable tick source | Both sources exist in `engine`; the pipeline uses one tick per sentence only |
| Supplied peak next to the computed one | The rule accepts a supplied peak; nothing in the pipeline sends one |
| Lexicons and taxonomies loaded from `data/` | Not built. Void labels come only from the questions file |
| Literature | Not built. Intended use of the same core |
| Evaluation | None. The only data is three synthetic articles |
| Continuous integration and releases | None yet |

## Principles

- **Network-free core:** the core packages never touch the network. Model providers such as a local laya server live in separate, explicitly listed packages and can either be called during a run or write annotation files the core reads.
- **Deterministic:** the same input and config give byte-identical output. Fixed-point integers in the cell rule, no randomness in the core, no wall clock, canonical JSONL output.
- **No global state:** no `init()` and no package-level variables, except the version stamp in the command; modules are built from explicit config.
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
    actors/                  gazetteer of actors and name matching
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

Add `-actors ../data/actors/sample.jsonl` to either run to fill in owners. The gazetteer is JSONL, one actor per line:

```
{"id":"acme","name":"Acme Corp","aliases":["Acme"],"tickers":["ACME"]}
```

Compare two answer files:

```bash
cd app && go run ./src/cmd/motivation compare -a ../data/annotations/live.jsonl -b ../data/annotations/other.jsonl
```

How a run works: documents are ordered by timestamp, the headline is sentence 0 and the body is split into sentences. Each sentence gets a fingerprint of its text. Each sentence is put to the questions in `config/questions.json`. The file must contain a choice question named `void` and a score question named `intensity`; a yes/no question named `pursuit` is optional, and so is a choice question named `capability` (is the actor presented as able to get what it wants: able, unable, not stated). Capability answers are recorded and replayed but not yet read into evidence, so they change no cell; they are there to measure first whether news states capability often enough, and separately enough from pursuit, to model. A misnamed or mistyped question is an error, not an empty result. A sentence becomes evidence when its void is not the `none` label and its yes-probability for `pursuit` is at least `pursuit_min` (per-mille). The owner of a sentence is the first gazetteer actor named in it; a sentence that names nobody takes the most recent actor named earlier in the same document, and has no owner if there is none. The first evidence for an owner and void in a document creates a document cell (level 1). The first document cell for an owner and void also creates a thread cell (level 2) for that owner and void across all documents, and every such document cell is linked to it (kind 3). A thread takes no evidence of its own: on every tick it reads its document cells and places each one's summed intensity at that document's position in time order, so its peak says where in the timeline the motivation is strongest and its slope and bend say whether it is rising. Sentences without an owner create document cells but no thread. Document cells with the same void are linked (kind 1), and with the same owner (kind 2). One tick runs per sentence, plus one closing tick so threads see the last evidence. The result is JSONL with one `run` line, then `cell` lines (with `level`), `evidence` lines (a `sentence` for document cells; a `position`, `doc` and `at` for threads) and `link` lines.

Recorded answers carry the sentence fingerprint, and a replay matches answers to sentences by that fingerprint, not by position. Sentences that only moved (for example because one was inserted above them) keep their answers. A sentence whose text has no recorded answers, in a document that was recorded, stops the replay with an error; re-record after editing the text or changing the splitter.

The answers depend on the laya checkpoint and on the exact question wording, and are uncalibrated. Set `LAYA_API_KEY` if the server requires a key.

## Modules

- `cell`: the contract. A `Rule[S, E]` maps a cell's state, its neighbours' states and the events delivered this tick to the next state.
- `engine`: steps every cell once per tick, synchronously or in ascending cell order. Structure arrives one way: as cells and links, at construction and on any later tick. A failed step changes nothing. Ticks come from a source: one per text position, or one per distinct timestamp.
- `lattice`: builders that produce links for common shapes (`Graph`, `Line`, `Layers`); `lattice.Edges` turns one into the links the engine takes.
- `model`, `motive`: a Motivation Cell holds a level, owners, a void, evidence and a peak. A document cell collects evidence by sentence; a thread cell computes its evidence from its document cells, one point per document in time order. The peak keeps a supplied value next to a computed field: the position with the highest summed intensity, the position of the strongest single item, the weighted centre, and the slope and bend at the latest evidence. The coarse-over-fine layering borrows the idea of Next-Concept Modeling (Kumbong et al., arXiv 2610.14162), not its model.
- `reading`: turns a document into numbered sentences with fingerprints. The only place that decides sentence numbers.
- `actors`: parses the gazetteer and finds actors in text by whole-word, case-sensitive match on name, aliases and tickers, longest name first. A name claimed by two actors is an error.
- `evidence`: parses and validates the questions file and decides whether a sentence's answers amount to evidence. The only place that knows question roles, yes/no polarity and the threshold.
- `annotate`, `laya`: the answer format with a file provider, a recorder and a comparison; and the live client.
- `narrative`: `Build(docs, provider, spec, gazetteer)` returns the timeline, the cells, their evidence and the links as values.
- `report`: writes that result as canonical JSONL.
- `guard`: a test that fails on unregistered packages, forbidden imports, `init()`, package-level variables and wall-clock calls, and in the core packages on floats, maps, channels and goroutines.

## Research

Design background and sources: [research/core-cell-network.md](research/core-cell-network.md).

## Security

See [SECURITY.md](SECURITY.md).

## License

MIT © 2026 Botond Biro. See [LICENSE](LICENSE).
