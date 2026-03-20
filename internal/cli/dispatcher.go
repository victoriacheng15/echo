package cli

import (
	"echo/internal/service"
	"flag"
	"fmt"
	"os"
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
	fmt.Fprintf(os.Stderr, "  delete    Remove a memory by ID or content matching\n")
	fmt.Fprintf(os.Stderr, "  maintain  Perform database maintenance and analytics sync\n")
	fmt.Fprintf(os.Stderr, "  help      Show this help message\n")
}

// Subcommand Handlers (Skeleton for internal logic)

func (d *Dispatcher) handleStore(args []string) error {
	fs := flag.NewFlagSet("store", flag.ExitOnError)
	// TODO: Define store flags
	_ = fs.Parse(args)
	fmt.Println("internal/cli: handleStore logic goes here.")
	return nil
}

func (d *Dispatcher) handleRecall(args []string) error {
	fs := flag.NewFlagSet("recall", flag.ExitOnError)
	// TODO: Define recall flags
	_ = fs.Parse(args)
	fmt.Println("internal/cli: handleRecall logic goes here.")
	return nil
}

func (d *Dispatcher) handleSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	// TODO: Define search flags
	_ = fs.Parse(args)
	fmt.Println("internal/cli: handleSearch logic goes here.")
	return nil
}

func (d *Dispatcher) handleDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	// TODO: Define delete flags
	_ = fs.Parse(args)
	fmt.Println("internal/cli: handleDelete logic goes here.")
	return nil
}

func (d *Dispatcher) handleMaintain(args []string) error {
	fs := flag.NewFlagSet("maintain", flag.ExitOnError)
	// TODO: Define maintain flags
	_ = fs.Parse(args)
	fmt.Println("internal/cli: handleMaintain logic goes here.")
	return nil
}
