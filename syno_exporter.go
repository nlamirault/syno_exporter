// Copyright (C) 2016 Nicolas Lamirault <nicolas.lamirault@gmail.com>

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	prom_version "github.com/prometheus/common/version"

	"github.com/nlamirault/syno_exporter/syno"
	"github.com/nlamirault/syno_exporter/version"
)

const (
	namespace = "syno"
)

var (
	systemStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "system_status"),
		"Diskstation system status.",
		nil, nil,
	)
	systemTemperature = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "system_temperature"),
		"DiskStation temperature.",
		nil, nil,
	)
	systemPowerStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "system_power_status"),
		"Returns error if power supplies fail.",
		nil, nil,
	)
	systemFanStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "system_fan_status"),
		"Returns error if system fan fails.",
		nil, nil,
	)
	systemCPUFanStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "system_cpu_status"),
		"Returns error if CPU fan fails.",
		nil, nil,
	)
	systemUpgradeAvailable = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "system_upgrade_available"),
		"Checks whether a new version or update of DSM is available",
		nil, nil,
	)

	memTotalSwap = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_total_swap"),
		"The total amount of swap space configured for this host.",
		nil, nil,
	)
	memAvailSwap = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_avail_swap"),
		"The amount of swap space currently unused or available.",
		nil, nil,
	)
	memTotalReal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_total_real"),
		"The total amount of real/physical memory installed on this host.",
		nil, nil,
	)
	memAvailReal = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_avail_real"),
		"The amount of real/physical memory currently unused or available.",
		nil, nil,
	)
	memTotalFree = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_total_free"),
		"The total amount of memory free or available for use on this host.",
		nil, nil,
	)
	memShared = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_shared"),
		"The total amount of real or virtual memory currently allocated for use as shared memory.",
		nil, nil,
	)
	memBuffer = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_buffer"),
		"The total amount of real or virtual memory currently allocated for use as memory buffers.",
		nil, nil,
	)
	memCached = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "mem_cached"),
		"The total amount of real or virtual memory currently allocated for use as cached memory.",
		nil, nil,
	)

	loadShort = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "load_short"),
		"1 minute Load",
		nil, nil,
	)
	loadMid = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "load_mid"),
		"5 minute Load",
		nil, nil,
	)
	loadLong = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "load_long"),
		"15 minute Load",
		nil, nil,
	)

	cpuUser = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cpu_user"),
		"The number of 'ticks' spent processing user-level code.",
		nil, nil,
	)
	cpuNice = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cpu_nice"),
		"The number of 'ticks' spent processing reduced-priority code.",
		nil, nil,
	)
	cpuSystem = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cpu_system"),
		"The number of 'ticks' spent processing system-level code.",
		nil, nil,
	)
	cpuIdle = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cpu_idle"),
		"The number of 'ticks' spent processing idle.",
		nil, nil,
	)
	cpuWait = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cpu_wait"),
		"The number of 'ticks' spent waiting for IO",
		nil, nil,
	)
	cpuKernel = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cpu_kernel"),
		"The number of 'ticks' spent processing kernel-level code.",
		nil, nil,
	)
	cpuInterrupt = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cpu_interrupt"),
		"The number of 'ticks' spent processing hardware interrupts.",
		nil, nil,
	)

	netIn = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "net_in"),
		"The total number of octets received on the interface",
		nil, nil,
	)
	netOut = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "net_out"),
		"The total number of octets transmitted out of the interface",
		nil, nil,
	)
)

// Exporter collects Syno stats from the given server and exports them using
// the prometheus metrics package.
type Exporter struct {
	Client *syno.Client
}

// NewExporter returns an initialized Exporter.
func NewExporter(dsIP string, interval time.Duration) (*Exporter, error) {
	log.Infof("Setup Syno client using diskstation: %s and interval %s\n", dsIP, interval)
	client, err := syno.NewClient(dsIP, interval)
	if err != nil {
		return nil, fmt.Errorf("Can't create the Syno client: %s", err)
	}

	log.Debugln("Init exporter")
	return &Exporter{
		Client: client,
	}, nil
}

