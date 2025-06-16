package main

import (
	"fmt"
	"math"
)

func formatCPU(millicores int) string {
	return fmt.Sprintf("%dm", millicores)
}

// func formatMemory(bytes int64) string {
// 	// Convert bytes to Mi
// 	mi := float64(bytes) / (1024 * 1024)
// 	return fmt.Sprintf("%.0fMi", mi)
// }

func formatMemory(bytes int64) string {
	mi := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%dMi", int64(math.Round(mi)))
}
