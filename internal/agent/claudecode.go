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

// ClaudeCodeRuntime manages a claude CLI subprocess.
// It communicates via structured JSON on stdin/stdout using the claude --output-format json flag.
type ClaudeCodeRuntime struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	outputCh   chan string  // raw stdout lines
	controlCh  chan string  // control message lines
	doneCh     chan struct{} // closed when process exits
	paused     bool
	chatReplyCh chan string
	pid        int
	tokens     int64

	// OutputHandler is called for every non-control stdout line (task output).
	OutputHandler func(line string)
	// ControlHandler is called for every parsed control message.
	ControlHandler func(prefix string, payload json.RawMessage)
}

func NewClaudeCodeRuntime(outputHandler func(string), controlHandler func(string, json.RawMessage)) *ClaudeCodeRuntime {
	return &ClaudeCodeRuntime{
		outputCh:       make(chan string, 256),
		controlCh:      make(chan string, 64),
		doneCh:         make(chan struct{}),
		chatReplyCh:    make(chan string, 1),
		OutputHandler:  outputHandler,
		ControlHandler: controlHandler,
	}
}

var controlPrefixes = []string{
	ControlHire,
	ControlEscalate,
	ControlDone,
	ControlBlocked,
	ControlHeartbeat,
}

func (r *ClaudeCodeRuntime) Spawn(ctx context.Context, config AgentConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	args := []string{
		"--output-format", "stream-json",
		"--system", config.SystemPrompt,
		"--no-interactive",
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = config.WorkDir

	// Inject agent-specific env vars
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("CONDUCTOR_AGENT_ID=%s", config.AgentID.String()))
	for k, v := range config.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	r.cmd = cmd
	r.stdin = stdin
	r.pid = cmd.Process.Pid
	close(r.doneCh)
	r.doneCh = make(chan struct{})

	go r.readStdout(stdout)
	go r.dispatchOutput()
	go func() {
		cmd.Wait() //nolint
		close(r.doneCh)
	}()

	return nil
}

func (r *ClaudeCodeRuntime) readStdout(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		isControl := false
		for _, prefix := range controlPrefixes {
			if strings.HasPrefix(line, prefix+" ") || line == prefix {
				isControl = true
				select {
				case r.controlCh <- line:
				default:
				}
				break
			}
		}
		if !isControl {
			select {
			case r.outputCh <- line:
			default:
			}
		}
	}
}

func (r *ClaudeCodeRuntime) dispatchOutput() {
	for {
		select {
		case line, ok := <-r.outputCh:
			if !ok {
				return
			}
			// Parse token usage from claude's JSON stream output
			r.tryParseTokens(line)

			r.mu.Lock()
			paused := r.paused
			r.mu.Unlock()

			if paused {
				// During pause, route output as chat reply
				select {
				case r.chatReplyCh <- line:
				default:
				}
			} else if r.OutputHandler != nil {
				r.OutputHandler(line)
			}

		case line, ok := <-r.controlCh:
			if !ok {
				return
			}
			r.handleControl(line)
		}
	}
}

func (r *ClaudeCodeRuntime) tryParseTokens(line string) {
	// claude --output-format stream-json emits usage objects
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
			rest := strings.TrimPrefix(line, prefix)
			rest = strings.TrimSpace(rest)
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

func (r *ClaudeCodeRuntime) SendTask(ctx context.Context, issue store.Issue) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stdin == nil {
		return fmt.Errorf("agent not running")
	}

	msg := map[string]interface{}{
		"type":        "user",
		"issue_id":    issue.ID.String(),
		"title":       issue.Title,
		"description": issue.Description,
	}
	data, _ := json.Marshal(msg)
	_, err := fmt.Fprintf(r.stdin, "%s\n", data)
	return err
}

func (r *ClaudeCodeRuntime) SendChat(ctx context.Context, message string) (string, error) {
	r.mu.Lock()
	if r.stdin == nil {
		r.mu.Unlock()
		return "", fmt.Errorf("agent not running")
	}
	r.mu.Unlock()

	msg := map[string]interface{}{
		"type":    "human_chat",
		"content": message,
	}
	data, _ := json.Marshal(msg)
	if _, err := fmt.Fprintf(r.stdin, "%s\n", data); err != nil {
		return "", err
	}

	select {
	case reply := <-r.chatReplyCh:
		return reply, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (r *ClaudeCodeRuntime) Heartbeat(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil || r.cmd.Process == nil {
		return fmt.Errorf("not running")
	}
	// Process.Signal(0) checks if process is alive
	return r.cmd.Process.Signal(nil) //nolint — nil signal checks liveness
}

func (r *ClaudeCodeRuntime) Pause(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = true

	if r.stdin == nil {
		return fmt.Errorf("agent not running")
	}
	msg := map[string]string{"type": "pause"}
	data, _ := json.Marshal(msg)
	_, err := fmt.Fprintf(r.stdin, "%s\n", data)
	return err
}

func (r *ClaudeCodeRuntime) Resume(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = false

	if r.stdin == nil {
		return fmt.Errorf("agent not running")
	}
	msg := map[string]string{"type": "resume"}
	data, _ := json.Marshal(msg)
	_, err := fmt.Fprintf(r.stdin, "%s\n", data)
	return err
}

func (r *ClaudeCodeRuntime) Kill(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	return r.cmd.Process.Kill()
}

func (r *ClaudeCodeRuntime) PID() int {
	return r.pid
}

func (r *ClaudeCodeRuntime) TokensUsed() int {
	return int(atomic.LoadInt64(&r.tokens))
}
