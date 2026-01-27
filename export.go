package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Exporter struct{}

func NewExporter() *Exporter {
	return &Exporter{}
}

func (e *Exporter) ExportJSON(sessions []Session) (string, error) {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (e *Exporter) ExportCSV(sessions []Session) (string, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"Session ID", "Command ID", "Timestamp", "Duration", "Command", "Directory", "Category", "Base Command"}
	if err := writer.Write(header); err != nil {
		return "", err
	}

	// Write data
	for _, session := range sessions {
		for _, cmd := range session.Commands {
			record := []string{
				fmt.Sprintf("%d", session.ID),
				fmt.Sprintf("%d", cmd.ID),
				cmd.Timestamp.Format(time.RFC3339),
				fmt.Sprintf("%d", cmd.Duration),
				cmd.Command,
				cmd.Directory,
				string(cmd.Category),
				cmd.BaseCommand,
			}
			if err := writer.Write(record); err != nil {
				return "", err
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (e *Exporter) ExportMarkdown(sessions []Session) string {
	var buf strings.Builder

	buf.WriteString("# Shell History Sessions\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC1123)))

	for _, session := range sessions {
		buf.WriteString(fmt.Sprintf("## Session %d: %s\n\n", session.ID, session.Description))
		buf.WriteString(fmt.Sprintf("- **Start:** %s\n", session.StartTime.Format(time.RFC1123)))
		buf.WriteString(fmt.Sprintf("- **End:** %s\n", session.EndTime.Format(time.RFC1123)))
		buf.WriteString(fmt.Sprintf("- **Duration:** %s\n", session.Duration.Round(time.Second)))
		buf.WriteString(fmt.Sprintf("- **Commands:** %d\n", len(session.Commands)))
		buf.WriteString(fmt.Sprintf("- **Directories:** %s\n\n", strings.Join(session.Directories, ", ")))

		if len(session.Categories) > 0 {
			buf.WriteString("### Categories\n\n")
			for cat, count := range session.Categories {
				buf.WriteString(fmt.Sprintf("- %s: %d commands\n", cat, count))
			}
			buf.WriteString("\n")
		}

		buf.WriteString("### Commands\n\n")
		for _, cmd := range session.Commands {
			buf.WriteString(fmt.Sprintf("#### %s - %s\n\n", cmd.Timestamp.Format("15:04:05"), cmd.BaseCommand))
			buf.WriteString(fmt.Sprintf("**Directory:** `%s`\n\n", cmd.Directory))
			buf.WriteString(fmt.Sprintf("**Category:** %s\n\n", cmd.Category))
			buf.WriteString("```bash\n")
			buf.WriteString(cmd.Command)
			buf.WriteString("\n```\n\n")
		}

		buf.WriteString("---\n\n")
	}

	return buf.String()
}

func (e *Exporter) ExportZshHistory(entries []HistoryEntry) string {
	var buf strings.Builder

	for _, entry := range entries {
		// Format: : <timestamp>:<duration>;<command>
		buf.WriteString(fmt.Sprintf(": %d:%d;%s\n", 
			entry.Timestamp.Unix(),
			entry.Duration,
			entry.Command,
		))
	}

	return buf.String()
}

func (e *Exporter) ExportSessionsToZsh(sessions []Session) string {
	var entries []HistoryEntry
	for _, session := range sessions {
		entries = append(entries, session.Commands...)
	}
	return e.ExportZshHistory(entries)
}
