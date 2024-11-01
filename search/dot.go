package search

func Dot(vec1 []float64, vec2 []float64) float64 {
	dot := 0.0
	for i := 0; i < len(vec1); i++ {
		dot += vec1[i] * vec2[i]
	}

	return dot
}
