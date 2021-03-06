package main

import (
	"fmt"
	"github.com/dotcloud/beam"
	"io"
	"os"
	"os/exec"
)

func worker() {
	worker := beam.NewWorker(&beam.NetTransport{"tcp", ":6379"}, "/jobs")
	worker.RegisterJob("exec", func(name string, args []string, env map[string]string, streams beam.Streamer, db beam.DB) error {
		var (
			cmdName string
			cmdArgs []string
		)
		if len(args) >= 1 {
			cmdName = args[0]
		} else {
			return fmt.Errorf("Not enough arguments")
		}
		if len(args) > 1 {
			cmdArgs = args[1:]
		}
		p := exec.Command(cmdName, cmdArgs...)

		out, err := streams.OpenWrite("stdout")
		if err != nil {
			return err
		}
		errStr, err := streams.OpenWrite("stderr")
		if err != nil {
			return err
		}
		defer out.Close()
		defer errStr.Close()

		p.Stdout = out
		p.Stderr = errStr

		return p.Run()
	})
	worker.Work()
}

func createJob(client *beam.Client) *beam.Job {
	job, err := client.NewJob("exec", os.Args[1:]...)
	if err != nil {
		panic(err)
	}
	job.Env = []string{"DEBUG=1"}
	go func() {
		stdout, err := job.OpenRead("stdout")
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(os.Stdout, stdout); err != nil {
			panic(err)
		}
	}()
	go func() {
		stderr, err := job.OpenRead("stderr")
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(os.Stderr, stderr); err != nil {
			panic(err)
		}
	}()
	return job
}

func main() {
	go worker()

	client, err := beam.NewClient(&beam.NetTransport{"tcp", ":6379"})
	if err != nil {
		panic(err)
	}
	defer client.Close()
	job := createJob(client)

	if err := job.Start(); err != nil {
		panic(err)
	}
	if err := job.Wait(); err != nil {
		panic(err)
	}
}
