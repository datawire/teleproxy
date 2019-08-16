package dtest

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

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

func dockerPs(args ...string) ([]string, error) {
	cmdline := append([]string{"docker", "ps", "--quiet", "--filter=label=scope=" + scope}, args...)
	cmd := supervisor.Command(prefix, cmdline[0], cmdline[1:]...)
	output, err := cmd.Capture(nil)
	if err != nil {
		return nil, err
	}
	return lines(output), nil
}

var errorNoContainer = errors.New("no container")

func tag2id(tag string) (string, error) {
	ids, err := dockerPs("--filter=label=" + tag)
	if err != nil {
		return "", err
	}
	switch len(ids) {
	case 0:
		return "", errorNoContainer
	case 1:
		return ids[0], nil
	default:
		return "", errors.Errorf("expected 0 or 1 containers with label %q and label %q, got %d", "scope="+scope, tag, len(ids))
	}
}

func dockerUp(tag string, args ...string) (string, error) {
	var id string
	var err error

	WithNamedMachineLock("docker", func() {
		id, err = tag2id(tag)
		if err == errorNoContainer {
			cmdline := append([]string{"docker", "run", "--detach", "--label=scope=" + scope, "--label=" + tag, "--rm"}, args...)
			cmd := supervisor.Command(prefix, cmdline[0], cmdline[1:]...)
			var out string
			out, _err := cmd.Capture(nil)
			if _err != nil {
				err = _err
				return
			}
			id = strings.TrimSpace(out)[:12] // XXX: what if 'out' isn't 12 bytes long?
			err = nil
		}
	})

	return id, err
}

func dockerKill(ids ...string) {
	if len(ids) > 0 {
		cmdline := append([]string{"docker", "kill", "--"}, ids...)
		cmd := supervisor.Command(prefix, cmdline[0], cmdline[1:]...)
		_, _ = cmd.Capture(nil)
	}
}

var requiredResources = []string{
	"bindings",
	"componentstatuses",
	"configmaps",
	"endpoints",
	"events",
	"limitranges",
	"namespaces",
	"nodes",
	"persistentvolumeclaims",
	"persistentvolumes",
	"pods",
	"podtemplates",
	"replicationcontrollers",
	"resourcequotas",
	"secrets",
	"serviceaccounts",
	"services",
	"mutatingwebhookconfigurations.admissionregistration.k8s.io",
	"validatingwebhookconfigurations.admissionregistration.k8s.io",
	"customresourcedefinitions.apiextensions.k8s.io",
	"apiservices.apiregistration.k8s.io",
	"controllerrevisions.apps",
	"daemonsets.apps",
	"deployments.apps",
	"replicasets.apps",
	"statefulsets.apps",
	"tokenreviews.authentication.k8s.io",
	"localsubjectaccessreviews.authorization.k8s.io",
	"selfsubjectaccessreviews.authorization.k8s.io",
	"selfsubjectrulesreviews.authorization.k8s.io",
	"subjectaccessreviews.authorization.k8s.io",
	"horizontalpodautoscalers.autoscaling",
	"cronjobs.batch",
	"jobs.batch",
	"certificatesigningrequests.certificates.k8s.io",
	"leases.coordination.k8s.io",
	"daemonsets.extensions",
	"deployments.extensions",
	"ingresses.extensions",
	"networkpolicies.extensions",
	"podsecuritypolicies.extensions",
	"replicasets.extensions",
	"helmcharts.helm.cattle.io",
	"addons.k3s.cattle.io",
	"listenerconfigs.k3s.cattle.io",
	"ingresses.networking.k8s.io",
	"networkpolicies.networking.k8s.io",
	"runtimeclasses.node.k8s.io",
	"poddisruptionbudgets.policy",
	"podsecuritypolicies.policy",
	"clusterrolebindings.rbac.authorization.k8s.io",
	"clusterroles.rbac.authorization.k8s.io",
	"rolebindings.rbac.authorization.k8s.io",
	"roles.rbac.authorization.k8s.io",
	"priorityclasses.scheduling.k8s.io",
	"csidrivers.storage.k8s.io",
	"csinodes.storage.k8s.io",
	"storageclasses.storage.k8s.io",
	"volumeattachments.storage.k8s.io",
}

