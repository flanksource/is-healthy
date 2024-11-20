package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanCase(t *testing.T) {
	assert.Equal(t, HumanCase("MemoryPressure"), "Memory Pressure")
}
