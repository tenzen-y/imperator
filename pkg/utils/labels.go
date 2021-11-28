package utils

import (
	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	"github.com/tenzen-y/imperator/pkg/version"
	"strings"
)

func BaseLabels() map[string]string {
	return map[string]string{
		consts.K8sAppNameKey:    consts.K8sAppNameValue,
		consts.K8sAppVersionKey: version.Version,
	}
}

func GetMachineTypeLabelTaintKey(machineType string) string {
	return strings.Join([]string{imperatorv1alpha1.GroupVersion.Group, machineType}, "/")
}