func isK3sReady() bool {
	kubeconfig, err := getKubeconfigPath()
	if err != nil {
		return false
	}

	cmd := supervisor.Command(prefix, "kubectl", "--kubeconfig="+kubeconfig, "api-resources", "--output=name")
	output, err := cmd.Capture(nil)
	if err != nil {
		return false
	}

	resources := make(map[string]struct{})
	for _, line := range strings.Split(output, "\n") {
		resources[strings.TrimSpace(line)] = struct{}{}
	}

	for _, req := range requiredResources {
		_, exists := resources[req]
		if !exists {
			return false
		}
	}

	get := supervisor.Command(prefix, "kubectl", "--kubeconfig", kubeconfig, "get", "namespace", "default")
	err = get.Start()
	if err != nil {
		panic(err)
	}
	_ = get.Wait()
	return get.ProcessState.ExitCode() == 0
}

const k3sConfigPath = "/etc/rancher/k3s/k3s.yaml"

// GetKubeconfig returns the kubeconfig contents for the running k3s
// cluster as a string. It will return the empty string if no cluster
// is running.
func GetKubeconfig() (string, error) {
	id, err := tag2id("k3s")
	if err != nil {
		return "", err
	}

	cmd := supervisor.Command(prefix, "docker", "exec", "--interactive", "--", id, "cat", "--", k3sConfigPath)
	kubeconfig, err := cmd.Capture(nil)
	if err != nil {
		return "", err
	}
	kubeconfig = strings.ReplaceAll(kubeconfig, "localhost:6443", net.JoinHostPort(dockerIP(), k3sPort))
	return kubeconfig, nil
}

func getKubeconfigPath() (string, error) {
	id, err := tag2id("k3s")
	if err != nil {
		return "", err
	}

	kubeconfig := filepath.Join(os.TempDir(), fmt.Sprintf("dtest-kubeconfig-%d-%s.yaml", os.Getuid(), id))
	contents, err := GetKubeconfig()
	if err != nil {
		return "", err
	}

	if err = ioutil.WriteFile(kubeconfig, []byte(contents), 0644); err != nil {
		return "", err
	}

	return kubeconfig, nil
}

const dtestRegistry = "DTEST_REGISTRY"
const registryPort = "5000"

// RegistryUp will launch if necessary and return the docker id of a
// container running a docker registry.
func RegistryUp() (string, error) {
	return dockerUp("registry",
		"--publish="+k3sPort+":6443",
		"--publish="+registryPort+":"+registryPort,
		"--env=REGISTRY_HTTP_ADDR="+net.JoinHostPort("0.0.0.0", registryPort),
		"--",
		"registry:2")
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

	return net.JoinHostPort(dockerIP(), registryPort)
}

const dtestKubeconfig = "DTEST_KUBECONFIG"
const k3sPort = "6443"
const k3sImage = "rancher/k3s:v0.6.1"

const k3sMsg = `
kubeconfig does not exist: %s

  Make sure DTEST_KUBECONFIG is either unset or points to a valid kubeconfig file.

`

// Kubeconfig returns a path referencing a kubeconfig file suitable for use in tests.
func Kubeconfig() (string, error) {
	kubeconfig := os.Getenv(dtestKubeconfig)
	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			return "", err
		}

		return kubeconfig, nil
	}

	if _, _, err := K3sUp(); err != nil {
		return "", err
	}

	for !isK3sReady() {
		time.Sleep(time.Second)
	}

	return getKubeconfigPath()
}

// K3sUp will launch if necessary and return the docker id of a
// container running a k3s cluster.
func K3sUp() (regid, k3sid string, err error) {
	regid, err = RegistryUp()
	if err != nil {
		return "", "", err
	}
	k3sid, err = dockerUp("k3s",
		"--privileged", "--network=container:"+regid,
		"--",
		k3sImage,
		"server", "--node-name=localhost", "--no-deploy=traefik")
	if err != nil {
		return "", "", err
	}
	return regid, k3sid, nil
}

// K3sDown shuts down the k3s cluster.
func K3sDown() string {
	id, _ := tag2id("k3s")
	if id != "" {
		dockerKill(id)
	}
	return id
}

// RegistryDown shutsdown the test registry.
func RegistryDown() string {
	id, _ := tag2id("registry")
	if id != "" {
		dockerKill(id)
	}
	return id
}
