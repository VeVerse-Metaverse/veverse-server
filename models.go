package main

import (
	"github.com/gofrs/uuid"
	goRuntime "runtime"
	"time"
)

// binarySuffixes list of suffixes of known|supported entrypoint binaries
var binarySuffixes = map[string]bool{
	"Server-Debug":     true,
	"Server-DebugGame": true,
	"Server":           true,
	"Server-Test":      true,
	"Server-Shipping":  true,
}

func getBinarySuffix() string {
	if //goland:noinspection GoBoolExpressions
	goRuntime.GOOS == "windows" {
		if pEnvironment == "debug" {
			return "Server-DebugGame.exe"
		} else if pEnvironment == "dev" {
			return "Server.exe"
		} else if pEnvironment == "test" {
			return "Server-Test.exe"
		} else if pEnvironment == "prod" {
			return "Server-Shipping.exe"
		}
		return "Server.exe"
	} else {
		if pEnvironment == "debug" {
			return "Server-DebugGame"
		} else if pEnvironment == "dev" {
			return "Server"
		} else if pEnvironment == "test" {
			return "Server-Test"
		} else if pEnvironment == "prod" {
			return "Server-Shipping"
		}
		return "Server"
	}
}

type Identifier struct {
	Id *uuid.UUID `json:"id,omitempty"`
}

type EntityTrait struct {
	Identifier
	EntityId *uuid.UUID `json:"entityId,omitempty"`
}

type Timestamps struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type File struct {
	EntityTrait

	Type         string     `json:"type"`
	Url          string     `json:"url"`
	Mime         *string    `json:"mime,omitempty"`
	Size         *int64     `json:"size,omitempty"`
	Version      int        `json:"version,omitempty"`        // version of the file if versioned
	Deployment   string     `json:"deploymentType,omitempty"` // server or client if applicable
	Platform     string     `json:"platform,omitempty"`       // platform if applicable
	UploadedBy   *uuid.UUID `json:"uploadedBy,omitempty"`     // user that uploaded the file
	Width        *int       `json:"width,omitempty"`
	Height       *int       `json:"height,omitempty"`
	CreatedAt    time.Time  `json:"createdAt,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	Variation    int        `json:"variation,omitempty"`    // variant of the file if applicable (e.g. PDF pages)
	OriginalPath string     `json:"originalPath,omitempty"` // original relative path to maintain directory structure (e.g. for releases)

	Timestamps
}

type ReleaseMetadata struct {
	Identifier
	AppId       uuid.UUID `json:"appId"`
	AppName     string    `json:"appName"`
	Version     string    `json:"version"`
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Files       []File    `json:"files,omitempty"`
}

type ReleaseMetadataContainer struct {
	ReleaseMetadata `json:"data"`
}
