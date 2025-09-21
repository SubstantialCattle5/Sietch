package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

// ListAvailableTemplates lists all available templates from config directory
// This function assumes EnsureDefaultTemplates() has been called first
func ListAvailableTemplates() ([]string, error) {
	templatesDir, err := GetTemplatesDirectory()
	if err != nil {
		return nil, err
	}

	var templates []string

	// Read templates from user config directory
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

	return templates, nil
}

func GetBuiltInTemplates() []string {
	// list all files in template directory
	templateDir := "template"
	files, err := os.ReadDir(templateDir)
	if err != nil {
		// Fallback to hardcoded list if template directory doesn't exist
		return []string{}
	}

	var templates []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			templateName := file.Name()[:len(file.Name())-5] // Remove .json extension
			templates = append(templates, templateName)
		}
	}

	return templates
}

// LoadTemplate loads a template from user config directory
// This function assumes EnsureDefaultTemplates() has been called first
func LoadTemplate(templateName string) (*Template, error) {
	// Load template from user config directory
	templatesDir, err := GetTemplatesDirectory()
	if err != nil {
		return nil, err
	}

	templatePath := filepath.Join(templatesDir, templateName+".json")
	if _, err := os.Stat(templatePath); err != nil {
		return nil, fmt.Errorf("template '%s' not found in user config directory (%s)", templateName, templatesDir)
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

	builtInTemplates := GetBuiltInTemplates()

	for _, templateName := range builtInTemplates {
		userTemplatePath := filepath.Join(templatesDir, templateName+".json")

		// Copy from built-in location (always copy when this function is called)
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

	// Copy README.md file if it exists
	readmeSourcePath := filepath.Join("template", "README.md")
	if _, err := os.Stat(readmeSourcePath); err == nil {
		readmeData, err := os.ReadFile(readmeSourcePath)
		if err != nil {
			return fmt.Errorf("failed to read README.md: %v", err)
		}

		readmeDestPath := filepath.Join(templatesDir, "README.md")
		if err := os.WriteFile(readmeDestPath, readmeData, 0644); err != nil {
			return fmt.Errorf("failed to copy README.md to user config: %v", err)
		}
	}

	return nil
}

// EnsureDefaultTemplates ensures default templates exist in user config
// If the templates directory is empty or doesn't exist, copies all templates from /template
func EnsureDefaultTemplates() error {
	templatesDir, err := GetTemplatesDirectory()
	if err != nil {
		return err
	}

	// Check if templates directory exists and has templates
	hasTemplates, err := hasExistingTemplates(templatesDir)
	if err != nil {
		return err
	}

	// If no templates exist, copy all default templates
	if !hasTemplates {
		return CopyDefaultTemplates()
	}

	return nil
}

// hasExistingTemplates checks if the templates directory exists and contains .json files
func hasExistingTemplates(templatesDir string) (bool, error) {
	// Check if directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return false, nil
	}

	// Check if directory contains any .json files
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return false, fmt.Errorf("failed to read templates directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			return true, nil
		}
	}

	return false, nil
}
