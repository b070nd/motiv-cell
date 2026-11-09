# PROTOCOL

Exchange interface between the local builder (Claude Code) and the reviewer (ChatGPT). Reviewer: answer in section 6, change statuses in section 3, never edit sections 1-2. Builder updates sections 3-5 after each pass.

Last updated: 2026-11-09 · Pass: 2 (research done, grilling rounds 1-2 answered)

## 1. Goal and constraints

- Air-gapped cellular automaton that detects timelines, narrative, actors and their motivations in text (novels, fiction, essays, news). First target: a Motivation Cell and a network of cells evaluating stock exchange news.
- Language: Go (go 1.27.1 installed). Layout mirrors cvewatch v2.0.0: `app/{go.mod,config,src/<pkg>,src/cmd/<bin>}`, `data/` gitignored.
- No globals, no `init()`, no package-level mutable state. Every module is constructed from explicit config and injected dependencies (rng, clock).
- Deterministic: same input + seed + config gives byte-identical output. No `time.Now`, no unordered map iteration into output, no goroutine-order dependence.
- Air-gapped: no network imports anywhere in `app/src`. Data enters only as files from `data/`.
- Scaffold pass: package clauses only. No code, no documentation (this file excepted).
- Model routing: easy tasks Haiku, mid and hard tasks Sonnet.

## 2. Module map (`app/src`)

Allowed dependencies: a package may import only packages listed in its row. `model` imports nothing project-internal.

| Pkg | Role | May import | Difficulty |
|---|---|---|---|
| model | immutable value types: Doc, Span, Actor, Void, Motivation, Event | none | mid |
| clock | injected time (fixed / replay), no wall clock | none | easy |
| config | typed config load and validate | model | easy |
| cell | one Motivation Cell: state + pure transition `(state, inputs) -> state` | model | hard |
| lattice | network topology: cells, edges, same-void links, neighbour order | model, cell | hard |
| engine | deterministic stepper over a lattice, tick by tick | model, cell, lattice, clock | hard |
| ingest | offline readers for files under `data/` | model | easy |
| newsfeed | stock-news item normalisation into Doc | model, ingest | mid |
| market | exchange domain vocabulary: instruments, venues, event kinds | model | mid |
| segment | text to sentences and spans | model | mid |
| lexicon | goal-pursuit and force lexicon loading | model, config | mid |
| actors | owner identification | model, lexicon, market | hard |
| voids | void labelling and same-void matching | model, lexicon | hard |
| evidence | goal-pursuit spans and intensity | model, lexicon | hard |
| peak | peak location of a motivation | model | mid |
| timeline | event ordering, discourse vs event time | model | hard |
| narrative | assemble actors, motivations, timeline | model, actors, voids, evidence, peak, timeline | hard |
| annotate | annotation record formats shared by humans and models | model | mid |
| agreement | Krippendorff alpha family | annotate | hard |
| store | file persistence under `data/` | model | mid |
| report | render output | model, narrative, agreement | mid |
| golden | golden-file test harness | none | easy |
| testkit | fixture builders | model | easy |
| version | build version | none | easy |
| tui | terminal UI | model, narrative, report | mid |
| cmd/motivation | CLI entry, the only place that wires packages together | all | mid |
| cmd/motivation-tui | TUI entry | all | easy |

## 3. Tasks

Status: `todo` · `doing` · `done` · `review` · `blocked`. Model: H Haiku, S Sonnet.

| ID | Task | Diff | Model | Status | Needs |
|---|---|---|---|---|---|
| T-01 | Directory tree, go.mod, .gitignore, package stubs | easy | H | done | |
| T-02 | Import guard test: reject network and `time.Now`, `math/rand` global, `init()` | mid | S | todo | T-01, Q-09 |
| T-03 | `model` value types | mid | S | todo | Q-06 |
| T-04 | `clock` | easy | H | todo | |
| T-05 | `config` schema and loader | easy | H | todo | Q-13 |
| T-06 | `cell` generic state and pure fixed-point transition | hard | S | todo | T-03, Q-07 |
| T-07 | `lattice` topologies: graph, grid, layered | hard | S | todo | T-06 |
| T-08 | `engine` stepper with injected tick source, replay determinism test | hard | S | todo | T-07, T-04 |
| T-09 | `ingest` + `newsfeed` for JSONL news export | mid | S | todo | |
| T-10 | `market` vocabulary | mid | S | todo | Q-05 |
| T-11 | `segment`, `lexicon` | mid | S | todo | Q-04 |
| T-12 | `actors`, `voids`, `evidence`, `peak` | hard | S | todo | T-03, T-10, T-11 |
| T-13 | `timeline`, `narrative` | hard | S | todo | T-12, Q-03 |
| T-14 | `annotate` record formats | mid | S | todo | Q-10 |
| T-15 | `agreement` (alpha: nominal, interval, unitizing) with hand-computed fixtures | hard | S | todo | T-14 |
| T-16 | `store`, `report` | mid | S | todo | T-13 |
| T-17 | `golden` + `testkit`, end-to-end test through the CLI seam | mid | S | todo | T-16 |
| T-18 | `cmd/motivation` wiring, `tui` | mid | S | todo | T-16, Q-12 |

