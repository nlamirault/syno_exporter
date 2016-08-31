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

var (
	oidSystem = ".1.3.6.1.4.1.6574.1"
)

type SystemPlugin struct{}

func (p SystemPlugin) Fetch(snmp *gosnmp.GoSNMP) (map[string]float64, error) {
	oids := []string{
		fmt.Sprintf("%s.1", oidSystem),   // systemStatus
		fmt.Sprintf("%s.2", oidSystem),   // temperature
		fmt.Sprintf("%s.3", oidSystem),   // powerStatus
		fmt.Sprintf("%s.4.1", oidSystem), // systemFanStatus
		fmt.Sprintf("%s.4.2", oidSystem), // cpuFanStatus
		// fmt.Sprintf("%s.5.1", oidSystem), // modelName
		// fmt.Sprintf("%s.5.1", oidSystem), // serialNumber
		// fmt.Sprintf("%s.5.3", oidSystem), // version
		fmt.Sprintf("%s.5.4", oidSystem), // upgradeAvailable
	}
	log.Infof("[CPU Plugin] Get SNMP data")
	result, err := snmp.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("[CPU Plugin] SNMP Error: %v", err)
	}
	return map[string]float64{
		"system-status":          float64(result.Variables[0].Value.(uint)),
		"system-temperature":     float64(result.Variables[1].Value.(uint)),
		"system-powerStatus":     float64(result.Variables[2].Value.(uint)),
		"system-systemFanStatus": float64(result.Variables[3].Value.(uint)),
		"system-cpuFanStatus":    float64(result.Variables[4].Value.(uint)),
		// "system-modelName":        float64(result.Variables[5].Value.(string)),
		// "system-serialNumber":     float64(result.Variables[6].Value.(string)),
		// "system-version":          float64(result.Variables[7].Value.(string)),
		"system-upgradeAvailable": float64(result.Variables[8].Value.(uint)),
	}, nil
}
