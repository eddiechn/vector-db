package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// VectorDatabase is the main database implementation
type VectorDatabase struct {
	mu           sync.RWMutex
	index        *HNSWIndex
	vectors      map[string]*VectorMetadata
	config       DatabaseConfig
	stats        *StatsTracker
	dimensions   int
	persistPath  string
	autoSave     bool
	saveInterval time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Dimensions     int            `json:"dimensions"`
	DistanceMetric DistanceMetric `json:"distance_metric"`
	IndexConfig    IndexConfig    `json:"index_config"`
	HNSWConfig     HNSWConfig     `json:"hnsw_config"`
	PersistPath    string         `json:"persist_path"`
	AutoSave       bool           `json:"auto_save"`
	SaveInterval   time.Duration  `json:"save_interval"`
}

// DefaultDatabaseConfig returns a default database configuration
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Dimensions:     32,
		DistanceMetric: CosineSimilarity,
		IndexConfig: IndexConfig{
			Type:       "hnsw",
			Parameters: make(map[string]interface{}),
		},
		HNSWConfig:   DefaultHNSWConfig(),
		PersistPath:  "vectordb_data",
		AutoSave:     true,
		SaveInterval: 5 * time.Minute,
	}
}

// NewVectorDatabase creates a new vector database
func NewVectorDatabase(config DatabaseConfig) (*VectorDatabase, error) {
	if config.Dimensions <= 0 {
		return nil, &InvalidConfigError{
			Field:  "dimensions",
			Value:  config.Dimensions,
			Reason: "must be positive",
		}
	}

	index := NewHNSWIndex(config.HNSWConfig, config.Dimensions, config.DistanceMetric)

	db := &VectorDatabase{
		index:        index,
		vectors:      make(map[string]*VectorMetadata),
		config:       config,
		stats:        NewStatsTracker(),
		dimensions:   config.Dimensions,
		persistPath:  config.PersistPath,
		autoSave:     config.AutoSave,
		saveInterval: config.SaveInterval,
		stopChan:     make(chan struct{}),
	}

	// Update initial stats
	db.stats.UpdateDimensions(config.Dimensions)

	// Start auto-save goroutine if enabled
	if config.AutoSave {
		db.wg.Add(1)
		go db.autoSaveWorker()
	}

	return db, nil
}

// Insert adds a vector to the database
func (db *VectorDatabase) Insert(req InsertRequest) error {
	start := time.Now()
	defer func() {
		db.stats.UpdateLatency(time.Since(start))
		db.stats.IncrementInsertRequests()
	}()

	db.mu.Lock()
	defer db.mu.Unlock()

	if len(req.Vector.Data) != db.dimensions {
		return &DimensionMismatchError{
			Expected: db.dimensions,
			Actual:   len(req.Vector.Data),
		}
	}

	// Check if vector already exists
	if _, exists := db.vectors[req.Vector.ID]; exists {
		return fmt.Errorf("vector with ID %s already exists", req.Vector.ID)
	}

	// Insert into index
	err := db.index.Insert(req.Vector.ID, req.Vector.Data, req.Metadata)
	if err != nil {
		return &DatabaseError{
			Operation: "insert",
			Cause:     err,
		}
	}

	// Store metadata
	db.vectors[req.Vector.ID] = &VectorMetadata{
		CreatedAt: time.Now(),
		Tags:      req.Metadata,
	}

	// Update stats
	db.stats.UpdateVectorCount(int64(len(db.vectors)))

	return nil
}

// Search performs similarity search
func (db *VectorDatabase) Search(req SearchRequest) ([]SearchResult, error) {
	start := time.Now()
	defer func() {
		db.stats.UpdateLatency(time.Since(start))
		db.stats.IncrementSearchRequests()
	}()

	if len(req.Vector) != db.dimensions {
		return nil, &DimensionMismatchError{
			Expected: db.dimensions,
			Actual:   len(req.Vector),
		}
	}

	if req.K <= 0 {
		req.K = 10 // Default to 10 results
	}

	// Calculate optimal ef for search
	ef := CalculateOptimalEf(req.K, db.config.HNSWConfig.EfSearch)

	return db.index.Search(req.Vector, req.K, ef)
}

