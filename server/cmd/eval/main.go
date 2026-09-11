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
	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/llm"
	"github.com/joho/godotenv"
)

func main() {
	coasDir := flag.String("coas", "testdata/coas", "directory of COA PDFs")
	expectedDir := flag.String("expected", "testdata/expected", "directory of golden JSON answer keys")
	promptPath := flag.String("prompt", "prompts/extract_coa_v5.md", "extraction prompt")
	schemaPath := flag.String("schema", "schemas/coa_analysis_v2.json", "extraction JSON schema")
	flag.Parse()

	godotenv.Load()
	model := os.Getenv("LLM_MODEL")

	llmClient, err := llm.NewOpenRouterClient(llm.OpenRouterConfig{
		APIKey: os.Getenv("OPENROUTER_API_KEY"),
		Model:  model,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to configure OpenRouter client:", err)
		os.Exit(1)
	}

	svc, err := coa.NewService(llmClient, *promptPath, *schemaPath)
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
	totalUsage := coa.Usage{}
	documentsScored, documentsChecked := 0, 0

	fmt.Printf("MODEL   %s\nPROMPT  %s\nSCHEMA  %s\n\n", model, *promptPath, *schemaPath)

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
		documentsChecked++

		actual, usage, err := analyze(ctx, svc, filepath.Join(*coasDir, name))
		totalUsage.InputTokens += usage.InputTokens
		totalUsage.OutputTokens += usage.OutputTokens
		totalUsage.CostUSD += usage.CostUSD
		if err != nil {
			totalChecked += eval.FieldCount(expected)
			fmt.Printf("%-24s SKIP (%v)  tokens=%d/%d cost=$%.6f\n",
				name, err, usage.InputTokens, usage.OutputTokens, usage.CostUSD)
			continue
		}

		correct, checked, missed := eval.Score(expected, actual)
		documentsScored++
		totalCorrect += correct
		totalChecked += checked
		fmt.Printf("%-24s %d/%d  tokens=%d/%d cost=$%.6f\n",
			name, correct, checked, usage.InputTokens, usage.OutputTokens, usage.CostUSD)
		for _, m := range missed {
			fmt.Printf("    %-26s want %q  got %q\n", m.Field, m.Expected, m.Got)
		}
	}

	if documentsChecked == 0 {
		fmt.Printf("\nno documents scored\nUSAGE    tokens=%d/%d cost=$%.6f\n",
			totalUsage.InputTokens, totalUsage.OutputTokens, totalUsage.CostUSD)
		return
	}
	fmt.Printf("\nDOCUMENTS %d/%d scored\n", documentsScored, documentsChecked)
	fmt.Printf("\nOVERALL  %d/%d  (%.1f%%)\n",
		totalCorrect, totalChecked, float64(totalCorrect)/float64(totalChecked)*100)
	fmt.Printf("USAGE    tokens=%d/%d cost=$%.6f\n",
		totalUsage.InputTokens, totalUsage.OutputTokens, totalUsage.CostUSD)
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

func analyze(ctx context.Context, svc *coa.Service, path string) (*coa.Analysis, coa.Usage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, coa.Usage{}, err
	}
	defer f.Close()

	actual, usage, err := svc.AnalyzeCOA(ctx, f, filepath.Base(path))
	if err != nil {
		return nil, usage, err
	}
	return actual, usage, nil
}
