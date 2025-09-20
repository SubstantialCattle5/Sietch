package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/substantialcattle5/sietch/internal/fs"
)

// Template represents a vault template structure
type Template struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Author      string         `json:"author"`
	Tags        []string       `json:"tags"`
	Config      TemplateConfig `json:"config"`
	Directories []string       `json:"directories"`
	Files       []TemplateFile `json:"files"`
}

// TemplateConfig represents default vault configuration in template
type TemplateConfig struct {
	ChunkingStrategy  string `json:"chunking_strategy"`
	ChunkSize         string `json:"chunk_size"`
	HashAlgorithm     string `json:"hash_algorithm"`
	Compression       string `json:"compression"`
	SyncMode          string `json:"sync_mode"`
	EnableDedup       bool   `json:"enable_dedup"`
	DedupStrategy     string `json:"dedup_strategy"`
	DedupMinSize      string `json:"dedup_min_size"`
	DedupMaxSize      string `json:"dedup_max_size"`
	DedupGCThreshold  int    `json:"dedup_gc_threshold"`
	DedupIndexEnabled bool   `json:"dedup_index_enabled"`
	DedupCrossFile    bool   `json:"dedup_cross_file"`
}

// TemplateFile represents a file to be created in the vault
type TemplateFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode"`
}

// GetTemplatesDirectory returns the path to templates directory
// which would be ~/.config/sietch/templates
func GetTemplatesDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "sietch", "templates"), nil
}

// EnsureConfigDirectories ensures all necessary config directories exist
func EnsureConfigDirectories() error {
	templatesDir, err := GetTemplatesDirectory()
	if err != nil {
		return err
	}
	return fs.EnsureDirectory(templatesDir)
}

// ListAvailableTemplates lists all available templates from config directory and built-in templates
func ListAvailableTemplates() ([]string, error) {
	templatesDir, err := GetTemplatesDirectory()
	if err != nil {
		return nil, err
	}

	var templates []string

	// Check if templates directory exists
	if _, err := os.Stat(templatesDir); !os.IsNotExist(err) {
		entries, err := os.ReadDir(templatesDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read templates directory: %v", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
				templateName := entry.Name()[:len(entry.Name())-5] // Remove .json extension
				templates = append(templates, templateName)
			}
		}
	}

	// TODO: Add built-in templates from bundled templates
	builtInTemplates := []string{"photo-vault"}
	for _, template := range builtInTemplates {
		// Only add if not already in user templates
		found := slices.Contains(templates, template)
		if !found {
			templates = append(templates, template)
		}
	}

	return templates, nil
}

// LoadTemplate loads a template from file system
func LoadTemplate(templateName string) (*Template, error) {
	var templatePath string

	// First check user config directory
	templatesDir, err := GetTemplatesDirectory()
	if err != nil {
		return nil, err
	}

	userTemplatePath := filepath.Join(templatesDir, templateName+".json")
	if _, err := os.Stat(userTemplatePath); err == nil {
		templatePath = userTemplatePath
	} else {
		// Check built-in templates (in project template directory for now)
		// In production, this would be embedded or in a system directory
		builtInPath := filepath.Join("template", templateName+".json")
		if _, err := os.Stat(builtInPath); err == nil {
			templatePath = builtInPath
		} else {
			return nil, fmt.Errorf("template '%s' not found in user config or built-in templates", templateName)
		}
	}

	// Read and parse template file
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %v", err)
	}

	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template file: %v", err)
	}

	return &template, nil
}

// ValidateTemplate validates template name and returns the loaded template
func ValidateTemplate(templateName string) (*Template, error) {
	template, err := LoadTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("template validation failed: %v", err)
	}
	return template, nil
}

// CopyDefaultTemplates copies built-in templates to user config directory
func CopyDefaultTemplates() error {
	templatesDir, err := GetTemplatesDirectory()
	if err != nil {
		return err
	}

	// Ensure templates directory exists
	if err := fs.EnsureDirectory(templatesDir); err != nil {
		return err
	}

	// List of built-in templates (in production, these would be embedded)
	builtInTemplates := []string{"photoVault"}

	for _, templateName := range builtInTemplates {
		userTemplatePath := filepath.Join(templatesDir, templateName+".json")

		// Skip if user already has this template
		if _, err := os.Stat(userTemplatePath); err == nil {
			continue
		}

		// Copy from built-in location
		builtInPath := filepath.Join("template", templateName+".json")
		if _, err := os.Stat(builtInPath); err == nil {
			data, err := os.ReadFile(builtInPath)
			if err != nil {
				return fmt.Errorf("failed to read built-in template %s: %v", templateName, err)
			}

			if err := os.WriteFile(userTemplatePath, data, 0644); err != nil {
				return fmt.Errorf("failed to copy template %s to user config: %v", templateName, err)
			}
		}
	}

	return nil
}

// EnsureDefaultTemplates ensures default templates exist in user config
func EnsureDefaultTemplates() error {
	return CopyDefaultTemplates()
}
