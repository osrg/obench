package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/osrg/obench/latency"
	"github.com/osrg/obench/master"
	"github.com/osrg/obench/worker"
)

var (
	rootCmd = &cobra.Command{
		Use:        "obench",
		Short:      "A benchmark for container orchestrators",
		SuggestFor: []string{"obench"},
	}
)

func init() {
	rootCmd.AddCommand(
		master.NewMasterCommand(),
		worker.NewWorkerCommand(),

		latency.NewLatencyCommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("failed to execute command: %s\n", err)
		os.Exit(1)
	}
}
