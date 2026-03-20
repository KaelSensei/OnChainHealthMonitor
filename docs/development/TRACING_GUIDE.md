# Distributed Tracing Guide

## What Is Distributed Tracing?

When a single user request flows through multiple microservices - say from the API, to the analyzer, to the collector - it's hard to know where time is being spent or where something went wrong. Distributed tracing solves this by attaching a unique trace ID to each request and recording every step it takes as a series of **spans**. You end up with a timeline showing exactly which service did what, for how long, and in what order - like a flight plan for your data.

---

## Opening the Jaeger UI

Once the stack is running (`docker compose up`), open your browser and navigate to:

```
http://localhost:16686
```

You'll land on the Jaeger search page.

---

## Searching for Traces

1. In the **Service** dropdown on the left, select a service (e.g. `onchain-api`, `onchain-analyzer`).
2. Optionally set a time range or operation filter.
3. Click **Find Traces**.
4. A list of matching traces appears - each row is one end-to-end request.

---

## Reading a Trace

Click any trace to open the timeline view. Here's what you'll see:

| Concept | What It Means |
|---|---|
| **Trace** | The full journey of one request across all services |
| **Span** | A single unit of work within a trace (one function call, one HTTP request, one DB query) |
| **Duration** | How long that span took - shown as a colored bar |
| **Parent span** | Spans nest inside each other, showing causality (which call triggered which) |
| **Attributes / Tags** | Key-value metadata attached to a span (HTTP status code, error message, DeFi protocol name, etc.) |

Wide bars = slow operations. Nested bars = a call chain. Red spans = errors.

---

## How the OTel Collector Works

The OpenTelemetry Collector acts as a **pipeline between your services and Jaeger**:

```
Go service  →  OTel Collector (otel-collector:4317)  →  Jaeger (jaeger:4317)  →  Jaeger UI
```

1. Your Go service sends spans over OTLP/gRPC to `otel-collector:4317`.
2. The collector **batches** them (reduces network chatter) and forwards to Jaeger.
3. Jaeger stores and indexes the traces.
4. You query them in the UI at `localhost:16686`.

The collector config lives in `observability/otel/otel-collector-config.yaml`. It also emits a `logging` exporter so you can see raw span data in the collector's container logs - useful for debugging.

---

## Adding a New Span to an Existing Service

Below is a minimal example using the OpenTelemetry Go SDK. First, make sure your service has the dependencies:

```bash
go get go.opentelemetry.io/otel \
       go.opentelemetry.io/otel/trace \
       go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc \
       go.opentelemetry.io/otel/sdk/trace
```

### Bootstrap a tracer (once, at startup)

```go
package main

import (
    "context"
    "log"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func initTracer(ctx context.Context) func() {
    endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if endpoint == "" {
        endpoint = "otel-collector:4317"
    }

    conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("failed to connect to OTel Collector: %v", err)
    }

    exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
    if err != nil {
        log.Fatalf("failed to create trace exporter: %v", err)
    }

    res := resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName("onchain-collector"), // change per service
    )

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )
    otel.SetTracerProvider(tp)

    return func() { _ = tp.Shutdown(ctx) }
}
```

Call it in `main()`:

```go
shutdown := initTracer(ctx)
defer shutdown()
```

### Creating a span

```go
tracer := otel.Tracer("onchain-collector")

func fetchProtocolData(ctx context.Context, protocolID string) (Data, error) {
    ctx, span := tracer.Start(ctx, "fetchProtocolData")
    defer span.End()

    // Add useful attributes
    span.SetAttributes(
        attribute.String("protocol.id", protocolID),
    )

    data, err := callDownstream(ctx, protocolID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return Data{}, err
    }

    return data, nil
}
```

Pass `ctx` through your call chain so child spans automatically nest under the parent trace.

---

## Tips

- Set `OTEL_SERVICE_NAME` env var to give your service a readable name in Jaeger.
- Use `span.SetAttributes()` to attach DeFi-specific context (protocol name, chain, TVL value).
- Check the OTel Collector logs (`docker compose logs otel-collector`) if spans aren't appearing in Jaeger.
- The zpages debug endpoint is available at `http://localhost:55679/debug/tracez`.
