package model

import (
	"fmt"
)

// SublayerValue is just a span-metric placeholder for a given sublayer val
type SublayerValue struct {
	Metric string
	Tag    Tag
	Value  float64
}

// String returns a description of a sublayer value.
func (v SublayerValue) String() string {
	if v.Tag.Name == "" && v.Tag.Value == "" {
		return fmt.Sprintf("SublayerValue{%q, %v}", v.Metric, v.Value)
	}

	return fmt.Sprintf("SublayerValue{%q, %v, %v}", v.Metric, v.Tag, v.Value)
}

// GoString returns a description of a sublayer value.
func (v SublayerValue) GoString() string {
	return v.String()
}

// ComputeSublayers compute sublayers durations and return both a slice of
// sublayer stats and a map of individual span durations.
func ComputeSublayers(trace *Trace) ([]SublayerValue, map[uint64]float64) {
	spanDuration := make(map[uint64]float64)
	typeDurations := make(map[string]float64)
	serviceDurations := make(map[string]float64)

	childrenMap := trace.ChildrenMap()

	for i := range *trace {
		span := &(*trace)[i]

		children := childrenMap[span.SpanID]

		// In-place filtering
		nonAsyncChildren := Spans(children[:0])
		for _, child := range children {
			end := child.End()
			if end < span.Start {
				// It should never happen, but better safe than sorry
				continue
			}

			if end <= span.End() {
				nonAsyncChildren = append(nonAsyncChildren, child)
			}
		}

		childrenDuration := nonAsyncChildren.CoveredDuration(span.Start)
		duration := float64(span.Duration - childrenDuration)

		spanDuration[span.SpanID] = duration

		if span.Type != "" {
			typeDurations[span.Type] += duration
		}

		if span.Service != "" {
			serviceDurations[span.Service] += duration
		}
	}

	// Generate sublayers values
	var values []SublayerValue

	for spanType, duration := range typeDurations {
		value := SublayerValue{
			Metric: "_sublayers.duration.by_type",
			Tag:    Tag{"sublayer_type", spanType},
			Value:  duration,
		}

		values = append(values, value)
	}

	for service, duration := range serviceDurations {
		value := SublayerValue{
			Metric: "_sublayers.duration.by_service",
			Tag:    Tag{"sublayer_service", service},
			Value:  duration,
		}

		values = append(values, value)
	}

	values = append(values, SublayerValue{
		Metric: "_sublayers.span_count",
		Value:  float64(len(*trace)),
	})

	return values, spanDuration
}

// SetSublayersOnSpan takes some sublayers and pins them on the given span.Metrics
func SetSublayersOnSpan(span *Span, values []SublayerValue) {
	if span.Metrics == nil {
		span.Metrics = make(map[string]float64, len(values))
	}

	for _, value := range values {
		name := value.Metric

		if value.Tag.Name != "" {
			name = name + "." + value.Tag.Name + ":" + value.Tag.Value
		}

		span.Metrics[name] = value.Value
	}
}

// SetSpanDurationsOnTrace sets a sublayer duration metric for each span in a
// trace.
func SetSpanDurationsOnTrace(trace Trace, durations map[uint64]float64) {
	for i := range trace {
		span := &trace[i]
		if span.Metrics == nil {
			span.Metrics = make(map[string]float64)
		}

		duration, ok := durations[span.SpanID]
		if !ok {
			continue
		}

		span.Metrics["_sublayers.duration"] = duration
	}
}
