package main

import (
	"fmt"
	"time"
)

var functionalTests = []string{
	"/build",
	"/certificates",
	"/container",
	"/docs",
	"/exec",
	"/multiple_k3s_clusters",
	"/nomad",
	"/single_file",
	"/single_k3s_cluster",
	"/terraform",
}

var runtimes = []string{"docker", "podman"}

type job struct {
	workingDirectory string
	runtime          string
}

func main() {
	jobCount := len(functionalTests) * len(runtimes)
	jobs := make(chan job, jobCount)
	errors := make(chan error, jobCount)

	// start the workers
	for w := 0; w < 3; w++ {
		fmt.Println("Starting worker", w)
		go startTestWorker(w, jobs, errors)
	}

	// add the jobs
	for _, runtime := range runtimes {
		for _, ft := range functionalTests {
			jobs <- job{workingDirectory: ft, runtime: runtime}
		}
	}
	close(jobs)

	for i := 0; i < jobCount; i++ {
		<-errors
		fmt.Println("Done", i)
	}
}

func startTestWorker(id int, jobs <-chan job, errors chan<- error) {
	for j := range jobs {
		fmt.Println("Running test", id, j.workingDirectory, j.runtime)
		time.Sleep(1 * time.Second)

		errors <- nil
	}
}
