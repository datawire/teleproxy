package dtest

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"github.com/datawire/teleproxy/pkg/supervisor"
)

const scope = "dtest"
const prefix = "DTEST"

func lines(str string) []string {
	var result []string

	for _, l := range strings.Split(str, "\n") {
		l := strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}

	return result
}

func dockerPs(args ...string) []string {
	cmd := supervisor.Command(prefix, "docker", append([]string{"ps", "-q", "-f", fmt.Sprintf("label=scope=%s", scope)},
		args...)...)
	return lines(cmd.MustCapture(nil))
}

func tag2id(tag string) string {
	result := dockerPs("-f", fmt.Sprintf("label=%s", tag))
	switch len(result) {
	case 0:
		return ""
	case 1:
		return result[0]
	default:
		panic(fmt.Sprintf("expecting zero or one containers with label scope=%s and label %s", scope, tag))
	}
}

func dockerUp(tag string, args ...string) string {
	var id string

	WithNamedMachineLock("docker", func() {
		id = tag2id(tag)

		if id == "" {
			cmd := supervisor.Command(prefix, "docker", append([]string{"run", "-d", "-l",
				fmt.Sprintf("scope=%s", scope), "-l", tag, "--rm"}, args...)...)
			out := cmd.MustCapture(nil)
			id = strings.TrimSpace(out)[:12]
		}
	})

	return id
}

func dockerKill(ids ...string) {
	if len(ids) > 0 {
		cmd := supervisor.Command(prefix, "docker", append([]string{"kill"}, ids...)...)
		cmd.MustCapture(nil)
	}
}

func isK3sStarted() bool {
	id := tag2id("k3s")
	if id == "" {
		return false
	}

	cmd := supervisor.Command(prefix, "docker", "logs", id)
	output := cmd.MustCaptureErr(nil)
	return strings.Contains(output, "Wrote kubeconfig")
}

const k3sConfigPath = "/etc/rancher/k3s/k3s.yaml"

func getKubeconfig() string {
	id := tag2id("k3s")

	if id == "" {
		return ""
	}

	cmd := supervisor.Command(prefix, "sh", "-c", fmt.Sprintf("docker cp \"%s:%s\" - | tar -xO", id, k3sConfigPath))
	kubeconfig := cmd.MustCapture(nil)
	kubeconfig = strings.ReplaceAll(kubeconfig, "localhost:6443", fmt.Sprintf("%s:%s", dockerIP(), k3sPort))
	return kubeconfig
}

const dtestRegistry = "DTEST_REGISTRY"
const registryPort = "5000"

func regUp() string {
	return dockerUp("registry",
		"-p", fmt.Sprintf("%s:6443", k3sPort),
		"-p", fmt.Sprintf("%s:%s", registryPort, registryPort),
		"-e", fmt.Sprintf("REGISTRY_HTTP_ADDR=0.0.0.0:%s", registryPort),
		"registry:2")
}

func dockerIP() string {
	if runtime.GOOS == "darwin" {
		return supervisor.Command(prefix, "docker-machine", "ip").MustCapture(nil)
	}

	return "localhost"
}

// DockerRegistry returns a docker registry suitable for use in tests.
func DockerRegistry() string {
	registry := os.Getenv(dtestRegistry)
	if registry != "" {
		return registry
	}

	regUp()

	return fmt.Sprintf("%s:%s", dockerIP(), registryPort)
}

const dtestKubeconfig = "DTEST_KUBECONFIG"
const k3sPort = "6443"
const k3sImage = "rancher/k3s:v0.6.1"

const msg = `
kubeconfig does not exist: %s

  Make sure DTEST_KUBECONFIG is either unset or points to a valid kubeconfig file.

`

// Kubeconfig returns a path referencing a kubeconfig file suitable for use in tests.
func Kubeconfig() string {
	kubeconfig := os.Getenv(dtestKubeconfig)
	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			fmt.Printf(msg, kubeconfig)
			os.Exit(1)
		}

		return kubeconfig
	}

	regid := regUp()

	id := dockerUp("k3s", "--privileged", "--network", fmt.Sprintf("container:%s", regid),
		k3sImage, "server", "--node-name", "localhost", "--no-deploy", "traefik")
	for {
		if isK3sStarted() {
			break
		} else {
			time.Sleep(time.Second)
		}
	}

	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	kubeconfig = fmt.Sprintf("/tmp/dtest-kubeconfig-%s-%s.yaml", user.Username, id)
	contents := getKubeconfig()

	err = ioutil.WriteFile(kubeconfig, []byte(contents), 0644)

	if err != nil {
		panic(err)
	}

	return kubeconfig
}
