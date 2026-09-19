package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var tracer = otel.Tracer("service-b")

func main() {
	ctx := context.Background()

	tp, err := setupJaegerTracer(ctx)
	if err != nil {
		log.Fatalf("Jaeger tracer setup error: %v", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Printf("Jaeger tracer shutdown error: %v", err)
		}
	}()

	otel.SetTextMapPropagator(propagation.TraceContext{})

	mux := http.NewServeMux()
	mux.Handle("/random-num", otelhttp.NewHandler(http.HandlerFunc(getRandomNum), "GET /random-num"))

	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("Web server start error: %v", err)
	}
}

func setupJaegerTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("simplest-collector.default.svc.cluster.local:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlptracegrpc.New call error: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceName("service-b")),
	)
	if err != nil {
		return nil, fmt.Errorf("resource.New call error: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp, nil
}

func getRandomNum(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "generate-random-num")

	defer span.End()

	n := rand.Intn(100) + 1
	_, err := fmt.Fprint(w, n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
