package receiver

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func LogHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	contentType := r.Header.Get("Content-Type")

	var logData collogpb.ExportLogsServiceRequest

	switch {
	case strings.Contains(contentType, "application/json"):
		err = protojson.Unmarshal(body, &logData)
		if err != nil {
			http.Error(w, "failed to parse json trace data", http.StatusBadRequest)
			return
		}
	case strings.Contains(contentType, "application/x-protobuf"),
		strings.Contains(contentType, "application/protobuf"):

		err = proto.Unmarshal(body, &logData)
		if err != nil {
			http.Error(w, "failed to parse protobuf trace data", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	processLogData(&logData)
}

func processLogData(logData *collogpb.ExportLogsServiceRequest) {
	now := time.Now().Format("2006-01-02 15:04:05.000")
	reset := "\033[0m"
	bold := "\033[1m"
	dim := "\033[2m"
	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	blue := "\033[34m"

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

	humanDur := func(d time.Duration) string {
		switch {
		case d >= time.Hour:
			return fmt.Sprintf("%.1fh", d.Hours())
		case d >= time.Minute:
			return fmt.Sprintf("%.1fm", d.Minutes())
		case d >= time.Second:
			return fmt.Sprintf("%.1fs", d.Seconds())
		case d >= time.Millisecond:
			return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
		default:
			return "0ms"
		}
	}

	hr := strings.Repeat("─", 78)
	fmt.Printf("\n%s┌%s%s┐%s\n", bold, hr, reset, bold)
	fmt.Printf("%s│%s  LOG BATCH · %s%s│%s\n", bold, reset, now, bold, reset)
	fmt.Printf("%s└%s%s┘%s\n\n", bold, hr, reset, bold)

	if len(logData.ResourceLogs) == 0 {
		fmt.Printf("  %s(empty)%s\n\n", dim, reset)
		_ = protojson.Format
		return
	}

	nowNs := uint64(time.Now().UnixNano())

	for _, rl := range logData.ResourceLogs {
		fmt.Printf("  %sResource%s  ", bold, reset)
		if rl.Resource != nil {
			svc, ver, env := "", "", ""
			for _, a := range rl.Resource.Attributes {
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

		for _, sl := range rl.ScopeLogs {
			if sl.Scope != nil && sl.Scope.Name != "" {
				fmt.Printf("  %sScope%s    %s %s\n\n", dim, reset, sl.Scope.Name, sl.Scope.Version)
			}

			for _, rec := range sl.LogRecords {
				sevStr := strings.TrimPrefix(rec.SeverityNumber.String(), "SEVERITY_NUMBER_")
				sevColor := dim
				switch {
				case strings.HasPrefix(sevStr, "INFO"):
					sevColor = green
				case strings.HasPrefix(sevStr, "WARN"):
					sevColor = yellow
				case strings.HasPrefix(sevStr, "ERROR"), strings.HasPrefix(sevStr, "FATAL"):
					sevColor = red
				case strings.HasPrefix(sevStr, "DEBUG"), strings.HasPrefix(sevStr, "TRACE"):
					sevColor = blue
				}

				ts := ""
				if rec.TimeUnixNano > 0 && rec.TimeUnixNano < nowNs {
					ago := time.Duration(nowNs - rec.TimeUnixNano)
					ts = fmt.Sprintf("%s%-6s%s ", dim, humanDur(ago), reset)
				}

				body := anyValStr(rec.Body)
				fmt.Printf("  %s%-6s%s %s%s%s\n", sevColor, sevStr, reset, ts, body, reset)

				for _, a := range rec.Attributes {
					fmt.Printf("   %s%s:%s %s\n", dim, a.Key, reset, anyValStr(a.Value))
				}
				if len(rec.EventName) > 0 {
					fmt.Printf("   %sevent:%s %s\n", dim, reset, rec.EventName)
				}
				fmt.Println()
			}
		}
	}
	_ = protojson.Format
}
