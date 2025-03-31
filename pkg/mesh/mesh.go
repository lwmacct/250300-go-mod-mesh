package mesh

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Config 定义了可选参数的配置结构体
type Config struct {
	Cmd     string        `note:"cmd" default:"-"`
	Shell   string        `note:"shell" default:"bash"`
	Timeout time.Duration `note:"timeout" default:"60s"`
	Env     []string      `note:"envVars" default:"system"`
}

// NewConfig 返回一个包含默认值的 Config 实例
func NewConfig() *Config {
	return &Config{
		Cmd:     "",               // 默认命令
		Shell:   "bash",           // 默认 Shell
		Timeout: 60 * time.Second, // 默认超时时间
		Env:     os.Environ(),     // 默认环境变量
	}
}

type Ts struct {
	Cfg      *Config
	stdout   string
	stderr   string
	exitCode int
}

func New(cmdStr string, config ...*Config) *Ts {
	var cfg *Config
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	} else {
		cfg = NewConfig()
	}
	if cfg.Cmd == "" {
		cfg.Cmd = cmdStr
	}
	return &Ts{
		Cfg: cfg,
	}
}

// SetEnv 追加或覆盖某些环境变量
func (t *Ts) SetEnv(envVars map[string]string) *Ts {
	newEnv := make([]string, 0, len(t.Cfg.Env)+len(envVars))
	existingKeys := make(map[string]bool, len(envVars))
	for k := range envVars {
		existingKeys[k] = false
	}

	for _, kv := range t.Cfg.Env {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k, _ := parts[0], parts[1]
		if _, ok := envVars[k]; ok {
			continue
		}
		newEnv = append(newEnv, kv)
	}
	for k, v := range envVars {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", k, v))
	}

	t.Cfg.Env = newEnv
	return t
}

func (t *Ts) GetEnv() []string {
	envCopy := make([]string, len(t.Cfg.Env))
	copy(envCopy, t.Cfg.Env)
	return envCopy
}

func (t *Ts) Lines() []string {
	trimSpace := t.Stdout()
	if trimSpace == "" {
		return []string{}
	}
	return strings.Split(trimSpace, "\n")
}

func (t *Ts) Fields(expectedLen int) [][]string {
	lines := t.Lines()
	result := make([][]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < expectedLen {
			padding := make([]string, expectedLen-len(fields))
			fields = append(fields, padding...)
		} else if len(fields) > expectedLen {
			fields = fields[:expectedLen]
		}
		result = append(result, fields)
	}
	return result
}

func (t *Ts) ToMap(expectedLen int) map[string][]string {
	fields := t.Fields(expectedLen)
	data := make(map[string][]string, len(fields))
	for _, field := range fields {
		if len(field) > 0 {
			key := field[0]
			data[key] = field[1:]
		}
	}
	return data
}

func (t *Ts) Exec() *Ts {
	if t.exitCode != 0 && t.exitCode != -1 {
		return t
	}

	shell := t.Cfg.Shell
	if _, err := exec.LookPath(shell); err != nil {
		shell = "sh"
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.Cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell)
	cmd.Env = t.Cfg.Env
	cmd.Stdin = strings.NewReader(t.Cfg.Cmd)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.stderr = err.Error()
	}

	if cmd.ProcessState != nil {
		t.exitCode = cmd.ProcessState.ExitCode()
	} else {
		t.exitCode = -1
	}

	t.stdout = strings.TrimSpace(out.String())
	t.stderr = strings.TrimSpace(stderr.String())

	if ctx.Err() == context.DeadlineExceeded {
		t.stderr = fmt.Sprintf("Error: Command execution timed out after %s.", t.Cfg.Timeout)
		t.exitCode = -1
	}
	return t
}

// 状态

func (t *Ts) Stdout() string {
	return t.stdout
}

func (t *Ts) Stderr() string {
	return t.stderr
}

func (t *Ts) ExitCode() int {
	return t.exitCode
}

func (t *Ts) Show() map[string]any {
	ret := map[string]any{
		"stdout":   t.stdout,
		"stderr":   t.stderr,
		"exitCode": t.exitCode,
		"envVars":  t.Cfg.Env,
		"cmdStr":   t.Cfg.Cmd,
	}
	return ret
}
