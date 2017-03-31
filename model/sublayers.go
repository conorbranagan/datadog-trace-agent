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

// ComputeSublayers extracts sublayer values by type and service for a trace
func ComputeSublayers(trace *Trace) []SublayerValue {
	typeDuration := make(map[string]float64)
	serviceDuration := make(map[string]float64)

	childrenMap := trace.ChildrenMap()

	for i := range *trace {
		span := &(*trace)[i]

		children := childrenMap[span.SpanID]

		// In-place filtering
		nonAsyncChildren := Spans(children[:0])
		for _, child := range children {
			if child.End() <= span.End() {
				nonAsyncChildren = append(nonAsyncChildren, child)
			}
		}

		duration := span.Duration - nonAsyncChildren.CoveredDuration()

		typeDuration[span.Type] += float64(duration)
		serviceDuration[span.Service] += float64(duration)
	}

	// Generate sublayers values
	var values []SublayerValue

	for spanType, duration := range typeDuration {
		value := SublayerValue{
			Metric: "_sublayers.duration.by_type",
			Tag:    Tag{"sublayer_type", spanType},
			Value:  duration,
		}

		values = append(values, value)
	}

	for service, duration := range serviceDuration {
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

	return values
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
