package health

import (
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defaultCertExpiryWarningPeriod  = time.Hour * 24 * 2
	defaultCertRenewalWarningPeriod = time.Minute * 30
)

var (
	certExpiryWarningPeriod  = defaultCertExpiryWarningPeriod
	certRenewalWarningPeriod = defaultCertRenewalWarningPeriod
)

// https://github.com/cert-manager/cert-manager/blob/cb20920fcf80c73ab6310470d5464d40e22db492/internal/controller/certificates/policies/constants.go
const (
	// DoesNotExist is a policy violation reason for a scenario where
	// Certificate's spec.secretName secret does not exist.
	DoesNotExist string = "DoesNotExist"
	// MissingData is a policy violation reason for a scenario where
	// Certificate's spec.secretName secret has missing data.
	MissingData string = "MissingData"
	// InvalidKeyPair is a policy violation reason for a scenario where public
	// key of certificate does not match private key.
	InvalidKeyPair string = "InvalidKeyPair"
	// InvalidCertificate is a policy violation whereby the signed certificate in
	// the Input Secret could not be parsed or decoded.
	InvalidCertificate string = "InvalidCertificate"
	// InvalidCertificateRequest is a policy violation whereby the CSR in
	// the Input CertificateRequest could not be parsed or decoded.
	InvalidCertificateRequest string = "InvalidCertificateRequest"

	// SecretMismatch is a policy violation reason for a scenario where Secret's
	// private key does not match spec.
	SecretMismatch string = "SecretMismatch"
	// IncorrectIssuer is a policy violation reason for a scenario where
	// Certificate has been issued by incorrect Issuer.
	IncorrectIssuer string = "IncorrectIssuer"
	// IncorrectCertificate is a policy violation reason for a scenario where
	// the Secret referred to by this Certificate's spec.secretName,
	// already has a `cert-manager.io/certificate-name` annotation
	// with the name of another Certificate.
	IncorrectCertificate string = "IncorrectCertificate"
	// RequestChanged is a policy violation reason for a scenario where
	// CertificateRequest not valid for Certificate's spec.
	RequestChanged string = "RequestChanged"
	// Renewing is a policy violation reason for a scenario where
	// Certificate's renewal time is now or in the past.
	Renewing string = "Renewing"
	// Expired is a policy violation reason for a scenario where Certificate has
	// expired.
	Expired string = "Expired"
	// SecretTemplateMisMatch is a policy violation whereby the Certificate's
	// SecretTemplate is not reflected on the target Secret, either by having
	// extra, missing, or wrong Annotations or Labels.
	SecretTemplateMismatch string = "SecretTemplateMismatch"
	// SecretManagedMetadataMismatch is a policy violation whereby the Secret is
	// missing labels that should have been added by cert-manager
	SecretManagedMetadataMismatch string = "SecretManagedMetadataMismatch"

	// AdditionalOutputFormatsMismatch is a policy violation whereby the
	// Certificate's AdditionalOutputFormats is not reflected on the target
	// Secret, either by having extra, missing, or wrong values.
	AdditionalOutputFormatsMismatch string = "AdditionalOutputFormatsMismatch"
	// ManagedFieldsParseError is a policy violation whereby cert-manager was
	// unable to decode the managed fields on a resource.
	ManagedFieldsParseError string = "ManagedFieldsParseError"
	// SecretOwnerRefMismatch is a policy violation whereby the Secret either has
	// a missing owner reference to the Certificate, or has an owner reference it
	// shouldn't have.
	SecretOwnerRefMismatch string = "SecretOwnerRefMismatch"
)

func GetCertificateRequestHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var certReq certmanagerv1.CertificateRequest
	if err := convertFromUnstructured(obj, &certReq); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured certificateRequest to typed: %w", err)
	}

	conditionMap := make(map[certmanagerv1.CertificateRequestConditionType]certmanagerv1.CertificateRequestCondition)
	for _, condition := range certReq.Status.Conditions {
		if condition.Type == certmanagerv1.CertificateRequestConditionReady &&
			condition.Status == cmmeta.ConditionFalse {
			// If the ready condition hasn't been met, look for failures or denial.
			// Can initially be approved but then get failed (eg: CommonName mismatch)
			switch condition.Reason {
			case certmanagerv1.CertificateRequestReasonFailed, certmanagerv1.CertificateRequestReasonDenied:
				return &HealthStatus{
					Health:  HealthUnhealthy,
					Message: condition.Message,
					Status:  HealthStatusCode(condition.Reason),
					Ready:   true,
				}, nil

			case certmanagerv1.CertificateRequestReasonPending:
				health := HealthUnknown

				durationInPendingState := time.Since(obj.GetCreationTimestamp().Time)
				if durationInPendingState > certRenewalWarningPeriod {
					health = HealthUnhealthy
				}

				return &HealthStatus{
					Health:  health,
					Message: condition.Message,
					Status:  HealthStatusCode(condition.Reason),
				}, nil
			}
		}

		if condition.Status == cmmeta.ConditionTrue {
			conditionMap[condition.Type] = condition
		}
	}

	if cr, ok := conditionMap[certmanagerv1.CertificateRequestConditionDenied]; ok {
		return &HealthStatus{
			Health:  HealthUnhealthy,
			Message: cr.Message,
			Status:  HealthStatusCode(cr.Type),
			Ready:   true,
		}, nil
	}

	if cr, ok := conditionMap[certmanagerv1.CertificateRequestConditionInvalidRequest]; ok {
		return &HealthStatus{
			Health:  HealthUnhealthy,
			Message: cr.Message,
			Status:  HealthStatusCode(cr.Type),
			Ready:   true,
		}, nil
	}

	if cr, ok := conditionMap[certmanagerv1.CertificateRequestConditionReady]; ok {
		return &HealthStatus{
			Health:  HealthHealthy,
			Message: cr.Message,
			Status:  HealthStatusCode(cr.Reason),
			Ready:   true,
		}, nil
	}

	if cr, ok := conditionMap[certmanagerv1.CertificateRequestConditionApproved]; ok {
		// approved but not issued
		h := &HealthStatus{
			Health:  HealthHealthy,
			Message: cr.Message,
			Status:  HealthStatusCode(cr.Type),
			Ready:   false,
		}

		if time.Since(certReq.CreationTimestamp.Time) > time.Hour {
			h.Health = HealthUnhealthy
		}

		return h, nil
	}

	status, err := GetDefaultHealth(obj)
	if err != nil {
		return status, err
	}

	return status, nil
}

