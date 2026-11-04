package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	messagesURL = "https://api.anthropic.com/v1/messages"
	apiVersion  = "2023-06-01"
	maxTokens   = 1024
	turnTimeout = 60 * time.Second
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func loadEnv(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s, %w", path, err)
	}

	values := make(map[string]string)

	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("%s:%d: expected KEY=VALUE", path, i+1)
		}

		values[key] = strings.TrimSpace(value)
	}

	return values, nil
}

func callMessages(
	ctx context.Context,
	client *http.Client,
	apiKey string,
	model string,
	history []Message,
) (messagesResponse, error) {
	var reply messagesResponse

	body, err := json.Marshal(messagesRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  history,
	})
	if err != nil {
		return reply, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, messagesURL, bytes.NewReader(body),
	)
	if err != nil {
		return reply, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := client.Do(req)
	if err != nil {
		return reply, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var failure struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}

		err := json.NewDecoder(resp.Body).Decode(&failure)
		if err != nil && failure.Error.Message != "" {
			return reply, fmt.Errorf("Anthropic HTTP %d: %s",
				resp.StatusCode, failure.Error.Message)
		}

		return reply, fmt.Errorf("Anthropic HTTP %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return reply, fmt.Errorf("decode response: %w", err)
	}

	return reply, nil
}

func run() error {
	settings, err := loadEnv(".env")
	if err != nil {
		return err
	}

	apiKey := settings["ANTHROPIC_API_KEY"]
	model := settings["ANTHROPIC_MODEL"]

	if apiKey == "" || apiKey == "replace-with-your-key" {
		return fmt.Errorf("set ANTHROPIC_API_KEY in .env")
	}
	if model == "" || model == "replace-with-your-model-id" {
		return fmt.Errorf("set ANTHROPIC_MODEL in .env")
	}

	client := &http.Client{}
	scanner := bufio.NewScanner(os.Stdin)

	scanner.Buffer(make([]byte, 4096), 1024*1024)

	var history []Message

	fmt.Printf("Go Harness v1 | model=%s | /quit to exit\n", model)

	for {
		fmt.Print("\n>you ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "/quit" {
			return nil
		}
		if input == "" {
			continue
		}

		pending := append(history, Message{Role: "user", Content: input})

		ctx, cancel := context.WithTimeout(context.Background(), turnTimeout)
		reply, err := callMessages(ctx, client, apiKey, model, pending)
		cancel()

		if err != nil {
			fmt.Fprintln(os.Stderr, "[error]", err)
			continue
		}

		var parts []string
		for _, block := range reply.Content {
			if block.Type == "text" {
				parts = append(parts, block.Text)
			}
		}
		answer := strings.Join(parts, "")

		fmt.Printf("[usage] input=%d output=%d | stop=%s\n",
			reply.Usage.InputTokens, reply.Usage.OutputTokens, reply.StopReason)

		if strings.TrimSpace(answer) == "" {
			fmt.Fprintln(os.Stderr, "[error] API returned no text")
			continue
		}

		history = append(pending, Message{Role: "assistant", Content: answer})
		fmt.Printf("assistant> %s\n", answer)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read terminal input: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "[error]", err)
		os.Exit(1)
	}
}
