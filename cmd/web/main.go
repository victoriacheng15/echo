package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Header struct {
		ProjectName string `yaml:"project_name"`
		SiteUrl     string `yaml:"site_url"`
	} `yaml:"header"`
	SystemSpecification struct {
		Objective           string `yaml:"objective"`
		Stack               string `yaml:"stack"`
		Pattern             string `yaml:"pattern"`
		EntryPoint          string `yaml:"entry_point"`
		PersistenceStrategy string `yaml:"persistence_strategy"`
		Observability       string `yaml:"observability"`
	} `yaml:"system_specification"`
	Hero struct {
		Headline         string `yaml:"headline"`
		SubHeadline      string `yaml:"sub_headline"`
		BriefDescription string `yaml:"brief_description"`
		CtaText          string `yaml:"cta_text"`
		CtaLink          string `yaml:"cta_link"`
		SecondaryCtaText string `yaml:"secondary_cta_text"`
		SecondaryCtaLink string `yaml:"secondary_cta_link"`
	} `yaml:"hero"`
	WhatIsEcho struct {
		Title   string   `yaml:"title"`
		Content []string `yaml:"content"`
	} `yaml:"what_is_echo"`
	KeyFeatures struct {
		Title    string `yaml:"title"`
		Features []struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Icon        string `yaml:"icon"`
		} `yaml:"features"`
	} `yaml:"key_features"`
	WhyItMatters struct {
		Title  string   `yaml:"title"`
		Points []string `yaml:"points"`
	} `yaml:"why_it_matters"`
	Footer struct {
		Author       string `yaml:"author"`
		GithubLink   string `yaml:"github_link"`
		LinkedinLink string `yaml:"linkedin_link"`
	} `yaml:"footer"`
}

type EvolutionConfig struct {
	Title    string    `yaml:"title"`
	Chapters []Chapter `yaml:"chapters"`
}

type Chapter struct {
	Title    string         `yaml:"title"`
	Intro    string         `yaml:"intro"`
	Timeline []TimelineItem `yaml:"timeline"`
}

type TimelineItem struct {
	Date        string     `yaml:"date"`
	Title       string     `yaml:"title"`
	Description string     `yaml:"description"`
	Points      []string   `json:"points"`
	Artifacts   []Artifact `yaml:"artifacts"`
}

type Artifact struct {
	Name string `yaml:"name"`
	Url  string `yaml:"url"`
}

func main() {
	if err := Run("web", "dist"); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(webDir, distDir string) error {
	// Load landing config
	var appConfig Config
	landingPath := filepath.Join(webDir, "content/landing.yml")
	landingData, err := os.ReadFile(landingPath)
	if err != nil {
		return fmt.Errorf("reading landing config: %w", err)
	}
	if err := yaml.Unmarshal(landingData, &appConfig); err != nil {
		return fmt.Errorf("unmarshalling landing config: %w", err)
	}

	// Load evolution config
	var evolutionConfig EvolutionConfig
	evolutionPath := filepath.Join(webDir, "content/evolution.yml")
	evolutionData, err := os.ReadFile(evolutionPath)
	if err != nil {
		return fmt.Errorf("reading evolution config: %w", err)
	}
	if err := yaml.Unmarshal(evolutionData, &evolutionConfig); err != nil {
		return fmt.Errorf("unmarshalling evolution config: %w", err)
	}

	// Process the data
	processEvolutionData(&evolutionConfig)

	if err := os.MkdirAll(distDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating dist directory: %w", err)
	}

	templateData := struct {
		Config      Config
		Evolution   EvolutionConfig
		CurrentYear int
	}{
		Config:      appConfig,
		Evolution:   evolutionConfig,
		CurrentYear: time.Now().Year(),
	}

	commonFiles := []string{
		filepath.Join(webDir, "base.html"),
		
		
	}

	// Generate index.html
	if err := generateIsolatedPage(webDir, "index.html", distDir, commonFiles, templateData); err != nil {
		return err
	}

	// Generate evolution.html
	if err := generateIsolatedPage(webDir, "evolution.html", distDir, commonFiles, templateData); err != nil {
		return err
	}

	// Generate llms.txt (doesn't use base)
	llmsTmpl, err := template.ParseFiles(filepath.Join(webDir, "llms.txt"))
	if err != nil {
		return fmt.Errorf("parsing llms.txt: %w", err)
	}
	llmsOut, err := os.Create(filepath.Join(distDir, "llms.txt"))
	if err != nil {
		return err
	}
	defer llmsOut.Close()
	if err := llmsTmpl.Execute(llmsOut, templateData); err != nil {
		return err
	}

	// Generate robots.txt
	robotsTmpl, err := template.ParseFiles(filepath.Join(webDir, "robots.txt"))
	if err != nil {
		return fmt.Errorf("parsing robots.txt: %w", err)
	}
	robotsOut, err := os.Create(filepath.Join(distDir, "robots.txt"))
	if err != nil {
		return err
	}
	defer robotsOut.Close()
	if err := robotsTmpl.Execute(robotsOut, templateData); err != nil {
		return err
	}

	// Generate evolution-registry.json under dist/api/
	apiDir := filepath.Join(distDir, "api")
	if err := os.MkdirAll(apiDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating api directory: %w", err)
	}

	registryPath := filepath.Join(apiDir, "evolution-registry.json")
	registryFile, err := os.Create(registryPath)
	if err != nil {
		return fmt.Errorf("creating evolution-registry.json: %w", err)
	}
	defer registryFile.Close()
	
	encoder := json.NewEncoder(registryFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evolutionConfig); err != nil {
		return fmt.Errorf("encoding evolution-registry.json: %w", err)
	}

	log.Printf("Static site and registry generated successfully in %s/", distDir)
	return nil
}

func generateIsolatedPage(webDir, pageName, distDir string, commonFiles []string, data interface{}) error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}

	files := append([]string{filepath.Join(webDir, pageName)}, commonFiles...)
	tmpl, err := template.New(pageName).Funcs(funcMap).ParseFiles(files...)
	if err != nil {
		return fmt.Errorf("parsing templates for %s: %w", pageName, err)
	}

	outputPath := filepath.Join(distDir, pageName)
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute the "pageName" template which calls "base" which uses the blocks
	if err := tmpl.ExecuteTemplate(outputFile, pageName, data); err != nil {
		return fmt.Errorf("executing template %s: %w", pageName, err)
	}
	return nil
}

func processEvolutionData(cfg *EvolutionConfig) {
	// Reverse chapters
	for i, j := 0, len(cfg.Chapters)-1; i < j; i, j = i+1, j-1 {
		cfg.Chapters[i], cfg.Chapters[j] = cfg.Chapters[j], cfg.Chapters[i]
	}

	// Process each chapter
	for i := range cfg.Chapters {
		chapter := &cfg.Chapters[i]
		
		// Reverse timeline
		for i, j := 0, len(chapter.Timeline)-1; i < j; i, j = i+1, j-1 {
			chapter.Timeline[i], chapter.Timeline[j] = chapter.Timeline[j], chapter.Timeline[i]
		}

		// Parse descriptions
		for j := range chapter.Timeline {
			item := &chapter.Timeline[j]
			item.Points = parseDescription(item.Description)
		}
	}
}

func parseDescription(desc string) []string {
	var points []string
	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			trimmed = strings.TrimPrefix(trimmed, "-")
			points = append(points, strings.TrimSpace(trimmed))
		}
	}
	return points
}
