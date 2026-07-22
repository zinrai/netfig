package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// netfig validates a YAML description of a network and emits SVG on
// stdout. Coordinates are computed directly from the validated
// (band, location) layout, so there is no external layout engine in
// the pipeline.
//
// Typical use:
//
//	netfig topology.yaml > diagram.svg
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "netfig: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		printVersion()
		return nil
	}

	path, err := resolveInputPath(flag.Args())
	if err != nil {
		return err
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}

	if err := ValidateLegend(cfg); err != nil {
		return fmt.Errorf("validate legend: %w", err)
	}

	if err := ValidateTopology(cfg); err != nil {
		return fmt.Errorf("validate topology: %w", err)
	}

	info, err := ValidateLayout(cfg)
	if err != nil {
		return fmt.Errorf("validate layout: %w", err)
	}

	if err := ValidateGroups(cfg, info); err != nil {
		return fmt.Errorf("validate groups: %w", err)
	}

	svgSrc := GenerateSVG(cfg, info)
	if _, err := io.WriteString(os.Stdout, svgSrc); err != nil {
		return fmt.Errorf("write stdout: %w", err)
	}

	return nil
}

// resolveInputPath validates that exactly one positional argument was
// given and returns it. netfig requires the input to be a file path
// because its YAML inputs are files that authors edit and keep under
// version control, not streams from upstream tools, and because file
// paths give error messages somewhere concrete to point at.
func resolveInputPath(args []string) (string, error) {
	switch len(args) {
	case 0:
		return "", fmt.Errorf("missing input file; usage: netfig FILE")
	case 1:
		return args[0], nil
	default:
		return "", fmt.Errorf("too many arguments; expected one input file, got %d", len(args))
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  netfig FILE
  netfig -version

Reads a YAML topology description from FILE and emits SVG on stdout.
Coordinates are computed directly from the validated layout; there
is no external rendering engine in the pipeline.

Examples:
  netfig topology.yaml > diagram.svg

`)
}
