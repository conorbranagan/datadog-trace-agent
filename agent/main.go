package main

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	_ "net/http/pprof" // register debugger
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/raclette/config"
	"github.com/DataDog/raclette/statsd"
	log "github.com/cihub/seelog"
)

// handleSignal closes a channel to exit cleanly from routines
func handleSignal(exit chan struct{}) {
	sigChan := make(chan os.Signal, 10)
	signal.Notify(sigChan)
	for signal := range sigChan {
		switch signal {
		case syscall.SIGINT, syscall.SIGTERM:
			log.Info("Received interruption signal")
			close(exit)
		}
	}
}

// opts are the command-line options
var opts struct {
	configFile string
	debug      bool
	topology   bool
	version    bool
}

// version info sourced from build flags
var (
	Version   string
	BuildDate string
	GitCommit string
	GitBranch string
	GoVersion string
)

// versionString returns the version information filled in at build time
func versionString() string {
	var buf bytes.Buffer

	if Version != "" {
		fmt.Fprintf(&buf, "Version: %s\n", Version)
	}
	if GitCommit != "" {
		fmt.Fprintf(&buf, "Git hash: %s\n", GitCommit)
	}
	if GitBranch != "" {
		fmt.Fprintf(&buf, "Git branch: %s\n", GitBranch)
	}
	if BuildDate != "" {
		fmt.Fprintf(&buf, "Build date: %s\n", BuildDate)
	}
	if GoVersion != "" {
		fmt.Fprintf(&buf, "Go Version: %s\n", GoVersion)
	}

	return buf.String()
}

// main is the entrypoint of our code
func main() {
	flag.StringVar(&opts.configFile, "config", "/etc/datadog/trace-agent.ini", "Trace agent ini config file.")
	flag.BoolVar(&opts.debug, "debug", false, "Turn on debug mode")
	flag.BoolVar(&opts.topology, "topology", false, "Use TCP conns info to draw network topology")
	flag.BoolVar(&opts.version, "version", false, "Show version information and exit")
	flag.Parse()

	if opts.version {
		fmt.Print(versionString())
		os.Exit(0)
	}

	// Instantiate the config
	var conf *config.File
	var agentConf *config.AgentConfig

	conf, err := config.New(opts.configFile)
	if conf == nil {
		fmt.Printf("No valid config file found. Using defaults")
		agentConf = config.NewDefaultAgentConfig()
	} else {
		agentConf, err = config.NewAgentConfig(conf)
		if err != nil {
			panic(err)
		}
	}

	// Initialize logging
	err = config.NewLoggerLevel(opts.debug)
	if err != nil {
		panic(fmt.Errorf("error with logger: %v", err))
	}
	defer log.Flush()
	agentConf.Topology = opts.topology

	// Initialize dogstatsd client
	err = statsd.Configure(agentConf)
	if err != nil {
		fmt.Errorf("Error configuring dogstatsd: %v", err)
	}

	// Seed rand
	rand.Seed(time.Now().UTC().UnixNano())

	agent := NewAgent(agentConf)

	// Handle stops properly
	go handleSignal(agent.exit)

	agent.Run()
}
