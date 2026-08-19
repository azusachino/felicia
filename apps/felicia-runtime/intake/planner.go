// Package intake contains provider-independent draft planning.
package intake

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"

	"github.com/azusachino/felicia/core/domain"
)

const (
	// PlanSchema identifies the JSON contract returned by BuildPlan.
	PlanSchema = "felicia.intake.plan"
	// DefaultDerivationVersion identifies the local visit algorithm.
	DefaultDerivationVersion = "gpx-stops-v1"
	// DefaultMinimumStopDwell is the minimum observed dwell for a derived stop.
	DefaultMinimumStopDwell = 20 * time.Minute
	// DefaultMaximumStopRadiusM is the maximum cluster radius for a derived stop.
	DefaultMaximumStopRadiusM = 250.0
	// DefaultMediaMatchWindow is the time tolerance for attaching media.
	DefaultMediaMatchWindow = 30 * time.Minute
)

// IssueSeverity describes whether an intake result is informational or needs
// review. Issues do not automatically make a plan invalid.
type IssueSeverity string

const (
	// IssueInfo records non-blocking planner context.
	IssueInfo IssueSeverity = "info"
	// IssueWarning records a result that needs author review.
	IssueWarning IssueSeverity = "warning"
	// IssueError records invalid input that prevents a trustworthy result.
	IssueError IssueSeverity = "error"
)

// Issue is an explainable planner diagnostic.
type Issue struct {
	Severity IssueSeverity `json:"severity"`
	Code     string        `json:"code"`
	Message  string        `json:"message"`
}

// PlanInput contains already-normalized source values. Adapters and storage
// are intentionally outside this package.
type PlanInput struct {
	JourneyID         uuid.UUID
	SourceFingerprint string
	Routes            []domain.Route
	Visits            []domain.Visit
	Media             []domain.MediaAsset
}

// PlanConfig controls deterministic planning thresholds. The defaults are
// conservative and can be replaced by a future user-visible configuration.
type PlanConfig struct {
	DerivationVersion  string
	MinimumStopDwell   time.Duration
	MaximumStopRadiusM float64
	MediaMatchWindow   time.Duration
}

// DraftPlan is the read-only result of intake planning.
type DraftPlan struct {
	JourneyID         uuid.UUID                 `json:"journey_id"`
	Schema            string                    `json:"schema"`
	Version           string                    `json:"version"`
	SourceFingerprint string                    `json:"source_fingerprint,omitempty"`
	Routes            []domain.Route            `json:"routes"`
	Visits            []domain.Visit            `json:"visits"`
	Stops             []domain.StopCandidate    `json:"stops"`
	Mementos          []domain.MementoCandidate `json:"mementos"`
	Issues            []Issue                   `json:"issues"`

	// DateStart and DateEnd are the journey's date bounds as derived from the
	// input's timestamps. They are a *default* for a journey whose dates the
	// author has not set by hand — the planner itself writes nothing, so
	// honouring that is the persistence step's job (Service.Apply). Both are
	// zero when no dated source was supplied.
	DateStart time.Time `json:"date_start,omitzero"`
	DateEnd   time.Time `json:"date_end,omitzero"`
}

// DefaultConfig returns the deterministic planner defaults.
func DefaultConfig() PlanConfig {
	return PlanConfig{
		DerivationVersion:  DefaultDerivationVersion,
		MinimumStopDwell:   DefaultMinimumStopDwell,
		MaximumStopRadiusM: DefaultMaximumStopRadiusM,
		MediaMatchWindow:   DefaultMediaMatchWindow,
	}
}

// BuildPlan derives reviewable stops and memento candidates without writing to
// a database or changing authored content. Supplied visits take precedence;
// timestamped route points are only used as a fallback.
func BuildPlan(input PlanInput, config PlanConfig) (DraftPlan, error) {
	if input.JourneyID == uuid.Nil {
		return DraftPlan{}, fmt.Errorf("journey ID is required")
	}
	config = withDefaults(config)
	plan := DraftPlan{
		JourneyID:         input.JourneyID,
		Schema:            PlanSchema,
		Version:           "1",
		SourceFingerprint: input.SourceFingerprint,
		Routes:            input.Routes,
		Visits:            input.Visits,
		Stops:             make([]domain.StopCandidate, 0),
		Mementos:          make([]domain.MementoCandidate, 0),
		Issues:            make([]Issue, 0),
	}

	visits := input.Visits
	if len(visits) == 0 {
		var derivedIssues []Issue
		visits, derivedIssues = deriveVisits(input.Routes, config)
		plan.Issues = append(plan.Issues, derivedIssues...)
	}
	plan.Visits = visits
	for index, visit := range visits {
		stop := stopFromVisit(input.JourneyID, visit, index, config.DerivationVersion)
		stop.Evidence = append(stop.Evidence, visitEvidence(visit, index))
		matched, mediaEvidence := mediaForStop(visit, input.Media, config.MediaMatchWindow, config.MaximumStopRadiusM)
		stop.Evidence = append(stop.Evidence, mediaEvidence...)
		if stop.Label == "" {
			plan.Issues = append(plan.Issues, Issue{Severity: IssueWarning, Code: "stop_label_missing", Message: fmt.Sprintf("stop %s has no source label", stop.Identity.Key)})
		}
		stop.Confidence = stopConfidence(visit.Confidence, visit.Arrive, visit.Depart, len(matched), config)
		plan.Stops = append(plan.Stops, stop)
		if len(matched) > 0 {
			plan.Mementos = append(plan.Mementos, mementoFromStop(stop, matched))
		}
	}
	plan.DateStart, plan.DateEnd = dateBoundsFrom(input)
	return plan, nil
}

