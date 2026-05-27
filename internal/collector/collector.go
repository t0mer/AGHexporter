package collector

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/t0mer/AGHexporter/internal/adguard"
	"github.com/t0mer/AGHexporter/internal/instances"
)

const scrapeTimeout = 10 * time.Second

// Collector implements prometheus.Collector for one or more AdGuard Home instances.
type Collector struct {
	instances []instances.Instance
	clients   []*adguard.Client
	errors    []atomic.Uint64
}

// New creates a Collector for the given instances.
func New(insts []instances.Instance) *Collector {
	c := &Collector{
		instances: insts,
		clients:   make([]*adguard.Client, len(insts)),
		errors:    make([]atomic.Uint64, len(insts)),
	}
	for i, inst := range insts {
		c.clients[i] = adguard.NewClient(inst)
	}
	return c
}

// Describe sends all descriptor definitions to ch.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range allDescs() {
		ch <- d
	}
}

// Collect fans out one goroutine per instance and streams metrics to ch.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup
	for i := range c.instances {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c.scrapeInstance(idx, ch)
		}(i)
	}
	wg.Wait()
}

func (c *Collector) scrapeInstance(idx int, ch chan<- prometheus.Metric) {
	inst := c.instances[idx]
	name := inst.Name
	start := time.Now()

	status, err := c.clients[idx].GetStatus()
	if err != nil {
		slog.Warn("scrape failed: GetStatus", "instance", name, "error", err)
		c.errors[idx].Add(1)
		ch <- gauge(descUp, 0, name)
		ch <- counter(descScrapeErrors, float64(c.errors[idx].Load()), name)
		ch <- gauge(descScrapeDuration, time.Since(start).Seconds(), name)
		return
	}

	stats, err := c.clients[idx].GetStats()
	if err != nil {
		slog.Warn("scrape failed: GetStats", "instance", name, "error", err)
		c.errors[idx].Add(1)
		ch <- gauge(descUp, 0, name)
		ch <- counter(descScrapeErrors, float64(c.errors[idx].Load()), name)
		ch <- gauge(descScrapeDuration, time.Since(start).Seconds(), name)
		return
	}

	ch <- gauge(descUp, 1, name)
	ch <- boolGauge(descProtectionEnabled, status.ProtectionEnabled, name)
	ch <- gauge(descDNSQueries, float64(stats.NumDNSQueries), name)
	ch <- gauge(descBlockedFiltering, float64(stats.NumBlockedFiltering), name)
	ch <- gauge(descBlockedSafebrowsing, float64(stats.NumReplacedSafebrowsing), name)
	ch <- gauge(descBlockedParental, float64(stats.NumReplacedParental), name)
	ch <- gauge(descEnforcedSafesearch, float64(stats.NumReplacedSafesearch), name)
	ch <- gauge(descAvgProcessingTime, stats.AvgProcessingTime, name)

	emitTopN(ch, descTopClients, name, stats.TopClients)
	emitTopN(ch, descTopQueriedDomains, name, stats.TopQueriedDomains)
	emitTopN(ch, descTopBlockedDomains, name, stats.TopBlockedDomains)
	emitTopN(ch, descTopUpstreams, name, stats.TopUpstreamsResponses)
	emitTopN(ch, descTopUpstreamsAvgTime, name, stats.TopUpstreamsAvgTime)

	ch <- counter(descScrapeErrors, float64(c.errors[idx].Load()), name)
	ch <- gauge(descScrapeDuration, time.Since(start).Seconds(), name)
}

func emitTopN(ch chan<- prometheus.Metric, desc *prometheus.Desc, instance string, entries []map[string]float64) {
	for _, entry := range entries {
		for k, v := range entry {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, instance, k)
		}
	}
}

func gauge(desc *prometheus.Desc, val float64, labelVals ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, val, labelVals...)
}

func counter(desc *prometheus.Desc, val float64, labelVals ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.CounterValue, val, labelVals...)
}

func boolGauge(desc *prometheus.Desc, b bool, labelVals ...string) prometheus.Metric {
	v := 0.0
	if b {
		v = 1.0
	}
	return gauge(desc, v, labelVals...)
}

func allDescs() []*prometheus.Desc {
	return []*prometheus.Desc{
		descUp, descProtectionEnabled, descDNSQueries, descBlockedFiltering,
		descBlockedSafebrowsing, descBlockedParental, descEnforcedSafesearch,
		descAvgProcessingTime, descScrapeDuration, descScrapeErrors,
		descTopClients, descTopQueriedDomains, descTopBlockedDomains,
		descTopUpstreams, descTopUpstreamsAvgTime,
	}
}
