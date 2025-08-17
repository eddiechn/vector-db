package internal

import (
	"sync"
	"time"
)

// Vector represents a vector with an ID and its data
type Vector struct {
	ID   string    `json:"id"`
	Data []float32 `json:"data"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
}

// VectorMetadata contains additional metadata for vectors
type VectorMetadata struct {
	CreatedAt time.Time              `json:"created_at"`
	Tags      map[string]interface{} `json:"tags,omitempty"`
}

// DatabaseStats holds statistics about the database
type DatabaseStats struct {
	VectorCount    int64         `json:"vector_count"`
	Dimensions     int           `json:"dimensions"`
	IndexType      string        `json:"index_type"`
	SearchRequests int64         `json:"search_requests"`
	InsertRequests int64         `json:"insert_requests"`
	DeleteRequests int64         `json:"delete_requests"`
	AverageLatency time.Duration `json:"average_latency"`
	LastUpdated    time.Time     `json:"last_updated"`
	MemoryUsage    int64         `json:"memory_usage_bytes"`
}

// DistanceMetric represents different distance calculation methods
type DistanceMetric int

const (
	CosineSimilarity DistanceMetric = iota
	EuclideanDistance
	DotProduct
	ManhattanDistance
)

func (d DistanceMetric) String() string {
	switch d {
	case CosineSimilarity:
		return "cosine"
	case EuclideanDistance:
		return "euclidean"
	case DotProduct:
		return "dot_product"
	case ManhattanDistance:
		return "manhattan"
	default:
		return "unknown"
	}
}

// SearchRequest represents a search query
type SearchRequest struct {
	Vector         []float32              `json:"vector"`
	K              int                    `json:"k"`
	DistanceMetric DistanceMetric         `json:"distance_metric"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
}

// InsertRequest represents a vector insertion request
type InsertRequest struct {
	Vector   Vector                 `json:"vector"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// IndexConfig holds configuration for different index types
type IndexConfig struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// HNSWConfig holds HNSW-specific configuration
type HNSWConfig struct {
	M              int     `json:"m"`               // Maximum number of connections for each node
	EfConstruction int     `json:"ef_construction"` // Size of dynamic candidate list
	EfSearch       int     `json:"ef_search"`       // Size of dynamic candidate list during search
	MaxM           int     `json:"max_m"`           // Maximum connections for layer 0
	MaxM0          int     `json:"max_m0"`          // Maximum connections for higher layers
	Ml             float64 `json:"ml"`              // Level generation factor
}

// DefaultHNSWConfig returns a default HNSW configuration
func DefaultHNSWConfig() HNSWConfig {
	return HNSWConfig{
		M:              16,
		EfConstruction: 200,
		EfSearch:       50,
		MaxM:           16,
		MaxM0:          32,
		Ml:             1.0 / 2.303, // 1/ln(2)
	}
}

// Node represents a node in the HNSW graph
type HNSWNode struct {
	ID          string                 `json:"id"`
	Vector      []float32              `json:"vector"`
	Connections []map[string]bool      `json:"connections"` // connections per layer
	Level       int                    `json:"level"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PriorityQueue for efficient k-nearest neighbor search
type PriorityQueueItem struct {
	ID       string
	Distance float32
	Index    int
}

type PriorityQueue []*PriorityQueueItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Distance < pq[j].Distance
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

// Thread-safe statistics tracker
type StatsTracker struct {
	mu    sync.RWMutex
	stats DatabaseStats
}

func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		stats: DatabaseStats{
			LastUpdated: time.Now(),
			IndexType:   "hnsw",
		},
	}
}

func (s *StatsTracker) IncrementSearchRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.SearchRequests++
	s.stats.LastUpdated = time.Now()
}

func (s *StatsTracker) IncrementInsertRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.InsertRequests++
	s.stats.LastUpdated = time.Now()
}

func (s *StatsTracker) IncrementDeleteRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.DeleteRequests++
	s.stats.LastUpdated = time.Now()
}

func (s *StatsTracker) UpdateVectorCount(count int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.VectorCount = count
	s.stats.LastUpdated = time.Now()
}

func (s *StatsTracker) UpdateDimensions(dims int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Dimensions = dims
	s.stats.LastUpdated = time.Now()
}

func (s *StatsTracker) UpdateLatency(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Simple moving average (you could implement exponential moving average for production)
	if s.stats.AverageLatency == 0 {
		s.stats.AverageLatency = latency
	} else {
		s.stats.AverageLatency = (s.stats.AverageLatency + latency) / 2
	}
	s.stats.LastUpdated = time.Now()
}

func (s *StatsTracker) GetStats() DatabaseStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// EmbedRequest represents a request to embed text and store as vector
type EmbedRequest struct {
	ID       string                 `json:"id"`
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TextSearchRequest represents a request to search using text input
type TextSearchRequest struct {
	Text string `json:"text"`
	K    int    `json:"k,omitempty"`
}
