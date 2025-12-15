package goadstc

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics defines the interface for collecting operational metrics.
// Implementations can export metrics to various backends (Prometheus, StatsD, etc.).
type Metrics interface {
	// Connection metrics
	ConnectionAttempts()
	ConnectionSuccesses()
	ConnectionFailures()
	ConnectionActive(active bool)
	Reconnections()

	// Operation metrics
	OperationStarted(operation string)
	OperationCompleted(operation string, duration time.Duration, err error)

	// Data transfer metrics
	BytesSent(bytes int64)
	BytesReceived(bytes int64)

	// Notification metrics
	NotificationReceived()
	NotificationDropped()
	SubscriptionsActive(count int)

	// Error metrics
	ErrorOccurred(category ErrorCategory, operation string)

	// Health metrics
	HealthCheckStarted()
	HealthCheckCompleted(success bool)
}

// noopMetrics implements Metrics with no-op operations for minimal overhead.
type noopMetrics struct{}

func (n *noopMetrics) ConnectionAttempts()                                                    {}
func (n *noopMetrics) ConnectionSuccesses()                                                   {}
func (n *noopMetrics) ConnectionFailures()                                                    {}
func (n *noopMetrics) ConnectionActive(active bool)                                           {}
func (n *noopMetrics) Reconnections()                                                         {}
func (n *noopMetrics) OperationStarted(operation string)                                      {}
func (n *noopMetrics) OperationCompleted(operation string, duration time.Duration, err error) {}
func (n *noopMetrics) BytesSent(bytes int64)                                                  {}
func (n *noopMetrics) BytesReceived(bytes int64)                                              {}
func (n *noopMetrics) NotificationReceived()                                                  {}
func (n *noopMetrics) NotificationDropped()                                                   {}
func (n *noopMetrics) SubscriptionsActive(count int)                                          {}
func (n *noopMetrics) ErrorOccurred(category ErrorCategory, operation string)                 {}
func (n *noopMetrics) HealthCheckStarted()                                                    {}
func (n *noopMetrics) HealthCheckCompleted(success bool)                                      {}

var (
	// DefaultMetrics is a no-op metrics collector to minimize overhead when metrics are not configured.
	DefaultMetrics Metrics = &noopMetrics{}
)

// InMemoryMetrics provides a simple in-memory metrics collector for testing and debugging.
type InMemoryMetrics struct {
	mu sync.RWMutex

	// Connection metrics
	ConnectionAttemptsCount  atomic.Int64
	ConnectionSuccessesCount atomic.Int64
	ConnectionFailuresCount  atomic.Int64
	ConnectionActiveState    atomic.Bool
	ReconnectionsCount       atomic.Int64

	// Operation metrics
	OperationCounts    map[string]*atomic.Int64
	OperationDurations map[string][]time.Duration
	OperationErrors    map[string]*atomic.Int64

	// Data transfer metrics
	BytesSentCount     atomic.Int64
	BytesReceivedCount atomic.Int64

	// Notification metrics
	NotificationsReceivedCount atomic.Int64
	NotificationsDroppedCount  atomic.Int64
	SubscriptionsActiveCount   atomic.Int64

	// Error metrics
	ErrorsByCategory  map[ErrorCategory]*atomic.Int64
	ErrorsByOperation map[string]*atomic.Int64

	// Health metrics
	HealthChecksStartedCount atomic.Int64
	HealthChecksSuccessCount atomic.Int64
	HealthChecksFailureCount atomic.Int64
}

// NewInMemoryMetrics creates a new in-memory metrics collector.
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		OperationCounts:    make(map[string]*atomic.Int64),
		OperationDurations: make(map[string][]time.Duration),
		OperationErrors:    make(map[string]*atomic.Int64),
		ErrorsByCategory:   make(map[ErrorCategory]*atomic.Int64),
		ErrorsByOperation:  make(map[string]*atomic.Int64),
	}
}

func (m *InMemoryMetrics) ConnectionAttempts() {
	m.ConnectionAttemptsCount.Add(1)
}

func (m *InMemoryMetrics) ConnectionSuccesses() {
	m.ConnectionSuccessesCount.Add(1)
}

func (m *InMemoryMetrics) ConnectionFailures() {
	m.ConnectionFailuresCount.Add(1)
}

func (m *InMemoryMetrics) ConnectionActive(active bool) {
	m.ConnectionActiveState.Store(active)
}

func (m *InMemoryMetrics) Reconnections() {
	m.ReconnectionsCount.Add(1)
}

func (m *InMemoryMetrics) OperationStarted(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.OperationCounts[operation]; !exists {
		m.OperationCounts[operation] = &atomic.Int64{}
	}
	m.OperationCounts[operation].Add(1)
}

