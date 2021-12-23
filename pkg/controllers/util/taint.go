/*
Copyright 2021 Yuki Iwai (@tenzen-y)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

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
