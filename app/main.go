package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func makeExecutable(path string) {
	var err error
	// change mode
	err = os.Chmod(path, 0777)
	if err != nil {
		log.Fatal("CHMOD", err)
	}
}

func getBinaryPath(binary string) string {
	cmd := exec.Command("which", binary)
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("WHICH", err)
	}
	return string(output)
}

func copyFile(sourceFilePath, destinationFilePath string) {
	var err error
	var source *os.File
	source, err = os.Open(sourceFilePath)
	if err != nil {
		log.Fatal("OPEN", err)
	}
	defer source.Close()

	var destination *os.File
	destination, err = os.Create(destinationFilePath)
	if err != nil {
		log.Fatal("CREATE", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		log.Fatal("COPY", err)
	}
}

func initalize(dir string, binary string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dir, 0777)
		if errDir != nil {
			log.Fatal("MKDIR", err)
		}
	}
	err = os.MkdirAll(filepath.Join(dir, "/dev/null"), 0777)
	err = os.MkdirAll(filepath.Join(dir, "/usr/local/bin/"), 0777)
	err = os.MkdirAll(filepath.Join(dir, "/bin"), 0777)
	err = os.MkdirAll(filepath.Join(dir, "/tmp"), 0777)
	source := strings.TrimSpace(getBinaryPath(binary))
	destination := filepath.Join(dir, source)
	abs, err := filepath.Abs(destination)
	copyFile(source, destination)
	makeExecutable(abs)
	return
}

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// fmt.Println("Your code goes here!")

	// Uncomment this block to pass the first stage!
	// fmt.Printf(os.Args)
	// fmt.Printf("%v\n", os.Args)

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	directory := "./sandbox"
	os.Remove(directory)
	initalize(directory, command)
	// fmt.Println(command)
	// fmt.Println(args)
	// make the current directory the chroot jail
	absPath, _ := filepath.Abs("./sandbox")
	// fmt.Println(absPath)
	chError := syscall.Chroot(absPath)
	if chError != nil {
		log.Fatal("Chroot Error", chError)
	}
	cmd := exec.Command(command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	fmt.Fprintf(os.Stdout, outStr)
	fmt.Fprintf(os.Stderr, errStr)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
	}

}
