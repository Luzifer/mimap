package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Luzifer/rconfig/v2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	cfg = struct {
		Listen         string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func init() {
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("mimap %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/map.png", mapHandler)

	log.WithError(http.ListenAndServe(cfg.Listen, nil)).Fatal("HTTP server quit")
}

func uploadHandler(res http.ResponseWriter, r *http.Request) {
	if err := storeFile(r, "map", "/data/navmap.ppm"); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := storeFile(r, "slam", "/data/slam.log"); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	go processFile()

	http.Error(res, "Received your files, generating map...", http.StatusCreated)
}

func mapHandler(res http.ResponseWriter, r *http.Request) {
	res.Header().Set("Cache-Control", "private, max-age=60")
	http.ServeFile(res, r, "/data/map.png")
}

func storeFile(r *http.Request, field, outname string) error {
	f, _, err := r.FormFile(field)
	if err != nil {
		return errors.Wrapf(err, "Unable to retrieve %q file", field)
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, f); err != nil {
		return errors.Wrapf(err, "Unable to read %q file", field)
	}

	if hash := fmt.Sprintf("%x", sha1.Sum(buf.Bytes())); hash != r.FormValue("sum_"+field) {
		return fmt.Errorf("File hash for %q did not match: %q != %q", field, hash, r.FormValue("sum_"+field))
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(outname), ".mimap-*")
	if err != nil {
		return fmt.Errorf("creating tempfile: %w", err)
	}
	// final cleanup, we don't really care about whether it succeeds
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err = io.Copy(tmpFile, buf); err != nil {
		return errors.Wrapf(err, "Unable to copy %q file", field)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("closing tempfile: %w", err)
	}

	if err = os.Rename(tmpFile.Name(), outname); err != nil {
		return fmt.Errorf("moving tempfile into place: %w", err)
	}

	return nil
}

func processFile() {
	log.Info("Generating map...")

	cmd := exec.Command("python", "/src/build_map.py", "-slam", "/data/slam.log", "-map", "/data/navmap.ppm", "-out", "/data/map.png")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.WithError(err).Error("Map generation failed")
		return
	}

	log.Info("Map generated")
}
