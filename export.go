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

// Wrapper methods for native UI compatibility

func (e *Exporter) ToCSV(sessions []Session) string {
	result, _ := e.ExportCSV(sessions)
	return result
}

func (e *Exporter) ToMarkdown(sessions []Session) string {
	return e.ExportMarkdown(sessions)
}

func (e *Exporter) ToZshHistory(sessions []Session) string {
	return e.ExportSessionsToZsh(sessions)
}

func (e *Exporter) ToBashScript(commands []string) string {
	var buf strings.Builder
	buf.WriteString("#!/bin/bash\n\n")
	buf.WriteString("# Generated from zsh history\n")
	buf.WriteString("# Generated at: " + time.Now().Format(time.RFC1123) + "\n\n")
	buf.WriteString("set -e\n\n")
	for i, cmd := range commands {
		buf.WriteString(fmt.Sprintf("# Command %d\n", i+1))
		buf.WriteString(cmd + "\n\n")
	}
	return buf.String()
}

func (e *Exporter) ToPythonScript(commands []string) string {
	var buf strings.Builder
	buf.WriteString("#!/usr/bin/env python3\n\n")
	buf.WriteString("import subprocess\n")
	buf.WriteString("import sys\n\n")
	buf.WriteString("# Generated from zsh history\n")
	buf.WriteString("# Generated at: " + time.Now().Format(time.RFC1123) + "\n\n")
	buf.WriteString("commands = [\n")
	for _, cmd := range commands {
		escaped := strings.ReplaceAll(cmd, "\"", "\\\"")
		buf.WriteString(fmt.Sprintf("    \"%s\",\n", escaped))
	}
	buf.WriteString("]\n\n")
	buf.WriteString("for i, cmd in enumerate(commands, 1):\n")
	buf.WriteString("    print(f'Executing command {i}/{len(commands)}: {cmd}')\n")
	buf.WriteString("    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)\n")
	buf.WriteString("    if result.returncode != 0:\n")
	buf.WriteString("        print(f'Error: {result.stderr}', file=sys.stderr)\n")
	buf.WriteString("        sys.exit(result.returncode)\n")
	buf.WriteString("    print(result.stdout)\n")
	return buf.String()
}

func (e *Exporter) ToJavaProgram(commands []string) string {
	var buf strings.Builder
	buf.WriteString("import java.io.*;\n")
	buf.WriteString("import java.util.*;\n\n")
	buf.WriteString("public class HistoryCommands {\n")
	buf.WriteString("    public static void main(String[] args) throws IOException, InterruptedException {\n")
	buf.WriteString("        String[] commands = {\n")
	for i, cmd := range commands {
		escaped := strings.ReplaceAll(cmd, "\"", "\\\"")
		if i < len(commands)-1 {
			buf.WriteString(fmt.Sprintf("            \"%s\",\n", escaped))
		} else {
			buf.WriteString(fmt.Sprintf("            \"%s\"\n", escaped))
		}
	}
	buf.WriteString("        };\n\n")
	buf.WriteString("        for (int i = 0; i < commands.length; i++) {\n")
	buf.WriteString("            System.out.println(\"Executing command \" + (i+1) + \"/\" + commands.length + \": \" + commands[i]);\n")
	buf.WriteString("            Process process = new ProcessBuilder(\"sh\", \"-c\", commands[i])\n")
	buf.WriteString("                .redirectErrorStream(true)\n")
	buf.WriteString("                .start();\n")
	buf.WriteString("            int exitCode = process.waitFor();\n")
	buf.WriteString("            if (exitCode != 0) {\n")
	buf.WriteString("                System.err.println(\"Command failed with exit code: \" + exitCode);\n")
	buf.WriteString("                System.exit(exitCode);\n")
	buf.WriteString("            }\n")
	buf.WriteString("        }\n")
	buf.WriteString("    }\n")
	buf.WriteString("}\n")
	return buf.String()
}

func (e *Exporter) ToGoProgram(commands []string) string {
	var buf strings.Builder
	buf.WriteString("package main\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"os\"\n")
	buf.WriteString("\t\"os/exec\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("func main() {\n")
	buf.WriteString("\tcommands := []string{\n")
	for _, cmd := range commands {
		escaped := strings.ReplaceAll(cmd, "`", "\\`")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		buf.WriteString(fmt.Sprintf("\t\t`%s`,\n", cmd))
	}
	buf.WriteString("\t}\n\n")
	buf.WriteString("\tfor i, cmdStr := range commands {\n")
	buf.WriteString("\t\tfmt.Printf(\"Executing command %d/%d: %s\\n\", i+1, len(commands), cmdStr)\n")
	buf.WriteString("\t\tcmd := exec.Command(\"sh\", \"-c\", cmdStr)\n")
	buf.WriteString("\t\tcmd.Stdout = os.Stdout\n")
	buf.WriteString("\t\tcmd.Stderr = os.Stderr\n")
	buf.WriteString("\t\tif err := cmd.Run(); err != nil {\n")
	buf.WriteString("\t\t\tfmt.Fprintf(os.Stderr, \"Command failed: %v\\n\", err)\n")
	buf.WriteString("\t\t\tos.Exit(1)\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")
	return buf.String()
}
