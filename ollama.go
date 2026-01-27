package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OllamaClient struct {
	baseURL string
	model   string
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
	}
}

func (c *OllamaClient) GeneratePrompt(action, commands string) string {
	prompts := map[string]string{
		"explain": fmt.Sprintf("Explain what these shell commands do:\n\n%s\n\nProvide a clear, concise explanation.", commands),
		"rewrite": fmt.Sprintf("Rewrite these shell commands to be more efficient or follow best practices:\n\n%s\n\nProvide the improved version with a brief explanation of changes.", commands),
		"script":  fmt.Sprintf("Convert these shell commands into a bash script with proper error handling and comments:\n\n%s\n\nProvide a complete, well-documented script.", commands),
		"python":  fmt.Sprintf("Translate these shell commands to Python code:\n\n%s\n\nProvide equivalent Python code using standard libraries when possible.", commands),
		"java":    fmt.Sprintf("Translate these shell commands to Java code:\n\n%s\n\nProvide equivalent Java code with appropriate libraries and error handling.", commands),
		"golang":  fmt.Sprintf("Translate these shell commands to Go code:\n\n%s\n\nProvide equivalent Go code using standard libraries and proper error handling.", commands),
	}

	if prompt, ok := prompts[action]; ok {
		return prompt
	}
	return fmt.Sprintf("Analyze these commands:\n\n%s", commands)
}

func (c *OllamaClient) Generate(prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(c.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Ollama returns newline-delimited JSON, even with stream=false
	// We need to parse each line and concatenate the responses
	var fullResponse strings.Builder
	lines := strings.Split(string(body), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var ollamaResp OllamaResponse
		if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
			return "", fmt.Errorf("failed to parse Ollama response: %w (body: %s)", err, line)
		}
		
		fullResponse.WriteString(ollamaResp.Response)
	}

	return strings.TrimSpace(fullResponse.String()), nil
}

func (c *OllamaClient) AnalyzeCommands(action string, commands []string) (string, error) {
	commandStr := strings.Join(commands, "\n")
	prompt := c.GeneratePrompt(action, commandStr)
	return c.Generate(prompt)
}

func (c *OllamaClient) AnalyzeSession(action string, session *Session) (string, error) {
	commands := make([]string, len(session.Commands))
	for i, cmd := range session.Commands {
		commands[i] = cmd.Command
	}
	return c.AnalyzeCommands(action, commands)
}

func (c *OllamaClient) AnalyzeSessionWithPrompt(customPrompt string, session *Session) (string, error) {
	commands := make([]string, len(session.Commands))
	for i, cmd := range session.Commands {
		commands[i] = cmd.Command
	}
	commandStr := strings.Join(commands, "\n")
	prompt := fmt.Sprintf("%s\n\nCommands:\n%s", customPrompt, commandStr)
	return c.Generate(prompt)
}
