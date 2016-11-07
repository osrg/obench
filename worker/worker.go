package worker

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	masterURL  string
)

func NewWorkerCommand() *cobra.Command {
	mc := &cobra.Command{
		Use:   "worker",
		Short: "start worker of obench",
		Run:   runWorker,
	}

	mc.Flags().StringVar(&masterURL, "master-url", "http://localhost:8080/worker", "A URL of obench master")

	return mc
}

func runWorker(cmd *cobra.Command, args []string) {
	fmt.Printf("obench worker started\n")

	_, err := http.Get(masterURL)
	if err != nil {
		fmt.Printf("failed to send GET request to master (%s): %s\n", masterURL, err)
		os.Exit(1)
	}
}