// dateBoundsFrom returns the first and last calendar day covered by the
// input's dated sources — route spans and samples, visits, and media capture
// times — so a journey can default its dates to the trip it actually
// contains instead of making the author type them in.
//
// Each bound is truncated in its *own* timestamp's location, so a photo taken
// at 08:00 in Tokyo counts as that Tokyo day rather than the previous UTC one.
// That is the best available answer until coordinates can be resolved to a
// timezone (issue #58); sources that hand us a bare UTC timestamp are still
// interpreted as UTC.
func dateBoundsFrom(input PlanInput) (start, end time.Time) {
	observe := func(at time.Time) {
		if at.IsZero() {
			return
		}
		day := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
		if start.IsZero() || day.Before(start) {
			start = day
		}
		if end.IsZero() || day.After(end) {
			end = day
		}
	}
	for _, route := range input.Routes {
		observe(route.From)
		observe(route.To)
		for _, point := range route.Points {
			observe(point.At)
		}
	}
	for _, visit := range input.Visits {
		observe(visit.Arrive)
		observe(visit.Depart)
	}
	for _, asset := range input.Media {
		observe(asset.At)
	}
	return start, end
}

func withDefaults(config PlanConfig) PlanConfig {
	defaults := DefaultConfig()
	if config.DerivationVersion == "" {
		config.DerivationVersion = defaults.DerivationVersion
	}
	if config.MinimumStopDwell <= 0 {
		config.MinimumStopDwell = defaults.MinimumStopDwell
	}
	if config.MaximumStopRadiusM <= 0 {
		config.MaximumStopRadiusM = defaults.MaximumStopRadiusM
	}
	if config.MediaMatchWindow <= 0 {
		config.MediaMatchWindow = defaults.MediaMatchWindow
	}
	return config
}

func deriveVisits(routes []domain.Route, config PlanConfig) ([]domain.Visit, []Issue) {
	points := make([]domain.TrackPoint, 0)
	for _, route := range routes {
		points = append(points, route.Points...)
	}
	sort.SliceStable(points, func(i, j int) bool { return points[i].At.Before(points[j].At) })
	if len(points) < 2 {
		return nil, []Issue{{Severity: IssueWarning, Code: "visit_derivation_unavailable", Message: "no timestamped route points are available for local visit derivation"}}
	}

	visits := make([]domain.Visit, 0)
	cluster := make([]domain.TrackPoint, 0)
	flush := func() {
		if len(cluster) < 2 {
			return
		}
		arrive, depart := cluster[0].At, cluster[len(cluster)-1].At
		if depart.Sub(arrive) < config.MinimumStopDwell {
			return
		}
		coord := centroid(cluster)
		// The cluster is the observation, so it carries a real source identity
		// -- the same one visitEvidence derives. Leaving provenance's source
		// blank would publish an unattributable observation (ADR-0010).
		reference := fmt.Sprintf("derived-route:cluster-%03d", len(visits)+1)
		source := domain.SourceIdentity{System: "local-track", ExternalID: reference}
		visits = append(visits, domain.Visit{
			Coord:      coord,
			Arrive:     arrive,
			Depart:     depart,
			Confidence: 0.5,
			SourceRef:  reference,
			Provenance: domain.Provenance{Source: source, ObservedAt: arrive, Confidence: 0.5},
		})
	}
	for _, point := range points {
		if len(cluster) == 0 {
			cluster = append(cluster, point)
			continue
		}
		if geo.Distance(centroid(cluster), point.Coord) <= config.MaximumStopRadiusM {
			cluster = append(cluster, point)
			continue
		}
		flush()
		cluster = []domain.TrackPoint{point}
	}
	flush()
	return visits, nil
}

