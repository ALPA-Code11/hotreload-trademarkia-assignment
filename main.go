package main

import (
	"context" // Added this to fix your error
	"flag"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Runner struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	buildCmd   string
	runCmd     string
	cancelFunc context.CancelFunc // Used to cancel a build in progress
}

func main() {
	root := flag.String("root", ".", "folder to watch")
	buildCmd := flag.String("build", "", "build command")
	execCmd := flag.String("exec", "", "run command")
	flag.Parse()

	if *buildCmd == "" || *execCmd == "" {
		slog.Error("Missing --build or --exec flags")
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Watcher error", "err", err)
		os.Exit(1)
	}
	defer watcher.Close()

	r := &Runner{
		buildCmd: *buildCmd,
		runCmd:   *execCmd,
	}

	// Recursive watch
	addWatches := func(target string) {
		_ = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return err
			}
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "bin" {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		})
	}
	addWatches(*root)

	trigger := make(chan bool, 1)
	trigger <- true // Initial start

	go func() {
		var timer *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					if event.Has(fsnotify.Create) {
						if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
							addWatches(event.Name)
						}
					}
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(300*time.Millisecond, func() {
						select {
						case trigger <- true:
						default:
						}
					})
				}
			case err := <-watcher.Errors:
				slog.Error("Watcher error", "err", err)
			}
		}
	}()

	slog.Info("Hotreload watching...", "root", *root)

	for range trigger {
		slog.Info("--- Restarting ---")
		if err := r.restart(); err != nil {
			slog.Error("Process error", "err", err)
		}
	}
}

func (r *Runner) restart() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. Cancel previous build if it's still running
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	// 2. Kill old server process (using our helper)
	if r.cmd != nil {
		slog.Info("Stopping old server...")
		killProcessGroup(r.cmd)
		_ = r.cmd.Wait()
	}

	// 3. Setup Context for the Build
	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFunc = cancel

	// 4. Run Build
	slog.Info("Building...", "cmd", r.buildCmd)
	bParts := strings.Fields(r.buildCmd)
	build := exec.CommandContext(ctx, bParts[0], bParts[1:]...)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return err
	}

	// 5. Run Server
	slog.Info("Starting server...", "cmd", r.runCmd)
	rParts := strings.Fields(r.runCmd)
	r.cmd = exec.Command(rParts[0], rParts[1:]...)
	r.cmd.Stdout = os.Stdout
	r.cmd.Stderr = os.Stderr

	// Set process group (using our helper)
	setProcessGroup(r.cmd)

	return r.cmd.Start()
}
