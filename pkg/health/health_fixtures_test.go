package health_test

import (
	"testing"

	"github.com/bmatcuk/doublestar/v4"
)

func TestFixtures(t *testing.T) {
	files, err := doublestar.FilepathGlob("testdata/*/**/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test files found")
	}

	for _, file := range files {
		// if file != "testdata/Kubernetes/MissionControl/scrapeConfig-minor-delay.yaml" {
		// 	continue
		// }

		testFixture(t, file)
	}
}
