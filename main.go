package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:  "tenta",
	Long: "Fast and easy local LAN proxy cache",
	RunE: run,
}

var args struct {
	debug        bool
	dataDir      string
	maxCacheAge  int
	cronSchedule string
	httpPort     int
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

func run(cmd *cobra.Command, argv []string) error {
	log.Println("Starting Tenta!")
	if args.debug {
		log.Println("Debug logging enabled")
		log.Printf("Cache dir: %s", args.dataDir)
	}
	StartCron()
	StartMetrics()
	StartHTTP()
	return nil
}
