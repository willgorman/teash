package labels

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const testYAML = `cluster-a:
  server1.example.com:
    environment: production
    role: web
  server2.example.com:
    environment: staging
cluster-b:
  server3.example.com:
    role: database
`

const testTOML = `[cluster-a]
[cluster-a."server1.example.com"]
environment = "production"
role = "web"

[cluster-a."server2.example.com"]
environment = "staging"

[cluster-b]
[cluster-b."server3.example.com"]
role = "database"
`

func TestImportFromFile_YAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "labels.yaml")
	if err := os.WriteFile(path, []byte(testYAML), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	result, err := ImportFromFile(path)
	if err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	if result["cluster-a"]["server1.example.com"]["environment"] != "production" {
		t.Error("expected production environment for server1")
	}
	if result["cluster-a"]["server1.example.com"]["role"] != "web" {
		t.Error("expected web role for server1")
	}
	if result["cluster-b"]["server3.example.com"]["role"] != "database" {
		t.Error("expected database role for server3")
	}
}

func TestImportFromFile_TOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "labels.toml")
	if err := os.WriteFile(path, []byte(testTOML), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	result, err := ImportFromFile(path)
	if err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	if result["cluster-a"]["server1.example.com"]["environment"] != "production" {
		t.Error("expected production environment for server1")
	}
	if result["cluster-b"]["server3.example.com"]["role"] != "database" {
		t.Error("expected database role for server3")
	}
}

func TestImportFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		w.Write([]byte(testYAML))
	}))
	defer server.Close()

	result, err := ImportFromURL(server.URL)
	if err != nil {
		t.Fatalf("ImportFromURL: %v", err)
	}

	if result["cluster-a"]["server1.example.com"]["environment"] != "production" {
		t.Error("expected production environment for server1")
	}
}

func TestImportFromURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := ImportFromURL(server.URL)
	if err == nil {
		t.Error("expected error for HTTP 404")
	}
}

func TestExportToYAML_Roundtrip(t *testing.T) {
	original := ImportFile{
		"cluster-a": {
			"server1": {"env": "production", "role": "web"},
		},
	}

	var buf bytes.Buffer
	if err := ExportToYAML(&buf, original); err != nil {
		t.Fatalf("ExportToYAML: %v", err)
	}

	parsed, err := parseImportData(buf.Bytes())
	if err != nil {
		t.Fatalf("parsing exported YAML: %v", err)
	}

	if parsed["cluster-a"]["server1"]["env"] != "production" {
		t.Error("roundtrip mismatch for env")
	}
	if parsed["cluster-a"]["server1"]["role"] != "web" {
		t.Error("roundtrip mismatch for role")
	}
}

func TestExportToTOML_Roundtrip(t *testing.T) {
	original := ImportFile{
		"cluster-a": {
			"server1": {"env": "production", "role": "web"},
		},
	}

	var buf bytes.Buffer
	if err := ExportToTOML(&buf, original); err != nil {
		t.Fatalf("ExportToTOML: %v", err)
	}

	parsed, err := parseImportData(buf.Bytes())
	if err != nil {
		t.Fatalf("parsing exported TOML: %v", err)
	}

	if parsed["cluster-a"]["server1"]["env"] != "production" {
		t.Error("roundtrip mismatch for env")
	}
	if parsed["cluster-a"]["server1"]["role"] != "web" {
		t.Error("roundtrip mismatch for role")
	}
}

func TestImportFromFile_NotFound(t *testing.T) {
	_, err := ImportFromFile("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestImportFromFile_InvalidFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.txt")
	if err := os.WriteFile(path, []byte("this is not yaml or toml ][}{"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	_, err := ImportFromFile(path)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}
