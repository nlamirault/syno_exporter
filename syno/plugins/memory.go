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

type MemoryPlugin struct{}

func (p MemoryPlugin) Fetch(snmp *gosnmp.GoSNMP) (map[string]float64, error) {
	oids := []string{
		".1.3.6.1.4.1.2021.4.3.0",  // memTotalSwap
		".1.3.6.1.4.1.2021.4.4.0",  // memAvailSwap
		".1.3.6.1.4.1.2021.4.5.0",  // memTotalReal
		".1.3.6.1.4.1.2021.4.6.0",  // memAvailReal
		".1.3.6.1.4.1.2021.4.11.0", // memTotalFree
		".1.3.6.1.4.1.2021.4.13.0", // memShared
		".1.3.6.1.4.1.2021.4.14.0", // memBuffer
		".1.3.6.1.4.1.2021.4.15.0", // memCached
	}
	log.Infof("[Memory Plugin] Get SNMP data")
	result, err := snmp.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("[Memory Plugin] SNMP Error: %v", err)
	}
	log.Debugf("SNMP System result: %s", result)
	printSNMPResult(result)

	return map[string]float64{
		"mem-total-swap": float64(gosnmp.ToBigInt(result.Variables[0].Value).Int64()),
		"mem-avail-swap": float64(gosnmp.ToBigInt(result.Variables[1].Value).Int64()),
		"mem-total-real": float64(gosnmp.ToBigInt(result.Variables[2].Value).Int64()),
		"mem-avail-real": float64(gosnmp.ToBigInt(result.Variables[3].Value).Int64()),
		"mem-total-free": float64(gosnmp.ToBigInt(result.Variables[4].Value).Int64()),
		"mem-shared":     float64(gosnmp.ToBigInt(result.Variables[5].Value).Int64()),
		"mem-buffer":     float64(gosnmp.ToBigInt(result.Variables[6].Value).Int64()),
		"mem-cached":     float64(gosnmp.ToBigInt(result.Variables[7].Value).Int64()),
	}, nil
}
