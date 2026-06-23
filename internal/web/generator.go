package web

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

type HeaderConfig struct {
	ProjectName string `yaml:"project_name"`
	SiteUrl     string `yaml:"site_url"`
}

type LLMSConfig struct {
	Objective           string `yaml:"objective"`
	Stack               string `yaml:"stack"`
	Pattern             string `yaml:"pattern"`
	EntryPoint          string `yaml:"entry_point"`
	PersistenceStrategy string `yaml:"persistence_strategy"`
	Observability       string `yaml:"observability"`
}

type ArchitectureConfig struct {
	DiagramASCII string `yaml:"diagram_ascii"`
}

type TechComponent struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type ProofComponent struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type HumblePivot struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type ObjectiveClarity struct {
	Description string `yaml:"description"`
}

type VerifiableOutput struct {
	Title          string `yaml:"title"`
	TerminalOutput string `yaml:"terminal_output"`
}

type ReachConfig struct {
	HumblePivots      []HumblePivot      `yaml:"humble_pivots"`
	ObjectiveClarity  ObjectiveClarity   `yaml:"objective_clarity"`
	VerifiableOutputs []VerifiableOutput `yaml:"verifiable_outputs"`
}

type FooterConfig struct {
	Author       string `yaml:"author"`
	GithubLink   string `yaml:"github_link"`
	LinkedinLink string `yaml:"linkedin_link"`
}

type Landing struct {
	Header       HeaderConfig       `yaml:"header"`
	LLMS         LLMSConfig         `yaml:"llms"`
	Architecture ArchitectureConfig `yaml:"architecture"`
	Tech         []TechComponent    `yaml:"tech"`
	Proof        []ProofComponent   `yaml:"proof"`
	Reach        ReachConfig        `yaml:"reach"`
	Footer       FooterConfig       `yaml:"footer"`
}

type Evolution struct {
	PageTitle string    `yaml:"page_title"`
	IntroText string    `yaml:"intro_text"`
	Chapters  []Chapter `yaml:"chapters"`
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

// Generate generates the static site and the evolution registry.
func Generate(webDir, distDir string) error {
	// Load landing config
	var landing Landing
	landingPath := filepath.Join(webDir, "templates/content/landing.yml")
	landingData, err := os.ReadFile(landingPath)
	if err != nil {
		return fmt.Errorf("reading landing config: %w", err)
	}
	if err := yaml.Unmarshal(landingData, &landing); err != nil {
		return fmt.Errorf("unmarshalling landing config: %w", err)
	}

	// Load evolution config
	var evolution Evolution
	evolutionPath := filepath.Join(webDir, "templates/content/evolution.yml")
	evolutionData, err := os.ReadFile(evolutionPath)
	if err != nil {
		return fmt.Errorf("reading evolution config: %w", err)
	}
	if err := yaml.Unmarshal(evolutionData, &evolution); err != nil {
		return fmt.Errorf("unmarshalling evolution config: %w", err)
	}

	// Process the data
	processEvolutionData(&evolution)

	if err := os.MkdirAll(distDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating dist directory: %w", err)
	}

	templateData := struct {
		Landing     Landing
		Evolution   Evolution
		CurrentYear int
	}{
		Landing:     landing,
		Evolution:   evolution,
		CurrentYear: time.Now().Year(),
	}

	commonFiles := []string{
		filepath.Join(webDir, "templates/base.html"),
	}

	// Generate index.html
	if err := generateIsolatedPage(webDir, "index.html", distDir, commonFiles, templateData); err != nil {
		return err
	}

	// Generate evolution.html
	if err := generateIsolatedPage(webDir, "evolution.html", distDir, commonFiles, templateData); err != nil {
		return err
	}

	// Generate plain text pages
	if err := generatePlainPage(webDir, "llms.txt", distDir, templateData); err != nil {
		return err
	}
	if err := generatePlainPage(webDir, "robots.txt", distDir, templateData); err != nil {
		return err
	}

	// Generate evolution-registry.json
	if err := generateEvolutionRegistry(distDir, evolution); err != nil {
		return err
	}

	log.Printf("Static site and registry generated successfully in %s/", distDir)
	return nil
}

func generatePlainPage(webDir, pageName, distDir string, data interface{}) error {
	tmplPath := filepath.Join(webDir, "templates", pageName)
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", pageName, err)
	}

	outputPath := filepath.Join(distDir, pageName)
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", pageName, err)
	}
	defer out.Close()

	if err := tmpl.Execute(out, data); err != nil {
		return fmt.Errorf("executing %s: %w", pageName, err)
	}
	return nil
}

func generateEvolutionRegistry(distDir string, evolution Evolution) error {
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
	if err := encoder.Encode(evolution); err != nil {
		return fmt.Errorf("encoding evolution-registry.json: %w", err)
	}
	return nil
}

func generateIsolatedPage(webDir, pageName, distDir string, commonFiles []string, data interface{}) error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}

	files := append([]string{filepath.Join(webDir, "templates", pageName)}, commonFiles...)
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

func processEvolutionData(cfg *Evolution) {
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
