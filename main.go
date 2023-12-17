package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gofrs/uuid"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	goRuntime "runtime"
	"strings"
	"syscall"
)

var (
	pEnvironment string // environment (development, test or production), will use a corresponding API to get jobs for processing
	appId        string
	api2Root     string
)

func init() {
	flag.StringVar(&pEnvironment, "env", "", "Environment: dev, test or prod")
	flag.Parse()

	//region Parse environment variables

	if pEnvironment == "" {
		pEnvironment = "dev"
	}

	appId = os.Getenv("VE_APP_ID")
	if appId == "" {
		log.Fatalf("invalid VE_APP_ID env\n")
	}

	api2Root = os.Getenv("VE_API2_ROOT_URL")
	if api2Root == "" {
		log.Fatalf("invalid VE_API2_ROOT_URL env\n")
	}

	//endregion
}

func getPlatformName() string {
	//goland:noinspection GoBoolExpressions
	if goRuntime.GOOS == "windows" {
		return "Win64"
	} else if goRuntime.GOOS == "darwin" {
		return "Mac"
	} else if goRuntime.GOOS == "linux" {
		return "Linux"
	} else {
		log.Fatalf("unsupported OS: %s\n", goRuntime.GOOS)
	}
	return ""
}

func main() {

	fmt.Println("Welcome to VeVerse server launcher")

	//region Authenticate and get the JWT

	token, err := login()
	if err != nil {
		log.Fatalf("failed to login: %s\n", err.Error())
	}

	//endregion

	//region Request the latest release from the API using the AppID

	url := fmt.Sprintf("%s/apps/%s/releases/latest?platform=%s&deployment=Server&configuration=%s", api2Root, appId, getPlatformName(), getConfiguration())
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode >= 400 {
		log.Fatalf("failed to fetch the latest release, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	//endregion

	//region Parse the release metadata

	var container ReleaseMetadataContainer
	err = json.Unmarshal(body, &container)
	if err != nil {
		log.Fatalf("failed to parse release metadata: %s\n", err.Error())
	}

	var metadata = container.ReleaseMetadata
	metadata.AppId, err = uuid.FromString(appId)
	if err != nil {
		log.Printf("failed to parse appId: %s\n", err.Error())
	}

	//endregion

	//region Download the server binaries

	for _, f := range metadata.Files {
		if f.OriginalPath == "" {
			f.OriginalPath = f.Id.String()
		}

		var size int64 = 0
		if f.Size != nil {
			size = *f.Size
		}

		err = downloadFile(f.OriginalPath, f.Url, size)
		if err != nil {
			log.Printf("failed to download a file: %s", err.Error())
		}
	}

	//endregion

	//region Entrypoint

	entrypoint, err := findEntrypoint(".")
	if err != nil || entrypoint == "" {
		log.Fatalf("failed to find an entrypoint: %s\n", err.Error())
	}

	projectName := getProjectName(entrypoint)

	// Get the PROJECT_DIR basing on the entrypoint as "../../../"
	projectDir := path.Dir(path.Dir(path.Dir(path.Dir(entrypoint)))) + "/"
	// Check if we need to normalize the entrypoint to the PROJECT_DIR
	if strings.Count(entrypoint, "/") > 3 {
		// Check if we need to remove excessive path prefix
		if !strings.HasPrefix(entrypoint, projectName) {
			// Normalize entrypoint to the PROJECT_DIR by removing excessive prefix
			entrypoint = strings.Replace(entrypoint, projectDir, "", 1)
		}
	}

	log.Printf("using entrypoint: %s\n", entrypoint)

	//endregion

	//region Command arguments

	// Set the first command line argument as the project name
	args := []string{"-Log", "-Verbose"}
	// Append additional command line arguments if any of them present
	args = append(args, os.Args[1:]...)

	//endregion

	//region Prepare and run the server command

	cmd := exec.Command(entrypoint, args...)
	cmd.Dir = projectDir // Change the current working directory for the process to the PROJECT_DIR
	cmd.Env = os.Environ()
	rd, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to attach to a command stdout pipe: %s\n", err.Error())
	}

	// Read output from the server process
	go func() {
		b := make([]byte, 2048)
		for {
			nn, err := rd.Read(b)
			if nn > 0 {
				log.Printf("%s", b[:nn])
			}
			if err != nil {
				if err == io.EOF {
					log.Printf("the server process has exited\n")
				} else {
					log.Fatalf("failed to read the server process pipe: %s\n", err.Error())
				}
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start() error: %v\n", err)
	}

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			// This usually means that the server process has crashed
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				log.Fatalf("server exit code: %v\n", status.ExitStatus())
			}
		} else {
			log.Fatalf("server exit error: %v\n", err)
		}
	} else {
		log.Printf("server exited normally\n")
	}

	//endregion
}