## 4. Findings

- F-01 The pasted Gate-1 spec assumes Python, 30-60 ELTeC literary excerpts, human annotators and an LLM arm. This project is Go, starts from stock-exchange news, and is air-gapped. The spec is used here only as the source of the Motivation Cell data model (owner, void, evidence spans with intensity, peak location, pairwise same-void judgement) and the agreement metrics.
- F-02 The spec's CHR 2027 deadline (2026-08-14) and 2026-07-13 start have passed (today 2026-11-08). Its timeline is stale.
- F-03 The spec cites an `AGENTS.md` and `/setup-matt-pocock-skills`; neither exists in this repo.
- F-04 The spec forbids coreference, NER and event extraction in the pilot. A cell network over news needs actors and events, so that restriction cannot carry over unchanged (Q-05).
- F-05 The spec's LLM arm conflicts with "air-gapped" unless the model runs locally or is replayed from cache (Q-09).
- F-06 cvewatch v2.0.0 layout confirmed from its tree: `app/config`, `app/src/<pkg>`, `app/src/cmd/<bin>`, `app/src/golden/testdata`, `docs`, `skills`, GoReleaser, `compose.yaml`. Not copied here: Docker, GoReleaser, CI, docs, skills (out of scope for the scaffold).
- F-07 Scaffold-only deviation from "no code": each `cmd/*/main.go` holds an empty `func main() {}`, otherwise `go build ./...` fails.

## 4a. Decisions (user, 2026-11-09)

- D-01 (Q-02) Cell model is not fixed. The core is a generic cell with a pure transition function; topology is pluggable so graph, text-position lattice and layered (lattice under graph) are all expressible. The choice among them is deferred to domain wiring, not the core.
- D-02 (Q-03) Tick source is injected and pluggable: discourse order and event timestamp implementations both exist.
- D-03 (Q-08) Cell rule uses fixed-point integers (scaled int64). No floats in state or transition.
- D-04 (Q-04) First news corpus is the user's own export, JSONL `{id, ts, source, headline, body}`, dropped into `data/news`.
- D-05 Core is domain-agnostic: stock-news sentiment and literature storyline detection are domain plug-ins over the same engine.
- D-06 (research Q2) Licence posture: plug-in lexicons. Core contains no restricted data; lexicons load from `data/` with a licence manifest.
- D-07 (Q-05) Actors and events: gazetteer + name-variant rules + lexicons is the default provider. Providers sit behind one interface so an external preprocessor writing annotation files, or an adapter around a vendored tagger, can be added without touching the core. Vendored tagger adapters live outside the guarded core (prose/v3 has `init()` and globals).
- D-08 (Q-02/Q-07) Own Motivation Cell (owner, void, evidence, peak, same-void edges), no derived numbers yet, link kinds kept open so plot-unit states and m/a/t/e links can be added later.
- D-09 (Q-17) Dependencies: stdlib plus a short vetted vendored allowlist (candidates: x/text, uniseg, snowball) with a hash manifest checked by the guard.
- D-10 (Q-06) Void labelling is configurable and extensible: normaliser and matcher are plug-ins; closed label sets are configuration.
- D-11 (research Q8) Evaluation is plug-in based and configurable; alpha, span F1, unitizing coefficient are candidate plug-ins.
- D-12 (Q-08) No RNG in the core. Rationale: assume least, avoid false positives and drift. Sampling-based evaluation is out of scope for now. `rng` package is therefore removed from the plan (T-04 reduced to `clock`).
- D-13 Canonical output is an own JSONL writer: one record per line, sorted keys, integers and strings only, no deep nesting. No `encoding/json` in output paths.
- D-14 (research Q12/Q13/Q6, confirmed by user 2026-11-09) Go toolchain pinned to 1.27.1 with golden regeneration on upgrade; English first with the language set left to config; tick sources per D-02, fabula/Allen time deferred.

- Consequence: `lattice` hosts topologies (graph, grid, layered) behind one interface; `cell` and `engine` are generic over state and input. T-06/T-07/T-08 are re-scoped accordingly.

## 4b. Research

