package helpers

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-getter"
	//"github.com/hashicorp/go-getter/v2"
)

func getWd() string {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return pwd
}

var parent = filepath.Dir(getWd())

// ProgressTracker tracks download progress and reports bytes transferred.
type ProgressTracker struct {
	Total int64
}

// TrackProgress implements the getter.ProgressTracker interface, printing download progress.
func (p *ProgressTracker) TrackProgress(src string, current, total int64, stream io.ReadCloser) io.ReadCloser {
	p.Total = total
	if total > 0 {
		fmt.Printf("\rDownloading: %d/%d bytes (%.2f%%)", current, total, float64(current)/float64(total)*100)
	} else {
		fmt.Printf("\rDownloading: %d bytes", current)
	}
	return stream
}

// DownloadLibs downloads the standard library files from the fo-lang repository.
func DownloadLibs() error {
	var err error = nil
	dst := filepath.Join(parent, "lib")
	os.MkdirAll(dst, 0755)
	errors := false
	// Git repository URL (supports various formats)
	src := "github.com/samkrao/fo-lang/frontend/frontend.git//lib"
	// Alternative formats:
	// src := "git::https://github.com/hashicorp/go-getter.git//examples"
	// src := "git@github.com:hashicorp/go-getter.git//examples"

	// Cleanup in case of failure
	defer func() {
		fmt.Println((errors))
		if errors {
			if err = os.RemoveAll(dst); err != nil {
				fmt.Println("Cleanup error: %v", err)
			}
		}
	}()

	// Create the client
	client := &getter.Client{
		Src:  src,
		Dst:  dst,
		Mode: getter.ClientModeDir, // Download as directory
		Options: []getter.ClientOption{
			getter.WithProgress(&ProgressTracker{}),
		},
	}

	// Run in a goroutine to allow cancellation
	done := make(chan struct{})
	go func() {
		defer close(done)
		err = client.Get()
		if err != nil {
			log.Printf("Download failed: %v", err)
			errors = true
		}
	}()
	select {
	case <-time.After(5 * time.Minute):
		done <- struct{}{}
		fmt.Println("Download cancelled after 5 Minutes")
	case <-done:
		fmt.Println("Download completed successfully")
	}

	return err
}
