package internal

import (
	"container/heap"
	"math/rand"
	"sync"
	"time"
)

// HNSWIndex implements the Hierarchical Navigable Small World algorithm
type HNSWIndex struct {
	mu              sync.RWMutex
	nodes           map[string]*HNSWNode
	entryPoint      *HNSWNode
	config          HNSWConfig
	dimensions      int
	distanceFunc    DistanceMetric
	maxLevel        int
	levelMultiplier float64
	rng             *rand.Rand
}

// NewHNSWIndex creates a new HNSW index
func NewHNSWIndex(config HNSWConfig, dimensions int, distanceFunc DistanceMetric) *HNSWIndex {
	return &HNSWIndex{
		nodes:           make(map[string]*HNSWNode),
		config:          config,
		dimensions:      dimensions,
		distanceFunc:    distanceFunc,
		levelMultiplier: config.Ml,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Insert adds a new vector to the HNSW index
func (h *HNSWIndex) Insert(id string, vector []float32, metadata map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(vector) != h.dimensions {
		return &DimensionMismatchError{
			Expected: h.dimensions,
			Actual:   len(vector),
		}
	}

	// Generate random level for the new node
	level := h.getRandomLevel()

	// Create new node
	node := &HNSWNode{
		ID:          id,
		Vector:      make([]float32, len(vector)),
		Connections: make([]map[string]bool, level+1),
		Level:       level,
		Metadata:    metadata,
	}

	copy(node.Vector, vector)

	// Initialize connections for each level
	for i := 0; i <= level; i++ {
		node.Connections[i] = make(map[string]bool)
	}

	// If this is the first node, make it the entry point
	if h.entryPoint == nil {
		h.entryPoint = node
		h.nodes[id] = node
		h.maxLevel = level
		return nil
	}

	// Find entry points for insertion
	ep := []*HNSWNode{h.entryPoint}

	// Search from top level down to level+1
	for currentLevel := h.maxLevel; currentLevel > level; currentLevel-- {
		ep = h.searchLayer(vector, ep, 1, currentLevel)
	}

	// Insert from level down to 0
	for currentLevel := min(level, h.maxLevel); currentLevel >= 0; currentLevel-- {
		candidates := h.searchLayer(vector, ep, h.config.EfConstruction, currentLevel)

		// Select M neighbors
		maxConn := h.config.M
		if currentLevel == 0 {
			maxConn = h.config.MaxM0
		}

		selectedNeighbors := h.selectNeighbors(candidates, maxConn, vector, currentLevel)

		// Add bidirectional connections
		for _, neighbor := range selectedNeighbors {
			h.addConnection(node, neighbor, currentLevel)
			h.addConnection(neighbor, node, currentLevel)

			// Prune connections if necessary
			h.pruneConnections(neighbor, currentLevel)
		}

		ep = selectedNeighbors
	}

	// Update entry point if necessary
	if level > h.maxLevel {
		h.maxLevel = level
		h.entryPoint = node
	}

	h.nodes[id] = node
	return nil
}

// Search performs k-nearest neighbor search
func (h *HNSWIndex) Search(vector []float32, k int, ef int) ([]SearchResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(vector) != h.dimensions {
		return nil, &DimensionMismatchError{
			Expected: h.dimensions,
			Actual:   len(vector),
		}
	}

	if h.entryPoint == nil {
		return []SearchResult{}, nil
	}

	if ef < k {
		ef = k
	}

	// Search from top level down to level 1
	ep := []*HNSWNode{h.entryPoint}
	for currentLevel := h.maxLevel; currentLevel > 0; currentLevel-- {
		ep = h.searchLayer(vector, ep, 1, currentLevel)
	}

	// Search at level 0 with ef
	candidates := h.searchLayer(vector, ep, ef, 0)

	// Return top k results
	results := make([]SearchResult, 0, k)
	for i, node := range candidates {
		if i >= k {
			break
		}
		distance := CalculateDistance(vector, node.Vector, h.distanceFunc)
		results = append(results, SearchResult{
			ID:    node.ID,
			Score: distance,
		})
	}

	return results, nil
}

// Delete removes a vector from the index
func (h *HNSWIndex) Delete(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	node, exists := h.nodes[id]
	if !exists {
		return &VectorNotFoundError{ID: id}
	}

	// Remove all connections to this node
	for level := 0; level <= node.Level; level++ {
		for neighborID := range node.Connections[level] {
			if neighbor, exists := h.nodes[neighborID]; exists {
				delete(neighbor.Connections[level], id)
			}
		}
	}

	// If this was the entry point, find a new one
	if h.entryPoint == node {
		h.findNewEntryPoint()
	}

	delete(h.nodes, id)
	return nil
}

// searchLayer performs search at a specific layer
func (h *HNSWIndex) searchLayer(query []float32, entryPoints []*HNSWNode, ef int, level int) []*HNSWNode {
	visited := make(map[string]bool)
	candidates := &PriorityQueue{}
	w := &PriorityQueue{}

	heap.Init(candidates)
	heap.Init(w)

	// Initialize with entry points
	for _, ep := range entryPoints {
		distance := CalculateDistance(query, ep.Vector, h.distanceFunc)
		heap.Push(candidates, &PriorityQueueItem{
			ID:       ep.ID,
			Distance: distance,
		})
		heap.Push(w, &PriorityQueueItem{
			ID:       ep.ID,
			Distance: -distance, // Negative for max-heap behavior
		})
		visited[ep.ID] = true
	}

	for candidates.Len() > 0 {
		current := heap.Pop(candidates).(*PriorityQueueItem)

		// Check if we should stop (current distance > furthest in w)
		if w.Len() >= ef {
			furthest := (*w)[0]
			if current.Distance > -furthest.Distance {
				break
			}
		}

		currentNode := h.nodes[current.ID]

		// Check all connections at this level
		if level < len(currentNode.Connections) {
			for neighborID := range currentNode.Connections[level] {
				if !visited[neighborID] {
					visited[neighborID] = true
					neighbor := h.nodes[neighborID]
					distance := CalculateDistance(query, neighbor.Vector, h.distanceFunc)

					if w.Len() < ef {
						heap.Push(candidates, &PriorityQueueItem{
							ID:       neighborID,
							Distance: distance,
						})
						heap.Push(w, &PriorityQueueItem{
							ID:       neighborID,
							Distance: -distance,
						})
					} else {
						furthest := (*w)[0]
						if distance < -furthest.Distance {
							heap.Push(candidates, &PriorityQueueItem{
								ID:       neighborID,
								Distance: distance,
							})
							heap.Pop(w)
							heap.Push(w, &PriorityQueueItem{
								ID:       neighborID,
								Distance: -distance,
							})
						}
					}
				}
			}
		}
	}

	// Convert results to node slice
	result := make([]*HNSWNode, 0, w.Len())
	for w.Len() > 0 {
		item := heap.Pop(w).(*PriorityQueueItem)
		result = append([]*HNSWNode{h.nodes[item.ID]}, result...)
	}

	return result
}

// selectNeighbors selects the best neighbors using heuristic
func (h *HNSWIndex) selectNeighbors(candidates []*HNSWNode, m int, query []float32, level int) []*HNSWNode {
	if len(candidates) <= m {
		return candidates
	}

	// Simple strategy: select m closest neighbors
	// In production, you might want to implement the heuristic from the paper
	selected := make([]*HNSWNode, 0, m)
	distances := make([]float32, len(candidates))

	for i, candidate := range candidates {
		distances[i] = CalculateDistance(query, candidate.Vector, h.distanceFunc)
	}

	// Sort by distance and take top m
	for i := 0; i < m && i < len(candidates); i++ {
		minIdx := i
		for j := i + 1; j < len(candidates); j++ {
			if distances[j] < distances[minIdx] {
				minIdx = j
			}
		}
		if minIdx != i {
			candidates[i], candidates[minIdx] = candidates[minIdx], candidates[i]
			distances[i], distances[minIdx] = distances[minIdx], distances[i]
		}
		selected = append(selected, candidates[i])
	}

	return selected
}

// addConnection adds a connection between two nodes at a specific level
func (h *HNSWIndex) addConnection(node1, node2 *HNSWNode, level int) {
	if level < len(node1.Connections) {
		node1.Connections[level][node2.ID] = true
	}
}

// pruneConnections ensures a node doesn't have too many connections
func (h *HNSWIndex) pruneConnections(node *HNSWNode, level int) {
	maxConn := h.config.M
	if level == 0 {
		maxConn = h.config.MaxM0
	}

	if level < len(node.Connections) && len(node.Connections[level]) > maxConn {
		// Simple pruning: remove random connections
		// In production, implement distance-based pruning
		connections := make([]string, 0, len(node.Connections[level]))
		for id := range node.Connections[level] {
			connections = append(connections, id)
		}

		// Keep only maxConn connections
		for i := len(connections) - 1; i >= maxConn; i-- {
			removeIdx := h.rng.Intn(i + 1)
			removeID := connections[removeIdx]
			delete(node.Connections[level], removeID)
			connections[removeIdx] = connections[i]
		}
	}
}

// getRandomLevel generates a random level for a new node
func (h *HNSWIndex) getRandomLevel() int {
	level := 0
	for h.rng.Float64() < h.levelMultiplier && level < 16 { // Cap at level 16
		level++
	}
	return level
}

// findNewEntryPoint finds a new entry point after deletion
func (h *HNSWIndex) findNewEntryPoint() {
	h.entryPoint = nil
	h.maxLevel = 0

	for _, node := range h.nodes {
		if node.Level > h.maxLevel {
			h.maxLevel = node.Level
			h.entryPoint = node
		}
	}
}

// GetStats returns statistics about the HNSW index
func (h *HNSWIndex) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"node_count":  len(h.nodes),
		"max_level":   h.maxLevel,
		"dimensions":  h.dimensions,
		"entry_point": nil,
		"config":      h.config,
	}

	if h.entryPoint != nil {
		stats["entry_point"] = h.entryPoint.ID
	}

	// Calculate average connections per level
	levelStats := make(map[int]map[string]interface{})
	for level := 0; level <= h.maxLevel; level++ {
		nodeCount := 0
		totalConnections := 0

		for _, node := range h.nodes {
			if node.Level >= level {
				nodeCount++
				if level < len(node.Connections) {
					totalConnections += len(node.Connections[level])
				}
			}
		}

		avgConnections := 0.0
		if nodeCount > 0 {
			avgConnections = float64(totalConnections) / float64(nodeCount)
		}

		levelStats[level] = map[string]interface{}{
			"node_count":        nodeCount,
			"total_connections": totalConnections,
			"avg_connections":   avgConnections,
		}
	}

	stats["level_stats"] = levelStats
	return stats
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