// Describe describes all the metrics ever exported by the Syno exporter.
// It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- systemStatus
	ch <- systemTemperature
	ch <- systemPowerStatus
	ch <- systemFanStatus
	ch <- systemCPUFanStatus
	ch <- systemUpgradeAvailable

	ch <- memTotalSwap
	ch <- memAvailSwap
	ch <- memTotalReal
	ch <- memAvailReal
	ch <- memTotalFree
	ch <- memShared
	ch <- memBuffer
	ch <- memCached

	ch <- loadShort
	ch <- loadMid
	ch <- loadLong

	ch <- cpuUser
	ch <- cpuNice
	ch <- cpuSystem
	ch <- cpuIdle
	ch <- cpuWait
	ch <- cpuKernel
	ch <- cpuInterrupt

	ch <- netIn
	ch <- netOut
}

// Collect fetches the stats from configured Syno location and delivers them
// as Prometheus metrics.
// It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	log.Infof("Syno exporter starting")
	if e.Client == nil {
		log.Errorf("Syno client not configured.")
		return
	}
	err := e.Client.Connect()
	if err != nil {
		log.Errorln("Can't connect to Synology for SNMP: %s", err)
		return
	}
	e.collectSystemMetrics(ch)
	e.collectCPUMetrics(ch)
	e.collectLoadMetrics(ch)
	e.collectMemoryMetrics(ch)
	e.collectNetworkMetrics(ch)
	e.collectDiskMetrics(ch)

	log.Infof("Syno exporter finished")
}

