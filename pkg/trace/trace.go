package trace

import (
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var Tracer trace.Tracer

func Init(serviceName string) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	Tracer = otel.Tracer(serviceName)
	log.Printf("[Trace] Initialized tracer for: %s", serviceName)
}

func GetTracer() trace.Tracer {
	if Tracer == nil { Init("unknown") }
	return Tracer
}
