package health

import "time"

// Properties provides the settings read by health checks.
type Properties interface {
	Duration(time.Duration, ...string) time.Duration
	Int(int, string) int
}

// RegisterPropertyListener opts into property updates from the supplied registrar.
// For commons, pass properties.RegisterListener or an instance's RegisterListener method.
// Settings are applied when the registrar invokes the callback, not at registration.
func RegisterPropertyListener[P Properties](register func(func(P))) {
	register(func(p P) {
		if v := p.Duration(defaultCertExpiryWarningPeriod, "health.cert-manager.expiryGracePeriod"); v != 0 {
			certExpiryWarningPeriod = v
		}

		if v := p.Duration(defaultCertRenewalWarningPeriod, "health.cert-manager.renewalGracePeriod"); v != 0 {
			certRenewalWarningPeriod = v
		}

		if v := p.Duration(defaultAzureClientSecretExpiry, "health.azure.clientSecretExpiryGracePeriod"); v != 0 {
			azureClientSecretExpiry = v
		}

		if v := p.Int(defaultMaxMessageLength, "health.maxMessageLength"); v != 0 {
			maxMessageLength = v
		}
	})
}
