package receiver

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func MetricHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")

	var metricData colmetricpb.ExportMetricsServiceRequest

	switch {
	case strings.Contains(contentType, "application/json"):
		err = protojson.Unmarshal(body, &metricData)
		if err != nil {
			http.Error(w, "failed to parse json trace data", http.StatusBadRequest)
			return
		}
	case strings.Contains(contentType, "application/x-protobuf"),
		strings.Contains(contentType, "application/protobuf"):

		err = proto.Unmarshal(body, &metricData)
		if err != nil {
			http.Error(w, "failed to parse protobuf trace data", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}

	processMetricData(&metricData)
}

func processMetricData(metricData *colmetricpb.ExportMetricsServiceRequest) {
	now := time.Now().Format("2006-01-02 15:04:05.000")
	reset := "\033[0m"
	bold := "\033[1m"
	dim := "\033[2m"
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"

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

	numValStr := func(dp interface {
		GetAsDouble() float64
		GetAsInt() int64
	}) string {
		if d := dp.GetAsDouble(); d != 0 {
			return fmt.Sprintf("%.2f", d)
		}
		return fmt.Sprintf("%d", dp.GetAsInt())
	}

	hr := strings.Repeat("─", 78)
	fmt.Printf("\n%s┌%s%s┐%s\n", bold, hr, reset, bold)
	fmt.Printf("%s│%s  METRIC BATCH · %s%s│%s\n", bold, reset, now, bold, reset)
	fmt.Printf("%s└%s%s┘%s\n\n", bold, hr, reset, bold)

	if len(metricData.ResourceMetrics) == 0 {
		fmt.Printf("  %s(empty)%s\n\n", dim, reset)
		_ = protojson.Format
		return
	}

	for _, rm := range metricData.ResourceMetrics {
		fmt.Printf("  %sResource%s  ", bold, reset)
		if rm.Resource != nil {
			svc, ver, env := "", "", ""
			for _, a := range rm.Resource.Attributes {
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

		for _, sm := range rm.ScopeMetrics {
			if sm.Scope != nil && sm.Scope.Name != "" {
				fmt.Printf("  %sScope%s    %s %s\n\n", dim, reset, sm.Scope.Name, sm.Scope.Version)
			}

			for _, m := range sm.Metrics {
				typeTag, typeColor := "", dim
				switch {
				case m.GetGauge() != nil:
					typeTag = "Gauge "
					typeColor = green
				case m.GetSum() != nil:
					if m.GetSum().GetIsMonotonic() {
						typeTag = "Sum:↑ "
					} else {
						typeTag = "Sum   "
					}
					typeColor = cyan
				case m.GetHistogram() != nil:
					typeTag = "Histo "
					typeColor = yellow
				default:
					typeTag = "?     "
				}

				unit := ""
				if m.Unit != "" {
					unit = fmt.Sprintf(" %s·%s %s", dim, reset, m.Unit)
				}

				fmt.Printf("  %s[%s]%s %s%s%s%s%s\n",
					typeColor, typeTag, reset,
					bold, m.Name, reset,
					unit, dim)

				if m.Description != "" {
					fmt.Printf("         %s%s%s\n", dim, m.Description, reset)
				}

				switch {
				case m.GetGauge() != nil:
					for _, dp := range m.GetGauge().GetDataPoints() {
						val := numValStr(dp)
						var attrPairs []string
						for _, a := range dp.GetAttributes() {
							attrPairs = append(attrPairs, fmt.Sprintf("%s%s%s", dim, anyValStr(a.Value), reset))
						}
						fmt.Printf("         %s%s%s  %s\n", bold, val, reset, strings.Join(attrPairs, " "))
					}

				case m.GetSum() != nil:
					for _, dp := range m.GetSum().GetDataPoints() {
						val := numValStr(dp)
						var attrPairs []string
						for _, a := range dp.GetAttributes() {
							attrPairs = append(attrPairs, fmt.Sprintf("%s%s%s:%s %s", dim, a.Key, reset, bold, anyValStr(a.Value), reset))
						}
						fmt.Printf("         %s%s%s  %s\n", bold, val, reset, strings.Join(attrPairs, "  "))
					}

				case m.GetHistogram() != nil:
					for _, dp := range m.GetHistogram().GetDataPoints() {
						var attrPairs []string
						for _, a := range dp.GetAttributes() {
							attrPairs = append(attrPairs, fmt.Sprintf("%s%s%s:%s %s", dim, a.Key, reset, bold, anyValStr(a.Value), reset))
						}
						fmt.Printf("         %s\n", strings.Join(attrPairs, "  "))

						parts := []string{fmt.Sprintf("count:%s%d%s", bold, dp.Count, reset)}
						if dp.Sum != nil {
							parts = append(parts, fmt.Sprintf("sum:%s%.2f%s", bold, *dp.Sum, reset))
						}
						if dp.Min != nil {
							parts = append(parts, fmt.Sprintf("min:%s%.2f%s", bold, *dp.Min, reset))
						}
						if dp.Max != nil {
							parts = append(parts, fmt.Sprintf("max:%s%.2f%s", bold, *dp.Max, reset))
						}
						fmt.Printf("         %s\n", strings.Join(parts, "  "))

						bounds := dp.ExplicitBounds
						counts := dp.BucketCounts
						maxCount := uint64(1)
						for _, c := range counts {
							if c > maxCount {
								maxCount = c
							}
						}
						barW := 25

						for i, c := range counts {
							var boundStr string
							if i < len(bounds) {
								boundStr = fmt.Sprintf("%s%7.1f%s", bold, bounds[i], reset)
							} else {
								boundStr = fmt.Sprintf("%s    inf%s", dim, reset)
							}
							barLen := int(float64(c) / float64(maxCount) * float64(barW))
							if c > 0 && barLen == 0 {
								barLen = 1
							}
							bar := strings.Repeat("█", barLen)
							fmt.Printf("        %s │ %s%4d%s  %s%s%s\n", boundStr, bold, c, reset, yellow, bar, reset)
						}
					}
				}
				fmt.Println()
			}
		}
	}
	_ = protojson.Format
}
