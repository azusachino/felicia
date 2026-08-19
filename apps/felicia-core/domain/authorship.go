package domain

import "slices"

// Authorship is the A+E boundary expressed as two write kinds:
//
//   - an *ingest* write seeds source-derived fields and may never touch a field
//     the author has claimed, nor shrink the authored mask;
//   - an *authoring* write may set any field and claims it as authored.
//
// The decision lives here — not in provider SQL — so both persistence backends
// behave identically (the same reasoning runtime/intake already applies to
// derived date bounds).

// IngestableFields returns the subset of mask that an ingest write may apply to
// a row whose current authored mask is authored. Order is preserved so callers
// get a deterministic field list.
func IngestableFields(mask, authored []string) []string {
	if len(mask) == 0 || len(authored) == 0 {
		return mask
	}
	allowed := make([]string, 0, len(mask))
	for _, field := range mask {
		if slices.Contains(authored, field) {
			continue
		}
		allowed = append(allowed, field)
	}
	return allowed
}

// IngestJourneyPatch is an explicit source operation on a journey. It mirrors
// IngestMementoPatch: its fields are chosen by the importer, and it can never
// add, remove, or overwrite authored ownership. Journey authoring keeps using
// UpsertJourney, which a shared upsert cannot distinguish from an import — the
// split port is what makes the two intents distinguishable.
type IngestJourneyPatch struct {
	Journey *Journey
	Fields  []string
}

// MergeIngestJourney copies the patch's masked fields onto dst, skipping every
// field dst already claims as authored. dst.AuthoredFields is never read as
// input to a write and never modified, so an ingest can neither widen nor reset
// the mask.
func MergeIngestJourney(dst *Journey, patch *IngestJourneyPatch) {
	if dst == nil || patch == nil || patch.Journey == nil {
		return
	}
	src := patch.Journey
	for _, field := range IngestableFields(patch.Fields, dst.AuthoredFields) {
		switch field {
		case "slug":
			dst.Slug = src.Slug
		case "source_ref":
			dst.SourceRef = src.SourceRef
		case "title":
			dst.Title = src.Title
		case "place":
			dst.Place = src.Place
		case "country":
			dst.Country = src.Country
		case "region":
			dst.Region = src.Region
		case "date_start":
			dst.DateStart = src.DateStart
		case "date_end":
			dst.DateEnd = src.DateEnd
		case "gps_route":
			dst.GPSRoute = src.GPSRoute
		}
	}
}
