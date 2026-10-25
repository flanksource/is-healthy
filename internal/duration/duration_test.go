package duration

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDuration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Copied duration formatting")
}

var _ = DescribeTable(
	"formats durations without changing units or significant zero digits",
	func(input time.Duration, expected string) {
		Expect(Duration(input).String()).To(Equal(expected))
	},
	Entry("zero", time.Duration(0), "0s"),
	Entry("nanoseconds", 100*time.Nanosecond, "100ns"),
	Entry("microseconds", 500*time.Microsecond, "500µs"),
	Entry("milliseconds", 320*time.Millisecond, "320ms"),
	Entry("fractional seconds", 90*time.Second+123*time.Millisecond, "1m30.123s"),
	Entry("thirty seconds", 30*time.Second, "30s"),
	Entry("thirty minutes", 30*time.Minute, "30m"),
	Entry("negative", -90*time.Minute, "-1h30m"),
	Entry("one day", 24*time.Hour, "24h"),
	Entry("over one day", 24*time.Hour+time.Minute, "1d0h1m"),
	Entry("one week", 7*24*time.Hour, "7d0h"),
	Entry("two weeks", 14*24*time.Hour, "2w0d0h"),
)
