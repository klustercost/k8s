package utils

import (
	"fmt"
	"strings"
)

// Helper function to convert values of a map[string]string to a csv string.
// Map key and value are returned separated by comma key=value,key=value.
func MapToString(labels map[string]string) string {
	var sb strings.Builder

	i := 0
	for key, value := range labels {
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(value)
		if i < len(labels)-1 {
			sb.WriteString(",")
		}
		i++
	}
	return sb.String()
}

func FindAppLabel(m map[string]string) string {
	for key, value := range m {
		if strings.HasPrefix(key, "app") {
			return fmt.Sprintf("%s", value)
		}
	}
	return ""
}
