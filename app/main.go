package main

import (
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
	output, err := cmd.Output()
	outputDev := os.Stdout
	if err != nil {
		outputDev = os.Stderr
		output = []byte(err.Error())
	}
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, err.Error())
	// 	// log.Fatal(err)

	// } else {
	fmt.Fprintf(outputDev, string(output))
	// }
}
