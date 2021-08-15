package utils

import (
	"github.com/tenzen-y/imperator"
	"github.com/tenzen-y/imperator/pkg/consts"
)

func BaseLabels() map[string]string {
	return map[string]string{
		consts.K8sAppNameKey:    consts.K8sAppNameValue,
		consts.K8sAppVersionKey: imperator.Version,
	}
}
