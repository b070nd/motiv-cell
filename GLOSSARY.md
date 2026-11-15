# Glossary

**Motivation Cell**: one motivation in one document: who wants something (owners), what they lack (void), the evidence for it, and where it peaks.

**Void**: what an actor lacks or wants. Labelled from a fixed option list in the questions file.

**Evidence**: one sentence that shows a void being pursued, with a sentence index and an integer intensity (per-mille).

**Peak**: where a motivation is strongest. Holds a supplied value and a computed field (highest total, strongest single item, centre, slope, bend).

**Same-void link**: a two-way link between cells in different documents that share a void label.

**Reading**: a document turned into numbered sentences, each with a mark.

**Mark**: the fingerprint of a sentence's text. Travels with every answer so a replay can detect that the text changed.

**Answer**: what a provider says about one sentence for one question: a label and a score in per-mille.

**Provider**: anything that returns answers for sentences. Live (laya) or file (recorded answers).

**Finding**: the reading of one sentence's answers as evidence: a void and an intensity, or nothing.

**Narrative**: the result of a run: the cells, their evidence and the links between them.

**Tick**: one step of the engine. In a run, one tick per sentence in reading order.
