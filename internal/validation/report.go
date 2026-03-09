package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteReportJSON writes the report to a JSON file.
func WriteReportJSON(report *Report, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	path := filepath.Join(outputDir, report.ScenarioName+".json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

// WriteReportMarkdown writes a human-readable Markdown summary.
func WriteReportMarkdown(report *Report, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	path := filepath.Join(outputDir, report.ScenarioName+".md")
	content := FormatReportMarkdown(report)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

// FormatReportMarkdown returns the Markdown content for a report.
func FormatReportMarkdown(report *Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Validation Report: %s\n\n", report.ScenarioName))
	b.WriteString(fmt.Sprintf("**Status:** %s\n\n", report.Status))
	b.WriteString(fmt.Sprintf("**Started:** %s\n\n", report.StartTime.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("**Ended:** %s\n\n", report.EndTime.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("**Duration:** %d ms\n\n", report.DurationMs))
	b.WriteString(fmt.Sprintf("**Assertions:** %d passed, %d failed\n\n", report.AssertionsPassed, report.AssertionsFailed))
	if report.FinalRobotState != "" {
		b.WriteString(fmt.Sprintf("**Final Robot State:** %s\n\n", report.FinalRobotState))
	}
	if report.Error != "" {
		b.WriteString(fmt.Sprintf("**Error:** %s\n\n", report.Error))
	}
	if report.Notes != "" {
		b.WriteString(fmt.Sprintf("**Notes:** %s\n\n", report.Notes))
	}

	b.WriteString("## Assertion Results\n\n")
	b.WriteString("| Type | Passed | Message |\n")
	b.WriteString("|------|--------|--------|\n")
	for _, r := range report.Results {
		passStr := "FAIL"
		if r.Passed {
			passStr = "PASS"
		}
		msg := strings.ReplaceAll(r.Message, "|", "\\|")
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", r.AssertionType, passStr, msg))
	}

	if len(report.EmittedEvents) > 0 {
		b.WriteString("\n## Emitted Events\n\n")
		for _, e := range report.EmittedEvents {
			b.WriteString(fmt.Sprintf("- %s\n", e.Type))
		}
	}
	if len(report.ContractViolations) > 0 {
		b.WriteString("\n## Adapter Contract Violations\n\n")
		for _, v := range report.ContractViolations {
			b.WriteString(fmt.Sprintf("- **%s** %s: %s\n", v.Category, v.Field, v.Message))
		}
	}
	return b.String()
}

// WriteSummaryMarkdown writes a summary of multiple reports.
func WriteSummaryMarkdown(reports []*Report, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	path := filepath.Join(outputDir, "summary.md")
	var b strings.Builder
	b.WriteString("# Validation Summary\n\n")
	b.WriteString(fmt.Sprintf("**Total scenarios:** %d\n\n", len(reports)))
	passed := 0
	for _, r := range reports {
		if r.Status == "PASS" {
			passed++
		}
	}
	b.WriteString(fmt.Sprintf("**Passed:** %d\n\n", passed))
	b.WriteString(fmt.Sprintf("**Failed:** %d\n\n", len(reports)-passed))
	b.WriteString("| Scenario | Status | Duration (ms) |\n")
	b.WriteString("|----------|--------|---------------|\n")
	for _, r := range reports {
		b.WriteString(fmt.Sprintf("| %s | %s | %d |\n", r.ScenarioName, r.Status, r.DurationMs))
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", fmt.Errorf("write summary: %w", err)
	}
	return path, nil
}
