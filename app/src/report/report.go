package report

import (
	"io"

	"github.com/b070nd/motiv-cell/src/canon"
	"github.com/b070nd/motiv-cell/src/narrative"
)

func Write(w io.Writer, result narrative.Result) error {
	writer := canon.NewWriter(w)
	if err := writer.Write(
		canon.Str("type", "run"),
		canon.Int("docs", result.Docs),
		canon.Int("sentences", result.Sentences),
		canon.Int("answers", result.Answers),
		canon.Int("cells", int64(len(result.Cells))),
		canon.Int("links", int64(len(result.Links))),
		canon.Int("ticks", result.Ticks),
	); err != nil {
		return err
	}
	for i, c := range result.Cells {
		fields := []canon.Field{
			canon.Str("type", "cell"),
			canon.Int("cell", int64(i)),
			canon.Str("id", c.State.ID),
			canon.Str("doc", c.Doc),
			canon.Str("owner", c.Owner),
			canon.Str("void", c.State.VoidLabel),
			canon.Int("evidence", int64(len(c.State.Evidence))),
		}
		if c.State.Peak.Known {
			fields = append(fields,
				canon.Int("peak_total", c.State.Peak.Total),
				canon.Int("peak_single", c.State.Peak.Single),
				canon.Int("peak_centroid", c.State.Peak.Centroid),
				canon.Int("peak_slope", c.State.Peak.Slope),
				canon.Int("peak_bend", c.State.Peak.Bend),
			)
		}
		if c.State.Peak.HasSupplied {
			fields = append(fields, canon.Int("peak_supplied", c.State.Peak.Supplied))
		}
		if err := writer.Write(fields...); err != nil {
			return err
		}
		for _, item := range c.State.Evidence {
			if err := writer.Write(
				canon.Str("type", "evidence"),
				canon.Int("cell", int64(i)),
				canon.Int("sentence", item.Sentence),
				canon.Int("intensity", item.Intensity),
			); err != nil {
				return err
			}
		}
	}
	for _, edge := range result.Links {
		if err := writer.Write(
			canon.Str("type", "link"),
			canon.Int("from", int64(edge.From)),
			canon.Int("to", int64(edge.To)),
			canon.Int("kind", int64(edge.Link)),
		); err != nil {
			return err
		}
	}
	return nil
}
