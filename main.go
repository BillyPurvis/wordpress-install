package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Songmu/prompter"
)

const downloadDir string = "./set-up"
const fileName string = "wordpress.tar.gz"

// downloadFile downloads a file and places in the target directory
func downloadFile(destination, fileName, url string) error {
	// Check if we have the install dir
	if _, err := os.Stat(destination); os.IsNotExist(err) {
		os.Mkdir(destination, os.ModePerm)
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err == nil {
		fmt.Println("File Downloaded")
	}
	return err
}

// unTar takes a destination path and a reader; a tar reader loops over the tar file
// creating directories and writing files
func unTar(destination string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	// Defer all reader streams
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Essentially a while loop
	for {
		header, err := tr.Next()

		switch {
		// check for end of file
		case err == io.EOF:
			return nil

		// return any other errors
		case err != nil:
			return err

		// sometimes the header is empty, it's whack
		case header == nil:
			continue
		}

		target := filepath.Join(destination, header.Name)

		switch header.Typeflag {

		// Create Dir if doesn't exist
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		case tar.TypeReg:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tr); err != nil {
				return err
			}

			file.Close()
		}
	}
}

// cleanUp deletes the tar file as we no longer need it
func cleanUp(target string) error {
	err := os.RemoveAll(target)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	// Download the file, because you kinda need it
	err := downloadFile(downloadDir, fmt.Sprintf("%v/%v", downloadDir, fileName), "https://wordpress.org/latest.tar.gz")
	if err != nil {
		panic(err)
	}

	// Open the file to return *File that satisfies IO.Reader
	file, err := os.Open(fmt.Sprintf("%s/wordpress.tar.gz", downloadDir))
	if err != nil {
		panic(err)
	}

	// UnTar the downloaded file
	fmt.Println(fmt.Sprintf("UnTaring into %s directory %s", downloadDir, string(129302)))
	var fileReader io.ReadCloser = file
	err = unTar(downloadDir, fileReader)
	if err != nil {
		panic(err)
	}

	// Move directory to app dir
	oldDir := fmt.Sprintf("%s/wordpress/wp-admin", downloadDir)
	newDir := "./wp-admin"
	err = os.Rename(oldDir, newDir)
	if err != nil {
		panic(err)
	}

	// Delete set up directories
	fmt.Println(fmt.Sprintf("Deleting %s directory and all it's children %s...", downloadDir, string(128561)))
	err = cleanUp(downloadDir)
	if err != nil {
		panic(err)
	}

	// Ask if they want Docker to start detached
	if prompter.YN(fmt.Sprintf("Do you want to start Docker Compose? %s", string(129300)), true) {
		fmt.Println("Starting docker-compose in dev mode!\r")
		cmd := exec.Command("/usr/local/bin/docker-compose", "-f", "docker-compose.yml", "-f", "docker-compose-dev.yml", "up", "-d")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
			fmt.Println(stderr.String())
		}
		fmt.Println("Docker Started!")
	}
}
