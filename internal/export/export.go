package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

// ExportFormat represents the export format
type ExportFormat string

const (
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportWorkflowStats exports workflow statistics to a file
func ExportWorkflowStats(stats *github.WorkflowRunStats, format ExportFormat, outputPath string) error {
	switch format {
	case FormatCSV:
		return exportStatsCSV(stats, outputPath)
	case FormatJSON:
		return exportStatsJSON(stats, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportStatsCSV(stats *github.WorkflowRunStats, outputPath string) error {
	file, err := createPrivateFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Header
	if err := writer.Write([]string{"Metric", "Value"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Data
	rows := [][]string{
		{"Total Runs", fmt.Sprintf("%d", stats.TotalRuns)},
		{"Success Rate", fmt.Sprintf("%.2f%%", stats.SuccessRate)},
		{"Failure Count", fmt.Sprintf("%d", stats.FailureCount)},
		{"Avg Duration", stats.AvgDuration.String()},
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return flushCSV(writer)
}

func exportStatsJSON(stats *github.WorkflowRunStats, outputPath string) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExportComments exports comments to a file
func ExportComments(comments []github.Comment, format ExportFormat, outputPath string) error {
	switch format {
	case FormatCSV:
		return exportCommentsCSV(comments, outputPath)
	case FormatJSON:
		return exportCommentsJSON(comments, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportCommentsCSV(comments []github.Comment, outputPath string) error {
	file, err := createPrivateFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Header
	if err := writer.Write([]string{"Repository", "PR", "Author", "Path", "Line", "Body", "Created"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Data
	for _, c := range comments {
		if err := writer.Write([]string{
			c.Repository,
			fmt.Sprintf("%d", c.PRNumber),
			c.Author,
			c.Path,
			fmt.Sprintf("%d", c.Line),
			c.Body,
			c.CreatedAt.Format(time.RFC3339),
		}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return flushCSV(writer)
}

func exportCommentsJSON(comments []github.Comment, outputPath string) error {
	data, err := json.MarshalIndent(comments, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExportProtectionRules exports protection rules to a file
func ExportProtectionRules(rules []*github.ProtectionRule, format ExportFormat, outputPath string) error {
	switch format {
	case FormatCSV:
		return exportProtectionCSV(rules, outputPath)
	case FormatJSON:
		return exportProtectionJSON(rules, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportProtectionCSV(rules []*github.ProtectionRule, outputPath string) error {
	file, err := createPrivateFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	// Header
	if err := writer.Write([]string{"Repository", "Branch", "Required Reviews", "Code Owner Reviews", "Enforce Admins"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Data
	for _, rule := range rules {
		if err := writer.Write([]string{
			rule.Repository,
			rule.Branch,
			fmt.Sprintf("%d", rule.RequiredReviews),
			fmt.Sprintf("%v", rule.RequireCodeOwnerReviews),
			fmt.Sprintf("%v", rule.EnforceAdmins),
		}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return flushCSV(writer)
}

func exportProtectionJSON(rules []*github.ProtectionRule, outputPath string) error {
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func createPrivateFile(outputPath string) (*os.File, error) {
	return os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600) // #nosec G304 -- output path is an explicit user-requested export target.
}

func flushCSV(writer *csv.Writer) error {
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV: %w", err)
	}
	return nil
}
