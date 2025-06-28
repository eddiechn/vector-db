package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vectordb/internal"
)

// Server represents the HTTP API server
type Server struct {
	db     *internal.VectorDatabase
	mux    *http.ServeMux
	port   int
	logger *log.Logger
}

// NewServer creates a new API server
func NewServer(db *internal.VectorDatabase, port int, logger *log.Logger) *Server {
	server := &Server{
		db:     db,
		mux:    http.NewServeMux(),
		port:   port,
		logger: logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Add CORS and logging middleware
	handler := s.corsMiddleware(s.loggingMiddleware(s.mux))

	// Vector operations
	s.mux.HandleFunc("/vectors", s.handleVectors)
	s.mux.HandleFunc("/vectors/", s.handleVectorByID)
	s.mux.HandleFunc("/search", s.handleSearch)

	// Database operations
	s.mux.HandleFunc("/stats", s.handleStats)
	s.mux.HandleFunc("/config", s.handleConfig)
	s.mux.HandleFunc("/health", s.handleHealth)

	// Administrative operations
	s.mux.HandleFunc("/admin/save", s.handleSave)
	s.mux.HandleFunc("/admin/index-stats", s.handleIndexStats)

	// Documentation and dashboard
	s.mux.HandleFunc("/", s.handleDashboard)
	s.mux.HandleFunc("/api-docs", s.handleAPIDocs)

	// Serve the final handler
	http.Handle("/", handler)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Printf("Starting VectorDB server on %s", addr)
	s.logger.Printf("Dashboard available at: http://localhost%s", addr)
	s.logger.Printf("API documentation at: http://localhost%s/api-docs", addr)
	return http.ListenAndServe(addr, nil)
}

// Middleware

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		s.logger.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Handler functions

// handleVectors handles vector listing and insertion
func (s *Server) handleVectors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListVectors(w, r)
	case "POST":
		s.handleInsertVector(w, r)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListVectors lists vectors with pagination
func (s *Server) handleListVectors(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	ids, err := s.db.List(offset, limit)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to list vectors: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"vectors": ids,
		"offset":  offset,
		"limit":   limit,
		"count":   len(ids),
	}

	s.writeJSON(w, response)
}

// handleInsertVector inserts a new vector
func (s *Server) handleInsertVector(w http.ResponseWriter, r *http.Request) {
	var req internal.InsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Vector.ID == "" {
		s.writeError(w, "Vector ID is required", http.StatusBadRequest)
		return
	}

	if len(req.Vector.Data) == 0 {
		s.writeError(w, "Vector data is required", http.StatusBadRequest)
		return
	}

	if err := s.db.Insert(req); err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(*internal.DimensionMismatchError); ok {
			status = http.StatusBadRequest
		}
		s.writeError(w, fmt.Sprintf("Failed to insert vector: %v", err), status)
		return
	}

	response := map[string]interface{}{
		"message": "Vector inserted successfully",
		"id":      req.Vector.ID,
	}

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, response)
}

// handleVectorByID handles operations on specific vectors
func (s *Server) handleVectorByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		s.writeError(w, "Vector ID is required", http.StatusBadRequest)
		return
	}
	id := parts[1]

	switch r.Method {
	case "GET":
		s.handleGetVector(w, r, id)
	case "DELETE":
		s.handleDeleteVector(w, r, id)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetVector retrieves a specific vector
func (s *Server) handleGetVector(w http.ResponseWriter, r *http.Request, id string) {
	vector, metadata, err := s.db.Get(id)
	if err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(*internal.VectorNotFoundError); ok {
			status = http.StatusNotFound
		}
		s.writeError(w, fmt.Sprintf("Failed to get vector: %v", err), status)
		return
	}

	response := map[string]interface{}{
		"vector":   vector,
		"metadata": metadata,
	}

	s.writeJSON(w, response)
}

// handleDeleteVector deletes a specific vector
func (s *Server) handleDeleteVector(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.db.Delete(id); err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(*internal.VectorNotFoundError); ok {
			status = http.StatusNotFound
		}
		s.writeError(w, fmt.Sprintf("Failed to delete vector: %v", err), status)
		return
	}

	response := map[string]interface{}{
		"message": "Vector deleted successfully",
		"id":      id,
	}

	s.writeJSON(w, response)
}

