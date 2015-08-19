package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Hardcoded metric names for ease of reference
const (
	HITS   string = "hits"
	ERRORS        = "errors"
	TIMES         = "times"
)

// These represents the default stats we keep track of
var DefaultMetrics = [3]string{HITS, ERRORS, TIMES}

// Count represents one specific "metric" we track for a given tag set, it accumulates new values, optionally keeping track of the distribution
type Count struct {
	Name         string       `json:"name"`         // represents the entity we count, e.g. "hits", "errors", "time"
	Tags         []Tag        `json:"tags"`         // list of dimensions for which we account this Count
	Value        float64      `json:"value"`        // accumulated values
	Distribution Distribution `json:"distribution"` // optional, represents distribution of values and refs of traces for the spectrum of values
}

func NewCount(m string, tags *[]Tag, eps float64) *Count {
	// FIXME: how to handle tracking the distribution of other than DefaultMetrics?
	var d Distribution
	if m == TIMES {
		d = NewDistribution(eps)
	}
	return &Count{
		Name:         m,
		Tags:         *tags,
		Value:        0,
		Distribution: d,
	}
}

func (c *Count) Add(s *Span) bool {
	switch c.Name {
	case HITS:
		c.Value++
		return true
	case ERRORS:
		c.Value++
		return true // always keep error traces? probably stupid if errors are not aggregated in some way
	case TIMES:
		c.Value += s.Duration
		keep := c.Distribution.Insert(s.Duration, s.TraceID)
		return keep
	default:
		panic(errors.New(fmt.Sprintf("Don't know how to handle a '%s' count", c.Name)))
	}
}

type StatsBucket struct {
	Eps      float64           `json:"eps"`      // parameter used to guarantee epsilon-precision for the distribution
	Start    float64           `json:"start"`    // timestamp representing the start of the bucket
	Duration float64           `json:"duration"` // width of the time bucket
	Counts   map[string]*Count `json:"count"`    // actual representation of the data that is tracked, keyed by (metric,tags) strings
}

func CountKey(m string, tags []Tag) string {
	s := make([]string, len(tags))
	for i, t := range tags {
		s[i] = t.String()
	}
	sort.Strings(s)
	return fmt.Sprintf("metric:%s|tags:%s", m, strings.Join(s, ","))
}

// NewStatsBucket opens a new bucket at this time and initializes it properly
func NewStatsBucket(eps float64) *StatsBucket {
	m := make(map[string]*Count)
	return &StatsBucket{
		Eps:    eps,
		Start:  Now(),
		Counts: m,
	}
}

// HandleSpan adds the span to this bucket stats
func (b *StatsBucket) HandleSpan(s *Span) bool {
	keep := false

	// FIXME: clean and implement generic way of generating tag sets and metric names

	// by service
	sTag := Tag{Name: "service", Value: s.Service}
	byS := []Tag{sTag}
	if b.addInDimension(s, &byS) {
		keep = true
	}

	// by (service, resource)
	rTag := Tag{Name: "resource", Value: s.Resource}
	bySR := []Tag{sTag, rTag}
	if b.addInDimension(s, &bySR) {
		keep = true
	}

	return keep
}

func (b *StatsBucket) addInDimension(s *Span, tags *[]Tag) bool {
	// FIXME: here add the ability to add more than the DefaultMetrics?
	keep := false
	for _, m := range DefaultMetrics {
		if b.addToCount(m, s, tags) {
			keep = true
		}
	}

	return keep
}

func (b *StatsBucket) addToCount(metric string, s *Span, tags *[]Tag) bool {
	ckey := CountKey(metric, *tags)

	_, ok := b.Counts[ckey]
	if !ok {
		b.Counts[ckey] = NewCount(metric, tags, b.Eps)
	}

	return b.Counts[ckey].Add(s)
}