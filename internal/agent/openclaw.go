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

// OpenClawRuntime manages an openclaw CLI subprocess.
// OpenClaw also communicates via JSON on stdin/stdout.
type OpenClawRuntime struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	outputCh    chan string
	controlCh   chan string
	doneCh      chan struct{}
	paused      bool
	chatReplyCh chan string
	pid         int
	tokens      int64

	OutputHandler  func(line string)
	ControlHandler func(prefix string, payload json.RawMessage)
}

func NewOpenClawRuntime(outputHandler func(string), controlHandler func(string, json.RawMessage)) *OpenClawRuntime {
	return &OpenClawRuntime{
		outputCh:    make(chan string, 256),
		controlCh:   make(chan string, 64),
		doneCh:      make(chan struct{}),
		chatReplyCh: make(chan string, 1),
		OutputHandler:  outputHandler,
		ControlHandler: controlHandler,
	}
}

func (r *OpenClawRuntime) Spawn(ctx context.Context, config AgentConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.CommandContext(ctx, "openclaw",
		"--json-mode",
		"--system", config.SystemPrompt,
		"--work-dir", config.WorkDir,
	)

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
		return fmt.Errorf("start openclaw: %w", err)
	}

	r.cmd = cmd
	r.stdin = stdin
	r.pid = cmd.Process.Pid
	r.doneCh = make(chan struct{})

	go r.readStdout(stdout)
	go r.dispatchOutput()
	go func() {
		cmd.Wait() //nolint
		close(r.doneCh)
	}()

	return nil
}

func (r *OpenClawRuntime) readStdout(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		isControl := false
		for _, prefix := range controlPrefixes {
			if strings.HasPrefix(line, prefix) {
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

func (r *OpenClawRuntime) dispatchOutput() {
	for {
		select {
		case line, ok := <-r.outputCh:
			if !ok {
				return
			}
			r.tryParseTokens(line)
			r.mu.Lock()
			paused := r.paused
			r.mu.Unlock()
			if paused {
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

func (r *OpenClawRuntime) tryParseTokens(line string) {
	var msg struct {
		Type   string `json:"type"`
		Tokens int    `json:"tokens_used"`
	}
	if err := json.Unmarshal([]byte(line), &msg); err == nil && msg.Tokens > 0 {
		atomic.AddInt64(&r.tokens, int64(msg.Tokens))
	}
}

func (r *OpenClawRuntime) handleControl(line string) {
	if r.ControlHandler == nil {
		return
	}
	for _, prefix := range controlPrefixes {
		if strings.HasPrefix(line, prefix) {
			rest := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			raw := json.RawMessage("{}")
			if len(rest) > 0 {
				raw = json.RawMessage(rest)
			}
			go r.ControlHandler(prefix, raw)
			return
		}
	}
}

func (r *OpenClawRuntime) SendTask(ctx context.Context, issue store.Issue) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stdin == nil {
		return fmt.Errorf("not running")
	}
	msg := map[string]interface{}{
		"type":        "task",
		"issue_id":    issue.ID.String(),
		"title":       issue.Title,
		"description": issue.Description,
	}
	data, _ := json.Marshal(msg)
	_, err := fmt.Fprintf(r.stdin, "%s\n", data)
	return err
}

func (r *OpenClawRuntime) SendChat(ctx context.Context, message string) (string, error) {
	r.mu.Lock()
	if r.stdin == nil {
		r.mu.Unlock()
		return "", fmt.Errorf("not running")
	}
	r.mu.Unlock()
	msg := map[string]interface{}{"type": "chat", "content": message}
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

func (r *OpenClawRuntime) Heartbeat(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil || r.cmd.Process == nil {
		return fmt.Errorf("not running")
	}
	return r.cmd.Process.Signal(nil) //nolint
}

func (r *OpenClawRuntime) Pause(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = true
	msg := map[string]string{"type": "pause"}
	data, _ := json.Marshal(msg)
	_, err := fmt.Fprintf(r.stdin, "%s\n", data)
	return err
}

func (r *OpenClawRuntime) Resume(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paused = false
	msg := map[string]string{"type": "resume"}
	data, _ := json.Marshal(msg)
	_, err := fmt.Fprintf(r.stdin, "%s\n", data)
	return err
}

func (r *OpenClawRuntime) Kill(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	return r.cmd.Process.Kill()
}

func (r *OpenClawRuntime) PID() int { return r.pid }

func (r *OpenClawRuntime) TokensUsed() int {
	return int(atomic.LoadInt64(&r.tokens))
}
