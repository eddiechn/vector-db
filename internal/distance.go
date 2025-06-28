package internal

import (
	"math"
)

// CalculateDistance computes the distance between two vectors using the specified metric
func CalculateDistance(a, b []float32, metric DistanceMetric) float32 {
	switch metric {
	case CosineSimilarity:
		return 1.0 - CosineSimilarityScore(a, b) // Convert similarity to distance
	case EuclideanDistance:
		return EuclideanDistanceScore(a, b)
	case DotProduct:
		return -DotProductScore(a, b) // Negative for max-heap behavior
	case ManhattanDistance:
		return ManhattanDistanceScore(a, b)
	default:
		return EuclideanDistanceScore(a, b)
	}
}

// CosineSimilarityScore calculates cosine similarity between two vectors
func CosineSimilarityScore(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EuclideanDistanceScore calculates Euclidean distance between two vectors
func EuclideanDistanceScore(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// DotProductScore calculates dot product between two vectors
func DotProductScore(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}

	return sum
}

// ManhattanDistanceScore calculates Manhattan distance between two vectors
func ManhattanDistanceScore(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		sum += float32(math.Abs(float64(a[i] - b[i])))
	}

	return sum
}

// NormalizeVector normalizes a vector to unit length
func NormalizeVector(v []float32) []float32 {
	var norm float32
	for _, val := range v {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm == 0 {
		return v
	}

	normalized := make([]float32, len(v))
	for i, val := range v {
		normalized[i] = val / norm
	}

	return normalized
}

// VectorMagnitude calculates the magnitude (L2 norm) of a vector
func VectorMagnitude(v []float32) float32 {
	var sum float32
	for _, val := range v {
		sum += val * val
	}
	return float32(math.Sqrt(float64(sum)))
}

// AddVectors adds two vectors element-wise
func AddVectors(a, b []float32) []float32 {
	if len(a) != len(b) {
		return nil
	}

	result := make([]float32, len(a))
	for i := 0; i < len(a); i++ {
		result[i] = a[i] + b[i]
	}

	return result
}

// SubtractVectors subtracts vector b from vector a element-wise
func SubtractVectors(a, b []float32) []float32 {
	if len(a) != len(b) {
		return nil
	}

	result := make([]float32, len(a))
	for i := 0; i < len(a); i++ {
		result[i] = a[i] - b[i]
	}

	return result
}

// ScaleVector multiplies a vector by a scalar
func ScaleVector(v []float32, scalar float32) []float32 {
	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = val * scalar
	}

	return result
}

// CalculateOptimalEf calculates optimal ef parameter for HNSW search
func CalculateOptimalEf(k int, baseEf int) int {
	// Heuristic: ef should be at least k and typically 1.5-2x larger for good recall
	minEf := k
	if baseEf > minEf {
		return int(math.Max(float64(baseEf), float64(k)*1.5))
	}
	return int(float64(k) * 1.5)
}
