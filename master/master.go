package master

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

var (
	nrWorkers      int
	masterHTTPPort int
	workerPath     string
	resultPath     string

	workerArrivalCh chan *workerInfo
	startTime       time.Time

	result *ResultInfo
)

func NewMasterCommand() *cobra.Command {
	mc := &cobra.Command{
		Use:   "master",
		Short: "start master of obench",
		Run:   runMaster,
	}

	mc.Flags().IntVar(&masterHTTPPort, "master-http-port", 8080, "A port number of master HTTP server")
	mc.Flags().IntVar(&nrWorkers, "nr-workers", 0, "A number of workers")
	mc.Flags().StringVar(&workerPath, "worker-path", "/worker", "A path for workers")
	mc.Flags().StringVar(&resultPath, "result-path", "/result", "A path for result")

	return mc
}

type workerInfo struct {
	arrivalTime time.Time
}

type ResultInfo struct {
	Start        time.Time
	FirstArrival time.Time
	Latencies    []jsonDuration
}

func workerHandler(w http.ResponseWriter, r *http.Request) {
	arrivalTime := time.Now()
	io.WriteString(w, "OK")

	workerArrivalCh <- &workerInfo{arrivalTime: arrivalTime}
}

type jsonDuration time.Duration

func (t jsonDuration) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("\"%d\"", time.Duration(t).Nanoseconds())
	return []byte(s), nil
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	if result == nil {
		fmt.Printf("not completed yet\n")
		http.NotFound(w, r)
		return
	}

	// for i, l := range result.latencies {
	// 	fmt.Printf("%d %v\n", i, l)
	// }

	s, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("marshal error: %s\n", err)
	}
	fmt.Printf("marshal: %s\n", string(s))
	io.WriteString(w, string(s))
	// enc := json.NewEncoder(w)
	// enc.Encode(result)
}

func summary(res []*workerInfo) {
	latencies := make([]jsonDuration, len(res))
	for i := range res {
		latencies[i] = jsonDuration(res[i].arrivalTime.Sub(startTime))
	}

	sort.Sort(durationSlice(latencies))

	result = &ResultInfo{
		Start:     startTime,
		Latencies: latencies,
	}
}

type durationSlice []jsonDuration

func (d durationSlice) Len() int {
	return len(d)
}

func (d durationSlice) Less(i, j int) bool {
	return d[i] < d[j]
}

func (d durationSlice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func runMaster(cmd *cobra.Command, args []string) {
	fmt.Printf("launch latency master\n")

	if nrWorkers <= 0 {
		fmt.Printf("invalid number of workers: %d\n", nrWorkers)
		os.Exit(1)
	}

	workerArrivalCh = make(chan *workerInfo)
	startTime = time.Now()
	fmt.Printf("start time: %s\n", startTime)

	http.HandleFunc(workerPath, workerHandler)
	http.HandleFunc(resultPath, resultHandler)

	go func() {
		portStr := fmt.Sprintf(":%d", masterHTTPPort)
		http.ListenAndServe(portStr, nil)
	}()

	nrArrivedWorkers := 0
	workers := make([]*workerInfo, nrWorkers)
	for {
		worker := <-workerArrivalCh
		workers[nrArrivedWorkers] = worker
		nrArrivedWorkers++

		if nrArrivedWorkers == nrWorkers {
			break
		}
	}

	summary(workers)

	<-make(chan struct{})
}
