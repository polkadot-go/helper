package core

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	counters   map[string]*int64
	gauges     map[string]*int64
	histograms map[string]*Histogram
	mu         sync.RWMutex
}

type Histogram struct {
	values []float64
	mu     sync.Mutex
}

var metrics = &Metrics{
	counters:   make(map[string]*int64),
	gauges:     make(map[string]*int64),
	histograms: make(map[string]*Histogram),
}

func IncrCounter(name string) {
	metrics.mu.RLock()
	counter, ok := metrics.counters[name]
	metrics.mu.RUnlock()

	if !ok {
		metrics.mu.Lock()
		counter = new(int64)
		metrics.counters[name] = counter
		metrics.mu.Unlock()
	}

	atomic.AddInt64(counter, 1)
}

func SetGauge(name string, value int64) {
	metrics.mu.RLock()
	gauge, ok := metrics.gauges[name]
	metrics.mu.RUnlock()

	if !ok {
		metrics.mu.Lock()
		gauge = new(int64)
		metrics.gauges[name] = gauge
		metrics.mu.Unlock()
	}

	atomic.StoreInt64(gauge, value)
}

func RecordDuration(name string, start time.Time) {
	RecordValue(name, float64(time.Since(start).Microseconds()))
}

func RecordValue(name string, value float64) {
	metrics.mu.RLock()
	hist, ok := metrics.histograms[name]
	metrics.mu.RUnlock()

	if !ok {
		metrics.mu.Lock()
		hist = &Histogram{}
		metrics.histograms[name] = hist
		metrics.mu.Unlock()
	}

	hist.mu.Lock()
	hist.values = append(hist.values, value)
	if len(hist.values) > 10000 {
		hist.values = hist.values[1:]
	}
	hist.mu.Unlock()
}

func GetMetrics() map[string]interface{} {
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	result := make(map[string]interface{})

	for name, counter := range metrics.counters {
		result["counter."+name] = atomic.LoadInt64(counter)
	}

	for name, gauge := range metrics.gauges {
		result["gauge."+name] = atomic.LoadInt64(gauge)
	}

	for name, hist := range metrics.histograms {
		hist.mu.Lock()
		if len(hist.values) > 0 {
			sum := 0.0
			for _, v := range hist.values {
				sum += v
			}
			result["histogram."+name+".avg"] = sum / float64(len(hist.values))
			result["histogram."+name+".count"] = len(hist.values)
		}
		hist.mu.Unlock()
	}

	return result
}
