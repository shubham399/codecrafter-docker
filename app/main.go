//go:build linux
// +build linux

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
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

func initalize(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dir, 0777)
		if errDir != nil {
			log.Fatal("MKDIR", err)
		}
	}
	os.MkdirAll(filepath.Join(dir, "/dev/null"), 0777)
	// err = os.MkdirAll(filepath.Join(dir, "/usr/local/bin/"), 0777)

	// err = os.MkdirAll(filepath.Join(dir, "/tmp"), 0777)
	return
}

func copyBinary(dir string, binary string) {
	// err = os.MkdirAll(filepath.Join(dir, "/bin"), 0777)
	source := strings.TrimSpace(getBinaryPath(binary))
	destination := filepath.Join(dir, source)
	abs, _ := filepath.Abs(destination)
	copyFile(source, destination)
	makeExecutable(abs)
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
	// Fetch an image from the Docker Registry, uncompress it in the local directory
	imageName := os.Args[2]
	os.Remove(directory)
	initalize(directory)
	getImage(directory, imageName)
	copyBinary(directory, command)
	// fmt.Println(command)
	// fmt.Println(args)
	// make the current directory the chroot jail
	absPath, _ := filepath.Abs(directory)
	// fmt.Println(absPath)
	chError := syscall.Chroot(absPath)
	if chError != nil {
		log.Fatal("Chroot Error", chError)
	}
	cmd := exec.Command(command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// https://medium.com/@teddyking/namespaces-in-go-basics-e3f0fc1ff69a
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Cloneflags: syscall.CLONE_NEWNS |
	// 		syscall.CLONE_NEWUTS |
	// 		syscall.CLONE_NEWIPC |
	// 		syscall.CLONE_NEWPID |
	// 		syscall.CLONE_NEWNET |
	// 		syscall.CLONE_NEWUSER,
	// 	// syscall.CLONE_NEWPID,
	// }
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

func getImage(directory string, imageName string) {
	type AuthResponse struct {
		Token       string    `json:"token"`
		AccessToken string    `json:"access_token"`
		ExpiresIn   int       `json:"expires_in"`
		IssuedAt    time.Time `json:"issued_at"`
	}

	type ManifestResponse struct {
		FsLayers []struct {
			BlobSum string `json:"blobSum"`
		} `json:"fsLayers"`
	}

	imageAndTag := strings.Split(imageName, ":")
	image := imageAndTag[0]
	var tag string
	// if imageName isn't of image:tag form,
	// set tag as it's needed in the url later
	if len(imageAndTag) > 1 {
		tag = imageAndTag[1]
	} else {
		tag = "latest"
	}

	// add librairy in front of the image name
	// for docker published images. This assumes the script will
	// always be called with such images names.
	authUrl := "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/" + image + ":pull"
	client := &http.Client{}
	req, err := http.NewRequest("GET", authUrl, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // response body is []byte
	var result AuthResponse
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	layersUrl := "https://registry.hub.docker.com/v2/library/" + image + "/manifests/" + tag
	// Create a new request using http
	req, err = http.NewRequest("GET", layersUrl, nil)
	// Create a Bearer string by appending string access token
	var bearer = "Bearer " + result.AccessToken
	// add authorization header to the req
	req.Header.Add("Authorization", bearer)
	resp, err = client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}
	// log.Println("response " + string([]byte(body)))
	var manifest ManifestResponse
	if err := json.Unmarshal(body, &manifest); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}
	log.Println(manifest)
	// the manifest contains an FsLayers array
	// with the blobsum hash that points
	// to the image layer to request
	for i := 0; i < len(manifest.FsLayers); i++ {
		manifestUrl := "https://registry.hub.docker.com/v2/library/" + image + "/blobs/" + manifest.FsLayers[i].BlobSum
		req, err = http.NewRequest("GET", manifestUrl, nil)
		req.Header.Add("Authorization", bearer)
		resp, err = client.Do(req)
		if err != nil {
			log.Println("Error on response.\n[ERROR] -", err)
		}
		defer resp.Body.Close()
		// log.Println("BLOB", resp.Body)
		writeToFile(resp.Body, "./layersFile")

		var waitStatus syscall.WaitStatus

		cmd := exec.Command("tar", "--extract", "--file", "./layersFile", "-C", directory)
		var outbuf, errbuf bytes.Buffer
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf
		// run the command
		err := cmd.Run()
		cm2 := exec.Command("ls", directory)
		output2, _ := cm2.CombinedOutput()
		fmt.Println(string(output2))
		fmt.Print("OUTPUT", outbuf.String())
		fmt.Fprintf(os.Stderr, "ERROR"+errbuf.String())
		if err != nil {
			// if there's an error, print the command's output
			// and exit with the same error code
			if exitError, ok := err.(*exec.ExitError); ok {
				// fmt.Print(outbuf.String())
				// this doesn't actually print anything
				// fmt.Fprintf(os.Stderr, errbuf.String())
				waitStatus = exitError.Sys().(syscall.WaitStatus)
				os.Exit(waitStatus.ExitStatus())
			}
		}
		// print stdout and stderr
		// fmt.Println("Extract Done")
		os.Remove("./layersFile")
	}
}

func writeToFile(content io.Reader, filename string) {
	// Create the file
	file, err := os.Create(filename)
	if err != nil {
		log.Println("Error creating the file", err)
	}
	defer file.Close()
	// Write the body to file
	_, err = io.Copy(file, content)
	if err != nil {
		log.Println("Error writing to the file", err)
	}
}
