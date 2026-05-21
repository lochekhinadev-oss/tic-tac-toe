package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	generalInfo = "main.go"
	outputDir   = "docs"
)

var swagDirs = []string{
	"./cmd/app",
	"./app/application",
	"./app/domain",
	"./infrastructure/auth",
	"./infrastructure/postgres/datasource",
	"./infrastructure/postgres/mapper",
	"./infrastructure/postgres/repository",
	"./internal/di",
	"./internal/transport/http/handler",
	"./internal/transport/http/middleware",
	"./internal/transport/http/response",
	"./internal/transport/http/dto",
}

func main() {
	if _, err := exec.LookPath("swag"); err != nil {
		if docsExist() {
			fmt.Println("swag is not installed; using existing generated docs")
			return
		}

		fmt.Fprintln(os.Stderr, "swag is not installed and generated docs are missing")
		fmt.Fprintln(os.Stderr, "Install swag or generate docs before running this target")
		os.Exit(1)
	}

	cmd := exec.Command("swag", swagArgs()...)
	cmd.Env = append(os.Environ(), "GOCACHE="+cacheDir())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func docsExist() bool {
	paths := []string{
		"docs/docs.go",
		"docs/swagger.json",
		"docs/swagger.yaml",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

func cacheDir() string {
	if value := os.Getenv("GOCACHE"); value != "" {
		return value
	}
	return "/tmp/go-cache"
}

func swagArgs() []string {
	return []string{"init", "--generalInfo", generalInfo, "--output", outputDir, "--parseInternal", "--dir", strings.Join(swagDirs, ",")}
}
