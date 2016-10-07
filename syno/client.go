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

package syno

import (
	// "fmt"
	// "log"
	// "net"
	"time"

	"github.com/prometheus/common/log"
	"github.com/soniah/gosnmp"

	"github.com/nlamirault/syno_exporter/syno/plugins"
)

// Client defines the Synology SNMP client
type Client struct {
	Diskstation string
	Interval    time.Duration
	Plugins     map[string]plugins.Plugin
	SNMP        *gosnmp.GoSNMP
}

// NewClient defines a new client for the Synology Diskstation
func NewClient(dsIP string, interval time.Duration) (*Client, error) {
	log.Debugf("New SNMP Client for Synology Disksation: %s", dsIP)
	return &Client{
		Diskstation: dsIP,
		Interval:    interval,
		Plugins: map[string]plugins.Plugin{
			"disk":   plugins.DiskPlugin{},
			"load":   plugins.LoadPlugin{},
			"cpu":    plugins.CPUPlugin{},
			"mem":    plugins.MemoryPlugin{},
			"net":    plugins.NetworkPlugin{},
			"system": plugins.SystemPlugin{},
		},
		SNMP: &gosnmp.GoSNMP{
			Target:    dsIP,
			Port:      161,
			Community: "public",
			Version:   gosnmp.Version1,
			Timeout:   time.Duration(2) * time.Second,
		},
	}, nil
}

func (c *Client) Connect() error {
	return c.SNMP.Connect()
}

func (c *Client) SystemMetrics() (map[string]float64, error) {
	log.Infof("[Client] Collect System metrics")
	return c.collect(c.Plugins["system"])
}

func (c *Client) DiskMetrics() (map[string]float64, error) {
	log.Infof("[Client] Collect Disk metrics")
	return c.collect(c.Plugins["disk"])
}

func (c *Client) LoadMetrics() (map[string]float64, error) {
	log.Infof("[Client] Collect Load metrics")
	return c.collect(c.Plugins["load"])
}

func (c *Client) CPUMetrics() (map[string]float64, error) {
	log.Infof("[Client] Collect Cpu metrics")
	return c.collect(c.Plugins["cpu"])
}

func (c *Client) MemoryMetrics() (map[string]float64, error) {
	log.Infof("[Client] Collect Memory metrics")
	return c.collect(c.Plugins["mem"])
}

func (c *Client) NetworkMetrics() (map[string]float64, error) {
	log.Infof("[Client] Collect Network metrics")
	return c.collect(c.Plugins["net"])
}

func (c *Client) collect(plugin plugins.Plugin) (map[string]float64, error) {
	metrics, err := plugin.Fetch(c.SNMP)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

// // Collect will retrieve SNMP informations from the Diskstation
// func (c *Client) Collect() {
// 	for now := range time.Tick(c.Interval) {
// 		c.SNMP.Connect()
// 		defer c.SNMP.Conn.Close()
// 		for _, plugin := range c.Plugins {
// 			data, err := plugin.Fetch(c.SNMP)
// 			if err != nil {
// 				log.Errorf("[Client] Error retrieving SNMP values: %v", err)
// 			} else {
// 				for key, value := range data {
// 					metric := fmt.Sprintf("%s %v %d\n\r", key, value, now.Unix())
// 				}
// 			}
// 		}

// 	}
// }
