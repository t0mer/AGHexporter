package main

import (
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/t0mer/AGHexporter/internal/collector"
	"github.com/t0mer/AGHexporter/internal/instances"
	"github.com/t0mer/AGHexporter/internal/server"
	"github.com/t0mer/AGHexporter/internal/svc"
)

func main() {
	var (
		serviceAction string
		port          int
		instanceFlags []string
	)

	pflag.StringVar(&serviceAction, "service", "", "Service action: install, uninstall, start, stop, restart")
	pflag.IntVar(&port, "port", 9100, "Port to expose /metrics on (overridden by ADGUARD_EXPORTER_PORT)")
	pflag.StringArrayVar(&instanceFlags, "instance", nil,
		"Instance spec (repeatable): url=http://host,username=u,password=p[,name=n,skip_tls=true]")
	pflag.Parse()

	port = instances.ResolvePort(port)

	if serviceAction != "" {
		if err := svc.Dispatch(serviceAction, port, instanceFlags); err != nil {
			slog.Error("service action failed", "action", serviceAction, "error", err)
			os.Exit(1)
		}
		return
	}

	insts, err := instances.DiscoverInstances(instanceFlags)
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	slog.Info("starting AdGuard Home exporter", "port", port, "instances", len(insts))
	for _, inst := range insts {
		slog.Info("configured instance", "name", inst.Name, "url", inst.URL)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.New(insts))

	if err := server.Run(port, reg); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
