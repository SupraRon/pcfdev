package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	dockerID   string
	pwd        string
	binaryPath string
)

var _ = BeforeSuite(func() {
	var err error
	pwd, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	output, err := exec.Command("docker", "run", "-d", "-w", "/go/src/pcfdev", "-v", pwd+":/go/src/pcfdev", "golang:1.6", "sleep", "1000").Output()
	Expect(err).NotTo(HaveOccurred())
	dockerID = strings.TrimSpace(string(output))

	err = exec.Command("bash", "-c", "echo \"#!/bin/bash\necho 'Waiting for services to start...'\necho \\$@\" > "+pwd+"/provision-script").Run()
	Expect(err).NotTo(HaveOccurred())

	err = exec.Command("docker", "exec", dockerID, "chmod", "+x", "/go/src/pcfdev/provision-script").Run()
	Expect(err).NotTo(HaveOccurred())

	err = exec.Command("docker", "exec", dockerID, "go", "build", "-ldflags", "-X main.provisionScriptPath=/go/src/pcfdev/provision-script", "pcfdev").Run()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	os.RemoveAll(filepath.Join(pwd, "pcfdev"))
	os.RemoveAll(filepath.Join(pwd, "provision-script"))
	Expect(exec.Command("docker", "rm", dockerID, "-f").Run()).To(Succeed())
})

var _ = Describe("PCF Dev provision", func() {
	It("should provision PCF Dev", func() {
		session, err := gexec.Start(exec.Command("docker", "exec", dockerID, "/go/src/pcfdev/pcfdev"), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
	})

	It("should pass arguments along", func() {
		session, err := gexec.Start(exec.Command("docker", "exec", dockerID, "/go/src/pcfdev/pcfdev", "local.pcfdev.io"), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session).To(gbytes.Say("Waiting for services to start..."))
		Expect(session).To(gbytes.Say("local.pcfdev.io"))
	})

	Context("when provisioning fails", func() {
		BeforeEach(func() {
			err := exec.Command("bash", "-c", "echo \"#!/bin/bash\nexit 42\" > "+pwd+"/provision-script").Run()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should exit with the exit status of the provision script", func() {
			session, _ := gexec.Start(exec.Command("docker", "exec", dockerID, "/go/src/pcfdev/pcfdev"), GinkgoWriter, GinkgoWriter)
			Eventually(session).Should(gexec.Exit(42))
		})
	})
})
