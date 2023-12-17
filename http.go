package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

// login authenticates user with the API
func login() (string, error) {
	var (
		requestBody []byte
		err         error
	)

	requestBody, err = json.Marshal(map[string]string{
		"email":    os.Getenv("USER_EMAIL"),
		"password": os.Getenv("USER_PASSWORD"),
	})

	if err != nil {
		log.Fatalln(err)
	}

	url := fmt.Sprintf("%s/auth/login", api2Root)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var v map[string]string
	if err = json.Unmarshal(body, &v); err != nil {
		return "", err
	}

	if v["status"] == "error" {
		return "", errors.New(fmt.Sprintf("authentication error %d: %s\n", resp.StatusCode, v["message"]))
	} else if v["status"] == "ok" {
		return v["data"], nil
	}

	return "", errors.New(v["message"])
}

// downloadFile downloads file to the filepath from url
func downloadFile(filepath string, url string, size int64) (err error) {
	// Check if file exists
	stat, err := os.Stat(filepath)
	if err == nil {
		if size > 0 && stat.Size() == size {
			log.Printf("skipping, file exists: %s, size matches: %d", filepath, size)
			return nil
		}
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to send a HTTP GET request: %s\n", err.Error())
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}(resp.Body)

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file %s to %s: bad status: %s\n", url, filepath, resp.Status)
	}

	// Create the dir
	dir := path.Dir(filepath)
	err = os.MkdirAll(dir, 0750)
	if err != nil {
		return fmt.Errorf("failed to create a directory %s: %s\n", dir, err.Error())
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create a file downloaded %s to %s: %s\n", url, filepath, err.Error())
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			log.Printf("failed to close file: %v", err)
		}
	}(out)

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write a file downloaded %s to %s: %s\n", url, filepath, err.Error())
	}

	// Change a file mode for known binaries to make them executable
	for s, b := range binarySuffixes {
		if b && strings.HasSuffix(filepath, s) {
			err = os.Chmod(filepath, 0755)
			if err != nil {
				log.Printf("failed to change file mode for %s: %s\n", filepath, err.Error())
			}
		}
	}

	return nil
}
