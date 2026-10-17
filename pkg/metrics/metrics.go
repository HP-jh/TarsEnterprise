package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Registry Prometheus-style 指标注册表
type Registry struct {
	counters   map[string]*int64
	histograms map[string][]float64 // latencies in ms
	mu         sync.RWMutex
}

var defaultRegistry = &Registry{
	counters:   make(map[string]*int64),
	histograms: make(map[string][]float64),
}

func Start() {
	// Register default counters
	defaultRegistry.counter("te_agent_calls_total")
	defaultRegistry.counter("te_agent_errors_total")
}

func (r *Registry) counter(name string) *int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.counters[name]; ok {
		return v
	}
	var v int64
	r.counters[name] = &v
	return &v
}

func RecordAgentCall(agentID, clientID string) {
	atomic.AddInt64(defaultRegistry.counter("te_agent_calls_total"), 1)
	key := "te_agent_latency_" + agentID
	defaultRegistry.mu.Lock()
	defaultRegistry.histograms[key] = append(defaultRegistry.histograms[key], float64(time.Now().UnixNano()))
	defaultRegistry.mu.Unlock()
}

func RecordAgentError(agentID string) {
	atomic.AddInt64(defaultRegistry.counter("te_agent_errors_total"), 1)
}

func HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		defaultRegistry.mu.RLock()
		defer defaultRegistry.mu.RUnlock()
		for name, ptr := range defaultRegistry.counters {
			fmt.Fprintf(w, "# TYPE %s counter\n", name)
			fmt.Fprintf(w, "%s %d\n", name, atomic.LoadInt64(ptr))
		}
		for name, vals := range defaultRegistry.histograms {
			fmt.Fprintf(w, "# TYPE %s histogram\n", name)
			fmt.Fprintf(w, "%s_count %d\n", name, len(vals))
		}
	}
}

// TopAgents 返回调用量 Top-N 的智能体
func TopAgents(n int) []AgentStat {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	var stats []AgentStat
	for name := range defaultRegistry.histograms {
		if len(name) > 18 && name[:18] == "te_agent_latency_" {
			agentID := name[18:]
			var calls int64
			if ptr, ok := defaultRegistry.counters["te_agent_calls_total"]; ok {
				calls = atomic.LoadInt64(ptr)
			}
			stats = append(stats, AgentStat{
				AgentID: agentID,
				Calls:   calls,
				P50:     percentile(defaultRegistry.histograms[name], 0.5),
				P95:     percentile(defaultRegistry.histograms[name], 0.95),
			})
		}
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Calls > stats[j].Calls
	})
	if n > 0 && n < len(stats) {
		stats = stats[:n]
	}
	return stats
}

type AgentStat struct {
	AgentID string  `json:"agent_id"`
	Calls   int64   `json:"calls"`
	P50     float64 `json:"p50"`
	P95     float64 `json:"p95"`
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}
