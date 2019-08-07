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

// GetKubeconfig returns the kubeconfig contents for the running k3s
// cluster as a string. It will return the empty string if no cluster
// is running.
func GetKubeconfig() string {
	id := tag2id("k3s")

	if id == "" {
		return ""
	}

	cmd := supervisor.Command(prefix, "sh", "-c", fmt.Sprintf("docker cp \"%s:%s\" - | tar -xO", id, k3sConfigPath))
	kubeconfig := cmd.MustCapture(nil)
	kubeconfig = strings.ReplaceAll(kubeconfig, "localhost:6443", fmt.Sprintf("%s:%s", dockerIp(), k3sPort))
	return kubeconfig
}

const dtestRegistry = "DTEST_REGISTRY"
const registryPort = "5000"

// RegistryUp will launch if necessary and return the docker id of a
// container running a docker registry.
func RegistryUp() string {
	return dockerUp("registry",
		"-p", fmt.Sprintf("%s:6443", k3sPort),
		"-p", fmt.Sprintf("%s:%s", registryPort, registryPort),
		"-e", fmt.Sprintf("REGISTRY_HTTP_ADDR=0.0.0.0:%s", registryPort),
		"registry:2")
}

func isDockerMachine() bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	if os.Getenv("DOCKER_HOST") != "" {
		return true
	}

	if os.Getenv("DOCKER_MACHINE_NAME") != "" {
		return true
	}

	return false
}

func dockerIP() string {
	return "localhost"
}

// DockerRegistry returns a docker registry suitable for use in tests.
func DockerRegistry() string {
	registry := os.Getenv(dtestRegistry)
	if registry != "" {
		return registry
	}

	RegistryUp()

	return fmt.Sprintf("%s:%s", dockerIp(), registryPort)
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

	id := K3sUp()

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
	contents := GetKubeconfig()

	err = ioutil.WriteFile(kubeconfig, []byte(contents), 0644)

	if err != nil {
		panic(err)
	}

	return kubeconfig
}

// K3sUp will launch if necessary and return the docker id of a
// container running a k3s cluster.
func K3sUp() string {
	regid := RegistryUp()
	return dockerUp("k3s", "--privileged", "--network", fmt.Sprintf("container:%s", regid),
		k3sImage, "server", "--node-name", "localhost", "--no-deploy", "traefik")
}

// K3sDown shuts down the k3s cluster.
func K3sDown() string {
	id := tag2id("k3s")
	if id != "" {
		dockerKill(id)
	}
	return id
}

// RegistryDown shutsdown the test registry.
func RegistryDown() string {
	id := tag2id("registry")
	if id != "" {
		dockerKill(id)
	}
	return id
}
