# Otelpeek
### Use otelpeek to peek your OpenTelemetry data in a TUI
Otelpeek is an easy-to-use and extremely lightweight command-line tool designed to inspect OpenTelemetry data on any device, including edge devices, or to temporarily monitor your local telemetry data.

### Why Otelpeek?
In production or development environments, we often just need to temporarily inspect or verify OpenTelemetry data. For instance, after implementing OpenTelemetry metrics or traces in an application, you might want to quickly check if the telemetry is being exported successfully or get a rough overview of the data payload.

In such scenarios, setting up a full-blown standard observability backend (like Jaeger, Prometheus, or an Elastic stack) can feel complete overkill. For edge devices with constrained resources, otelpeek offers a much lighter and more efficient alternative for telemetry inspection.

### Getting Started

#### Installation

```shell
go install github.com/Modo-me/otel-peek@latest
```

#### Quick Start

Start otelpeek on a random available port — the actual address is printed to stdout on startup.

``` shell 
otelpeek
```

Start otelpeek on a specific port:

```shell
otelpeek -p 4318
```

This starts an HTTP server listening on `127.0.0.1:4318` with the following OTLP endpoints:

| Endpoint       | Description            |
|----------------|------------------------|
| `/v1/traces`   | OTLP trace ingestion   |
| `/v1/metrics`  | OTLP metric ingestion  |
| `/v1/logs`     | OTLP log ingestion     |

otelpeek accepts both `application/json` and `application/x-protobuf` content types.

### Preview
![截屏2026-05-18 14.30.43](assets/%E6%88%AA%E5%B1%8F2026-05-18%2014.30.43.png)



### Best Use Cases
#### Ideal scenarios for otelpeek:
- Ad-hoc Peeking: Temporarily capturing and inspecting observability data on the fly.

- Resource-Constrained Environments: Running on edge devices where performance and memory are limited.

#### Scenarios where otelpeek is NOT recommended:
- High-Volume/Long-Term Storage: Handling persistent, high-frequency OpenTelemetry data over extended periods. This is a job better suited for traditional, full-scale observability backends.