func (m *InMemoryMetrics) OperationCompleted(operation string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Track duration
	m.OperationDurations[operation] = append(m.OperationDurations[operation], duration)

	// Track errors
	if err != nil {
		if _, exists := m.OperationErrors[operation]; !exists {
			m.OperationErrors[operation] = &atomic.Int64{}
		}
		m.OperationErrors[operation].Add(1)
	}
}

func (m *InMemoryMetrics) BytesSent(bytes int64) {
	m.BytesSentCount.Add(bytes)
}

func (m *InMemoryMetrics) BytesReceived(bytes int64) {
	m.BytesReceivedCount.Add(bytes)
}

func (m *InMemoryMetrics) NotificationReceived() {
	m.NotificationsReceivedCount.Add(1)
}

func (m *InMemoryMetrics) NotificationDropped() {
	m.NotificationsDroppedCount.Add(1)
}

func (m *InMemoryMetrics) SubscriptionsActive(count int) {
	m.SubscriptionsActiveCount.Store(int64(count))
}

func (m *InMemoryMetrics) ErrorOccurred(category ErrorCategory, operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.ErrorsByCategory[category]; !exists {
		m.ErrorsByCategory[category] = &atomic.Int64{}
	}
	m.ErrorsByCategory[category].Add(1)

	if _, exists := m.ErrorsByOperation[operation]; !exists {
		m.ErrorsByOperation[operation] = &atomic.Int64{}
	}
	m.ErrorsByOperation[operation].Add(1)
}

func (m *InMemoryMetrics) HealthCheckStarted() {
	m.HealthChecksStartedCount.Add(1)
}

func (m *InMemoryMetrics) HealthCheckCompleted(success bool) {
	if success {
		m.HealthChecksSuccessCount.Add(1)
	} else {
		m.HealthChecksFailureCount.Add(1)
	}
}

// Snapshot returns a copy of current metrics for reporting.
func (m *InMemoryMetrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MetricsSnapshot{
		ConnectionAttempts:    m.ConnectionAttemptsCount.Load(),
		ConnectionSuccesses:   m.ConnectionSuccessesCount.Load(),
		ConnectionFailures:    m.ConnectionFailuresCount.Load(),
		ConnectionActive:      m.ConnectionActiveState.Load(),
		Reconnections:         m.ReconnectionsCount.Load(),
		BytesSent:             m.BytesSentCount.Load(),
		BytesReceived:         m.BytesReceivedCount.Load(),
		NotificationsReceived: m.NotificationsReceivedCount.Load(),
		NotificationsDropped:  m.NotificationsDroppedCount.Load(),
		SubscriptionsActive:   m.SubscriptionsActiveCount.Load(),
		HealthChecksStarted:   m.HealthChecksStartedCount.Load(),
		HealthChecksSuccess:   m.HealthChecksSuccessCount.Load(),
		HealthChecksFailure:   m.HealthChecksFailureCount.Load(),
		OperationCounts:       make(map[string]int64),
		OperationErrors:       make(map[string]int64),
		ErrorsByCategory:      make(map[ErrorCategory]int64),
		ErrorsByOperation:     make(map[string]int64),
	}

	for op, counter := range m.OperationCounts {
		snapshot.OperationCounts[op] = counter.Load()
	}

	for op, counter := range m.OperationErrors {
		snapshot.OperationErrors[op] = counter.Load()
	}

	for cat, counter := range m.ErrorsByCategory {
		snapshot.ErrorsByCategory[cat] = counter.Load()
	}

	for op, counter := range m.ErrorsByOperation {
		snapshot.ErrorsByOperation[op] = counter.Load()
	}

	return snapshot
}

// MetricsSnapshot represents a point-in-time snapshot of metrics.
type MetricsSnapshot struct {
	ConnectionAttempts    int64
	ConnectionSuccesses   int64
	ConnectionFailures    int64
	ConnectionActive      bool
	Reconnections         int64
	BytesSent             int64
	BytesReceived         int64
	NotificationsReceived int64
	NotificationsDropped  int64
	SubscriptionsActive   int64
	HealthChecksStarted   int64
	HealthChecksSuccess   int64
	HealthChecksFailure   int64
	OperationCounts       map[string]int64
	OperationErrors       map[string]int64
	ErrorsByCategory      map[ErrorCategory]int64
	ErrorsByOperation     map[string]int64
}

// WithMetrics returns a new option that sets the metrics collector for the client.
func WithMetrics(metrics Metrics) Option {
	return func(c *clientConfig) error {
		c.metrics = metrics
		return nil
	}
}
