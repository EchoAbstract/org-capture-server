package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func makeHandlerWithLog(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		fn(w, r)
		ellapsed := float64(time.Since(t)) * 1e-6
		log.Printf("%s %s complete in %0.2g ms", r.Method, r.URL.Path, ellapsed)

	}
}

func appendToFile(file string, contents []byte) (int, error) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatal("appendToFile: ", err)
	}

	defer f.Close()

	bytesWritten, err := f.Write(contents)
	if err != nil {
		log.Fatal("appendToFile: ", err)
	}

	return bytesWritten, err
}

type OrgEntry struct {
	Directory string
	File      string
	Entry     []byte
}

func (oe *OrgEntry) save() error {
	filename := oe.Directory + "/" + oe.File
	bytesWritten, err := appendToFile(filename, oe.Entry)

	if err != nil {
		return err
	} else {
		log.Printf("Added entry to %s (%d bytes written)", filename, bytesWritten)
		return nil
	}
}

type OrgConfig struct {
	OrgDir             string
	DefaultCaptureFile string
}

func makeOrgHandler(config *OrgConfig, fn func(http.ResponseWriter, *http.Request, *OrgConfig)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, config)
	}
}

type Capture struct {
	Tag         string `json:"tag,omitempty"`
	Headline    string `json:"headline,omitempty"`
	Body        string `json:"body,omitempty"`
	TagSpecific string `json:"tagData,omitempty"`
}

func dispatchOnTag(config *OrgConfig, capture Capture) error {
	contents := "** TODO " + capture.Headline + "\n" + capture.Body + "\n\n"
	entry := []byte(contents)
	oe := &OrgEntry{Directory: config.OrgDir, File: config.DefaultCaptureFile, Entry: entry}

	return oe.save()
}

func captureHandler(w http.ResponseWriter, req *http.Request, config *OrgConfig) {
	var capture Capture
	_ = json.NewDecoder(req.Body).Decode(&capture)

	if err := dispatchOnTag(config, capture); err != nil {
		w.WriteHeader(http.StatusInternalServerError)

	}

	w.WriteHeader(http.StatusCreated)
}

func getenvOrDefault(env string, defaultString string) string {
	value := os.Getenv(env)

	if len(value) == 0 {
		value = defaultString
	}

	return value
}

func main() {
	bindAddr := getenvOrDefault("HOST", "")
	bindPort := getenvOrDefault("PORT", "8080")
	homeDir := getenvOrDefault("HOME", "/tmp")
	orgDir := getenvOrDefault("ORG_DIR", homeDir+"/org")
	defaultCaptureFile := getenvOrDefault("ORG_DEFAULT_CAPTURE_FILE", "web-captures.org")

	orgConfig := &OrgConfig{OrgDir: orgDir, DefaultCaptureFile: defaultCaptureFile}

	http.HandleFunc("/api/v0/capture", makeHandlerWithLog(makeOrgHandler(orgConfig, captureHandler)))

	addr := bindAddr + ":" + bindPort

	log.Printf("Starting org-capture API server on %s", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}
