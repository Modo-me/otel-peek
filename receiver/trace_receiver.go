package receiver

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TraceHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")

	var traceData coltracepb.ExportTraceServiceRequest

	switch {
	case strings.Contains(contentType, "application/json"):
		err = protojson.Unmarshal(body, &traceData)
		if err != nil {
			http.Error(w, "failed to parse json trace data", http.StatusBadRequest)
			return
		}
	case strings.Contains(contentType, "application/x-protobuf"),
		strings.Contains(contentType, "application/protobuf"):

		err = proto.Unmarshal(body, &traceData)
		if err != nil {
			http.Error(w, "failed to parse protobuf trace data", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
	processTraceData(&traceData)
}

func processTraceData(traceData *coltracepb.ExportTraceServiceRequest) {
	now := time.Now().Format("2006-01-02 15:04:05.000")
	reset := "\033[0m"
	bold := "\033[1m"
	dim := "\033[2m"
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	magenta := "\033[35m"

	trunc := func(s string, n int) string {
		if len(s) > n {
			return s[:n-1] + "…"
		}
		return s
	}

	anyValStr := func(v interface {
		GetStringValue() string
		GetIntValue() int64
		GetDoubleValue() float64
		GetBoolValue() bool
	}) string {
		if s := v.GetStringValue(); s != "" {
			return s
		}
		if v.GetBoolValue() {
			return "true"
		}
		if d := v.GetDoubleValue(); d != 0 {
			return fmt.Sprintf("%.2f", d)
		}
		if i := v.GetIntValue(); i != 0 {
			return fmt.Sprintf("%d", i)
		}
		return "-"
	}

	hr := strings.Repeat("─", 78)
	fmt.Printf("\n%s┌%s%s┐%s\n", bold, hr, reset, bold)
	fmt.Printf("%s│%s  TRACE BATCH · %s%s│%s\n", bold, reset, now, bold, reset)
	fmt.Printf("%s└%s%s┘%s\n\n", bold, hr, reset, bold)

	if len(traceData.ResourceSpans) == 0 {
		fmt.Printf("  %s(empty)%s\n\n", dim, reset)
		_ = protojson.Format
		return
	}

	for _, rs := range traceData.ResourceSpans {
		fmt.Printf("  %sResource%s  ", bold, reset)
		if rs.Resource != nil {
			svc, ver, env := "", "", ""
			for _, a := range rs.Resource.Attributes {
				switch a.Key {
				case "service.name":
					svc = a.Value.GetStringValue()
				case "service.version":
					ver = a.Value.GetStringValue()
				case "deployment.environment":
					env = a.Value.GetStringValue()
				}
			}
			var parts []string
			if svc != "" {
				parts = append(parts, svc)
			}
			if ver != "" {
				parts = append(parts, ver)
			}
			if env != "" {
				parts = append(parts, env)
			}
			fmt.Print(strings.Join(parts, "  "))
		}
		fmt.Println()

		for _, ss := range rs.ScopeSpans {
			if ss.Scope != nil && ss.Scope.Name != "" {
				fmt.Printf("  %sScope%s    %s %s\n", dim, reset, ss.Scope.Name, ss.Scope.Version)
			}
			fmt.Println()

			for _, s := range ss.Spans {
				durMs := float64(s.EndTimeUnixNano-s.StartTimeUnixNano) / 1e6

				kindStr := strings.TrimPrefix(s.Kind.String(), "SPAN_KIND_")
				kindColor := dim
				switch s.Kind.String() {
				case "SPAN_KIND_SERVER":
					kindColor = cyan
				case "SPAN_KIND_CLIENT":
					kindColor = magenta
				case "SPAN_KIND_CONSUMER", "SPAN_KIND_PRODUCER":
					kindColor = yellow
				}

				statusStr := "?"
				statusColor := dim
				if s.Status != nil {
					statusStr = strings.TrimPrefix(s.Status.Code.String(), "STATUS_CODE_")
					switch s.Status.Code.String() {
					case "STATUS_CODE_OK":
						statusColor = green
					case "STATUS_CODE_ERROR":
						statusColor = red
					}
				}

				isRoot := true
				for _, b := range s.ParentSpanId {
					if b != 0 {
						isRoot = false
						break
					}
				}

				prefix := "  ▸"
				attrPrefix := "   │"
				if !isRoot {
					prefix = "    ├─"
					attrPrefix = "    │ "
				}

				fmt.Printf("%s %s%-44s%s %s%-12s%s %s%-4s%s %s%7.1fms%s\n",
					prefix, bold, trunc(s.Name, 44), reset,
					kindColor, kindStr, reset,
					statusColor, statusStr, reset,
					dim, durMs, reset)

				for _, a := range s.Attributes {
					fmt.Printf("%s %s%s:%s %s\n", attrPrefix, dim, a.Key, reset, anyValStr(a.Value))
				}
				for _, ev := range s.Events {
					evOff := float64(ev.TimeUnixNano-s.StartTimeUnixNano) / 1e6
					fmt.Printf("    ◆ %s%s%s  %s+%.1fms%s\n", yellow, ev.Name, reset, dim, evOff, reset)
					for _, a := range ev.Attributes {
						fmt.Printf("      %s%s:%s %s\n", dim, a.Key, reset, anyValStr(a.Value))
					}
				}
				fmt.Println()
			}
		}
	}
	_ = protojson.Format
}
