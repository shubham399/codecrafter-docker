package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// fmt.Println("Your code goes here!")

	// Uncomment this block to pass the first stage!
	// fmt.Printf(os.Args)
	// fmt.Printf("%v\n", os.Args)

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	// fmt.Println(command)
	// fmt.Println(args)

	cmd := exec.Command(command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
	}
	fmt.Fprintf(os.Stdout, outStr)
	fmt.Fprintf(os.Stderr, errStr)
}
