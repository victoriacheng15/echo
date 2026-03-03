package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single point with hyphen",
			input:    "- Point 1",
			expected: []string{"Point 1"},
		},
		{
			name: "Multiple points with hyphens and newlines",
			input: `- Point 1
- Point 2
  - Point 3`,
			expected: []string{"Point 1", "Point 2", "Point 3"},
		},
		{
			name: "Points without hyphens",
			input: `Point 1
Point 2`,
			expected: []string{"Point 1", "Point 2"},
		},
		{
			name: "Empty lines and spaces",
			input: `
  - Point 1  

- Point 2
`,
			expected: []string{"Point 1", "Point 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDescription(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseDescription() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProcessEvolutionData(t *testing.T) {
	input := &Evolution{
		Chapters: []Chapter{
			{
				Title: "Chapter 1",
				Timeline: []TimelineItem{
					{Title: "Event 1.1", Description: "- Point A"},
					{Title: "Event 1.2", Description: "- Point B"},
				},
			},
			{
				Title: "Chapter 2",
				Timeline: []TimelineItem{
					{Title: "Event 2.1", Description: "- Point C"},
				},
			},
		},
	}

	processEvolutionData(input)

	// Chapters should be reversed
	if input.Chapters[0].Title != "Chapter 2" {
		t.Errorf("Expected first chapter to be Chapter 2, got %s", input.Chapters[0].Title)
	}
	if input.Chapters[1].Title != "Chapter 1" {
		t.Errorf("Expected second chapter to be Chapter 1, got %s", input.Chapters[1].Title)
	}

	// Timeline in Chapter 1 (now index 1) should be reversed
	if input.Chapters[1].Timeline[0].Title != "Event 1.2" {
		t.Errorf("Expected first event in Chapter 1 to be Event 1.2, got %s", input.Chapters[1].Timeline[0].Title)
	}

	// Points should be parsed
	expectedPoints := []string{"Point C"}
	if !reflect.DeepEqual(input.Chapters[0].Timeline[0].Points, expectedPoints) {
		t.Errorf("Points not parsed correctly for Chapter 2: got %v, want %v", input.Chapters[0].Timeline[0].Points, expectedPoints)
	}
}

func TestRun(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "echo-web-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	webDir := filepath.Join(tempDir, "web")
	distDir := filepath.Join(tempDir, "dist")
	contentDir := filepath.Join(webDir, "content")

	// Create directory structure
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create dummy YAML files
	landingYAML := `
header:
  project_name: "Test Project"
hero:
  headline: "Headline"
  brief_description: "Description"
`
	if err := os.WriteFile(filepath.Join(contentDir, "landing.yml"), []byte(landingYAML), 0644); err != nil {
		t.Fatalf("Failed to write landing.yml: %v", err)
	}

	evolutionYAML := `
page_title: "Evolution"
intro_text: "Intro"
chapters:
  - title: "Chapter 1"
    timeline:
      - date: "2026-01-01"
        title: "Event"
        description: "Desc"
`
	if err := os.WriteFile(filepath.Join(contentDir, "evolution.yml"), []byte(evolutionYAML), 0644); err != nil {
		t.Fatalf("Failed to write evolution.yml: %v", err)
	}

	// Create dummy templates
	templates := map[string]string{
		"base.html":      `{{ define "base" }}{{ block "header" . }}{{ end }}{{ block "content" . }}{{ end }}{{ end }}`,
		"index.html":     `{{ define "index.html" }}{{ template "base" . }}{{ end }}`,
		"evolution.html": `{{ define "evolution.html" }}{{ template "base" . }}{{ end }}`,
		"llms.txt":       `{{ .Landing.Header.ProjectName }}`,
		"robots.txt":     `Allow: /`,
	}

	for name, content := range templates {
		if err := os.WriteFile(filepath.Join(webDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write template %s: %v", name, err)
		}
	}

	// Run the generator
	if err := Run(webDir, distDir); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify output files exist
	expectedFiles := []string{
		"index.html",
		"evolution.html",
		"llms.txt",
		"robots.txt",
		"api/evolution-registry.json",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(distDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", name)
		}
	}
}

func TestRunErrors(t *testing.T) {
	t.Run("MissingWebDir", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "echo-web-err-*")
		defer os.RemoveAll(tempDir)
		err := Run("non-existent-dir", filepath.Join(tempDir, "dist"))
		if err == nil {
			t.Error("Expected error for missing web dir, got nil")
		}
	})

	t.Run("InvalidLandingYAML", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "echo-web-err-*")
		defer os.RemoveAll(tempDir)

		contentDir := filepath.Join(tempDir, "content")
		os.MkdirAll(contentDir, 0755)
		os.WriteFile(filepath.Join(contentDir, "landing.yml"), []byte("invalid: yaml: :"), 0644)

		err := Run(tempDir, filepath.Join(tempDir, "dist"))
		if err == nil {
			t.Error("Expected error for invalid landing.yml, got nil")
		}
	})

	t.Run("NoTemplates", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "echo-web-err-*")
		defer os.RemoveAll(tempDir)

		contentDir := filepath.Join(tempDir, "content")
		os.MkdirAll(contentDir, 0755)
		os.WriteFile(filepath.Join(contentDir, "landing.yml"), []byte("header: {project_name: test}"), 0644)
		os.WriteFile(filepath.Join(contentDir, "evolution.yml"), []byte("title: test"), 0644)

		err := Run(tempDir, filepath.Join(tempDir, "dist"))
		if err == nil {
			t.Error("Expected error for no templates, got nil")
		}
	})

	t.Run("InvalidTemplate", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "echo-web-err-*")
		defer os.RemoveAll(tempDir)

		contentDir := filepath.Join(tempDir, "content")
		os.MkdirAll(contentDir, 0755)
		os.WriteFile(filepath.Join(contentDir, "landing.yml"), []byte("header: {project_name: test}"), 0644)
		os.WriteFile(filepath.Join(contentDir, "evolution.yml"), []byte("title: test"), 0644)

		// Create a malformed template
		os.WriteFile(filepath.Join(tempDir, "bad.html"), []byte("{{ .Invalid field }}"), 0644)

		err := Run(tempDir, filepath.Join(tempDir, "dist"))
		if err == nil {
			t.Error("Expected error for invalid template, got nil")
		}
	})
}
