package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"conductor/internal/store"
)

// ClaudeCodeRuntime spawns a fresh `claude` process per task.
// The claude CLI is a single-shot tool; it is not a persistent server.
// We start a new process for each task, passing the task as the initial prompt via stdin.
type ClaudeCodeRuntime struct {
	mu      sync.Mutex
	cmd     *exec.Cmd   // current running process, nil when idle
	config  AgentConfig // set by Spawn, reused per task
	pid     int
	tokens  int64
	alive   bool // true while a process is running

	OutputHandler  func(line string)
	ControlHandler func(prefix string, payload json.RawMessage)
}

func NewClaudeCodeRuntime(outputHandler func(string), controlHandler func(string, json.RawMessage)) *ClaudeCodeRuntime {
	return &ClaudeCodeRuntime{
		OutputHandler:  outputHandler,
		ControlHandler: controlHandler,
	}
}

// Spawn stores the agent config. No process is started yet.
func (r *ClaudeCodeRuntime) Spawn(ctx context.Context, config AgentConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = config
	return nil
}

// SendTask spawns a new claude process with the issue as the initial user message.
// Any in-progress task is killed first.
func (r *ClaudeCodeRuntime) SendTask(ctx context.Context, issue store.Issue) error {
	r.mu.Lock()
	if r.cmd != nil && r.cmd.Process != nil {
		r.cmd.Process.Kill() //nolint
	}
	r.mu.Unlock()

	// Build the task prompt — plain text so the LLM understands it naturally.
	taskText := fmt.Sprintf("Task ID: %s\nTitle: %s\n\n%s", issue.ID, issue.Title, issue.Description)

	args := []string{
		"--output-format", "stream-json",
		"--system", r.config.SystemPrompt,
		"--no-interactive",
		"--print", taskText,
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = r.config.WorkDir
	cmd.Env = append(cmd.Environ(),
		fmt.Sprintf("CONDUCTOR_AGENT_ID=%s", r.config.AgentID.String()),
		fmt.Sprintf("CONDUCTOR_COMPANY_ID=%s", r.config.CompanyID.String()),
	)
	for k, v := range r.config.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.pid = cmd.Process.Pid
	r.alive = true
	r.mu.Unlock()

	go r.readOutput(stdout)
	go io.Copy(io.Discard, stderr) //nolint
	go func() {
		cmd.Wait() //nolint
		r.mu.Lock()
		r.alive = false
		r.mu.Unlock()
	}()

	return nil
}

func (r *ClaudeCodeRuntime) readOutput(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for control messages first.
		isControl := false
		for _, prefix := range controlPrefixes {
			if strings.HasPrefix(line, prefix+" ") || line == prefix {
				isControl = true
				r.handleControl(line)
				break
			}
		}
		if isControl {
			continue
		}

		// Try to parse tokens from stream-json output.
		r.tryParseTokens(line)

		if r.OutputHandler != nil {
			r.OutputHandler(line)
		}
	}
}

func (r *ClaudeCodeRuntime) tryParseTokens(line string) {
	var msg struct {
		Type  string `json:"type"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal([]byte(line), &msg); err == nil && msg.Type == "usage" {
		total := int64(msg.Usage.InputTokens + msg.Usage.OutputTokens)
		atomic.AddInt64(&r.tokens, total)
	}
}

func (r *ClaudeCodeRuntime) handleControl(line string) {
	if r.ControlHandler == nil {
		return
	}
	for _, prefix := range controlPrefixes {
		if strings.HasPrefix(line, prefix) {
			rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			var raw json.RawMessage
			if len(rest) > 0 {
				raw = json.RawMessage(rest)
			} else {
				raw = json.RawMessage("{}")
			}
			go r.ControlHandler(prefix, raw)
			return
		}
	}
}

// Heartbeat returns nil if the process is currently running, error if idle.
func (r *ClaudeCodeRuntime) Heartbeat(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.alive && r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Signal(nil) //nolint
	}
	// No task running — that's fine, agent is idle. Treat as alive.
	return nil
}

func (r *ClaudeCodeRuntime) SendChat(ctx context.Context, message string) (string, error) {
	return "", fmt.Errorf("chat not supported while agent is idle; pause first")
}

func (r *ClaudeCodeRuntime) Pause(ctx context.Context) error  { return nil }
func (r *ClaudeCodeRuntime) Resume(ctx context.Context) error { return nil }

func (r *ClaudeCodeRuntime) Kill(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Kill()
	}
	return nil
}

func (r *ClaudeCodeRuntime) PID() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pid
}

func (r *ClaudeCodeRuntime) TokensUsed() int {
	return int(atomic.LoadInt64(&r.tokens))
}