func (e *Exporter) collectSystemMetrics(ch chan<- prometheus.Metric) {
	resp, err := e.Client.SystemMetrics()
	if err != nil {
		log.Errorf("[syno] Can't retrieve system metrics: %v", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		systemStatus, prometheus.GaugeValue, resp["system-status"],
	)
	ch <- prometheus.MustNewConstMetric(
		systemTemperature, prometheus.GaugeValue, resp["system-temperature"],
	)
	ch <- prometheus.MustNewConstMetric(
		systemPowerStatus, prometheus.GaugeValue, resp["system-powerStatus"],
	)
	ch <- prometheus.MustNewConstMetric(
		systemFanStatus, prometheus.GaugeValue, resp["system-systemFanStatus"],
	)
	ch <- prometheus.MustNewConstMetric(
		systemCPUFanStatus, prometheus.GaugeValue, resp["system-cpuFanStatus"],
	)
	ch <- prometheus.MustNewConstMetric(
		systemUpgradeAvailable, prometheus.GaugeValue, resp["system-upgradeAvailable"],
	)
}

func (e *Exporter) collectDiskMetrics(ch chan<- prometheus.Metric) {
	// resp, err := e.Client.DiskMetrics()
	// if err != nil {
	// 	log.Errorf("[syno] Can't retrieve Disk metrics: %v", err)
	// 	return
	// }
}

func (e *Exporter) collectLoadMetrics(ch chan<- prometheus.Metric) {
	resp, err := e.Client.LoadMetrics()
	if err != nil {
		log.Errorf("[syno] Can't retrieve Load metrics: %v", err)
		return
	}
	log.Debugf("SNMP Load response: %s", resp)
	ch <- prometheus.MustNewConstMetric(
		loadShort, prometheus.GaugeValue, resp["load.shortterm"],
	)
	ch <- prometheus.MustNewConstMetric(
		loadMid, prometheus.GaugeValue, resp["load.midterm"],
	)
	ch <- prometheus.MustNewConstMetric(
		loadLong, prometheus.GaugeValue, resp["load.longterm"],
	)
}

func (e *Exporter) collectCPUMetrics(ch chan<- prometheus.Metric) {
	resp, err := e.Client.CPUMetrics()
	if err != nil {
		log.Errorf("[syno] Can't retrieve CPU metrics: %v", err)
		return
	}
	log.Debugf("SNMP CPU response: %s", resp)
	ch <- prometheus.MustNewConstMetric(
		cpuUser, prometheus.GaugeValue, resp["cpu-0.cpu-user"],
	)
	ch <- prometheus.MustNewConstMetric(
		cpuNice, prometheus.GaugeValue, resp["cpu-0.cpu-nice"],
	)
	ch <- prometheus.MustNewConstMetric(
		cpuSystem, prometheus.GaugeValue, resp["cpu-0.cpu-system"],
	)
	ch <- prometheus.MustNewConstMetric(
		cpuIdle, prometheus.GaugeValue, resp["cpu-0.cpu-idle"],
	)
	ch <- prometheus.MustNewConstMetric(
		cpuWait, prometheus.GaugeValue, resp["cpu-0.cpu-wait"],
	)
	ch <- prometheus.MustNewConstMetric(
		cpuKernel, prometheus.GaugeValue, resp["cpu-0.cpu-kernel"],
	)
	ch <- prometheus.MustNewConstMetric(
		cpuInterrupt, prometheus.GaugeValue, resp["cpu-0.cpu-interrupt"],
	)
}

func (e *Exporter) collectMemoryMetrics(ch chan<- prometheus.Metric) {
	resp, err := e.Client.MemoryMetrics()
	if err != nil {
		log.Errorf("[syno] Can't retrieve Memory metrics: %v", err)
		return
	}
	log.Debugf("SNMP Memory response: %s", resp)
	ch <- prometheus.MustNewConstMetric(
		memTotalSwap, prometheus.GaugeValue, resp["mem-total-swap"],
	)
	ch <- prometheus.MustNewConstMetric(
		memAvailSwap, prometheus.GaugeValue, resp["mem-avail-swap"],
	)
	ch <- prometheus.MustNewConstMetric(
		memTotalReal, prometheus.GaugeValue, resp["mem-total-real"],
	)
	ch <- prometheus.MustNewConstMetric(
		memAvailReal, prometheus.GaugeValue, resp["mem-avail-real"],
	)
	ch <- prometheus.MustNewConstMetric(
		memTotalFree, prometheus.GaugeValue, resp["mem-total-free"],
	)
	ch <- prometheus.MustNewConstMetric(
		memShared, prometheus.GaugeValue, resp["mem-shared"],
	)
	ch <- prometheus.MustNewConstMetric(
		memBuffer, prometheus.GaugeValue, resp["mem-buffer"],
	)
	ch <- prometheus.MustNewConstMetric(
		memCached, prometheus.GaugeValue, resp["mem-cached"],
	)
}

func (e *Exporter) collectNetworkMetrics(ch chan<- prometheus.Metric) {
	resp, err := e.Client.NetworkMetrics()
	if err != nil {
		log.Errorf("[syno] Can't retrieve Network metrics: %v", err)
		return
	}
	log.Debugf("SNMP Network response: %s", resp)
	ch <- prometheus.MustNewConstMetric(
		netIn, prometheus.GaugeValue, resp["net-in"],
	)
	ch <- prometheus.MustNewConstMetric(
		netOut, prometheus.GaugeValue, resp["net-out"],
	)
}

func init() {
	prometheus.MustRegister(prom_version.NewCollector("syno_exporter"))
}

func main() {
	var (
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9111", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		diskstation   = flag.String("diskstation", "", "Disktation IP.")
		//interval      = flag.Int("interval", 60*time.Second, "Interval for metrics.")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Synology Prometheus exporter. v%s\n", version.Version)
		os.Exit(0)
	}

	log.Infoln("Starting syno_exporter", prom_version.Info())
	log.Infoln("Build context", prom_version.BuildContext())

	interval := 60 * time.Second
	exporter, err := NewExporter(*diskstation, interval)
	if err != nil {
		log.Errorf("Can't create exporter : %s", err)
		os.Exit(1)
	}
	log.Infoln("Register exporter")
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Syno Exporter</title></head>
             <body>
             <h1>Syno Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
