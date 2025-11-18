package api

import (
	"net/http"
	"runtime"
	"time"

	"haystack-lite/internal/storage"

	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	store     *storage.Store
	startTime time.Time
}

func NewMetricsHandler(store *storage.Store) *MetricsHandler {
	return &MetricsHandler{
		store:     store,
		startTime: time.Now(),
	}
}

func (h *MetricsHandler) Metrics(c *gin.Context) {
	status := h.store.Status()
	compactionStats := h.store.GetCompactionStats()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(h.startTime).Seconds()

	metrics := []string{
		"# HELP haystack_up Service is up",
		"# TYPE haystack_up gauge",
		"haystack_up 1",
		"",
		"# HELP haystack_uptime_seconds Service uptime in seconds",
		"# TYPE haystack_uptime_seconds counter",
		formatMetric("haystack_uptime_seconds", uptime),
		"",
		"# HELP haystack_files_total Total number of files",
		"# TYPE haystack_files_total gauge",
		formatMetric("haystack_files_total", status["total_files"]),
		"",
		"# HELP haystack_files_active Active files count",
		"# TYPE haystack_files_active gauge",
		formatMetric("haystack_files_active", status["active_files"]),
		"",
		"# HELP haystack_files_deleted Deleted files count",
		"# TYPE haystack_files_deleted gauge",
		formatMetric("haystack_files_deleted", status["deleted_files"]),
		"",
		"# HELP haystack_storage_bytes Total storage size in bytes",
		"# TYPE haystack_storage_bytes gauge",
		formatMetric("haystack_storage_bytes", status["total_size"]),
		"",
		"# HELP haystack_volumes_total Total number of volumes",
		"# TYPE haystack_volumes_total gauge",
		formatMetric("haystack_volumes_total", status["volume_count"]),
		"",
		"# HELP haystack_compaction_wasted_bytes Wasted space in bytes",
		"# TYPE haystack_compaction_wasted_bytes gauge",
		formatMetric("haystack_compaction_wasted_bytes", compactionStats["wasted_size"]),
		"",
		"# HELP haystack_compaction_wasted_ratio Wasted space ratio",
		"# TYPE haystack_compaction_wasted_ratio gauge",
		formatMetric("haystack_compaction_wasted_ratio", compactionStats["wasted_ratio"]),
		"",
		"# HELP haystack_memory_alloc_bytes Allocated memory in bytes",
		"# TYPE haystack_memory_alloc_bytes gauge",
		formatMetric("haystack_memory_alloc_bytes", m.Alloc),
		"",
		"# HELP haystack_memory_sys_bytes System memory in bytes",
		"# TYPE haystack_memory_sys_bytes gauge",
		formatMetric("haystack_memory_sys_bytes", m.Sys),
		"",
		"# HELP haystack_goroutines Number of goroutines",
		"# TYPE haystack_goroutines gauge",
		formatMetric("haystack_goroutines", runtime.NumGoroutine()),
		"",
	}

	c.String(http.StatusOK, joinMetrics(metrics))
}

func formatMetric(name string, value interface{}) string {
	return name + " " + toString(value)
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case int:
		return formatInt(int64(val))
	case int64:
		return formatInt(val)
	case uint32:
		return formatInt(int64(val))
	case uint64:
		return formatInt(int64(val))
	case float64:
		return formatFloat(val)
	default:
		return "0"
	}
}

func formatInt(v int64) string {
	return string(append([]byte{}, []byte(itoa(v))...))
}

func formatFloat(v float64) string {
	return string(append([]byte{}, []byte(ftoa(v))...))
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}

	neg := v < 0
	if neg {
		v = -v
	}

	var buf [20]byte
	i := len(buf) - 1

	for v > 0 {
		buf[i] = byte('0' + v%10)
		v /= 10
		i--
	}

	if neg {
		buf[i] = '-'
		i--
	}

	return string(buf[i+1:])
}

func ftoa(v float64) string {
	if v == 0 {
		return "0"
	}

	intPart := int64(v)
	fracPart := int64((v - float64(intPart)) * 1000000)

	if fracPart < 0 {
		fracPart = -fracPart
	}

	result := itoa(intPart) + "."

	fracStr := itoa(fracPart)
	for len(fracStr) < 6 {
		fracStr = "0" + fracStr
	}

	result += fracStr

	for len(result) > 0 && result[len(result)-1] == '0' {
		result = result[:len(result)-1]
	}
	if len(result) > 0 && result[len(result)-1] == '.' {
		result = result[:len(result)-1]
	}

	return result
}

func joinMetrics(metrics []string) string {
	result := ""
	for _, m := range metrics {
		result += m + "\n"
	}
	return result
}
