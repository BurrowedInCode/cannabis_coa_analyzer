package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/coa"
	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/eval"
	"github.com/joho/godotenv"
)

func main() {
	coasDir := flag.String("coas", "testdata/coas", "directory of COA PDFs")
	expectedDir := flag.String("expected", "testdata/expected", "directory of golden JSON answer keys")
	promptPath := flag.String("prompt", "prompts/extract_coa_v3.md", "extraction prompt")
	flag.Parse()

	godotenv.Load()

	svc, err := coa.NewService(*promptPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load service:", err)
		os.Exit(1)
	}

	entries, err := os.ReadDir(*coasDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read coas dir:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	totalCorrect, totalChecked := 0, 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			continue
		}
		name := entry.Name()
		base := strings.TrimSuffix(name, filepath.Ext(name))

		expected, err := loadGolden(filepath.Join(*expectedDir, base+".json"))
		if err != nil {
			fmt.Printf("%-24s SKIP (no answer key: %v)\n", name, err)
			continue
		}

		actual, err := analyze(ctx, svc, filepath.Join(*coasDir, name))
		if err != nil {
			fmt.Printf("%-24s SKIP (%v)\n", name, err)
			continue
		}

		correct, checked, missed := eval.Score(expected, actual)
		totalCorrect += correct
		totalChecked += checked
		fmt.Printf("%-24s %d/%d\n", name, correct, checked)
		for _, m := range missed {
			fmt.Printf("    %-26s want %q  got %q\n", m.Field, m.Expected, m.Got)
		}
	}

	if totalChecked == 0 {
		fmt.Println("\nno documents scored")
		return
	}
	fmt.Printf("\nOVERALL  %d/%d  (%.1f%%)\n",
		totalCorrect, totalChecked, float64(totalCorrect)/float64(totalChecked)*100)
}

func loadGolden(path string) (*coa.Analysis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a coa.Analysis
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func analyze(ctx context.Context, svc *coa.Service, path string) (*coa.Analysis, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	uploaded, err := svc.UploadCOA(ctx, f, filepath.Base(path), "application/pdf")
	if err != nil {
		return nil, err
	}
	defer svc.DeleteCOA(ctx, uploaded.ID)

	actual, _, err := svc.AnalyzeCOA(ctx, uploaded.ID)
	if err != nil {
		return nil, err
	}
	return actual, nil
}
