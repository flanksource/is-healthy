package lo

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Copied collection and string helpers")
}

var _ = Describe("Copied helpers", func() {
	DescribeTable(
		"truncates messages by Unicode code points",
		func(input string, limit int, expected string) {
			Expect(Ellipsis(input, limit)).To(Equal(expected))
		},
		Entry("trim", "  healthy  ", 7, "healthy"),
		Entry("emoji", "🙂🙂🙂🙂🙂🙂🙂", 6, "🙂🙂🙂..."),
		Entry("CJK", "资源正在等待就绪", 6, "资源正..."),
		Entry("whitespace at truncation", "ab cdefgh", 6, "ab..."),
		Entry("short limit", "abc", 1, "..."),
	)

	It("preserves Unicode title casing and zero-value pointer handling", func() {
		Expect(Capitalize("échec de mémoire")).To(Equal("Échec De Mémoire"))
		Expect(FromPtr[*string](nil)).To(BeNil())
		Expect(FromPtr(ToPtr("ready"))).To(Equal("ready"))
	})

	It("retains mapping order and removes zero values without deduplicating", func() {
		Expect(Map([]string{"alpha", "beta"}, func(value string, index int) string {
			return Ternary(index == 0, value, "last")
		})).To(Equal([]string{"alpha", "last"}))
		Expect(Compact([]string{"", "ready", "ready", ""})).To(Equal([]string{"ready", "ready"}))
		Expect(CoalesceOrEmpty("", "ready", "failed")).To(Equal("ready"))
		value, found := Coalesce[string]()
		Expect(value).To(BeZero())
		Expect(found).To(BeFalse())
		Expect(Keys(map[string]int{"ready": 1})).To(Equal([]string{"ready"}))
	})
})
