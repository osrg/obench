package latency

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func workerJobYaml(id int, masterURL string) string {
	return fmt.Sprintf(`
apiVersion: batch/v1
kind: Job
metadata:
  name: obench-worker-%d
spec:
  template:
    metadata:
      name: obench-worker-%d
    spec:
      containers:
      - name: obench-worker-%d
        image: mitake/obench
        command: ["obench", "worker"]
        args: ["--master-url", "%s"]
      restartPolicy: Never
`, id, id, id, masterURL)
}

func masterDeploymentYaml() string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: obench-master
  labels:
    app: obench
    tier: obench-master
spec:
  # if your cluster supports it, uncomment the following to automatically create
  # an external load-balanced IP for the frontend service.
  type: LoadBalancer
  ports:
    # the port that this service should serve on
  - port: 8080
  selector:
    app: obench
    tier: obench-master
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: obench-master
  # these labels can be applied automatically
  # from the labels in the pod template if not set
  # labels:
  #   app: obench
  #   tier: frontend
spec:
  # this replicas value is default
  # modify it according to your case
  replicas: 1
  # selector can be applied automatically
  # from the labels in the pod template if not set
  # selector:
  #   matchLabels:
  #     app: obench
  #     tier: frontend
  template:
    metadata:
      labels:
        app: obench
        tier: obench-master
    spec:
      containers:
      - name: obench
        image: mitake/obench
        command: ["obench", "master"]
        args: ["--nr-workers", "%d"]
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
          # If your cluster config does not include a dns service, then to
          # instead access environment variables to find service host
          # info, comment out the 'value: dns' line above, and uncomment the
          # line below.
          # value: env
        ports:
        - containerPort: 8080
