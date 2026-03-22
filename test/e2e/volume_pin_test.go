//go:build linux || freebsd

package integration

import (
	"fmt"

	. "github.com/containers/podman/v6/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman volume pin", func() {
	AfterEach(func() {
		session := podmanTest.Podman([]string{"volume", "rm", "-fa", "--include-pinned"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())
	})

	It("podman volume pin basic", func() {
		volName := "test-pin-vol"
		session := podmanTest.Podman([]string{"volume", "create", volName})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())

		session = podmanTest.Podman([]string{"volume", "pin", volName})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())
		Expect(session.OutputToString()).To(ContainSubstring("is now pinned"))

		inspect := podmanTest.Podman([]string{"volume", "inspect", "--format", "{{.Pinned}}", volName})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect).Should(ExitCleanly())
		Expect(inspect.OutputToString()).To(Equal("true"))
	})

	It("podman volume pin --unpin", func() {
		volName := "test-unpin-vol"
		podmanTest.PodmanExitCleanly("volume", "create", "--pinned", volName)

		session := podmanTest.Podman([]string{"volume", "pin", "--unpin", volName})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())
		Expect(session.OutputToString()).To(ContainSubstring("is now unpinned"))

		inspect := podmanTest.Podman([]string{"volume", "inspect", "--format", "{{.Pinned}}", volName})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect).Should(ExitCleanly())
		Expect(inspect.OutputToString()).To(Equal("false"))
	})

	It("podman volume pin idempotent", func() {
		volName := "test-pin-idempotent"
		podmanTest.PodmanExitCleanly("volume", "create", "--pinned", volName)

		session := podmanTest.Podman([]string{"volume", "pin", volName})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())
		Expect(session.OutputToString()).To(ContainSubstring("is now pinned"))

		inspect := podmanTest.Podman([]string{"volume", "inspect", "--format", "{{.Pinned}}", volName})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect).Should(ExitCleanly())
		Expect(inspect.OutputToString()).To(Equal("true"))
	})

	It("podman volume pin --unpin idempotent", func() {
		volName := "test-unpin-idempotent"
		podmanTest.PodmanExitCleanly("volume", "create", volName)

		session := podmanTest.Podman([]string{"volume", "pin", "--unpin", volName})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())

		inspect := podmanTest.Podman([]string{"volume", "inspect", "--format", "{{.Pinned}}", volName})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect).Should(ExitCleanly())
		Expect(inspect.OutputToString()).To(Equal("false"))
	})

	It("podman volume pin multiple volumes", func() {
		podmanTest.PodmanExitCleanly("volume", "create", "vol1")
		podmanTest.PodmanExitCleanly("volume", "create", "vol2")
		podmanTest.PodmanExitCleanly("volume", "create", "vol3")

		session := podmanTest.Podman([]string{"volume", "pin", "vol1", "vol2", "vol3"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())

		for _, name := range []string{"vol1", "vol2", "vol3"} {
			inspect := podmanTest.Podman([]string{"volume", "inspect", "--format", "{{.Pinned}}", name})
			inspect.WaitWithDefaultTimeout()
			Expect(inspect).Should(ExitCleanly())
			Expect(inspect.OutputToString()).To(Equal("true"))
		}
	})

	It("podman volume pin non-existent volume", func() {
		session := podmanTest.Podman([]string{"volume", "pin", "no-such-vol"})
		session.WaitWithDefaultTimeout()
		Expect(session).NotTo(ExitCleanly())
	})

	It("podman volume pin protects from volume rm", func() {
		volName := "pin-rm-protect"
		podmanTest.PodmanExitCleanly("volume", "create", volName)

		podmanTest.PodmanExitCleanly("volume", "pin", volName)

		session := podmanTest.Podman([]string{"volume", "rm", volName})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError(125, fmt.Sprintf("volume %s is pinned and cannot be removed without --include-pinned flag", volName)))
	})

	It("podman volume pin protects from volume prune", func() {
		volName := "pin-prune-protect"
		podmanTest.PodmanExitCleanly("volume", "create", volName)

		podmanTest.PodmanExitCleanly("volume", "pin", volName)

		session := podmanTest.Podman([]string{"volume", "prune", "--force"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(ExitCleanly())

		inspect := podmanTest.Podman([]string{"volume", "inspect", volName})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect).Should(ExitCleanly())
	})

	It("podman volume pin without args fails", func() {
		session := podmanTest.Podman([]string{"volume", "pin"})
		session.WaitWithDefaultTimeout()
		Expect(session).NotTo(ExitCleanly())
	})

	It("podman rm -v preserves pinned anonymous volume", func() {
		ctrName := "pinned-anon-ctr"
		podmanTest.PodmanExitCleanly("create", "--name", ctrName, "-v", "/data", ALPINE, "top")

		list := podmanTest.PodmanExitCleanly("volume", "ls", "-q")
		arr := list.OutputToStringArray()
		Expect(arr).To(HaveLen(1))
		anonVolName := arr[0]

		podmanTest.PodmanExitCleanly("volume", "pin", anonVolName)

		podmanTest.PodmanExitCleanly("rm", "-v", ctrName)

		list2 := podmanTest.PodmanExitCleanly("volume", "ls", "-q")
		Expect(list2.OutputToStringArray()).To(ContainElement(anonVolName))
	})

	It("podman pod rm preserves pinned anonymous volume", func() {
		podName := "pinned-anon-pod"
		podmanTest.PodmanExitCleanly("pod", "create", "--name", podName)
		podmanTest.PodmanExitCleanly("create", "--pod", podName, "-v", "/data", ALPINE, "top")

		list := podmanTest.PodmanExitCleanly("volume", "ls", "-q")
		arr := list.OutputToStringArray()
		Expect(arr).To(HaveLen(1))
		anonVolName := arr[0]

		podmanTest.PodmanExitCleanly("volume", "pin", anonVolName)

		podmanTest.PodmanExitCleanly("pod", "rm", podName)

		list2 := podmanTest.PodmanExitCleanly("volume", "ls", "-q")
		Expect(list2.OutputToStringArray()).To(ContainElement(anonVolName))
	})

	It("podman system prune --volumes preserves pinned volume", func() {
		useCustomNetworkDir(podmanTest, tempdir)
		pinnedVolName := "pinned-sys-prune-vol"
		unpinnedVolName := "unpinned-sys-prune-vol"

		podmanTest.PodmanExitCleanly("volume", "create", "--pinned", pinnedVolName)
		podmanTest.PodmanExitCleanly("volume", "create", unpinnedVolName)

		podmanTest.PodmanExitCleanly("system", "prune", "--force", "--volumes")

		list := podmanTest.PodmanExitCleanly("volume", "ls", "-q")
		Expect(list.OutputToStringArray()).To(ContainElement(pinnedVolName))
		Expect(list.OutputToStringArray()).To(Not(ContainElement(unpinnedVolName)))
	})

	It("podman image rm fails for pinned image-backed volume", func() {
		podmanTest.AddImageToRWStore(FEDORA_MINIMAL)
		volName := "pinned-img-vol"

		podmanTest.PodmanExitCleanly("volume", "create", "--driver", "image", "--opt", fmt.Sprintf("image=%s", FEDORA_MINIMAL), volName)
		podmanTest.PodmanExitCleanly("volume", "pin", volName)

		session := podmanTest.Podman([]string{"rmi", FEDORA_MINIMAL})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError(125, fmt.Sprintf("volume %s is pinned", volName)))
	})
})
