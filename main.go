// MiMAP Image Generation from Xiaomi Mi RoboRock Mapdata
package main

import (
	"bytes"
	"crypto/sha1" //#nosec:G505 // Only used for file integrity
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Luzifer/rconfig/v2"
	"github.com/sirupsen/logrus"

	"github.com/Luzifer/mimap/pkg/mapbuilder"
)

var (
	cfg = struct {
		Listen         string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func initApp() (err error) {
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		return fmt.Errorf("parsing CLI options: %w", err)
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("parsing log-level: %w", err)
	}
	logrus.SetLevel(l)

	return nil
}

func main() {
	var err error
	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		fmt.Printf("mimap %s\n", version) //nolint:forbidigo // Fine to print the version
		os.Exit(0)
	}

	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/map.png", mapHandler)

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: time.Second,
	}

	if err = server.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("running HTTP server")
	}
}

func readRequestFile(r *http.Request, field string) ([]byte, error) {
	expectedHash := r.FormValue(fmt.Sprintf("sum_%s", field))

	f, _, err := r.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("getting form-file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing request file")
		}
	}()

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading file contents: %w", err)
	}

	//#nosec:G401 // only used for file transfer integrity
	if hash := fmt.Sprintf("%x", sha1.Sum(raw)); hash != expectedHash {
		return nil, fmt.Errorf("content hash mismatch: %s != %s", hash, expectedHash)
	}

	return raw, nil
}

func mapHandler(res http.ResponseWriter, r *http.Request) {
	res.Header().Set("Cache-Control", "private, max-age=60")
	http.ServeFile(res, r, "/data/map.png")
}

func processFile(navmap, slamlog []byte) {
	logrus.Debug("starting map generation")

	mappng, err := mapbuilder.Build(slamlog, navmap)
	if err != nil {
		logrus.WithError(err).Error("generating map")
		return
	}

	if err = storeFile("/data/map.png", mappng); err != nil {
		logrus.WithError(err).Error("storing map to disk")
		return
	}

	logrus.Debug("map updated")
}

func storeFile(outname string, content []byte) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(outname), ".mimap-*")
	if err != nil {
		return fmt.Errorf("creating tempfile: %w", err)
	}
	// final cleanup, we don't really care about whether it succeeds
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err = io.Copy(tmpFile, bytes.NewReader(content)); err != nil {
		return fmt.Errorf("copying file content to disk: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("closing tempfile: %w", err)
	}

	if err = os.Rename(tmpFile.Name(), outname); err != nil {
		return fmt.Errorf("moving tempfile into place: %w", err)
	}

	return nil
}

func uploadHandler(res http.ResponseWriter, r *http.Request) {
	var (
		err             error
		navmap, slamlog []byte
	)

	if navmap, err = readRequestFile(r, "map"); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if slamlog, err = readRequestFile(r, "slam"); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	go processFile(navmap, slamlog)

	http.Error(res, "Received your files, generating map...", http.StatusCreated)
}
