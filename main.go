package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	_ "net/http/pprof"
)

var Cmd = &cobra.Command{
	Use:  "tenta",
	Long: "Fast and easy local LAN proxy cache",
	RunE: run,
}

var args struct {
	debug           bool
	dataDir         string
	maxCacheAge     int
	cronSchedule    string
	httpPort        int
	requestTimeout  int
	maxBodySize     int64
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.dataDir,
		"data-dir",
		"data/",
		"Directory to use for caching files",
	)
	flags.IntVar(
		&args.maxCacheAge,
		"max-cache-age",
		0,
		"Max age (in hours) of files. Value of 0 means no files will be deleted (default 0)",
	)
	flags.IntVar(
		&args.httpPort,
		"http-port",
		8080,
		"Port to use for the HTTP server",
	)

	flags.BoolVar(
		&args.debug,
		"debug",
		false,
		"Enable debug logging",
	)

	flags.StringVar(
		&args.cronSchedule,
		"cron-schedule",
		"* */1 * * *",
		"Cron schedule to use for cleaning up cache files",
	)

	flags.IntVar(
		&args.requestTimeout,
		"request-timeout",
		30,
		"Timeout (in seconds) for outbound HTTP requests",
	)

	flags.Int64Var(
		&args.maxBodySize,
		"max-body-size",
		1073741824, // 1GB default
		"Maximum size (in bytes) of response bodies to cache",
	)

	Cmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "prom"}, cobra.ShellCompDirectiveDefault
	})
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	if err := Cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(0)
}

// validateConfig validates all configuration parameters
func validateConfig() error {
	// Validate data directory
	info, err := os.Stat(args.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the directory if it doesn't exist
			log.Printf("Data directory %s does not exist, creating it", args.dataDir)
			if err := os.MkdirAll(args.dataDir, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %v", err)
			}
		} else {
			return fmt.Errorf("error accessing data directory: %v", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("data-dir must be a directory, got file: %s", args.dataDir)
	}

	// Validate max cache age
	if args.maxCacheAge < 0 {
		return fmt.Errorf("max-cache-age must be >= 0, got %d", args.maxCacheAge)
	}

	// Validate HTTP port
	if args.httpPort < 1 || args.httpPort > 65535 {
		return fmt.Errorf("http-port must be between 1 and 65535, got %d", args.httpPort)
	}

	// Validate request timeout
	if args.requestTimeout < 1 {
		return fmt.Errorf("request-timeout must be >= 1, got %d", args.requestTimeout)
	}

	// Validate max body size
	if args.maxBodySize < 1024 { // Minimum 1KB
		return fmt.Errorf("max-body-size must be at least 1024 bytes, got %d", args.maxBodySize)
	}

	// Note: Cron schedule validation happens in StartCron()
	// We don't validate it here to avoid delaying startup

	return nil
}

func run(cmd *cobra.Command, argv []string) error {
	log.Println("Starting Tenta!")

	// Validate configuration before starting
	if err := validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Printf("Configuration: dataDir=%s, maxCacheAge=%dh, httpPort=%d, cron=%s",
		args.dataDir, args.maxCacheAge, args.httpPort, args.cronSchedule)

	if args.debug {
		go func() {
			log.Println("Starting pprof server on port 6060")
			log.Println(http.ListenAndServe(":6060", nil))
		}()
	}

	StartCron()
	StartMetrics()
	StartHTTP()

	return nil
}
