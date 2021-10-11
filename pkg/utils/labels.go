package utils

import (
	"github.com/tenzen-y/imperator/pkg/consts"
	"github.com/tenzen-y/imperator/pkg/version"
)

func BaseLabels() map[string]string {
	return map[string]string{
		consts.K8sAppNameKey:    consts.K8sAppNameValue,
		consts.K8sAppVersionKey: version.Version,
	}
}
