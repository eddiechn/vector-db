#!/usr/bin/env python3
"""
VectorDB Python Client Example

This script demonstrates how to interact with VectorDB using Python.
Perfect for machine learning and data science workflows.
"""

import requests
import json
import numpy as np
import time
from typing import List, Dict, Any, Optional

class VectorDBClient:
    """Python client for VectorDB"""
    
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.session.headers.update({'Content-Type': 'application/json'})
    
    def health_check(self) -> bool:
        """Check if the server is healthy"""
        try:
            response = self.session.get(f"{self.base_url}/health")
            return response.status_code == 200
        except requests.RequestException:
            return False
    
    def insert_vector(self, vector_id: str, data: List[float], metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Insert a vector into the database"""
        payload = {
            "vector": {
                "id": vector_id,
                "data": data
            },
            "metadata": metadata or {}
        }
        
        response = self.session.post(f"{self.base_url}/vectors", json=payload)
        response.raise_for_status()
        return response.json()
    
    def search_vectors(self, query_vector: List[float], k: int = 10, distance_metric: int = 0) -> Dict[str, Any]:
        """Search for similar vectors"""
        payload = {
            "vector": query_vector,
            "k": k,
            "distance_metric": distance_metric
        }
        
        response = self.session.post(f"{self.base_url}/search", json=payload)
        response.raise_for_status()
        return response.json()
    
    def get_vector(self, vector_id: str) -> Dict[str, Any]:
        """Get a specific vector by ID"""
        response = self.session.get(f"{self.base_url}/vectors/{vector_id}")
        response.raise_for_status()
        return response.json()
    
    def delete_vector(self, vector_id: str) -> Dict[str, Any]:
        """Delete a vector by ID"""
        response = self.session.delete(f"{self.base_url}/vectors/{vector_id}")
        response.raise_for_status()
        return response.json()
    
    def list_vectors(self, offset: int = 0, limit: int = 50) -> Dict[str, Any]:
        """List vectors with pagination"""
        params = {"offset": offset, "limit": limit}
        response = self.session.get(f"{self.base_url}/vectors", params=params)
        response.raise_for_status()
        return response.json()
    
    def get_stats(self) -> Dict[str, Any]:
        """Get database statistics"""
        response = self.session.get(f"{self.base_url}/stats")
        response.raise_for_status()
        return response.json()
    
    def get_config(self) -> Dict[str, Any]:
        """Get database configuration"""
        response = self.session.get(f"{self.base_url}/config")
        response.raise_for_status()
        return response.json()

def generate_random_embeddings(num_vectors: int, dimensions: int) -> np.ndarray:
    """Generate random normalized embeddings"""
    # Generate random vectors
    vectors = np.random.randn(num_vectors, dimensions).astype(np.float32)
    
    # Normalize to unit length (common for embeddings)
    norms = np.linalg.norm(vectors, axis=1, keepdims=True)
    norms[norms == 0] = 1  # Avoid division by zero
    vectors = vectors / norms
    
    return vectors

def simulate_text_embeddings(texts: List[str], dimensions: int = 128) -> np.ndarray:
    """Simulate text embeddings (normally you'd use a real model)"""
    embeddings = []
    
    for i, text in enumerate(texts):
        # Simple hash-based embedding simulation
        # In practice, you'd use transformers, sentence-transformers, etc.
        np.random.seed(hash(text) % (2**32))
        embedding = np.random.randn(dimensions).astype(np.float32)
        
        # Add some structure based on text properties
        embedding[0] = len(text) / 100.0  # Text length feature
        embedding[1] = text.count(' ') / 50.0  # Word count feature
        
        # Normalize
        embedding = embedding / np.linalg.norm(embedding)
        embeddings.append(embedding)
    
    return np.array(embeddings)

def main():
    """Main demonstration function"""
    print("🐍 VectorDB Python Client Demo")
    print("===============================")
    
    # Initialize client
    client = VectorDBClient()
    
    # Check server health
    print("📡 Checking server health...")
    if not client.health_check():
        print("❌ Server is not running. Please start it with: ./vectordb")
        return
    print("✅ Server is healthy")
    
    # Get initial stats
    print("\n📊 Initial database stats:")
    stats = client.get_stats()
    print(f"  Vectors: {stats['vector_count']}")
    print(f"  Dimensions: {stats['dimensions']}")
    print(f"  Memory usage: {stats['memory_usage_bytes'] / 1024 / 1024:.2f} MB")
    
    # Example 1: Insert random vectors
    print("\n🔢 Example 1: Inserting random vectors")
    dimensions = stats['dimensions']
    random_vectors = generate_random_embeddings(5, dimensions)
    
    for i, vector in enumerate(random_vectors):
        vector_id = f"random_{i}"
        metadata = {
            "type": "random",
            "index": i,
            "norm": float(np.linalg.norm(vector))
        }
        
        result = client.insert_vector(vector_id, vector.tolist(), metadata)
        print(f"  ✅ Inserted {vector_id}")
    
    # Example 2: Text embeddings simulation
    print("\n📝 Example 2: Simulating text embeddings")
    texts = [
        "Machine learning is a subset of artificial intelligence",
        "Deep learning uses neural networks with multiple layers",
        "Natural language processing helps computers understand text",
        "Computer vision enables machines to interpret visual information",
        "Reinforcement learning trains agents through rewards and penalties"
    ]
    
    text_embeddings = simulate_text_embeddings(texts, dimensions)
    
    for i, (text, embedding) in enumerate(zip(texts, text_embeddings)):
        vector_id = f"text_{i}"
        metadata = {
            "type": "text",
            "content": text,
            "word_count": len(text.split()),
            "char_count": len(text)
        }
        
        result = client.insert_vector(vector_id, embedding.tolist(), metadata)
        print(f"  ✅ Inserted text embedding: {text[:50]}...")
    
    # Example 3: Similarity search
    print("\n🔍 Example 3: Similarity search")
    
    # Search for vectors similar to the first text embedding
    query_vector = text_embeddings[0]
    search_results = client.search_vectors(query_vector.tolist(), k=3, distance_metric=0)
    
    print(f"  Query: '{texts[0][:50]}...'")
    print(f"  Found {len(search_results['results'])} similar vectors:")
    
    for result in search_results['results']:
        vector_info = client.get_vector(result['id'])
        if 'content' in vector_info['metadata']:
            content = vector_info['metadata']['content']
            print(f"    {result['id']}: {content[:50]}... (score: {result['score']:.4f})")
        else:
            print(f"    {result['id']}: score {result['score']:.4f}")
    
    # Example 4: Distance metric comparison
    print("\n📏 Example 4: Comparing distance metrics")
    
    query_vector = random_vectors[0]
    metrics = [
        (0, "Cosine Similarity"),
        (1, "Euclidean Distance"),
        (2, "Dot Product"),
        (3, "Manhattan Distance")
    ]
    
    for metric_id, metric_name in metrics:
        results = client.search_vectors(query_vector.tolist(), k=3, distance_metric=metric_id)
        print(f"  {metric_name}: {len(results['results'])} results")
        for result in results['results'][:2]:  # Show top 2
            print(f"    {result['id']}: {result['score']:.4f}")
    
    # Example 5: Batch operations and performance
    print("\n⚡ Example 5: Batch operations performance")
    
    start_time = time.time()
    batch_size = 50
    batch_vectors = generate_random_embeddings(batch_size, dimensions)
    
    print(f"  Inserting {batch_size} vectors...")
    for i, vector in enumerate(batch_vectors):
        vector_id = f"batch_{i}"
        metadata = {"type": "batch", "batch_index": i}
        client.insert_vector(vector_id, vector.tolist(), metadata)
    
    insert_time = time.time() - start_time
    print(f"  ✅ Inserted {batch_size} vectors in {insert_time:.2f}s ({batch_size/insert_time:.1f} vectors/sec)")
    
    # Search performance
    start_time = time.time()
    search_count = 20
    
    for i in range(search_count):
        query = batch_vectors[i % len(batch_vectors)]
        client.search_vectors(query.tolist(), k=5)
    
    search_time = time.time() - start_time
    print(f"  ✅ Performed {search_count} searches in {search_time:.2f}s ({search_count/search_time:.1f} searches/sec)")
    
    # Final stats
    print("\n📊 Final database stats:")
    final_stats = client.get_stats()
    print(f"  Vectors: {final_stats['vector_count']}")
    print(f"  Search requests: {final_stats['search_requests']}")
    print(f"  Insert requests: {final_stats['insert_requests']}")
    print(f"  Memory usage: {final_stats['memory_usage_bytes'] / 1024 / 1024:.2f} MB")
    
    # Cleanup
    print("\n🧹 Cleaning up demo vectors...")
    vectors_to_clean = [f"random_{i}" for i in range(5)] + \
                      [f"text_{i}" for i in range(5)] + \
                      [f"batch_{i}" for i in range(batch_size)]
    
    cleaned = 0
    for vector_id in vectors_to_clean:
        try:
            client.delete_vector(vector_id)
            cleaned += 1
        except requests.RequestException:
            pass  # Vector might not exist
    
    print(f"  ✅ Cleaned up {cleaned} vectors")
    
    print("\n🎉 Python client demo completed!")
    print("💡 This demonstrates how to integrate VectorDB with ML workflows")

if __name__ == "__main__":
    main()
