package scaffold

import (
	"fmt"
)

func ListTemplates() error {
	// Ensure config directories exist
	if err := EnsureConfigDirectories(); err != nil {
		return fmt.Errorf("failed to ensure config directories: %v", err)
	}

	// Ensure default templates are available
	if err := EnsureDefaultTemplates(); err != nil {
		return fmt.Errorf("failed to ensure default templates: %v", err)
	}

	// Get available templates
	templates, err := ListAvailableTemplates()
	if err != nil {
		return fmt.Errorf("failed to list templates: %v", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates available.")
		return nil
	}

	fmt.Println("Available templates:")
	for _, templateName := range templates {
		// Try to load template for details
		if template, err := LoadTemplate(templateName); err == nil {
			fmt.Printf("  %s - %s (v%s)\n", templateName, template.Description, template.Version)
			if len(template.Tags) > 0 {
				fmt.Printf("    Tags: %v\n", template.Tags)
			}
		} else {
			fmt.Printf("  %s\n", templateName)
		}
	}

	templatesDir, _ := GetTemplatesDirectory()
	fmt.Printf("\nTemplates are stored in: %s\n", templatesDir)
	fmt.Println("You can edit templates or add new ones in this directory.")

	return nil
}
