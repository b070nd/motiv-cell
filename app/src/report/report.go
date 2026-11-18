package report

import (
	"io"
	"strconv"

	"github.com/b070nd/motiv-cell/src/canon"
	"github.com/b070nd/motiv-cell/src/model"
	"github.com/b070nd/motiv-cell/src/narrative"
)

type Error string

func (e Error) Error() string { return string(e) }

const ErrPosition Error = "report: thread evidence outside the timeline"

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
			canon.Int("level", int64(c.State.Level)),
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
			fields := []canon.Field{
				canon.Str("type", "evidence"),
				canon.Int("cell", int64(i)),
				canon.Int("intensity", item.Intensity),
			}
			if c.State.Level == model.LevelThread {
				if item.Position < 0 || item.Position >= int64(len(result.Timeline)) {
					return Error(string(ErrPosition) + ": " + strconv.FormatInt(item.Position, 10))
				}
				moment := result.Timeline[item.Position]
				fields = append(fields,
					canon.Int("position", item.Position),
					canon.Str("doc", moment.Doc),
					canon.Int("at", moment.At),
				)
			} else {
				fields = append(fields, canon.Int("sentence", item.Position))
			}
			if err := writer.Write(fields...); err != nil {
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
