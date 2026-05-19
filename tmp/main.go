package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	metricsv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

const port = 62133

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func floatPtr(v float64) *float64 { return &v }

func NewTraceDataInstance() *coltracepb.ExportTraceServiceRequest {
	now := uint64(time.Now().UnixNano())
	startTime := now - uint64(50*time.Millisecond)

	traceID := randomBytes(16)
	rootSpanID := randomBytes(8)
	cacheSpanID := randomBytes(8)
	dbSpanID := randomBytes(8)

	return &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "demo-service"}}},
						{Key: "service.version", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "1.0.0"}}},
						{Key: "deployment.environment", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "production"}}},
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Scope: &commonv1.InstrumentationScope{
							Name:    "demo-instrumentation",
							Version: "1.0.0",
						},
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								SpanId:            rootSpanID,
								Name:              "GET /api/users",
								Kind:              tracev1.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: startTime,
								EndTimeUnixNano:   now,
								Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
								Attributes: []*commonv1.KeyValue{
									{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "GET"}}},
									{Key: "http.url", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "https://api.example.com/api/users"}}},
									{Key: "http.status_code", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 200}}},
									{Key: "http.route", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "/api/users"}}},
								},
								Events: []*tracev1.Span_Event{
									{
										TimeUnixNano: startTime + uint64(5*time.Millisecond),
										Name:         "cache.miss",
										Attributes: []*commonv1.KeyValue{
											{Key: "cache.key", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "users:list"}}},
										},
									},
									{
										TimeUnixNano: startTime + uint64(10*time.Millisecond),
										Name:         "db.query.start",
										Attributes: []*commonv1.KeyValue{
											{Key: "db.statement", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "SELECT * FROM users LIMIT 100"}}},
										},
									},
								},
							},
							{
								TraceId:           traceID,
								SpanId:            cacheSpanID,
								ParentSpanId:      rootSpanID,
								Name:              "cache.lookup",
								Kind:              tracev1.Span_SPAN_KIND_INTERNAL,
								StartTimeUnixNano: startTime + uint64(2*time.Millisecond),
								EndTimeUnixNano:   startTime + uint64(8*time.Millisecond),
								Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
								Attributes: []*commonv1.KeyValue{
									{Key: "cache.system", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "redis"}}},
									{Key: "cache.hit", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_BoolValue{BoolValue: false}}},
								},
							},
							{
								TraceId:           traceID,
								SpanId:            dbSpanID,
								ParentSpanId:      rootSpanID,
								Name:              "SELECT users",
								Kind:              tracev1.Span_SPAN_KIND_CLIENT,
								StartTimeUnixNano: startTime + uint64(10*time.Millisecond),
								EndTimeUnixNano:   startTime + uint64(45*time.Millisecond),
								Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
								Attributes: []*commonv1.KeyValue{
									{Key: "db.system", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "postgresql"}}},
									{Key: "db.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "users_db"}}},
									{Key: "db.statement", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "SELECT * FROM users LIMIT 100"}}},
									{Key: "db.rows_returned", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 42}}},
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewLogDataInstance() *collogpb.ExportLogsServiceRequest {
	now := uint64(time.Now().UnixNano())
	traceID := randomBytes(16)
	spanID := randomBytes(8)

	return &collogpb.ExportLogsServiceRequest{
		ResourceLogs: []*logsv1.ResourceLogs{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "demo-service"}}},
						{Key: "service.version", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "1.0.0"}}},
						{Key: "deployment.environment", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "production"}}},
					},
				},
				ScopeLogs: []*logsv1.ScopeLogs{
					{
						Scope: &commonv1.InstrumentationScope{
							Name:    "demo-logger",
							Version: "1.0.0",
						},
						LogRecords: []*logsv1.LogRecord{
							{
								TimeUnixNano:         now - uint64(10*time.Second),
								ObservedTimeUnixNano: now,
								SeverityNumber:       logsv1.SeverityNumber_SEVERITY_NUMBER_INFO,
								SeverityText:         "INFO",
								Body:                 &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "Request processed successfully"}},
								Attributes: []*commonv1.KeyValue{
									{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "GET"}}},
									{Key: "http.route", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "/api/users"}}},
									{Key: "http.status_code", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 200}}},
									{Key: "duration_ms", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: 48.5}}},
								},
								TraceId: traceID,
								SpanId:  spanID,
							},
							{
								TimeUnixNano:         now - uint64(5*time.Second),
								ObservedTimeUnixNano: now,
								SeverityNumber:       logsv1.SeverityNumber_SEVERITY_NUMBER_WARN,
								SeverityText:         "WARN",
								Body:                 &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "High memory usage detected"}},
								Attributes: []*commonv1.KeyValue{
									{Key: "memory.usage_percent", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: 85.3}}},
									{Key: "memory.threshold_percent", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: 80.0}}},
									{Key: "pod.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "demo-service-7d4f8b9c-x2k9m"}}},
								},
							},
							{
								TimeUnixNano:         now - uint64(1*time.Second),
								ObservedTimeUnixNano: now,
								SeverityNumber:       logsv1.SeverityNumber_SEVERITY_NUMBER_ERROR,
								SeverityText:         "ERROR",
								Body:                 &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "Connection timeout to database"}},
								Attributes: []*commonv1.KeyValue{
									{Key: "exception.type", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "TimeoutError"}}},
									{Key: "exception.message", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "context deadline exceeded after 30s"}}},
									{Key: "db.system", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "postgresql"}}},
									{Key: "db.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "users_db"}}},
									{Key: "db.operation", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "SELECT"}}},
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewMetricDataInstance() *colmetricpb.ExportMetricsServiceRequest {
	now := uint64(time.Now().UnixNano())
	startTime := now - uint64(60*time.Second)

	return &colmetricpb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricsv1.ResourceMetrics{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						{Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "demo-service"}}},
						{Key: "service.version", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "1.0.0"}}},
						{Key: "deployment.environment", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "production"}}},
					},
				},
				ScopeMetrics: []*metricsv1.ScopeMetrics{
					{
						Scope: &commonv1.InstrumentationScope{
							Name:    "demo-meter",
							Version: "1.0.0",
						},
						Metrics: []*metricsv1.Metric{
							{
								Name:        "cpu.temperature",
								Description: "Current CPU temperature",
								Unit:        "Cel",
								Data: &metricsv1.Metric_Gauge{
									Gauge: &metricsv1.Gauge{
										DataPoints: []*metricsv1.NumberDataPoint{
											{
												TimeUnixNano: now,
												Value:        &metricsv1.NumberDataPoint_AsDouble{AsDouble: 42.5},
												Attributes: []*commonv1.KeyValue{
													{Key: "cpu.core", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "core-0"}}},
												},
											},
											{
												TimeUnixNano: now,
												Value:        &metricsv1.NumberDataPoint_AsDouble{AsDouble: 43.1},
												Attributes: []*commonv1.KeyValue{
													{Key: "cpu.core", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "core-1"}}},
												},
											},
										},
									},
								},
							},
							{
								Name:        "http.server.requests",
								Description: "Total number of HTTP requests served",
								Unit:        "1",
								Data: &metricsv1.Metric_Sum{
									Sum: &metricsv1.Sum{
										DataPoints: []*metricsv1.NumberDataPoint{
											{
												StartTimeUnixNano: startTime,
												TimeUnixNano:      now,
												Value:             &metricsv1.NumberDataPoint_AsInt{AsInt: 1024},
												Attributes: []*commonv1.KeyValue{
													{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "GET"}}},
													{Key: "http.route", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "/api/users"}}},
													{Key: "http.status_code", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 200}}},
												},
											},
											{
												StartTimeUnixNano: startTime,
												TimeUnixNano:      now,
												Value:             &metricsv1.NumberDataPoint_AsInt{AsInt: 128},
												Attributes: []*commonv1.KeyValue{
													{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "POST"}}},
													{Key: "http.route", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "/api/users"}}},
													{Key: "http.status_code", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_IntValue{IntValue: 201}}},
												},
											},
										},
										AggregationTemporality: metricsv1.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
										IsMonotonic:            true,
									},
								},
							},
							{
								Name:        "http.server.request.duration",
								Description: "Duration of HTTP requests",
								Unit:        "ms",
								Data: &metricsv1.Metric_Histogram{
									Histogram: &metricsv1.Histogram{
										DataPoints: []*metricsv1.HistogramDataPoint{
											{
												StartTimeUnixNano: startTime,
												TimeUnixNano:      now,
												Count:             1152,
												Sum:               floatPtr(68925.6),
												BucketCounts:      []uint64{120, 380, 420, 180, 40, 10, 2},
												ExplicitBounds:    []float64{10, 25, 50, 100, 250, 500, 1000},
												Attributes: []*commonv1.KeyValue{
													{Key: "http.method", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "GET"}}},
													{Key: "http.route", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "/api/users"}}},
												},
												Min: floatPtr(0.5),
												Max: floatPtr(1523.8),
											},
										},
										AggregationTemporality: metricsv1.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func main() {
	var traceData = NewTraceDataInstance()
	var logData = NewLogDataInstance()
	var metricData = NewMetricDataInstance()
	traceBuf, err := proto.Marshal(traceData)
	if err != nil {
		panic(err)
	}

	logBuf, err := proto.Marshal(logData)
	if err != nil {
		panic(err)
	}

	metricBuf, err := proto.Marshal(metricData)
	if err != nil {
		panic(err)
	}

	logBody := bytes.NewReader(logBuf)
	logurl := fmt.Sprintf("http://localhost:%d/v1/logs", port)
	_, err = http.Post(logurl, "application/x-protobuf", logBody)
	if err != nil {
		panic(err)
	}

	metricBody := bytes.NewReader(metricBuf)
	metricUrl := fmt.Sprintf("http://localhost:%d/v1/metrics", port)
	_, err = http.Post(metricUrl, "application/x-protobuf", metricBody)
	if err != nil {
		panic(err)
	}

	traceBody := bytes.NewReader(traceBuf)
	traceUrl := fmt.Sprintf("http://localhost:%d/v1/traces", port)
	_, err = http.Post(traceUrl, "application/x-protobuf", traceBody)
	if err != nil {
		panic(err)
	}
}
