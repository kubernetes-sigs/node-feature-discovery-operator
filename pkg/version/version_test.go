package version_test

import (
	"runtime/debug"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/node-feature-discovery-operator/pkg/version"
)

func TestPkgVersion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/version Suite")
}

var _ = Describe("GetWithVCSRevision", func() {
	const v = "1.2.3"

	It(`should append "undefined" when no VCS revision could be read`, func() {
		Expect(
			version.GetWithVCSRevision(v, &debug.BuildInfo{}),
		).To(
			Equal(v + "-undefined"),
		)
	})

	It("should append vcs.revision to the version if it is defined in the build settings", func() {
		bi := debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "somehash"},
			},
		}

		Expect(
			version.GetWithVCSRevision(v, &bi),
		).To(
			Equal(v + "-somehash"),
		)
	})

	bsRev := debug.BuildSetting{Key: "vcs.revision", Value: "abc"}

	DescribeTable("should append -dirty if vcs.dirty is true",
		func(bs []debug.BuildSetting, expected string) {
			Expect(
				version.GetWithVCSRevision(v, &debug.BuildInfo{Settings: bs}),
			).To(
				Equal(expected),
			)
		},
		Entry("no vcs.modified", []debug.BuildSetting{bsRev}, "1.2.3-abc"),
		Entry("vcs.modified=false", []debug.BuildSetting{bsRev, {Key: "vcs.modified", Value: "false"}}, "1.2.3-abc"),
		Entry("vcs.modified=true", []debug.BuildSetting{bsRev, {Key: "vcs.modified", Value: "true"}}, "1.2.3-abc-dirty"),
	)
})
