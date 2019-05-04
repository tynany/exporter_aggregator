# Exporter Aggregator
Exporter Aggregator will scape metrics from a list of Prometheus exporter endpoints and aggregate the values of any metrics with the same name and label/s.

## Why?
This project was driven by having to run multiple instances of HAProxy Exporter when using HAProxy in multiple process mode (`nbproc > 1`). It is very easy to have thousands of HAProxy metrics, and when running a large number of HAProxy processes, the number of metrics per scrape escalates quickly. For example, if each HAProxy process exposes 4,000 metrics, and you have 64 HAProxy processes (`nbproc 64`), you end up collecting 256,000 metrics per scrape. This exporter  reduces that number down to 4,000 without any drawbacks. I'm sure there might be other use cases too.

## Getting Started
Download your flavour of prebuilt binaries from the [releases](https://github.com/tynany/exporter_aggregator/releases) tab.

To run exporter_aggregator:
```
./exporter_aggregator [flags]
```

To view metrics on the default port (9299) and path (/metrics):
```
http://device:9299/metrics
```

To view available flags:
```
./exporter_aggregator --help
usage: exporter_aggregator --config.path=CONFIG.PATH [<flags>]

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
  -c, --config.path=CONFIG.PATH  Path of the YAML configuration file.
      --web.listen-address=":9299"
                                 Address on which to expose metrics and web interface.
      --web.telemetry-path="/metrics"
                                 Path under which to expose metrics.
      --version                  Show application version.

```

YAML formatted configuration file:
```
timeout: 3s # The timeout in duration format.
endpoints: # The list of endpoints to scrape.
  - http://127.0.0.1:9101/metrics
  - http://127.0.0.1:9102/metrics
  - http://127.0.0.1:9103/metrics
  - http://127.0.0.1:9104/metrics
  - http://127.0.0.1:9105/metrics
  - http://127.0.0.1:9106/metrics
  - http://127.0.0.1:9107/metrics
```

Promethues Collector configuraiton:
```
scrape_configs:
  - job_name: haproxy aggregate
    static_configs:
      - targets:
        - device1:9299
        - device2:9299
    relabel_configs:
      - source_labels: [__address__]
        regex: "(.*):\d+"
        target: instance
```

## exporter_aggregator_successful_endpoints
The `exporter_consolidator_successful_endpoints` metric exposes the number of endpoints were successfully scraped. This is useful to ensure all endpoints are up and their metrics are being successfully collected. For example, if you are scaping 64 HAProxy Exporter endpoints, the below alert could be configured, and if any of the HAProxy Exporter endpoints fail, an alert is triggered.
```
- alert: EnsureAllEndpointsAreScraped
  expr: exporter_consolidator_successful_endpoints < 64
```
