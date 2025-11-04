package testutil

func ContainsMarker(image []byte, marker byte) bool {
	for i := 0; i+1 < len(image); i++ {
		if image[i] == 0xFF && image[i+1] == marker {
			return true
		}
	}
	return false
}
