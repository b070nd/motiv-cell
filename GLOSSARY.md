# Glossary

**Motivation Cell**: one motivation: who wants something (owners), what they lack (void), the evidence for it, and where it peaks. It is either a document cell or a thread cell.

**Document cell**: a Motivation Cell for one owner and void in one document (level 1). Its evidence is sentences.

**Thread cell**: a Motivation Cell for one owner and void across all documents (level 2). It reads its document cells through part links; its evidence is one point per document, by position in time order.

**Part link**: a link (kind 3) from a document cell to its thread cell. The only link a cell reads.

**Timeline**: the documents of a run in timestamp order; a thread's positions index it.

**Void**: what an actor lacks or wants. Labelled from a fixed option list in the questions file.

**Evidence**: a position and an integer intensity (per-mille). In a document cell the position is a sentence that shows a void being pursued; in a thread cell it is a document on the timeline, with the document cell's summed intensity.

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

**Tick**: one step of the engine. In a run, one tick per sentence in reading order, plus one closing tick so thread cells see the last evidence.
