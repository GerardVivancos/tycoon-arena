package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"realtime-game-server/testutil"
)

func main() {
	// CLI flags
	scenarioPath := flag.String("scenario", "", "Path to scenario JSON file (relative to maps/scenarios/)")
	all := flag.Bool("all", false, "Render all scenarios in maps/scenarios/")
	outputDir := flag.String("output", "../../../maps/scenarios/visuals", "Output directory for SVG files")

	flag.Parse()

	if *scenarioPath == "" && !*all {
		fmt.Println("Usage: scenario-viz --scenario=<file.json> OR --all")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  scenario-viz --scenario=navigate_around_rock.json")
		fmt.Println("  scenario-viz --all")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	if *all {
		renderAllScenarios(*outputDir)
	} else {
		renderScenario(*scenarioPath, *outputDir)
	}
}

func renderScenario(scenarioFile, outputDir string) {
	// Construct full path
	scenarioPath := filepath.Join("../../../maps/scenarios", scenarioFile)

	// Load scenario
	scenario, err := testutil.LoadScenario(scenarioPath)
	if err != nil {
		log.Fatalf("Failed to load scenario: %v", err)
	}

	fmt.Printf("Rendering scenario: %s\n", scenario.Name)

	// Render SVG
	svg, err := testutil.RenderScenarioSVG(scenario, nil)
	if err != nil {
		log.Fatalf("Failed to render SVG: %v", err)
	}

	// Determine output filename
	outputFile := strings.TrimSuffix(filepath.Base(scenarioFile), ".json") + ".svg"
	outputPath := filepath.Join(outputDir, outputFile)

	// Write SVG file
	if err := os.WriteFile(outputPath, []byte(svg), 0644); err != nil {
		log.Fatalf("Failed to write SVG: %v", err)
	}

	fmt.Printf("✓ Generated: %s\n", outputPath)
}

func renderAllScenarios(outputDir string) {
	// Find all scenario JSON files
	scenarioFiles, err := filepath.Glob("../../../maps/scenarios/*.json")
	if err != nil {
		log.Fatalf("Failed to find scenarios: %v", err)
	}

	if len(scenarioFiles) == 0 {
		fmt.Println("No scenarios found in maps/scenarios/")
		return
	}

	fmt.Printf("Found %d scenarios\n\n", len(scenarioFiles))

	for _, scenarioPath := range scenarioFiles {
		scenario, err := testutil.LoadScenario(scenarioPath)
		if err != nil {
			log.Printf("⚠ Skipping %s: %v\n", filepath.Base(scenarioPath), err)
			continue
		}

		fmt.Printf("Rendering: %s\n", scenario.Name)

		svg, err := testutil.RenderScenarioSVG(scenario, nil)
		if err != nil {
			log.Printf("⚠ Failed to render: %v\n", err)
			continue
		}

		// Write SVG
		outputFile := strings.TrimSuffix(filepath.Base(scenarioPath), ".json") + ".svg"
		outputPath := filepath.Join(outputDir, outputFile)

		if err := os.WriteFile(outputPath, []byte(svg), 0644); err != nil {
			log.Printf("⚠ Failed to write SVG: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ %s\n", outputPath)
	}

	fmt.Printf("\nDone! All SVGs saved to: %s\n", outputDir)
}