// handleSearch performs similarity search
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req internal.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Vector) == 0 {
		s.writeError(w, "Query vector is required", http.StatusBadRequest)
		return
	}

	if req.K <= 0 {
		req.K = 10 // Default
	}

	results, err := s.db.Search(req)
	if err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(*internal.DimensionMismatchError); ok {
			status = http.StatusBadRequest
		}
		s.writeError(w, fmt.Sprintf("Search failed: %v", err), status)
		return
	}

	response := map[string]interface{}{
		"results": results,
		"query":   req,
		"count":   len(results),
	}

	s.writeJSON(w, response)
}

// handleStats returns database statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.db.GetStats()
	s.writeJSON(w, stats)
}

// handleConfig returns database configuration
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := s.db.GetConfig()
	s.writeJSON(w, config)
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.db.GetStats()
	health := map[string]interface{}{
		"status":       "healthy",
		"timestamp":    time.Now(),
		"vector_count": stats.VectorCount,
		"uptime":       time.Since(stats.LastUpdated),
	}

	s.writeJSON(w, health)
}

// handleSave manually triggers a database save
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.db.Save(); err != nil {
		s.writeError(w, fmt.Sprintf("Save failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "Database saved successfully",
		"timestamp": time.Now(),
	}

	s.writeJSON(w, response)
}

// handleIndexStats returns detailed index statistics
func (s *Server) handleIndexStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// This would require exposing GetStats from the HNSW index
	// For now, return basic stats
	stats := s.db.GetStats()
	s.writeJSON(w, stats)
}

