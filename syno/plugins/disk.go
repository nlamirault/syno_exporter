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

package plugins

import (
	"fmt"

	"github.com/prometheus/common/log"
	"github.com/soniah/gosnmp"
)

type DiskPlugin struct{}

func (p DiskPlugin) Fetch(snmp *gosnmp.GoSNMP) (map[string]float64, error) {
	metrics := map[string]float64{}
	temperatures, err := getTemperatures(snmp)
	if err != nil {
		return nil, fmt.Errorf("[Disk Plugin] SNMP Temperature error: %v", err)
	}
	for key, value := range temperatures {
		metrics[fmt.Sprintf("disk.disk-%v.temperature", key)] = value
	}
	return metrics, nil
}

func getTemperatures(snmp *gosnmp.GoSNMP) (map[int]float64, error) {
	log.Infof("[Disk Plugin] Get SNMP disk temperatures")
	result, err := snmp.Get([]string{
		".1.3.6.1.4.1.6574.2.1.1.6.0",
		".1.3.6.1.4.1.6574.2.1.1.6.1"})
	if err != nil {
		return nil, fmt.Errorf("[Disk Plugin] SNMP Error: %v", err)
	}
	temps := map[int]float64{}
	for i, variable := range result.Variables {
		temps[i] = float64(variable.Value.(int))
	}
	return temps, nil
}