- `research/core-cell-network.md` (areas A-I, VERIFIED/UNVERIFIED tagged). Key facts: float fused multiply-add differs across arm64 and amd64 (reproduced); Go 1.27 uses Unicode 17 tables and the json v2 implementation; vendored offline build works with stock Go; a syntactic import/init/time.Now guard is feasible but `go list -deps` is needed for transitive imports; Go interface methods cannot have type parameters, so heterogeneous cells need a non-generic seam.
- F-08 `rng` removal also resolves the unverified claim that raw PCG/ChaCha8 streams are stable across Go releases.

## 5. Open questions (questionnaire)

Answer by letter, or write your own. Default is what the builder assumes if unanswered. Hard-blocking questions are marked **B**.

**Q-01 Language.** The spec says Python, the brief says Go. Default: Go. (A) Go (B) Python.

**[ANSWERED D-01] Q-02 What is a "cell"?** (A) One motivation instance (owner, void, evidence, peak); the network is a graph linked by same-void edges. (B) A site on a lattice of text positions (one sentence or paragraph per site) with a local update rule, as in a classic automaton. (C) Hybrid: text-position lattice carries evidence, motivation cells form on top of it. Default: A.

**[ANSWERED D-02] Q-03 Tick axis.** What advances the automaton? (A) Discourse order (sentence by sentence). (B) Publication timestamp of each news item. (C) Inferred event time (fabula) separate from discourse order. Default: A for v0, B for the news network.

**[ANSWERED D-04] Q-04 Stock-news corpus.** Source, licence, language, size, period, granularity (headline, article, wire flash)? Air-gapped means you supply the files by hand. Format preference (JSONL, plain text per article, CSV)? Default: JSONL, one item per line, `{id, ts, source, headline, body}`.

**[ANSWERED D-07] Q-05 Actors in finance news.** Who counts as an owner? Companies, tickers, regulators, central banks, funds, people, "the market" as an abstract actor? Gazetteer supplied by you, derived from the corpus, or annotator-only as the spec says? Default: gazetteer file in `data/` plus annotator override.

**[ANSWERED D-10] Q-06 Void taxonomy.** Closed label set (e.g. capital, market share, regulatory approval, liquidity, reputation) or open free text plus normalised label as in the spec? Default: open text plus normalised label, closed set revisited after the first annotation round.

**[ANSWERED D-08] Q-07 State of a cell.** Does the cell hold numeric amplitude and phase now? The spec defers the derivation formula. (A) Cell carries evidence only, no derived numbers. (B) Cell carries amplitude = frequency x force with a placeholder formula. (C) Amplitude and phase with interference between same-void cells. Default: A.

**[ANSWERED D-03] Q-08 Determinism rules.** Floats allowed, or fixed-point integers only? Single goroutine throughout? Seed type uint64 in config? Default: integers/fixed-point in the cell rule, single goroutine, uint64 seed.

**Q-09 Air-gap and the model arm.** Keep an LLM arm at all? (A) None. (B) Local model run by an external process that reads task files and writes record files (the program itself never calls a model). (C) Replay of cached outputs only. Default: B, with C for tests. Also confirm the import guard may be strict: stdlib only, no `net`, `net/http`, `os/exec`.

**Q-10 Scope of this build.** Include the spec's Gate-1 seam (`sample`, `annotate-llm`, `report`) in this project, or only the cell and network for news and keep the annotation pilot as a later separate effort? Default: cell and network first; `annotate` and `agreement` stay as stubs.

**Q-11 Names.** Module path `github.com/b070nd/motivation`, binaries `motivation` and `motivation-tui`. Change?

**Q-12 TUI.** Stdlib-only (ANSI, minimal) or a library such as Bubble Tea (external dependency, would need vendoring for air-gap)? Default: stdlib-only.

**Q-13 Config format.** The Go stdlib reads JSON only. YAML or TOML needs a vendored dependency. Default: JSON.

**Q-14 Storage.** Plain JSONL files under `data/`, or an embedded database (pure-Go SQLite, vendored)? Default: JSONL.

**Q-15 How is correctness judged on news?** Hand-labelled fixtures only, agreement against your own annotations, or a market-reaction proxy (price move after the item)? The proxy invites investment-advice territory and leakage; default: hand-labelled fixtures only.

**Q-16 Languages.** English only for news, or also Hungarian or another second language from the start? Default: English only.

**[ANSWERED D-09] Q-17 Vendoring.** Vendor all dependencies into `app/vendor` and commit them, so builds work offline? Default: yes, once any dependency exists.

**Q-18 Pilot timeline.** F-02 says the spec's dates are stale. Is the next target venue and deadline known, or should the scaffold ignore the conference entirely? Default: ignore.

## 6. Reviewer notes

Reviewer (ChatGPT): write answers to section 5 and review findings below this line. Builder reads this section first on the next pass.

-