func GetCertificateHealth(obj *unstructured.Unstructured) (*HealthStatus, error) {
	var cert certmanagerv1.Certificate
	if err := convertFromUnstructured(obj, &cert); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured certificate to typed: %w", err)
	}

	for _, c := range cert.Status.Conditions {
		if c.Type == certmanagerv1.CertificateConditionIssuing {
			hs := &HealthStatus{
				Status:  HealthStatusCode(c.Reason),
				Ready:   false,
				Message: c.Message,
			}

			switch c.Reason {
			case "ManuallyTriggered":
				// We check for expiry below
				hs.Status = "Issuing"
				hs.Health = HealthUnknown

			case DoesNotExist:
				inIssuingState := time.Since(obj.GetCreationTimestamp().Time)
				if inIssuingState > time.Minute*30 {
					return &HealthStatus{
						Status:  HealthStatusCode(c.Reason),
						Health:  HealthUnhealthy,
						Message: c.Message,
						Ready:   false,
					}, nil
				}

				// We check for expiry below
				hs.Status = "Issuing"
				hs.Health = HealthUnknown

			case Renewing:
				renewalTime := cert.Status.RenewalTime.Time

				if time.Since(renewalTime) > certRenewalWarningPeriod {
					hs.Health = HealthWarning
					hs.Message = fmt.Sprintf(
						"Certificate has been in renewal state for > %s",
						time.Since(renewalTime).Truncate(time.Minute),
					)
				} else {
					hs.Health = HealthHealthy
				}

			case "Failed":
				return &HealthStatus{
					Status:  HealthStatusCode(c.Reason),
					Health:  HealthUnhealthy,
					Message: c.Message,
					Ready:   true,
				}, nil

			default:
				unhealthyReasons := map[string]string{
					MissingData:                     "Certificate secret has missing data",
					InvalidKeyPair:                  "Public key of certificate does not match private key",
					InvalidCertificate:              "Signed certificate could not be parsed or decoded",
					InvalidCertificateRequest:       "CSR could not be parsed or decoded",
					SecretMismatch:                  "Secret's private key does not match spec",
					IncorrectIssuer:                 "Certificate has been issued by incorrect Issuer",
					IncorrectCertificate:            "Certificate's secretName already has an annotation with another Certificate",
					Expired:                         "Certificate has expired",
					SecretTemplateMismatch:          "SecretTemplate is not reflected on the target Secret",
					SecretManagedMetadataMismatch:   "Secret is missing labels that should have been added by cert-manager",
					AdditionalOutputFormatsMismatch: "Certificate's AdditionalOutputFormats are not reflected on the target Secret",
					ManagedFieldsParseError:         "cert-manager was unable to decode the managed fields on a resource",
					SecretOwnerRefMismatch:          "Secret has an incorrect owner reference",
				}

				if msg, exists := unhealthyReasons[string(c.Reason)]; exists {
					return &HealthStatus{
						Health:  HealthUnhealthy,
						Status:  HealthStatusCode(c.Reason),
						Message: msg,
						Ready:   true,
					}, nil
				} else if cert.Status.NotBefore != nil {
					if overdue := time.Since(cert.Status.NotBefore.Time); overdue > time.Hour {
						hs.Health = HealthUnhealthy
						return hs, nil
					} else if overdue > time.Minute*15 {
						hs.Health = HealthWarning
						return hs, nil
					}
				}
			}

			// If we're issuing a new cert, at least ensure the existing cert hasn't expired
			if expiryHealth := certExpiryCheck(cert); expiryHealth != nil {
				return expiryHealth, nil
			} else {
				return hs, nil
			}
		}
	}

	if expiryHealth := certExpiryCheck(cert); expiryHealth != nil {
		return expiryHealth, nil
	}

	if cert.Status.RenewalTime != nil {
		renewalTime := cert.Status.RenewalTime.Time

		if time.Since(renewalTime) > certRenewalWarningPeriod {
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

func certExpiryCheck(cert certmanagerv1.Certificate) *HealthStatus {
	if cert.Status.NotAfter == nil {
		return nil
	}

	notAfterTime := cert.Status.NotAfter.Time
	if notAfterTime.Before(time.Now()) {
		return &HealthStatus{
			Health:  HealthUnhealthy,
			Status:  "Expired",
			Message: "Certificate has expired",
			Ready:   true,
		}
	}

	if time.Until(notAfterTime) < certExpiryWarningPeriod {
		return &HealthStatus{
			Health:  HealthWarning,
			Status:  HealthStatusWarning,
			Message: fmt.Sprintf("Certificate is expiring soon (%s)", notAfterTime),
			Ready:   true,
		}
	}

	return nil
}
