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
	"github.com/prometheus/common/log"
	"github.com/soniah/gosnmp"
)

// Plugin defines a SNMP receiver
type Plugin interface {
	Fetch(snmp *gosnmp.GoSNMP) (map[string]float64, error)
}

func printSNMPResult(result *gosnmp.SnmpPacket) {
	for i, variable := range result.Variables {
		log.Debugf("[Plugin] %d: oid: %s ", i, variable.Name)
		switch variable.Type {
		case gosnmp.OctetString:
			log.Debugf("[Plugin] string: %s", string(variable.Value.([]byte)))
		default:
			log.Debugf("[Plugin] number: %d", gosnmp.ToBigInt(variable.Value))
		}
	}
}
