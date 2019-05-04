package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tynany/exporter_aggregator/config"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	version       = "0.0.1"
	configPath    = kingpin.Flag("config.path", "Path of the YAML configuration file.").Short('c').Required().String()
	listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9299").String()
	telemetryPath = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	endpoints     []string
	timeout       time.Duration
)

func handler(w http.ResponseWriter, r *http.Request) {

	resultCh := make(chan map[string]float64, 100)
	wg := &sync.WaitGroup{}

	for _, endpoint := range endpoints {
		wg.Add(1)
		go getEndpoint(wg, resultCh, endpoint, timeout)
	}

	wg.Wait()
	close(resultCh)

	metrics, endpointCount := getMetrics(resultCh)

	for metricName, metricVal := range metrics {
		fmt.Fprintf(w, "%s %v\n", metricName, metricVal)
	}
	fmt.Fprintf(w, "exporter_aggregator_successful_endpoints %v\n", endpointCount)
}

func getMetrics(ch chan map[string]float64) (map[string]float64, float64) {
	metrics := make(map[string]float64, 100)
	count := float64(0)
	for {
		select {
		case result, more := <-ch:
			if !more {
				return metrics, count
			}
			for metricName, metricVal := range result {
				if _, exists := metrics[metricName]; exists {
					metrics[metricName] = metrics[metricName] + metricVal
				} else {
					metrics[metricName] = metricVal
				}
			}
			count++
		}
	}
}

func getEndpoint(wg *sync.WaitGroup, ch chan map[string]float64, url string, timeout time.Duration) {
	defer wg.Done()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("could not create request for %s: %s", url, err)
		return
	}

	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("could not do request for %s: %s", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Errorf("bad status code %d for request %s", resp.StatusCode, url)
		return
	}

	metrics := make(map[string]float64)
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		if !strings.HasPrefix(s.Text(), "#") {
			// Only handle metric lines.
			metricName, metricVal, err := processMetric(s.Text(), url)
			if err != nil {
				log.Error(err)
			}
			metrics[metricName] = metricVal
		}
	}
	if err := s.Err(); err != nil {
		log.Errorf("error reading response from %s: %s", url, s.Err().Error())
		return
	}

	if len(metrics) > 0 {
		ch <- metrics
	}
}

func processMetric(line string, url string) (string, float64, error) {

	// Split the metric name from the metric value.
	split := strings.Split(line, " ")

	// Convert the metric value to a floar64.
	metricVal, err := strconv.ParseFloat(split[1], 10)
	if err != nil {
		return "", 0, fmt.Errorf("unable to convert metric to float: %s", err)
	}

	// Handle exporter health metrics (e.g. go_*, process_*) by inserting the endpoint label.
	if strings.HasPrefix(split[0], "go_") || strings.HasPrefix(split[0], "process_") || strings.HasPrefix(split[0], "http_") || strings.HasPrefix(split[0], "process_") {
		var metricName string
		if strings.HasSuffix(split[0], "}") {
			// If the metric has any labels.
			metricName = fmt.Sprintf("%s,endpoint=\"%s\"}", strings.TrimRight(split[0], "}"), url)
		} else {
			// If the metric does not have any labels.
			metricName = fmt.Sprintf("%s{endpoint=\"%s\"}", split[0], url)
		}
		return metricName, metricVal, nil
	}

	// All endpoint metrics.
	return split[0], metricVal, nil
}

func main() {

	kingpin.Version(version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	conf, err := config.GetConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	endpoints = conf.Endpoints

	// No need to handle err here as the timeout value is validated in the config package.
	timeout, _ = time.ParseDuration(conf.Timeout)

	http.HandleFunc(*telemetryPath, handler)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
