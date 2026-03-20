package cli

import (
	"echo/internal/service"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Dispatcher coordinates CLI subcommand routing and service injection.
type Dispatcher struct {
	MemorySvc    *service.MemoryService
	AnalyticsSvc *service.AnalyticsService
	RateSvc      *service.RateService
	OutputFormat string
}

// NewDispatcher initializes a new CLI dispatcher.
func NewDispatcher(mem *service.MemoryService, analytics *service.AnalyticsService, rate *service.RateService, format string) *Dispatcher {
	return &Dispatcher{
		MemorySvc:    mem,
		AnalyticsSvc: analytics,
		RateSvc:      rate,
		OutputFormat: format,
	}
}

// Run parses the subcommand and delegates to the appropriate handler.
func (d *Dispatcher) Run(args []string) error {
	if len(args) < 1 {
		d.Usage()
		return fmt.Errorf("no command provided")
	}

	command := args[0]
	subArgs := args[1:]

	switch command {
	case "store":
		return d.handleStore(subArgs)
	case "recall":
		return d.handleRecall(subArgs)
	case "search":
		return d.handleSearch(subArgs)
	case "delete":
		return d.handleDelete(subArgs)
	case "maintain":
		return d.handleMaintain(subArgs)
	case "help":
		d.Usage()
		return nil
	default:
		d.Usage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// Usage prints the CLI help documentation.
func (d *Dispatcher) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: echo [global flags] <command> [subcommand flags]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  store     Save a new memory with specific tags and context\n")
	fmt.Fprintf(os.Stderr, "  recall    Retrieve memories by context and importance\n")
	fmt.Fprintf(os.Stderr, "  search    Perform a keyword search using FTS5\n")
	fmt.Fprintf(os.Stderr, "  delete    Remove a memory by ID\n")
	fmt.Fprintf(os.Stderr, "  maintain  Perform database maintenance and analytics sync\n")
	fmt.Fprintf(os.Stderr, "  help      Show this help message\n")
}

// Subcommand Handlers

func (d *Dispatcher) handleStore(args []string) error {
	fs := flag.NewFlagSet("store", flag.ExitOnError)
	content := fs.String("content", "", "The memory content (required)")
	context := fs.String("context", "global", "Context key (e.g., project:echo)")
	entryType := fs.String("type", "fact", "Entry type (fact, artifact, directive)")
	tagsStr := fs.String("tags", "", "Comma-separated list of tags")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *content == "" {
		return fmt.Errorf("content flag is required")
	}

	var tags []string
	if *tagsStr != "" {
		tags = strings.Split(*tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	d.MemorySvc.Source = "cli"
	if err := d.MemorySvc.StoreMemoryWithTags(*content, *context, *entryType, tags); err != nil {
		return fmt.Errorf("failed to store memory: %w", err)
	}

	fmt.Printf("Memory stored successfully in context %q.\n", *context)
	return nil
}

func (d *Dispatcher) handleRecall(args []string) error {
	fs := flag.NewFlagSet("recall", flag.ExitOnError)
	contextsStr := fs.String("contexts", "global", "Comma-separated context keys")
	limit := fs.Int("limit", 10, "Maximum number of memories to retrieve")

	if err := fs.Parse(args); err != nil {
		return err
	}

	contexts := strings.Split(*contextsStr, ",")
	for i := range contexts {
		contexts[i] = strings.TrimSpace(contexts[i])
	}

	d.MemorySvc.Source = "cli"
	memories, err := d.MemorySvc.RecallMemory(contexts, *limit)
	if err != nil {
		return fmt.Errorf("failed to recall memories: %w", err)
	}

	return d.formatOutput(memories)
}

func (d *Dispatcher) handleSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	query := fs.String("query", "", "Search query (required)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *query == "" {
		return fmt.Errorf("query flag is required")
	}

	d.MemorySvc.Source = "cli"
	memories, err := d.MemorySvc.SearchMemories(*query)
	if err != nil {
		return fmt.Errorf("failed to search memories: %w", err)
	}

	return d.formatOutput(memories)
}

func (d *Dispatcher) handleDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	id := fs.Int64("id", 0, "Memory ID to delete")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *id == 0 {
		return fmt.Errorf("id flag is required and must be non-zero")
	}

	if err := d.MemorySvc.DeleteMemoryByID(*id); err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	fmt.Printf("Memory %d deleted successfully.\n", *id)
	return nil
}

func (d *Dispatcher) handleMaintain(args []string) error {
	fs := flag.NewFlagSet("maintain", flag.ExitOnError)
	sync := fs.Bool("sync", false, "Synchronize event logs to DuckDB")
	rebuild := fs.Bool("rebuild", false, "Rebuild FTS5 search index")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if !*sync && !*rebuild {
		return fmt.Errorf("at least one flag (--sync or --rebuild) must be provided")
	}

	if *sync {
		if d.AnalyticsSvc == nil {
			return fmt.Errorf("analytics service is not initialized")
		}
		if err := d.AnalyticsSvc.SyncEvents(); err != nil {
			return fmt.Errorf("failed to sync events: %w", err)
		}
		fmt.Println("Analytics sync complete.")
	}

	if *rebuild {
		if err := d.MemorySvc.RebuildIndex(); err != nil {
			return fmt.Errorf("failed to rebuild index: %w", err)
		}
		fmt.Println("FTS5 index rebuild complete.")
	}

	return nil
}

// formatOutput handles the multi-format rendering logic.
func (d *Dispatcher) formatOutput(memories []service.Memory) error {
	switch d.OutputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(memories)

	case "csv":
		writer := csv.NewWriter(os.Stdout)
		defer writer.Flush()
		// Header
		writer.Write([]string{"ID", "Context", "Type", "Importance", "Created", "Content", "Tags"})
		for _, m := range memories {
			writer.Write([]string{
				fmt.Sprintf("%d", m.ID),
				m.ContextKey,
				m.EntryType,
				fmt.Sprintf("%d", m.ImportanceScore),
				m.CreatedAt,
				m.Content,
				strings.Join(m.Tags, ","),
			})
		}
		return nil

	case "table":
		fallthrough
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tCONTEXT\tTYPE\tIMP\tCREATED\tCONTENT")
		for _, m := range memories {
			// Truncate content for better table view
			content := m.Content
			if len(content) > 50 {
				content = content[:47] + "..."
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\t%s\n",
				m.ID, m.ContextKey, m.EntryType, m.ImportanceScore,
				m.CreatedAt[:10], content)
		}
		return w.Flush()
	}
}
