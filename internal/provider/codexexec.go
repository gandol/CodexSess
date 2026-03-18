package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ChatEvent struct {
	Type string
	Text string
}

type ChatResult struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

type CodexExec struct {
	Binary string
}

func NewCodexExec(binary string) *CodexExec {
	if strings.TrimSpace(binary) == "" {
		binary = "codex"
	}
	return &CodexExec{Binary: binary}
}

func (c *CodexExec) Chat(ctx context.Context, codexHome string, model string, prompt string) (ChatResult, error) {
	clean := resolveCleanExecMode()
	cmd := exec.CommandContext(ctx, c.Binary, c.buildExecArgs(model, prompt, clean)...)
	if dir := strings.TrimSpace(codexHome); dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = c.buildExecEnv(cmd.Environ(), codexHome, clean)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ChatResult{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ChatResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return ChatResult{}, err
	}
	stderrBuf, stderrDone := drainPipe(stderr)

	var out ChatResult
	var lastExecErr string
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var evt map[string]any
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}
		if msg := codexEventErrorMessage(evt); msg != "" {
			lastExecErr = msg
		}
		t, _ := evt["type"].(string)
		if t == "item.completed" {
			item, _ := evt["item"].(map[string]any)
			if itemType, _ := item["type"].(string); itemType == "agent_message" {
				if text, _ := item["text"].(string); text != "" {
					out.Text = text
				}
			}
		}
		if t == "turn.completed" {
			usage, _ := evt["usage"].(map[string]any)
			out.InputTokens = int(number(usage["input_tokens"]))
			out.OutputTokens = int(number(usage["output_tokens"]))
		}
	}
	if err := sc.Err(); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		<-stderrDone
		return ChatResult{}, err
	}
	if err := cmd.Wait(); err != nil {
		<-stderrDone
		stderrBytes := stderrBuf.Bytes()
		msg := firstNonEmpty(strings.TrimSpace(lastExecErr), strings.TrimSpace(string(stderrBytes)), err.Error())
		return ChatResult{}, fmt.Errorf("codex exec failed: %s", msg)
	}
	<-stderrDone
	if strings.TrimSpace(out.Text) == "" {
		return ChatResult{}, errors.New("empty response from codex")
	}
	return out, nil
}

func (c *CodexExec) StreamChat(ctx context.Context, codexHome string, model string, prompt string, onEvent func(ChatEvent) error) (ChatResult, error) {
	clean := resolveCleanExecMode()
	cmd := exec.CommandContext(ctx, c.Binary, c.buildExecArgs(model, prompt, clean)...)
	if dir := strings.TrimSpace(codexHome); dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = c.buildExecEnv(cmd.Environ(), codexHome, clean)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ChatResult{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return ChatResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return ChatResult{}, err
	}
	stderrBuf, stderrDone := drainPipe(stderr)

	var out ChatResult
	var lastExecErr string
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var evt map[string]any
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}
		if msg := codexEventErrorMessage(evt); msg != "" {
			lastExecErr = msg
		}
		t, _ := evt["type"].(string)
		if t == "item.completed" {
			item, _ := evt["item"].(map[string]any)
			if itemType, _ := item["type"].(string); itemType == "agent_message" {
				if text, _ := item["text"].(string); text != "" {
					out.Text = text
					if err := onEvent(ChatEvent{Type: "delta", Text: text}); err != nil {
						_ = cmd.Process.Kill()
						_ = cmd.Wait()
						<-stderrDone
						return ChatResult{}, err
					}
				}
			}
		}
		if t == "turn.completed" {
			usage, _ := evt["usage"].(map[string]any)
			out.InputTokens = int(number(usage["input_tokens"]))
			out.OutputTokens = int(number(usage["output_tokens"]))
		}
	}
	if err := sc.Err(); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		<-stderrDone
		return ChatResult{}, err
	}
	if err := cmd.Wait(); err != nil {
		<-stderrDone
		stderrBytes := stderrBuf.Bytes()
		msg := firstNonEmpty(strings.TrimSpace(lastExecErr), strings.TrimSpace(string(stderrBytes)), err.Error())
		return ChatResult{}, fmt.Errorf("codex exec failed: %s", msg)
	}
	<-stderrDone
	if strings.TrimSpace(out.Text) == "" {
		return ChatResult{}, errors.New("empty response from codex")
	}
	return out, nil
}

func drainPipe(rc io.ReadCloser) (*bytes.Buffer, <-chan struct{}) {
	buf := &bytes.Buffer{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if rc == nil {
			return
		}
		defer rc.Close()
		_, _ = io.Copy(buf, io.LimitReader(rc, 512*1024))
	}()
	return buf, done
}

func codexEventErrorMessage(evt map[string]any) string {
	if evt == nil {
		return ""
	}
	if t, _ := evt["type"].(string); strings.TrimSpace(t) == "error" {
		if msg, _ := evt["message"].(string); strings.TrimSpace(msg) != "" {
			return strings.TrimSpace(msg)
		}
	}
	if t, _ := evt["type"].(string); strings.TrimSpace(t) == "turn.failed" {
		if errObj, _ := evt["error"].(map[string]any); errObj != nil {
			if msg, _ := errObj["message"].(string); strings.TrimSpace(msg) != "" {
				return strings.TrimSpace(msg)
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func number(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func (c *CodexExec) buildExecArgs(model, prompt string, clean bool) []string {
	sandbox := resolveSandboxMode()
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--sandbox",
		sandbox,
		"-m",
		model,
		prompt,
	}
	if clean {
		args = append(args, "--ephemeral")
	}
	return args
}

func resolveSandboxMode() string {
	if v := strings.TrimSpace(os.Getenv("CODEXSESS_CODEX_SANDBOX")); v != "" {
		return v
	}
	// default should allow BrowserOS/tooling agents to create local sockets/temp files.
	return "workspace-write"
}

func resolveCleanExecMode() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CODEXSESS_CLEAN_EXEC")))
	if v == "" {
		return true
	}
	return v != "0" && v != "false" && v != "no"
}

func (c *CodexExec) buildExecEnv(base []string, codexHome string, clean bool) []string {
	env := append([]string{}, base...)
	env = append(env, "CODEX_HOME="+codexHome)
	if !clean {
		return env
	}
	homeRoot := filepath.Join(strings.TrimSpace(codexHome), ".codexsess-clean-home")
	if strings.TrimSpace(codexHome) == "" {
		homeRoot = filepath.Join(os.TempDir(), "codexsess-clean-home")
	}
	_ = os.MkdirAll(filepath.Join(homeRoot, ".config"), 0o700)
	_ = os.MkdirAll(filepath.Join(homeRoot, ".local", "share"), 0o700)
	env = append(env,
		"HOME="+homeRoot,
		"XDG_CONFIG_HOME="+filepath.Join(homeRoot, ".config"),
		"XDG_DATA_HOME="+filepath.Join(homeRoot, ".local", "share"),
	)
	return env
}
