# Glossary

**Motivation Cell**: one motivation of one owner in one document: who wants something (owners), what they lack (void), the evidence for it, and where it peaks.

**Void**: what an actor lacks or wants. Labelled from a fixed option list in the questions file.

**Evidence**: one sentence that shows a void being pursued, with a sentence index and an integer intensity (per-mille).

**Peak**: where a motivation is strongest. Holds a supplied value and a computed field (highest total, strongest single item, centre, slope, bend).

**Actor**: a company, institution or person listed in the gazetteer, with an id, a name, aliases and tickers.

**Gazetteer**: the user-supplied list of actors that owners are matched against.

**Owner**: the actor a Motivation Cell belongs to: the first actor named in the evidence sentence, or the most recent one named earlier in the document.

**Same-void link**: a two-way link (kind 1) between cells that share a void label.

**Same-owner link**: a two-way link (kind 2) between cells that share an owner.

**Reading**: a document turned into numbered sentences, each with a mark.

**Mark**: the fingerprint of a sentence's text. Travels with every answer so a replay can detect that the text changed.

**Answer**: what a provider says about one sentence for one question: a label and a score in per-mille.

**Provider**: anything that returns answers for sentences. Live (laya) or file (recorded answers).

**Finding**: the reading of one sentence's answers as evidence: a void and an intensity, or nothing.

**Narrative**: the result of a run: the cells, their evidence and the links between them.

**Tick**: one step of the engine. In a run, one tick per sentence in reading order.