// handleDashboard serves the interactive dashboard
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	html := `<!DOCTYPE html>
<html>
<head>
    <title>VectorDB Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin: 20px 0; }
        .stat-card { background: #f5f5f5; padding: 15px; border-radius: 8px; border: 1px solid #ddd; }
        .stat-value { font-size: 24px; font-weight: bold; color: #333; }
        .stat-label { color: #666; font-size: 14px; }
        .api-section { margin: 30px 0; }
        .endpoint { background: #f9f9f9; padding: 10px; margin: 10px 0; border-left: 4px solid #007cba; }
        .method { font-weight: bold; color: #007cba; }
        nav { background: #333; color: white; padding: 15px; border-radius: 8px; margin-bottom: 20px; }
        nav h1 { margin: 0; }
    </style>
</head>
<body>
    <div class="container">
        <nav>
            <h1>VectorDB - High-Performance Vector Database</h1>
            <p>Real-time statistics and API endpoints</p>
        </nav>

        <div id="stats-container">
            <h2>Database Statistics</h2>
            <div class="stats-grid" id="stats-grid">
                <!-- Stats will be loaded here -->
            </div>
        </div>

        <div class="api-section">
            <h2>Quick API Reference</h2>
            <div class="endpoint">
                <span class="method">POST</span> /vectors - Insert a new vector
            </div>
            <div class="endpoint">
                <span class="method">GET</span> /vectors - List all vectors
            </div>
            <div class="endpoint">
                <span class="method">POST</span> /search - Perform similarity search
            </div>
            <div class="endpoint">
                <span class="method">GET</span> /stats - Get database statistics
            </div>
            <div class="endpoint">
                <span class="method">GET</span> /health - Health check
            </div>
            <p><a href="/api-docs">View complete API documentation</a></p>
        </div>
    </div>

    <script>
        // Load statistics
        function loadStats() {
            fetch('/stats')
                .then(response => response.json())
                .then(data => {
                    const grid = document.getElementById('stats-grid');
                    grid.innerHTML = ` + "`" + `
                        <div class="stat-card">
                            <div class="stat-value">${data.vector_count}</div>
                            <div class="stat-label">Total Vectors</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">${data.dimensions}</div>
                            <div class="stat-label">Dimensions</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">${data.search_requests}</div>
                            <div class="stat-label">Search Requests</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">${data.insert_requests}</div>
                            <div class="stat-label">Insert Requests</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">${Math.round(data.memory_usage_bytes / 1024 / 1024)} MB</div>
                            <div class="stat-label">Memory Usage</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">${data.index_type}</div>
                            <div class="stat-label">Index Type</div>
                        </div>
                    ` + "`" + `;
                })
                .catch(error => console.error('Error loading stats:', error));
        }

        // Load stats on page load and refresh every 5 seconds
        loadStats();
        setInterval(loadStats, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleAPIDocs serves the API documentation
func (s *Server) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>VectorDB API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; line-height: 1.6; }
        .container { max-width: 1000px; margin: 0 auto; }
        .endpoint { background: #f9f9f9; padding: 15px; margin: 15px 0; border-left: 4px solid #007cba; }
        .method { font-weight: bold; color: #007cba; padding: 3px 8px; background: #e3f2fd; border-radius: 3px; }
        .path { font-family: monospace; font-size: 16px; margin: 5px 0; }
        .description { margin: 10px 0; }
        .example { background: #f5f5f5; padding: 10px; border-radius: 5px; font-family: monospace; white-space: pre-wrap; }
        nav { background: #333; color: white; padding: 15px; border-radius: 8px; margin-bottom: 20px; }
        nav h1 { margin: 0; }
        .section { margin: 30px 0; }
        .parameter { background: #fff3cd; padding: 5px 10px; margin: 5px 0; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="container">
        <nav>
            <h1>VectorDB API Documentation</h1>
            <p>Complete reference for the VectorDB REST API</p>
            <a href="/" style="color: #fff;">← Back to Dashboard</a>
        </nav>

        <div class="section">
            <h2>Vector Operations</h2>
            
            <div class="endpoint">
                <span class="method">POST</span>
                <div class="path">/vectors</div>
                <div class="description">Insert a new vector into the database</div>
                <div class="parameter"><strong>Request Body:</strong></div>
                <div class="example">{
  "vector": {
    "id": "vector_1",
    "data": [0.1, 0.2, 0.3, ...]
  },
  "metadata": {
    "category": "example",
    "source": "api"
  }
}</div>
            </div>

            <div class="endpoint">
                <span class="method">GET</span>
                <div class="path">/vectors?offset=0&limit=50</div>
                <div class="description">List vectors with pagination</div>
                <div class="parameter"><strong>Query Parameters:</strong> offset (default: 0), limit (default: 50, max: 1000)</div>
            </div>

            <div class="endpoint">
                <span class="method">GET</span>
                <div class="path">/vectors/{id}</div>
                <div class="description">Retrieve a specific vector by ID</div>
            </div>

            <div class="endpoint">
                <span class="method">DELETE</span>
                <div class="path">/vectors/{id}</div>
                <div class="description">Delete a specific vector by ID</div>
            </div>
        </div>

        <div class="section">
            <h2>Search Operations</h2>
            
            <div class="endpoint">
                <span class="method">POST</span>
                <div class="path">/search</div>
                <div class="description">Perform similarity search</div>
                <div class="parameter"><strong>Request Body:</strong></div>
                <div class="example">{
  "vector": [0.1, 0.2, 0.3, ...],
  "k": 10,
  "distance_metric": 0,
  "filters": {}
}</div>
                <div class="parameter"><strong>Distance Metrics:</strong> 0=Cosine, 1=Euclidean, 2=Dot Product, 3=Manhattan</div>
            </div>
        </div>

        <div class="section">
            <h2>Database Operations</h2>
            
            <div class="endpoint">
                <span class="method">GET</span>
                <div class="path">/stats</div>
                <div class="description">Get database statistics and performance metrics</div>
            </div>

            <div class="endpoint">
                <span class="method">GET</span>
                <div class="path">/config</div>
                <div class="description">Get database configuration</div>
            </div>

            <div class="endpoint">
                <span class="method">GET</span>
                <div class="path">/health</div>
                <div class="description">Health check endpoint</div>
            </div>
        </div>

        <div class="section">
            <h2>Administrative Operations</h2>
            
            <div class="endpoint">
                <span class="method">POST</span>
                <div class="path">/admin/save</div>
                <div class="description">Manually trigger database persistence</div>
            </div>

            <div class="endpoint">
                <span class="method">GET</span>
                <div class="path">/admin/index-stats</div>
                <div class="description">Get detailed index statistics</div>
            </div>
        </div>

        <div class="section">
            <h2>Example Usage</h2>
            <h3>Insert a Vector</h3>
            <div class="example">curl -X POST http://localhost:8080/vectors \
  -H "Content-Type: application/json" \
  -d '{
    "vector": {
      "id": "example_1",
      "data": [0.1, 0.2, 0.3, 0.4, 0.5]
    },
    "metadata": {
      "category": "example"
    }
  }'</div>

            <h3>Search for Similar Vectors</h3>
            <div class="example">curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, 0.4, 0.5],
    "k": 5,
    "distance_metric": 0
  }'</div>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Utility functions

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}
