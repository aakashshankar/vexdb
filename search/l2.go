package search

import "math"

func L2(vec1, vec2 []float64) float64 {
	diff := 0.0
	for i := 0; i < len(vec1); i++ {
		diff += math.Pow(vec1[i]-vec2[i], 2)
	}

	return math.Sqrt(diff)
}
