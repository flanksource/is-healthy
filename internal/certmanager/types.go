/*
Copyright The cert-manager Authors.

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

// Copied from github.com/cert-manager/cert-manager v1.20.0:
// https://github.com/cert-manager/cert-manager/blob/v1.20.0/pkg/apis/certmanager/v1/types_certificate.go
// https://github.com/cert-manager/cert-manager/blob/v1.20.0/pkg/apis/certmanager/v1/types_certificaterequest.go
// https://github.com/cert-manager/cert-manager/blob/v1.20.0/pkg/apis/meta/v1/types.go
// Adapted to one package retaining only metadata, health status fields, and conditions.
package certmanager

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            CertificateStatus `json:"status"`
}

type CertificateStatus struct {
	Conditions  []CertificateCondition `json:"conditions,omitempty"`
	NotAfter    *metav1.Time           `json:"notAfter,omitempty"`
	RenewalTime *metav1.Time           `json:"renewalTime,omitempty"`
}

type CertificateCondition struct {
	Type               CertificateConditionType `json:"type"`
	Status             ConditionStatus          `json:"status"`
	LastTransitionTime *metav1.Time             `json:"lastTransitionTime,omitempty"`
	Reason             string                   `json:"reason,omitempty"`
	Message            string                   `json:"message,omitempty"`
	ObservedGeneration int64                    `json:"observedGeneration,omitempty"`
}

type CertificateConditionType string

const (
	CertificateConditionReady   CertificateConditionType = "Ready"
	CertificateConditionIssuing CertificateConditionType = "Issuing"
)

type CertificateRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            CertificateRequestStatus `json:"status"`
}

type CertificateRequestStatus struct {
	Conditions []CertificateRequestCondition `json:"conditions,omitempty"`
}

type CertificateRequestCondition struct {
	Type               CertificateRequestConditionType `json:"type"`
	Status             ConditionStatus                 `json:"status"`
	LastTransitionTime *metav1.Time                    `json:"lastTransitionTime,omitempty"`
	Reason             string                          `json:"reason,omitempty"`
	Message            string                          `json:"message,omitempty"`
}

type CertificateRequestConditionType string

const (
	CertificateRequestConditionReady          CertificateRequestConditionType = "Ready"
	CertificateRequestConditionInvalidRequest CertificateRequestConditionType = "InvalidRequest"
	CertificateRequestConditionApproved       CertificateRequestConditionType = "Approved"
	CertificateRequestConditionDenied         CertificateRequestConditionType = "Denied"
)

const (
	CertificateRequestReasonPending = "Pending"
	CertificateRequestReasonFailed  = "Failed"
	CertificateRequestReasonDenied  = "Denied"
)
