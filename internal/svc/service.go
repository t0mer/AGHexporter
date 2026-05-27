package svc

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kardianos/service"
)

const (
	svcName        = "adguardhome-exporter"
	svcDisplayName = "AdGuard Home Prometheus Exporter"
	svcDescription = "Scrapes one or more AdGuard Home instances and exposes Prometheus metrics."
	defaultPort    = 9100
)

// program satisfies the kardianos/service interface.
type program struct{}

func (p *program) Start(s service.Service) error { return nil }
func (p *program) Stop(s service.Service) error  { return nil }

// Dispatch performs the requested service management action.
// It captures all ADGUARD_* environment variables and the given flags into the
// service definition so the OS service manager can provide them at runtime.
func Dispatch(action string, port int, instanceFlags []string) error {
	cfg := buildConfig(port, instanceFlags)

	svc, err := service.New(&program{}, cfg)
	if err != nil {
		return fmt.Errorf("creating service handle: %w", err)
	}

	switch action {
	case "install":
		return svc.Install()
	case "uninstall":
		return svc.Uninstall()
	case "start":
		return svc.Start()
	case "stop":
		return svc.Stop()
	case "restart":
		return svc.Restart()
	default:
		return fmt.Errorf("unknown service action %q; valid: install, uninstall, start, stop, restart", action)
	}
}

// buildConfig constructs the service.Config, embedding current ADGUARD_* env vars
// and any --port / --instance arguments so the service has them at runtime.
func buildConfig(port int, instanceFlags []string) *service.Config {
	envVars := make(map[string]string)
	for _, e := range os.Environ() {
		k, v, found := strings.Cut(e, "=")
		if found && strings.HasPrefix(k, "ADGUARD_") {
			envVars[k] = v
		}
	}

	var args []string
	if port != defaultPort {
		args = append(args, "--port", strconv.Itoa(port))
	}
	for _, f := range instanceFlags {
		args = append(args, "--instance", f)
	}

	return &service.Config{
		Name:        svcName,
		DisplayName: svcDisplayName,
		Description: svcDescription,
		EnvVars:     envVars,
		Arguments:   args,
	}
}