func stopFromVisit(journeyID uuid.UUID, visit domain.Visit, index int, derivationVersion string) domain.StopCandidate {
	key := visit.SourceRef
	if key == "" {
		key = fmt.Sprintf("visit-%03d-%d", index+1, visit.Arrive.Unix())
	}
	return domain.StopCandidate{
		JourneyID:  journeyID,
		Identity:   domain.CandidateIdentity{DerivationVersion: derivationVersion, Key: key},
		Label:      visit.Label,
		Coord:      visit.Coord,
		Arrive:     visit.Arrive,
		Depart:     visit.Depart,
		State:      domain.CandidateProposed,
		Provenance: []domain.Provenance{visit.Provenance},
	}
}

func visitEvidence(visit domain.Visit, index int) domain.EvidenceRef {
	if strings.HasPrefix(visit.SourceRef, "derived-route:") {
		return domain.EvidenceRef{
			Kind:    domain.EvidenceRoute,
			Source:  domain.SourceIdentity{System: "local-track", ExternalID: visit.SourceRef},
			Locator: visit.SourceRef,
		}
	}
	if visit.Provenance.Source.Valid() {
		return domain.EvidenceRef{Kind: domain.EvidenceVisit, Source: visit.Provenance.Source, Locator: visit.Provenance.Source.ExternalID}
	}
	source := domain.SourceIdentity{System: "derived-visit", ExternalID: fmt.Sprintf("%03d", index+1)}
	if visit.SourceRef != "" {
		source = domain.SourceIdentity{System: "visit", ExternalID: visit.SourceRef}
	}
	return domain.EvidenceRef{Kind: domain.EvidenceVisit, Source: source, Locator: source.ExternalID}
}

func mediaForStop(visit domain.Visit, media []domain.MediaAsset, window time.Duration, radiusM float64) ([]domain.MediaAsset, []domain.EvidenceRef) {
	matched := make([]domain.MediaAsset, 0)
	evidence := make([]domain.EvidenceRef, 0)
	for _, asset := range media {
		if asset.At.IsZero() || asset.At.Before(visit.Arrive.Add(-window)) || asset.At.After(visit.Depart.Add(window)) {
			continue
		}
		if asset.Coord != nil && geo.Distance(visit.Coord, *asset.Coord) > radiusM {
			continue
		}
		if asset.MemoryLinks == nil {
			// Same reason as the candidate's containers: adapters leave this
			// nil, and nil marshals to JSON null in the emitted plan.
			asset.MemoryLinks = []domain.MemoryLink{}
		}
		matched = append(matched, asset)
		if asset.SourceRef != "" {
			evidence = append(evidence, domain.EvidenceRef{Kind: domain.EvidenceMedia, Source: domain.SourceIdentity{System: "media", ExternalID: asset.SourceRef}, Locator: asset.ID})
		}
	}
	return matched, evidence
}

func stopConfidence(sourceConfidence float64, arrive, depart time.Time, mediaCount int, config PlanConfig) float64 {
	if sourceConfidence > 0 {
		return math.Min(1, sourceConfidence)
	}
	if depart.Before(arrive) || arrive.IsZero() || depart.IsZero() {
		return 0
	}
	dwell := math.Min(1, depart.Sub(arrive).Seconds()/config.MinimumStopDwell.Seconds())
	if mediaCount > 0 {
		return math.Min(1, dwell+0.1)
	}
	return dwell
}

func mementoFromStop(stop domain.StopCandidate, media []domain.MediaAsset) domain.MementoCandidate {
	source := domain.SourceIdentity{System: "derived-stop", ExternalID: stop.Identity.Key}
	// Kind and its kind_data stay unset until the author promotes the
	// candidate, but the containers are still emitted empty rather than nil:
	// a nil map/slice marshals to JSON null, which every consumer of the plan
	// would otherwise have to special-case.
	return domain.MementoCandidate{
		Source:      source,
		StopKey:     stop.Identity.Key,
		OccurredAt:  stop.Arrive,
		Geom:        stop.Coord,
		Title:       stop.Label,
		Place:       stop.Label,
		KindData:    map[string]any{},
		Media:       media,
		MemoryLinks: []domain.MemoryLink{},
		Provenance:  domain.Provenance{Source: source, ObservedAt: stop.Arrive, Confidence: stop.Confidence},
	}
}

func centroid(points []domain.TrackPoint) orb.Point {
	var longitude, latitude float64
	for _, point := range points {
		longitude += point.Coord[0]
		latitude += point.Coord[1]
	}
	return orb.Point{longitude / float64(len(points)), latitude / float64(len(points))}
}
