package health

import (
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var defaultCertExpiryWarningPeriod = time.Hour * 24 * 2

func SetDefaultCertificateExpiryWarningPeriod(p time.Duration) {
	defaultCertExpiryWarningPeriod = p
}

func GetCertificateRequestHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var certReq certmanagerv1.CertificateRequest
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &certReq); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured certificateRequest to typed: %w", err)
	}

	for _, cr := range certReq.Status.Conditions {
		if string(cr.Status) != string(v1.ConditionTrue) {
			continue
		}

		if cr.Type == "Approved" {
			return &HealthStatus{
				Health:  HealthHealthy,
				Message: cr.Message,
				Status:  HealthStatusCode(cr.Type),
				Ready:   true,
			}, nil
		}
	}

	status, err := GetDefaultHealth(obj)
	if err != nil {
		return status, err
	}

	return status, nil
}

func GetCertificateHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var cert certmanagerv1.Certificate
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &cert); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured certificate to typed: %w", err)
	}

	for _, c := range cert.Status.Conditions {
		if string(c.Status) != string(v1.ConditionTrue) {
			continue
		}

		if c.Type == "Issuing" && cert.Status.NotBefore != nil {
			hs := &HealthStatus{
				Status:  HealthStatusCode(c.Reason),
				Ready:   false,
				Message: c.Message,
			}

			if overdue := time.Since(cert.Status.NotBefore.Time); overdue > time.Hour {
				hs.Health = HealthUnhealthy
				return hs, nil
			} else if overdue > time.Minute*15 {
				hs.Health = HealthWarning
				return hs, nil
			}
		}
	}

	if cert.Status.NotAfter != nil {
		notAfterTime := cert.Status.NotAfter.Time
		if notAfterTime.Before(time.Now()) {
			return &HealthStatus{
				Health:  HealthUnhealthy,
				Status:  "Expired",
				Message: "Certificate has expired",
				Ready:   true,
			}, nil
		}

		if time.Until(notAfterTime) < defaultCertExpiryWarningPeriod {
			return &HealthStatus{
				Health:  HealthWarning,
				Status:  HealthStatusWarning,
				Message: fmt.Sprintf("Certificate is expiring soon (%s)", notAfterTime),
				Ready:   true,
			}, nil
		}
	}

	if cert.Status.RenewalTime != nil {
		renewalTime := cert.Status.RenewalTime.Time

		if time.Since(renewalTime) > time.Minute*5 {
			return &HealthStatus{
				Health:  HealthWarning,
				Status:  HealthStatusWarning,
				Message: fmt.Sprintf("Certificate should have been renewed at %s", renewalTime),
				Ready:   true,
			}, nil
		}
	}

	status, err := GetDefaultHealth(obj)
	if err != nil {
		return status, err
	}

	return status, nil
}