// Delete removes a vector from the database
func (db *VectorDatabase) Delete(id string) error {
	start := time.Now()
	defer func() {
		db.stats.UpdateLatency(time.Since(start))
		db.stats.IncrementDeleteRequests()
	}()

	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if vector exists
	if _, exists := db.vectors[id]; !exists {
		return &VectorNotFoundError{ID: id}
	}

	// Delete from index
	err := db.index.Delete(id)
	if err != nil {
		return &DatabaseError{
			Operation: "delete",
			Cause:     err,
		}
	}

	// Remove metadata
	delete(db.vectors, id)

	// Update stats
	db.stats.UpdateVectorCount(int64(len(db.vectors)))

	return nil
}

// Get retrieves a vector by ID
func (db *VectorDatabase) Get(id string) (*Vector, *VectorMetadata, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	metadata, exists := db.vectors[id]
	if !exists {
		return nil, nil, &VectorNotFoundError{ID: id}
	}

	// Get vector from index
	node, exists := db.index.nodes[id]
	if !exists {
		// This should not happen, but handle gracefully
		return nil, nil, &DatabaseError{
			Operation: "get",
			Cause:     fmt.Errorf("vector found in metadata but not in index"),
		}
	}

	vector := &Vector{
		ID:   id,
		Data: make([]float32, len(node.Vector)),
	}
	copy(vector.Data, node.Vector)

	return vector, metadata, nil
}

// List returns all vector IDs with pagination
func (db *VectorDatabase) List(offset, limit int) ([]string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	ids := make([]string, 0, len(db.vectors))
	for id := range db.vectors {
		ids = append(ids, id)
	}

	// Apply pagination
	if offset >= len(ids) {
		return []string{}, nil
	}

	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}

	return ids[offset:end], nil
}

// GetStats returns database statistics
func (db *VectorDatabase) GetStats() DatabaseStats {
	stats := db.stats.GetStats()

	// Add index-specific stats
	indexStats := db.index.GetStats()
	stats.MemoryUsage = db.estimateMemoryUsage()

	// You could add more detailed stats here
	_ = indexStats // Use indexStats if needed for more detailed reporting

	return stats
}

// GetConfig returns the database configuration
func (db *VectorDatabase) GetConfig() DatabaseConfig {
	return db.config
}

// Save persists the database to disk
func (db *VectorDatabase) Save() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if err := os.MkdirAll(db.persistPath, 0755); err != nil {
		return &PersistenceError{
			Operation: "create_directory",
			Path:      db.persistPath,
			Cause:     err,
		}
	}

	// Save configuration
	configPath := filepath.Join(db.persistPath, "config.json")
	configData, err := json.MarshalIndent(db.config, "", "  ")
	if err != nil {
		return &PersistenceError{
			Operation: "marshal_config",
			Path:      configPath,
			Cause:     err,
		}
	}

	if err := ioutil.WriteFile(configPath, configData, 0644); err != nil {
		return &PersistenceError{
			Operation: "write_config",
			Path:      configPath,
			Cause:     err,
		}
	}

	// Save vectors and metadata
	vectorsPath := filepath.Join(db.persistPath, "vectors.json")
	vectorData := make(map[string]interface{})

	for id, metadata := range db.vectors {
		if node, exists := db.index.nodes[id]; exists {
			vectorData[id] = map[string]interface{}{
				"vector":   node.Vector,
				"metadata": metadata,
			}
		}
	}

	vectorsJSON, err := json.MarshalIndent(vectorData, "", "  ")
	if err != nil {
		return &PersistenceError{
			Operation: "marshal_vectors",
			Path:      vectorsPath,
			Cause:     err,
		}
	}

	if err := ioutil.WriteFile(vectorsPath, vectorsJSON, 0644); err != nil {
		return &PersistenceError{
			Operation: "write_vectors",
			Path:      vectorsPath,
			Cause:     err,
		}
	}

	// Save index structure (simplified - in production you'd want more efficient serialization)
	indexPath := filepath.Join(db.persistPath, "index.json")
	indexData, err := json.MarshalIndent(db.index.nodes, "", "  ")
	if err != nil {
		return &PersistenceError{
			Operation: "marshal_index",
			Path:      indexPath,
			Cause:     err,
		}
	}

	if err := ioutil.WriteFile(indexPath, indexData, 0644); err != nil {
		return &PersistenceError{
			Operation: "write_index",
			Path:      indexPath,
			Cause:     err,
		}
	}

	return nil
}

