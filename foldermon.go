// Folder Monitor
// By Orochi, 2025 Saudi Arabia
//
// Dependencies
// - fsnotify
// - archive/zip
// - log
// - os
// - path/filepath
// - time

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	watchFolder  string
	backupFolder string
)

const (
	deleteAfterZip = false // Set to true to delete files after zipping
	logFilePath    = "foldermon.log"
)

// ------------------------------------------------------------------------------------------------------------
// Main function.
func main() {
	// Setup logging
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.Println("Starting folder monitor...")

	// Get folders from command line arguments.
	watchFolder, backupFolder, err := getFoldersFromArgs()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Watching folder: %s\n", watchFolder)
	fmt.Printf("Backup folder: %s\n", backupFolder)

	// Ensure backup folder exists
	os.MkdirAll(backupFolder, os.ModePerm)

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(watchFolder)
	if err != nil {
		log.Fatal(err)
	}

	// Monitor loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("Detected new file: %s\n", event.Name)
				time.Sleep(1 * time.Second) // Wait to ensure file is completely written

				// Call the zipAndMove function
				if err := zipAndMove(watchFolder, backupFolder); err != nil {
					fmt.Println("Error during zip and move:", err)
					os.Exit(1)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Watcher error:", err)
		}
	}
}

// ------------------------------------------------------------------------------------------------------------
// Zip the contents of the watch folder into a zip file and move it to the backup folder.
func zipAndMove(watchFolder, backupFolder string) error {
	timestamp := time.Now().Format("20060102_150405")
	zipFileName := fmt.Sprintf("backup_%s.zip", timestamp)
	zipFilePath := filepath.Join(backupFolder, zipFileName)

	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		log.Println("Failed to create zip:", err)
		return err
	}
	defer zipFile.Close()

	fmt.Printf("Zip file path: %s\n", zipFilePath)

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through files in the watch folder
	err = filepath.Walk(watchFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(watchFolder, path)
		if err != nil {
			return err
		}

		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		fileToZip, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		_, err = io.Copy(zipEntry, fileToZip)
		if err != nil {
			return err
		}

		log.Printf("Added to zip: %s\n", path)
		return nil
	})

	if err != nil {
		log.Println("Error creating zip archive:", err)
		return err
	}

	// Move zip to backup folder
	destPath := filepath.Join(backupFolder, zipFileName)
	err = os.Rename(zipFilePath, destPath)
	if err != nil {
		log.Println("Failed to move zip file:", err)
		return err
	}
	log.Printf("Moved zip to: %s\n", destPath)

	// Delete files if required
	if deleteAfterZip {
		err = filepath.Walk(watchFolder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				err = os.Remove(path)
				if err != nil {
					return err
				}
				log.Printf("Deleted: %s\n", path)
			}
			return nil
		})

		if err != nil {
			log.Println("Error deleting files:", err)
		}
	}
	return nil
}

// ------------------------------------------------------------------------------------------------------------
// getFoldersFromArgs retrieves the watchFolder and backupFolder from the command line arguments.
// It returns an error if the correct number of arguments are not provided.
func getFoldersFromArgs() (string, string, error) {
	if len(os.Args) != 3 {
		return "", "", fmt.Errorf("usage: %s <watchFolder> <backupFolder>", os.Args[0])
	}
	watchFolder = os.Args[1]
	backupFolder := os.Args[2]
	return watchFolder, backupFolder, nil
}
