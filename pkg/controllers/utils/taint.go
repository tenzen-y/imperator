package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func ExtractKeyValueFromTaint(taints []corev1.Taint) map[string]string {
	result := make(map[string]string)
	for _, t := range taints {
		result[t.Key] = t.Value
	}
	return result
}

func GetTaintKeyIndex(taints []corev1.Taint, key string) *int {
	for idx, t := range taints {
		if t.Key != key {
			continue
		}
		return pointer.Int(idx)
	}
	return nil
}