// Load restores the database from disk
func (db *VectorDatabase) Load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Load vectors and metadata
	vectorsPath := filepath.Join(db.persistPath, "vectors.json")
	if _, err := os.Stat(vectorsPath); os.IsNotExist(err) {
		// No saved data, start fresh
		return nil
	}

	vectorsData, err := ioutil.ReadFile(vectorsPath)
	if err != nil {
		return &PersistenceError{
			Operation: "read_vectors",
			Path:      vectorsPath,
			Cause:     err,
		}
	}

	var vectorData map[string]interface{}
	if err := json.Unmarshal(vectorsData, &vectorData); err != nil {
		return &PersistenceError{
			Operation: "unmarshal_vectors",
			Path:      vectorsPath,
			Cause:     err,
		}
	}

	// Rebuild the database
	for id, data := range vectorData {
		dataMap := data.(map[string]interface{})

		// Extract vector
		vectorSlice := dataMap["vector"].([]interface{})
		vector := make([]float32, len(vectorSlice))
		for i, v := range vectorSlice {
			vector[i] = float32(v.(float64))
		}

		// Extract metadata
		var metadata map[string]interface{}
		if metadataRaw, exists := dataMap["metadata"]; exists {
			if metadataMap, ok := metadataRaw.(map[string]interface{}); ok {
				if tagsRaw, exists := metadataMap["tags"]; exists {
					if tags, ok := tagsRaw.(map[string]interface{}); ok {
						metadata = tags
					}
				}
			}
		}

		// Insert into index
		if err := db.index.Insert(id, vector, metadata); err != nil {
			return &DatabaseError{
				Operation: "rebuild_index",
				Cause:     err,
			}
		}

		// Store metadata
		db.vectors[id] = &VectorMetadata{
			CreatedAt: time.Now(), // We lost the original timestamp, use current time
			Tags:      metadata,
		}
	}

	// Update stats
	db.stats.UpdateVectorCount(int64(len(db.vectors)))

	return nil
}

// Close gracefully shuts down the database
func (db *VectorDatabase) Close() error {
	// Stop auto-save worker
	close(db.stopChan)
	db.wg.Wait()

	// Final save
	if db.autoSave {
		return db.Save()
	}

	return nil
}

// autoSaveWorker runs in a goroutine to periodically save the database
func (db *VectorDatabase) autoSaveWorker() {
	defer db.wg.Done()

	ticker := time.NewTicker(db.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := db.Save(); err != nil {
				// Log error (in production, use proper logging)
				fmt.Printf("Auto-save failed: %v\n", err)
			}
		case <-db.stopChan:
			return
		}
	}
}

// estimateMemoryUsage provides a rough estimate of memory usage
func (db *VectorDatabase) estimateMemoryUsage() int64 {
	// Rough calculation: vectors + metadata + index overhead
	vectorMemory := int64(len(db.vectors)) * int64(db.dimensions) * 4 // 4 bytes per float32
	metadataMemory := int64(len(db.vectors)) * 100                    // Rough estimate for metadata
	indexOverhead := vectorMemory / 2                                 // HNSW typically has ~50% overhead

	return vectorMemory + metadataMemory + indexOverhead
}

// UpdateConfig allows updating certain configuration parameters
func (db *VectorDatabase) UpdateConfig(newConfig DatabaseConfig) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Only allow updating certain fields
	if newConfig.SaveInterval > 0 {
		db.config.SaveInterval = newConfig.SaveInterval
		db.saveInterval = newConfig.SaveInterval
	}

	if newConfig.PersistPath != "" {
		db.config.PersistPath = newConfig.PersistPath
		db.persistPath = newConfig.PersistPath
	}

	db.config.AutoSave = newConfig.AutoSave
	db.autoSave = newConfig.AutoSave

	return nil
}
