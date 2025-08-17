# Vector Database in Go

A custom-built, high-performance vector database implemented in Go, showcasing advanced systems programming and AI infrastructure capabilities.

## 🚀 Features

### Core Capabilities
- **In-memory storage** with automatic persistence to disk
- **Thread-safe concurrent operations** using RWMutex
- **Multiple distance metrics**: Cosine similarity, Euclidean distance, Dot product, Manhattan distance
- **Top-K similarity search** with configurable result limits
- **Automatic data persistence** with graceful shutdown handling
- **OpenAI Integration**: Text-to-vector embedding and text-based search using OpenAI's API

### Advanced Indexing
- **HNSW (Hierarchical Navigable Small World)** algorithm implementation
- Configurable index parameters for optimal performance
- Multi-layer graph structure for efficient nearest neighbor search

## 🏗️ Architecture

### Modular Design
```
vectordb/
├── internal/           # Core database implementation
│   ├── types.go       # Data structures and types
│   ├── distance.go    # Distance calculation functions
│   ├── hnsw.go        # HNSW index implementation
│   ├── database.go    # Main database logic
│   └── errors.go      # Custom error types
├── api/               # HTTP API layer
│   └── server.go      # REST API server
├── cmd/               # Application entry point
│   └── main.go        # CLI and initialization
└── web/               # Web interface (future expansion)
```

## 🚀 Quick Start

### Installation
```bash
git clone <repository>
cd vector-db-in-go
go build -o vectordb ./cmd
```

### Basic Usage
```bash
# Start the server with default settings
./vectordb

# Custom configuration
./vectordb -port 8080 -dimensions 1536 -data ./my_vectors

# View all options
./vectordb -help
```

### API Examples

#### Insert a Vector
```bash
curl -X POST http://localhost:8080/vectors \
  -H "Content-Type: application/json" \
  -d '{
    "vector": {
      "id": "example_1",
      "data": [0.1, 0.2, 0.3, 0.4, 0.5]
    },
    "metadata": {
      "category": "example",
      "source": "api"
    }
  }'
```

#### Search for Similar Vectors
```bash
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, 0.4, 0.5],
    "k": 10,
    "distance_metric": 0
  }'
```

#### Get Database Statistics
```bash
curl http://localhost:8080/stats
```

## 📊 Web Dashboard

Access the interactive dashboard at `http://localhost:8080/` to:
- View real-time database statistics
- Monitor performance metrics
- Browse API endpoints
- Access comprehensive documentation

## 🔧 Configuration

### Command-Line Options
- `-port`: HTTP server port (default: 8080)
- `-dimensions`: Vector dimensions (default: 128)
- `-data`: Data persistence path (default: ./vectordb_data)
- `-autosave`: Enable automatic saving (default: true)
- `-save-interval`: Automatic save interval (default: 5m)
- `-load`: Load existing data on startup (default: true)

### HNSW Parameters
- **M**: Maximum connections per node (default: 16)
- **efConstruction**: Dynamic candidate list size during construction (default: 200)
- **efSearch**: Dynamic candidate list size during search (default: 50)

### Distance Metrics
- **0**: Cosine Similarity
- **1**: Euclidean Distance
- **2**: Dot Product
- **3**: Manhattan Distance

## 🎯 API Reference

### Vector Operations
- `POST /vectors` - Insert a new vector
- `GET /vectors` - List vectors with pagination
- `GET /vectors/{id}` - Get specific vector
- `DELETE /vectors/{id}` - Delete vector

### Search Operations
- `POST /search` - Perform similarity search
- `POST /search/text` - Search using text input (requires OpenAI API key)

### String-Based Embedding Operations  
- `POST /embed` - Convert text to vector embedding and store (requires OpenAI API key)

### Database Operations
- `GET /stats` - Database statistics
- `GET /config` - Configuration
- `GET /health` - Health check

### Administrative
- `POST /admin/save` - Manual save
- `GET /admin/index-stats` - Detailed index statistics

