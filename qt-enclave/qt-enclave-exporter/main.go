package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

const (
	period = time.Second
	re     = `CpuUsage: (\d+\.\d+)%, MemTotal: (\d+) kB, MemFree: (\d+) kB, MemAvailable: (\d+) kB`
)

var (
	qlogData = make([]float64, 4)
	rwMutex  sync.RWMutex
	logFile  string
	port     int
	lastLog  string
)

// readLastLine read latest qlog
func readLastLine(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}
	fileSize := fileInfo.Size()

	if fileSize == 0 {
		return "", fmt.Errorf("file is nil")
	}

	buffer := make([]byte, 1)
	for i := fileSize - 1; i >= 0; i-- {
		_, err := file.Seek(i, io.SeekStart)
		if err != nil {
			return "", err
		}
		_, err = file.Read(buffer)
		if err != nil {
			return "", err
		}
		if buffer[0] == '\n' {
			if i == fileSize-1 {
				continue
			}
			break
		}
		if i == 0 {
			_, err := file.Seek(i, io.SeekStart)
			if err != nil {
				return "", err
			}
		}
	}

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil {
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return "", err
		}
		line, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
	}

	return line, nil
}

func watchResourceLog() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Warnf("failed to creating watcher: %v\n", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(logFile)
	if err != nil {
		logrus.Warnf("failed to add file into watcher: %v\n", err)
		return
	}
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logrus.Infof("failed to get file event")
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				logLine, err := readLastLine(logFile)
				if err != nil {
					logrus.Infof("failed to read resource file, err: %v", err)
					continue
				}
				if logLine == lastLog {
					logrus.Warn("get the last log, qlog is not update")
					continue
				}
				// regular matching related data
				dateRegex := regexp.MustCompile(re)
				matches := dateRegex.FindStringSubmatch(logLine)
				if len(matches) == 5 {
					cpuUsage, _ := strconv.ParseFloat(matches[1], 64)
					memTotal, _ := strconv.ParseFloat(matches[2], 64)
					memFree, _ := strconv.ParseFloat(matches[3], 64)
					memAvailable, _ := strconv.ParseFloat(matches[4], 64)

					logrus.Infof("last line of the log file: CpuUsage: %.2f, MemTotal: %.2f, MemFree: %.2f, MemAvailable: %.2f\n", cpuUsage, memTotal, memFree, memAvailable)
					rwMutex.Lock()
					qlogData[0] = cpuUsage
					qlogData[1] = memTotal
					qlogData[2] = memFree
					qlogData[3] = memAvailable
					rwMutex.Unlock()
					lastLog = logLine
				} else {
					logrus.Warn("failed to decode log, log format is not right")
				}
			} else {
				// log rotate happend, should restart fswatch
				logrus.Warnf("restart watch file")
				return
			}
		case err, _ := <-watcher.Errors:
			logrus.Warnf("failed to watch file: %v\n", err)
			return
		}
	}
}

func tryWatchLog() {
	for {
		watchResourceLog()
		time.Sleep(period)
	}
}

func registerPrometheusMetrics(errCh chan error) {
	gaugeFunc := []prometheus.GaugeFunc{
		// define CPUUsage GaugeFunc
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "qingtian",
				Name:      "cpu_usage_percent",
				Help:      "Current CPU usage percentage",
			},
			func() float64 {
				rwMutex.RLock()
				defer rwMutex.RUnlock()
				return qlogData[0]
			},
		),

		// define MemTotal GaugeFunc
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "qingtian",
				Name:      "memory_total",
				Help:      "Current total memory",
			},
			func() float64 {
				rwMutex.RLock()
				defer rwMutex.RUnlock()
				return qlogData[1]
			},
		),

		// define MemFree GaugeFunc
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "qingtian",
				Name:      "memory_free",
				Help:      "Current free memory",
			},
			func() float64 {
				rwMutex.RLock()
				defer rwMutex.RUnlock()
				return qlogData[2]
			},
		),

		// define MemAvailable GaugeFunc
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "qingtian",
				Name:      "memory_available",
				Help:      "Current available memory",
			},
			func() float64 {
				rwMutex.RLock()
				defer rwMutex.RUnlock()
				return qlogData[3]
			},
		),
	}

	// create a prometheus registry
	reg := prometheus.NewRegistry()
	for _, val := range gaugeFunc {
		if err := reg.Register(val); err != nil {
			logrus.Errorf("failed to register gauge: %v", err)
			return
		}
	}

	// start server with port, support metrics interface
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	logrus.Debug("starting server on %d", port)
	if err := http.ListenAndServe(fmt.Sprintf("%s%d", ":", port), nil); err != nil {
		logrus.Errorf("failed to start server: %v", err)
		errCh <- fmt.Errorf("failed to listen port %d: %w", port, err)
	}
}

func handleSignals(errCh chan error) {
	var sig os.Signal
	var handledSignals = []os.Signal{
		unix.SIGTERM,
		unix.SIGINT,
	}

	// Do not print message when dealing with SIGPIPE, which may cause
	// nested signals and consume lots of cpu bandwidth.
	signal.Ignore(unix.SIGPIPE)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, handledSignals...)

	select {
	case sig = <-signalChannel:
		logrus.Errorf("interrupted by a %v signal, process exiting", sig)
		return
	case err := <-errCh:
		logrus.Errorf("receive error: %v", err)
		return
	}
}

func runStart(_ *cobra.Command, _ []string) error {
	errCh := make(chan error)

	go tryWatchLog()
	go registerPrometheusMetrics(errCh)
	handleSignals(errCh)
	return fmt.Errorf("qt enclave export process exit")
}

func newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "qt-export",
		Short:                 "collect qlog and send to k8s",
		RunE:                  runStart,
		Args:                  cobra.NoArgs,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
	}

	cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	cmd.PersistentFlags().MarkShorthandDeprecated("help", "please use --help")

	flags := cmd.Flags()
	flags.IntVarP(&port, "port", "p", 9113, "prometheus listen port")
	flags.StringVarP(&logFile, "log-file", "l", "/var/log/qlog/resource.log", "qlog reoucese log file")

	return cmd
}

func main() {
	cmd := newExportCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
