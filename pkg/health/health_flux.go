package health

import (
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type fluxStatusType string

const (
	fluxHealthy     fluxStatusType = "Healthy"
	fluxReady       fluxStatusType = "Ready"
	fluxReconciling fluxStatusType = "Reconciling"
)

type fluxKustomization struct {
	Status struct {
		Conditions []struct {
			Type    fluxStatusType
			Status  v1.ConditionStatus
			Reason  string
			Message string
		}
	}
}

func getFluxKustomizationHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var k fluxKustomization
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &k)
	if err != nil {
		return nil, err
	}

	ranking := map[fluxStatusType]int{
		fluxHealthy:     3,
		fluxReady:       2,
		fluxReconciling: 1,
	}
	sort.Slice(k.Status.Conditions, func(i, j int) bool {
		return ranking[k.Status.Conditions[i].Type] > ranking[k.Status.Conditions[j].Type]
	})
	for _, c := range k.Status.Conditions {
		msg := fmt.Sprintf("%s: %s", c.Reason, c.Message)
		if c.Type == fluxHealthy {
			if c.Status == v1.ConditionTrue {
				return &HealthStatus{Health: HealthHealthy, Status: HealthStatusHealthy, Message: msg}, nil
			} else {
				return &HealthStatus{Health: HealthHealthy, Status: HealthStatusDegraded, Message: msg}, nil
			}
		}

		if c.Type == fluxReady {
			if c.Status == v1.ConditionTrue {
				return &HealthStatus{Ready: true, Health: HealthHealthy, Status: HealthStatusHealthy, Message: msg}, nil
			} else {
				return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusDegraded, Message: msg}, nil
			}
		}
		// All conditions apart from Healthy/Ready should be false
		if c.Status == v1.ConditionTrue {
			return &HealthStatus{
				Health:  HealthUnhealthy,
				Status:  HealthStatusDegraded,
				Message: msg,
			}, nil
		}
	}

	return &HealthStatus{Health: HealthUnknown, Status: HealthStatusUnknown, Message: ""}, nil
}

type helmStatusType string

const (
	helmReleased helmStatusType = "Released"
	helmReady    helmStatusType = "Ready"
)

type fluxHelmRelease struct {
	Status struct {
		Conditions []struct {
			Type    helmStatusType
			Status  v1.ConditionStatus
			Reason  string
			Message string
		}
	}
}

func getFluxHelmReleaseHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var hr fluxHelmRelease
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &hr)
	if err != nil {
		return nil, err
	}

	ranking := map[helmStatusType]int{
		helmReleased: 2,
		helmReady:    1,
	}
	sort.Slice(hr.Status.Conditions, func(i, j int) bool {
		return ranking[hr.Status.Conditions[i].Type] > ranking[hr.Status.Conditions[j].Type]
	})

	for _, c := range hr.Status.Conditions {
		msg := fmt.Sprintf("%s: %s", c.Reason, c.Message)
		if c.Type == helmReleased {
			if c.Status == v1.ConditionTrue {
				return &HealthStatus{Health: HealthHealthy, Status: HealthStatusHealthy, Message: msg}, nil
			} else {
				return &HealthStatus{Health: HealthHealthy, Status: HealthStatusDegraded, Message: msg}, nil
			}
		}

		if c.Type == helmReady {
			if c.Status == v1.ConditionTrue {
				return &HealthStatus{Ready: true, Health: HealthHealthy, Status: HealthStatusHealthy, Message: msg}, nil
			} else {
				return &HealthStatus{Health: HealthUnhealthy, Status: HealthStatusDegraded, Message: msg}, nil
			}
		}
		// All conditions apart from Healthy/Ready should be false
		if c.Status == v1.ConditionTrue {
			return &HealthStatus{
				Health:  HealthUnhealthy,
				Status:  HealthStatusDegraded,
				Message: msg,
			}, nil
		}
	}

	return &HealthStatus{Health: HealthUnknown, Status: HealthStatusUnknown, Message: ""}, nil
}

type fluxRepoStatusType string

const (
	fluxRepoReconciling       fluxRepoStatusType = "Reconciling"
	fluxRepoReady             fluxRepoStatusType = "Ready"
	fluxRepoFetchFailed       fluxRepoStatusType = "FetchFailed"
	fluxRepoArtifactInStorage helmStatusType     = "ArtifactInStorage"
)

type fluxRepo struct {
	Status struct {
		Conditions []struct {
			Type    fluxRepoStatusType
			Status  v1.ConditionStatus
			Reason  string
			Message string
		}
	}
}

func getFluxRepositoryHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var hr fluxRepo
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &hr)
	if err != nil {
		return nil, err
	}

	ranking := map[fluxRepoStatusType]int{
		fluxRepoReady:       3,
		fluxRepoFetchFailed: 2,
		fluxRepoReconciling: 1,
	}
	sort.Slice(hr.Status.Conditions, func(i, j int) bool {
		return ranking[hr.Status.Conditions[i].Type] > ranking[hr.Status.Conditions[j].Type]
	})

	for _, c := range hr.Status.Conditions {
		msg := fmt.Sprintf("%s: %s", c.Reason, c.Message)
		if c.Type == fluxRepoReady {
			if c.Status == v1.ConditionTrue {
				return &HealthStatus{Ready: true, Health: HealthHealthy, Status: HealthStatusHealthy, Message: msg}, nil
			} else {
				return &HealthStatus{Health: HealthHealthy, Status: HealthStatusDegraded, Message: msg}, nil
			}
		}

		// All conditions apart from Healthy/Ready should be false
		if c.Status == v1.ConditionTrue {
			return &HealthStatus{
				Health:  HealthUnhealthy,
				Status:  HealthStatusDegraded,
				Message: msg,
			}, nil
		}
	}

	return &HealthStatus{Health: HealthUnknown, Status: HealthStatusUnknown, Message: ""}, nil
}
