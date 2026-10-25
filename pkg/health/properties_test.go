package health

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProperties(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Health properties")
}

type testProperties struct {
	durations map[string]time.Duration
	integers  map[string]int
	listener  func(*testProperties)
}

func (p *testProperties) Duration(def time.Duration, keys ...string) time.Duration {
	for _, key := range keys {
		if value, ok := p.durations[key]; ok {
			return value
		}
	}
	return def
}

func (p *testProperties) Int(def int, key string) int {
	if value, ok := p.integers[key]; ok {
		return value
	}
	return def
}

func (p *testProperties) RegisterListener(listener func(*testProperties)) {
	p.listener = listener
}

var _ = Describe("Property listener registration", func() {
	var p *testProperties

	BeforeEach(func() {
		expiry, renewal := certExpiryWarningPeriod, certRenewalWarningPeriod
		azure, message := azureClientSecretExpiry, maxMessageLength
		DeferCleanup(func() {
			certExpiryWarningPeriod, certRenewalWarningPeriod = expiry, renewal
			azureClientSecretExpiry, maxMessageLength = azure, message
		})
		certExpiryWarningPeriod = defaultCertExpiryWarningPeriod
		certRenewalWarningPeriod = defaultCertRenewalWarningPeriod
		azureClientSecretExpiry = defaultAzureClientSecretExpiry
		maxMessageLength = defaultMaxMessageLength
		p = &testProperties{
			durations: map[string]time.Duration{
				"health.cert-manager.expiryGracePeriod":      time.Hour * 12,
				"health.cert-manager.renewalGracePeriod":     time.Minute * 10,
				"health.azure.clientSecretExpiryGracePeriod": time.Hour * 24 * 7,
			},
			integers: map[string]int{"health.maxMessageLength": 80},
		}
	})

	It("keeps defaults until the explicitly registered listener receives an update", func() {
		RegisterPropertyListener(p.RegisterListener)
		Expect(p.listener).NotTo(BeNil())
		Expect([]time.Duration{certExpiryWarningPeriod, certRenewalWarningPeriod, azureClientSecretExpiry}).To(Equal(
			[]time.Duration{
				defaultCertExpiryWarningPeriod,
				defaultCertRenewalWarningPeriod,
				defaultAzureClientSecretExpiry,
			},
		))
		Expect(maxMessageLength).To(Equal(defaultMaxMessageLength))

		p.listener(p)
		Expect([]time.Duration{certExpiryWarningPeriod, certRenewalWarningPeriod, azureClientSecretExpiry}).To(Equal(
			[]time.Duration{12 * time.Hour, 10 * time.Minute, 7 * 24 * time.Hour},
		))
		Expect(maxMessageLength).To(Equal(80))
	})

	It("applies subsequent updates and retains configured values when a property is zero", func() {
		RegisterPropertyListener(p.RegisterListener)
		Expect(p.listener).NotTo(BeNil())
		p.listener(p)
		p.durations["health.cert-manager.expiryGracePeriod"] = time.Hour * 6
		p.integers["health.maxMessageLength"] = 120
		p.listener(p)
		Expect(certExpiryWarningPeriod).To(Equal(time.Hour * 6))
		Expect(maxMessageLength).To(Equal(120))

		for key := range p.durations {
			p.durations[key] = 0
		}
		p.integers["health.maxMessageLength"] = 0
		p.listener(p)
		Expect([]time.Duration{certExpiryWarningPeriod, certRenewalWarningPeriod, azureClientSecretExpiry}).To(Equal(
			[]time.Duration{6 * time.Hour, 10 * time.Minute, 7 * 24 * time.Hour},
		))
		Expect(maxMessageLength).To(Equal(120))
	})

	It("restores defaults when properties are removed", func() {
		RegisterPropertyListener(p.RegisterListener)
		Expect(p.listener).NotTo(BeNil())
		p.listener(p)
		clear(p.durations)
		clear(p.integers)
		p.listener(p)
		Expect([]time.Duration{certExpiryWarningPeriod, certRenewalWarningPeriod, azureClientSecretExpiry}).To(Equal(
			[]time.Duration{
				defaultCertExpiryWarningPeriod,
				defaultCertRenewalWarningPeriod,
				defaultAzureClientSecretExpiry,
			},
		))
		Expect(maxMessageLength).To(Equal(defaultMaxMessageLength))
	})
})
