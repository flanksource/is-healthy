package certmanager

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCertificateTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Copied certificate health types")
}

var _ = Describe("Certificate health fields", func() {
	It("preserves certificate metadata and optional timestamps when decoding", func() {
		input := `{
			"metadata":{"name":"example","creationTimestamp":"2026-01-01T00:00:00Z"},
			"status":{"conditions":[{"type":"Issuing","status":"True"}]}
		}`
		var cert Certificate
		Expect(json.Unmarshal([]byte(input), &cert)).To(Succeed())
		Expect(cert.Name).To(Equal("example"))
		Expect(cert.CreationTimestamp.Time).To(BeTemporally("==", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		Expect(cert.Status.NotAfter).To(BeNil())
		Expect(cert.Status.RenewalTime).To(BeNil())
		Expect(cert.Status.Conditions).To(Equal([]CertificateCondition{{
			Type:   CertificateConditionIssuing,
			Status: ConditionTrue,
		}}))
	})

	It("preserves request conditions without requiring a transition timestamp", func() {
		var request CertificateRequest
		Expect(json.Unmarshal([]byte(`{"status":{"conditions":[{
			"type":"Ready","status":"False","reason":"Pending","message":"Waiting for approval"
		}]}}`), &request)).To(Succeed())
		Expect(request.Status.Conditions).To(Equal([]CertificateRequestCondition{{
			Type:    CertificateRequestConditionReady,
			Status:  ConditionFalse,
			Reason:  CertificateRequestReasonPending,
			Message: "Waiting for approval",
		}}))
	})

	DescribeTable(
		"rejects malformed consumed certificate fields",
		func(input string) {
			var cert Certificate
			Expect(json.Unmarshal([]byte(input), &cert)).NotTo(Succeed())
		},
		Entry("expiration", `{"status":{"notAfter":"invalid"}}`),
		Entry("condition type", `{"status":{"conditions":[{"type":123}]}}`),
		Entry("condition timestamp", `{"status":{"conditions":[{"lastTransitionTime":"invalid"}]}}`),
	)
})
