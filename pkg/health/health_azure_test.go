package health

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// mockNow sets the time for testing and returns a function to restore the original time.
func mockNow(mockTime time.Time) func() {
	originalNow := now
	now = func() time.Time { return mockTime }
	return func() {
		now = originalNow
	}
}

// Helper function to load test data from YAML files
func loadTestData(filePath string) (map[string]any, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read test data file %s: %w", absPath, err)
	}

	var obj map[string]any
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test data from %s: %w", absPath, err)
	}
	return obj, nil
}

func TestGetAzureHealth_ClientSecret(t *testing.T) {
	fixedNow := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	restoreOriginalNow := mockNow(fixedNow)
	defer restoreOriginalNow()

	testCases := []struct {
		name            string
		fixturePath     string
		configType      string
		expectedHealth  Health
		expectedStatus  string
		expectedMessage string
	}{
		{
			name:            "ClientSecret Healthy",
			fixturePath:     "testdata/azure-client-secret-healthy.yaml",
			configType:      "Azure::AppRegistration::ClientSecret",
			expectedHealth:  HealthHealthy,
			expectedStatus:  "Healthy",
			expectedMessage: "ClientSecret is valid",
		},
		{
			name:            "ClientSecret Expiring",
			fixturePath:     "testdata/azure-client-secret-expiring.yaml",
			configType:      "Azure::AppRegistration::ClientSecret",
			expectedHealth:  HealthWarning,
			expectedStatus:  "Expiring",
			expectedMessage: "ClientSecret is expiring in 2w0d0h",
		},
		{
			name:            "ClientSecret Expired",
			fixturePath:     "testdata/azure-client-secret-expired.yaml",
			configType:      "Azure::AppRegistration::ClientSecret",
			expectedHealth:  HealthUnhealthy,
			expectedStatus:  "Expired",
			expectedMessage: "ClientSecret has expired",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := loadTestData(tc.fixturePath)
			if err != nil {
				t.Fatalf("Failed to load test data from %s: %v", tc.fixturePath, err)
			}

			result := GetAzureHealth(tc.configType, obj)

			if result.Health != tc.expectedHealth {
				t.Errorf("Expected health %v, got %v", tc.expectedHealth, result.Health)
			}

			if string(result.Status) != tc.expectedStatus {
				t.Errorf("Expected status %s, got %s", tc.expectedStatus, string(result.Status))
			}

			if result.Message != tc.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tc.expectedMessage, result.Message)
			}
		})
	}
}
