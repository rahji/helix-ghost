package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// createTempFile creates a temp file and populates it with the provided string
func createTempFile(text string) (string, error) {
	tempFile, err := os.CreateTemp("", "*.txt")
	if err != nil {
		return "", err
	}
	if _, err := tempFile.Write([]byte(text)); err != nil {
		return tempFile.Name(), err
	}
	if err := tempFile.Close(); err != nil {
		return tempFile.Name(), err
	}
	return tempFile.Name(), nil
}

// openEditor opens an a specified file in an editor
func openEditor(editor string, fn string) error {
	cmd := exec.Command(editor, fn)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// watchFile watches a temp file for changes
// and sends the file's content into the channel
func watchFile(fn string, c chan<- FileChangeEvent) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Error creating file watcher: %v", err)
		return
	}
	defer watcher.Close()

	dir := filepath.Dir(fn)

	// Add directory (not the specific file) to watcher
	// This is because the file is not *changed* when saved by
	// the editor - it is *replaced*, using a temporary file
	err = watcher.Add(dir)
	if err != nil {
		log.Printf("Error adding %s to watcher: %v", dir, err)
		return
	}

	log.Printf("Watching folder %s for changes to %s", dir, fn)

	for {
		select {
		case event := <-watcher.Events:
			if event.Name != fn {
				break
			}
			log.Printf("File event: %s", event.Name)
			content, err := os.ReadFile(event.Name)
			if err != nil {
				log.Println("Error reading file: ", err)
				continue
			}
			c <- FileChangeEvent{
				Filename: event.Name,
				Content:  content,
			}
		case err, ok := <-watcher.Errors:
			// just in case, I guess
			if !ok {
				return
			}
			log.Println("Watcher error: ", err)
		}
	}
}

// func debugFileAccess(filename string) {
// 	log.Printf("=== Debugging file access for: %q ===", filename)

// 	// Current working directory
// 	wd, _ := os.Getwd()
// 	log.Printf("Working directory: %s", wd)

// 	// Absolute path
// 	abs, _ := filepath.Abs(filename)
// 	log.Printf("Absolute path: %s", abs)

// 	// Check if file exists
// 	if info, err := os.Stat(filename); err != nil {
// 		log.Printf("os.Stat error: %v", err)

// 		// Check directory
// 		dir := filepath.Dir(filename)
// 		if dirInfo, err := os.Stat(dir); err != nil {
// 			log.Printf("Directory %q stat error: %v", dir, err)
// 		} else {
// 			log.Printf("Directory %q exists, mode: %v", dir, dirInfo.Mode())

// 			// List directory contents
// 			if entries, err := os.ReadDir(dir); err != nil {
// 				log.Printf("ReadDir error: %v", err)
// 			} else {
// 				log.Printf("Directory contents:")
// 				for _, entry := range entries {
// 					log.Printf("  %q", entry.Name())
// 				}
// 			}
// 		}
// 	} else {
// 		log.Printf("File exists, size: %d, mode: %v", info.Size(), info.Mode())
// 	}

// 	// Try reading
// 	if content, err := os.ReadFile(filename); err != nil {
// 		log.Printf("os.ReadFile error: %v", err)
// 	} else {
// 		log.Printf("Successfully read %d bytes", len(content))
// 	}
// }