`, nrWorkers)
}

var (
	nrWorkers int
)

func getMasterResultURL() (string, error) {
	for {
		masterService := exec.Command("kubectl", "get", "services")
		masterServiceStdout, err := masterService.StdoutPipe()
		if err != nil {
			fmt.Printf("failed to start kubectl: %s\n", err)
			return "", err
		}

		masterService.Start()

		buf := make([]byte, 1024) // FIXME
		rbytes := 0
		for {
			r, err := masterServiceStdout.Read(buf[rbytes:])
			if err != nil && err != io.EOF {
				fmt.Printf("failed to read a result of kubectl get services: %s\n", err)
				return "", nil
			}
			rbytes += r
			if r == 0 {
				break
			}
		}

		masterService.Wait()

		result := string(buf)
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			columns := strings.Fields(line)
			if strings.Compare(columns[0], "obench-master") != 0 {
				continue
			}

			if len(columns) < 3 {
				break
			}

			extIP := columns[2]
			if strings.Compare(extIP, "<pending>") == 0 {
				break
			}

			return "http://" + extIP + ":8080/result", nil
		}

		<-time.After(1 * time.Second)
		// fmt.Printf("waiting... service external IP\n")
	}
}

func getMasterWorkerURL() (string, error) {
	for {
		getPod := exec.Command("kubectl", "get", "pods", "-o", "wide")
		getPodStdout, err := getPod.StdoutPipe()
		if err != nil {
			fmt.Printf("failed to start kubectl: %s\n", err)
			return "", err
		}

		getPod.Start()

		buf := make([]byte, 1024) // FIXME
		rbytes := 0
		for {
			r, err := getPodStdout.Read(buf[rbytes:])
			if err != nil && err != io.EOF {
				fmt.Printf("failed to read a result of kubectl get services: %s\n", err)
				return "", nil
			}
			rbytes += r
			if r == 0 {
				break
			}
		}

		getPod.Wait()

		result := string(buf)
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			columns := strings.Fields(line)
			matched, err := regexp.MatchString("obench-master*", columns[0])
			if err != nil {
				return "", err
			}
			if !matched {
				continue
			}

			if len(columns) < 6 {
				break
			}

			ip := columns[5]
			if strings.Compare(ip, "<pending>") == 0 {
				break
			}

			return "http://" + ip + ":8080/worker", nil
		}

		<-time.After(1 * time.Second)
		// fmt.Printf("waiting... pod ip\n")
	}
}

func launchMaster() (string, string, error) {
	masterCreate := exec.Command("kubectl", "create", "-f", "-")
	masterCreateStdin, err := masterCreate.StdinPipe()
	if err != nil {
		fmt.Printf("failed to execute kubectl: %s\n", err)
		return "", "", err
	}

	masterCreate.Start()
	_, err = masterCreateStdin.Write([]byte(masterDeploymentYaml()))
	if err != nil {
		fmt.Printf("failed to write a yaml of master deployment to kubectl stdin: %s\n", err)
		return "", "", err
	}
	err = masterCreateStdin.Close()
	if err != nil {
		fmt.Printf("failed to write a yaml of master deployment to kubectl stdin: %s\n", err)
		return "", "", err
	}

	masterCreate.Wait()

	masterResultURL, err := getMasterResultURL()
	if err != nil {
		return "", "", err
	}
	// fmt.Printf("masterResultURL: %s\n", masterResultURL)

	masterWorkerURL, err := getMasterWorkerURL()
	// fmt.Printf("masterWorkerURL: %s\n", masterWorkerURL)

	return masterResultURL, masterWorkerURL, nil
}

func deleteMaster() error {
	del := exec.Command("kubectl", "delete", "-f", "-")
	delStdin, err := del.StdinPipe()
	if err != nil {
		fmt.Printf("failed to execute kubectl: %s\n", err)
		return err
	}

	del.Start()
	_, err = delStdin.Write([]byte(masterDeploymentYaml()))
	if err != nil {
		fmt.Printf("failed to write a yaml of master deployment to kubectl stdin: %s\n", err)
		return err
	}
	err = delStdin.Close()
	if err != nil {
		fmt.Printf("failed to write a yaml of master deployment to kubectl stdin: %s\n", err)
		return err
	}

	del.Wait()

	del = exec.Command("kubectl", "delete", "services", "obench-master")
	del.Start()
	del.Wait()

	return nil
}

func launchWorker(id int, url string) error {
	workerCreate := exec.Command("kubectl", "create", "-f", "-")
	workerCreateStdin, err := workerCreate.StdinPipe()
	if err != nil {
		fmt.Printf("failed to execute kubectl: %s\n", err)
		return err
	}

	workerCreate.Start()
	_, err = workerCreateStdin.Write([]byte(workerJobYaml(id, url)))
	if err != nil {
		fmt.Printf("failed to write a yaml of worker job to kubectl stdin: %s\n", err)
		return err
	}
	workerCreateStdin.Close()

	// fmt.Printf("created worker %d\n", id)
	return nil
}

func deleteWorker(ch chan struct{}, id int, url string) error {
	workerCreate := exec.Command("kubectl", "delete", "-f", "-")
	workerCreateStdin, err := workerCreate.StdinPipe()
	if err != nil {
		fmt.Printf("failed to execute kubectl: %s\n", err)
		return err
	}

	workerCreate.Start()
	_, err = workerCreateStdin.Write([]byte(workerJobYaml(id, url)))
	if err != nil {
		fmt.Printf("failed to write a yaml of worker job to kubectl stdin: %s\n", err)
		return err
	}
	workerCreateStdin.Close()

	ch <- struct{}{}
	return nil
}

func NewLatencyCommand() *cobra.Command {
	lc := &cobra.Command{
		Use:   "latency",
		Short: "latency benchmark",
		Run:   runLatency,
	}

	lc.Flags().IntVar(&nrWorkers, "nr-workers", 0, "A number of workers")

	return lc
}

func runLatency(cmd *cobra.Command, args []string) {
	if nrWorkers <= 0 {
		fmt.Printf("invalid number of workers: %d\n", nrWorkers)
		return
	}

	resultURL, workerURL, err := launchMaster()
	if err != nil {
		fmt.Printf("failed to launch master: %s\n", err)
		return
	}

	for i := 0; i < nrWorkers; i++ {
		go func(id int) {
			err = launchWorker(id, workerURL)
			if err != nil {
				fmt.Printf("failed to launch workers: %s\n", err)
				return
			}
		}(i)
	}

	for {
		resp, err := http.Get(resultURL)
		if err != nil {
			fmt.Printf("failed to get result: %s\n", err)
			return
		}

		if resp.StatusCode == 404 {
			// fmt.Printf("waiting result\n")
			<-time.After(1 * time.Second)
			continue
		}

		buf := make([]byte, 1024*1024) // FIXME
		resp.Body.Read(buf)
		fmt.Printf(string(buf))
		break
	}

	deleteMaster()

	ch := make(chan struct{})
	for i := 0; i < nrWorkers; i++ {
		go deleteWorker(ch, i, workerURL)
	}

	for i := 0; i < nrWorkers; i++ {
		<-ch
	}
}
