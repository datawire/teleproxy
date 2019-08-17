package dns

import (
	"runtime"
	"strings"

	"github.com/datawire/teleproxy/pkg/supervisor"
	"github.com/datawire/teleproxy/pkg/logexec"
)

type searchDomains struct {
	Interface string
	Domains   string
}

func OverrideSearchDomains(p *supervisor.Process, domains string) (func(), error) {
	if runtime.GOOS != "darwin" {
		return func() {}, nil
	}

	ifaces, err := getIfaces(p)
	if err != nil {
		return nil, err
	}
	previous := []searchDomains{}

	for _, iface := range ifaces {
		// setup dns search path
		domain, err := getSearchDomains(p, iface)
		if err != nil {
			log("DNS: error getting search domain for interface %v: %v", iface, err)
		} else {
			setSearchDomains(p, iface, domains)
			previous = append(previous, searchDomains{iface, domain})
		}
	}

	// return function to restore dns search paths
	return func() {
		for _, prev := range previous {
			setSearchDomains(p, prev.Interface, prev.Domains)
		}
	}, nil
}

func getIfaces(p *supervisor.Process) (ifaces []string, err error) {
	outputBytes, err := logexec.CommandContext(p.Context(), "networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(outputBytes), "\n") {
		if strings.Contains(line, "*") {
			continue
		}
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			ifaces = append(ifaces, line)
		}
	}
	return
}

func getSearchDomains(p *supervisor.Process, iface string) (domains string, err error) {
	var domainsBytes []byte
	domainsBytes, err = logexec.CommandContext(p.Context(), "networksetup", "-getsearchdomains", iface).Output()
	domains = strings.TrimSpace(string(domainsBytes))
	return
}

func setSearchDomains(p *supervisor.Process, iface, domains string) (err error) {
	err = logexec.CommandContext(p.Context(), "networksetup", "-setsearchdomains", iface, domains).Run()
	return
}
