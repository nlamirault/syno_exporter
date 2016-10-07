#!/bin/bash

# Copyright (C) 2016 Nicolas Lamirault <nicolas.lamirault@gmail.com>

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# set -e
# set -x

NO_COLOR="\033[0m"
OK_COLOR="\033[32;01m"
ERROR_COLOR="\033[31;01m"
WARN_COLOR="\033[33;01m"

if [ $# -ne 2 ]; then
    echo -e "${ERROR_COLOR}$0 <DiskSation IP> <community>${NO_COLOR}"
    exit 1
fi

DS_STATION=$1
COMMUNITY=$2

echo -e "${OK_COLOR}=== Try SNMP CPU ===${NO_COLOR}"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.11.50.0"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.11.51.0"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.11.52.0"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.11.53.0"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.11.54.0"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.11.55.0"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.11.56.0"

echo -e "${OK_COLOR}=== Try SNMP Disk ===${NO_COLOR}"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.6574.2.1.1.6.0"
# snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.6574.2.1.1.6.1"

echo -e "${OK_COLOR}=== Try SNMP Load ===${NO_COLOR}"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.10.1.5.1"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.10.1.5.2"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.10.1.5.3"

echo -e "${OK_COLOR}=== Try SNMP Memory ===${NO_COLOR}"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.3.0" # memTotalSwap
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.4.0" # memAvailSwap
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.5.0" # memTotalReala
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.6.0" # memAvailReal
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.11.0" # memTotalFree
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.13.0" # memShared
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.14.0" # memBuffer
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.2021.4.15.0" # memCached

echo -e "${OK_COLOR}=== Try SNMP Network ===${NO_COLOR}"
echo "--> NOT WORK"
# snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.2.1.31.1.1.1.6"
# snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.2.1.31.1.1.1.10"

echo -e "${OK_COLOR}=== Try SNMP System ===${NO_COLOR}"
snmpget -v 1 -c ${COMMUNITY} ${DS_STATION} ".1.3.6.1.4.1.6574.1.1"
