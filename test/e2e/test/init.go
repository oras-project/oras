package test

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var OrasPath string
var Host string

func init() {
	OrasPath = os.Getenv("ORAS_PATH")
	Host = os.Getenv("REGISTRY_HOST")

	var _ = BeforeSuite(func() {
		if OrasPath == "" {
			// fallback to native build to facilitate locally debugging
			var err error
			OrasPath, err = gexec.Build("oras.land/oras/cmd/oras")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			DeferCleanup(gexec.CleanupBuildArtifacts)
			return
		}
		wd, err := os.Getwd()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		OrasPath = filepath.Join(wd, OrasPath)
		fmt.Printf("Testing based on binary locates in %q\n", OrasPath)
	})

}
