package embedding

import (
	"fmt"
	"math"
)

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(vec1, vec2 []float64) (float64, error) {
	if len(vec1) != len(vec2) {
		return 0.0, fmt.Errorf("vectors must have the same dimension: %d vs %d", len(vec1), len(vec2))
	}

	if len(vec1) == 0 {
		return 0.0, fmt.Errorf("vectors cannot be empty")
	}

	dotProduct := 0.0
	magnitude1 := 0.0
	magnitude2 := 0.0

	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		magnitude1 += vec1[i] * vec1[i]
		magnitude2 += vec2[i] * vec2[i]
	}

	magnitude1 = math.Sqrt(magnitude1)
	magnitude2 = math.Sqrt(magnitude2)

	if magnitude1 == 0 || magnitude2 == 0 {
		return 0.0, fmt.Errorf("one or both vectors have zero magnitude")
	}

	similarity := dotProduct / (magnitude1 * magnitude2)

	// Clamp to [-1, 1] range to handle floating point precision issues
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	return similarity, nil
}

// EuclideanDistance calculates the Euclidean distance between two vectors
func EuclideanDistance(vec1, vec2 []float64) (float64, error) {
	if len(vec1) != len(vec2) {
		return 0.0, fmt.Errorf("vectors must have the same dimension: %d vs %d", len(vec1), len(vec2))
	}

	if len(vec1) == 0 {
		return 0.0, fmt.Errorf("vectors cannot be empty")
	}

	sumSquaredDiff := 0.0
	for i := 0; i < len(vec1); i++ {
		diff := vec1[i] - vec2[i]
		sumSquaredDiff += diff * diff
	}

	return math.Sqrt(sumSquaredDiff), nil
}

// DotProduct calculates the dot product of two vectors
func DotProduct(vec1, vec2 []float64) (float64, error) {
	if len(vec1) != len(vec2) {
		return 0.0, fmt.Errorf("vectors must have the same dimension: %d vs %d", len(vec1), len(vec2))
	}

	product := 0.0
	for i := 0; i < len(vec1); i++ {
		product += vec1[i] * vec2[i]
	}

	return product, nil
}

// Magnitude calculates the magnitude (L2 norm) of a vector
func Magnitude(vec []float64) float64 {
	sumSquares := 0.0
	for _, v := range vec {
		sumSquares += v * v
	}
	return math.Sqrt(sumSquares)
}

// Normalize normalizes a vector to unit length
func Normalize(vec []float64) ([]float64, error) {
	if len(vec) == 0 {
		return nil, fmt.Errorf("vector cannot be empty")
	}

	mag := Magnitude(vec)
	if mag == 0 {
		return nil, fmt.Errorf("cannot normalize zero vector")
	}

	normalized := make([]float64, len(vec))
	for i, v := range vec {
		normalized[i] = v / mag
	}

	return normalized, nil
}
