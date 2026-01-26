package dns

func Same[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[T]int, len(a))
	for _, v := range a {
		counts[v]++
	}

	for _, v := range b {
		if counts[v] == 0 {
			return false
		}
		counts[v]--
	}

	return true
}
