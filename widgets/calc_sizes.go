package widgets

import "math"

func calcSizes(targetSize int, sizes []int, flexes []int) []int {
	result := make([]int, len(sizes))
	totalSize, totalFlex := 0, 0
	for i, size := range sizes {
		result[i] = size
		totalSize += size
		totalFlex += flexes[i]
	}
	for totalSize > targetSize {
		idx := 0
		maxSize := result[0]
		for i, size := range result {
			if maxSize < size {
				maxSize = size
				idx = i
			}
		}
		result[idx]--
		totalSize--
	}

	if totalFlex == 0 {
		return result
	}

	if totalSize < targetSize {
		diff := targetSize - totalSize
		remainders := make([]float64, len(sizes))
		for i, flex := range flexes {
			rate := float64(diff*flex) / float64(totalFlex)
			remainders[i] = rate - math.Floor(rate)
			result[i] += int(rate)
		}
		totalSize := 0
		for _, size := range result {
			totalSize += size
		}
		for i := range result {
			if totalSize == targetSize {
				break
			}
			if flexes[i] > 0 {
				result[i]++
				totalSize++
			}
		}
		for i := range result {
			if totalSize == targetSize {
				break
			}
			if flexes[i] == 0 {
				result[i]++
				totalSize++
			}
		}
	}

	return result
}
