package labels

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ImportFile maps cluster name → hostname → label key → label value.
type ImportFile = map[string]ServerLabels

// ImportFromFile reads a YAML or TOML file and returns the parsed label data.
// The file format is detected by attempting YAML first, then TOML.
func ImportFromFile(path string) (ImportFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading import file: %w", err)
	}
	return parseImportData(data)
}

// ImportFromURL fetches a URL and parses the content as YAML or TOML.
func ImportFromURL(url string) (ImportFile, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}

	return parseImportData(data)
}

func parseImportData(data []byte) (ImportFile, error) {
	// Try YAML first
	var result ImportFile
	if err := yaml.Unmarshal(data, &result); err == nil && len(result) > 0 {
		return result, nil
	}

	// Try TOML
	if _, err := toml.Decode(string(data), &result); err == nil && len(result) > 0 {
		return result, nil
	}

	return nil, fmt.Errorf("unable to parse as YAML or TOML")
}

// ExportToYAML writes label data in YAML format to the given writer.
func ExportToYAML(w io.Writer, data ImportFile) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding YAML: %w", err)
	}
	return enc.Close()
}

// ExportToTOML writes label data in TOML format to the given writer.
func ExportToTOML(w io.Writer, data ImportFile) error {
	enc := toml.NewEncoder(w)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding TOML: %w", err)
	}
	return nil
}
