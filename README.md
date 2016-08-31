# syno_exporter

[![License Apache 2][badge-license]](LICENSE)
[![GitHub version](https://badge.fury.io/gh/nlamirault%2Fsyno_exporter.svg)](https://badge.fury.io/gh/nlamirault%2Fsyno_exporter)

* Master : [![Circle CI](https://circleci.com/gh/nlamirault/syno_exporter/tree/master.svg?style=svg)](https://circleci.com/gh/nlamirault/syno_exporter/tree/master)
* Develop : [![Circle CI](https://circleci.com/gh/nlamirault/syno_exporter/tree/develop.svg?style=svg)](https://circleci.com/gh/nlamirault/syno_exporter/tree/develop)

This Prometheus exporter check the health of your Synology NAS:
* [ ] System status (Power, Fans)
* [ ] Disks status
* [ ] RAID status
* [ ] DSM update status
* [ ] Temperature Warning and Critical
* [ ] UPS information
* [ ] Storage percentage of use

Tested with DSM 6.0
Based on [Synology Diskstation MIB Guide](http://ukdl.synology.com/download/Document/MIBGuide/Synology_DiskStation_MIB_Guide.pdf )


## Installation

You can download the binaries :

* Architecture i386 [ [linux](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_linux_386) / [darwin](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_darwin_386) / [freebsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_freebsd_386) / [netbsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_netbsd_386) / [openbsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_openbsd_386) / [windows](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_windows_386.exe) ]
* Architecture amd64 [ [linux](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_linux_amd64) / [darwin](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_darwin_amd64) / [freebsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_freebsd_amd64) / [netbsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_netbsd_amd64) / [openbsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_openbsd_amd64) / [windows](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_windows_amd64.exe) ]
* Architecture arm [ [linux](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_linux_arm) / [freebsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_freebsd_arm) / [netbsd](https://bintray.com/artifact/download/nlamirault/oss/syno_exporter-0.2.0_netbsd_arm) ]


## Usage

Launch the Prometheus exporter :

    $ syno_exporter -log.level=debug -diskstation 192.168.1.11

Check SNMP informations from your Diskstation :

    # System load
    $ snmpget -v 1 -c "community" 192.168.1.11 .1.3.6.1.4.1.2021.10.1.3.1

    # Get available disk space for /
    $ snmpget -v 1 -c "community" 192.168.1.11 .1.3.6.1.4.1.2021.9.1.7.1


## Development

* Initialize environment

        $ make init

* Build tool :

        $ make build

* Launch unit tests :

        $ make test


## Local Deployment

* Launch Prometheus using the configuration file in this repository:

        $ prometheus -config.file=prometheus.yml

* Launch exporter:

        $ syno_exporter -log.level=debug -diskstation 192.168.1.11

* Check that Prometheus find the exporter on `http://localhost:9090/targets`


## Contributing

See [CONTRIBUTING](CONTRIBUTING.md).


## License

See [LICENSE](LICENSE) for the complete license.


## Changelog

A [changelog](ChangeLog.md) is available


## Contact

Nicolas Lamirault <nicolas.lamirault@gmail.com>

[badge-license]: https://img.shields.io/badge/license-Apache2-green.svg?style=flat
