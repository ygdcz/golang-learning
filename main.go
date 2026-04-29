package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ProcessInfo holds PID and command name for a target process
type ProcessInfo struct {
	PID     int
	Command string
}

// getProcessInfos parses command output lines into ProcessInfo list
// Expected format: "PID COMMAND" per line (from awk '{print $1, $2}')
func getProcessInfos(lines []string) ([]ProcessInfo, error) {
	var infos []ProcessInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on whitespace: first field is PID, rest is command
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid PID '%s': %w", parts[0], err)
		}

		// Join remaining parts as command (may contain spaces)
		command := ""
		if len(parts) > 1 {
			command = strings.Join(parts[1:], " ")
		}

		infos = append(infos, ProcessInfo{
			PID:     pid,
			Command: command,
		})
	}
	return infos, nil
}

// buildSearchCmds constructs the command pipeline to find processes
// Pipeline: ps -eo pid,comm | grep -F -- keyword | awk '{print $1, $2}'
func buildSearchCmds(keyword string) []*exec.Cmd {
	return []*exec.Cmd{
		exec.Command("ps", "-eo", "pid,comm"),
		exec.Command("grep", "-F", "--", keyword),
		exec.Command("awk", "{print $1, $2}"),
	}
}

// findProcesses searches for processes matching keyword
// Returns ProcessInfo list or error
func findProcesses(keyword string) ([]ProcessInfo, error) {
	cmds := buildSearchCmds(keyword)
	lines, err := runCmds(cmds)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	infos, err := getProcessInfos(lines)
	if err != nil {
		return nil, fmt.Errorf("failed to parse process info: %w", err)
	}

	return infos, nil
}

func runCmds(cmds []*exec.Cmd) ([]string, error) {
	if len(cmds) == 0 {
		return nil, errors.New("cmd slice is empty")
	}
	first := true
	var output []byte
	var err error
	for _, cmd := range cmds {
		fmt.Printf("Run command: %v\n", getCmdPlaintext(cmd))
		if !first {
			var stdinBuf bytes.Buffer
			stdinBuf.Write(output)
			cmd.Stdin = &stdinBuf
		}
		var stdoutBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		if err = cmd.Start(); err != nil {
			return nil, getError(err, cmd)
		}
		if err = cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && filepath.Base(cmd.Path) == "grep" && exitErr.ExitCode() == 1 {
				output = stdoutBuf.Bytes()
				if first {
					first = false
				}
				continue
			}
			return nil, getError(err, cmd)
		}
		output = stdoutBuf.Bytes()
		if first {
			first = false
		}
	}
	var lines []string
	var outputBuf bytes.Buffer
	outputBuf.Write(output)
	for {
		line, err := outputBuf.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, getError(err, nil)
			}
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}

func getCmdPlaintext(cmd *exec.Cmd) string {
	var buf bytes.Buffer
	buf.WriteString(cmd.Path)
	for _, arg := range cmd.Args[1:] {
		buf.WriteRune(' ')
		buf.WriteString(arg)
	}
	return buf.String()
}

func getError(err error, cmd *exec.Cmd, extraInfo ...string) error {
	var errMsg string
	if cmd != nil {
		errMsg = fmt.Sprintf("%s  [%s %v]", err, (*cmd).Path, (*cmd).Args)
	} else {
		errMsg = fmt.Sprintf("%s", err)
	}
	if len(extraInfo) > 0 {
		errMsg = fmt.Sprintf("%s (%v)", errMsg, extraInfo)
	}
	return errors.New(errMsg)
}

func parseSignal(sigName string) (syscall.Signal, error) {
	switch strings.ToLower(sigName) {
	case "term":
		return syscall.SIGTERM, nil
	case "quit":
		return syscall.SIGQUIT, nil
	default:
		return 0, fmt.Errorf("invalid signal '%s': must be 'term' or 'quit'", sigName)
	}
}

func sendSignals(infos []ProcessInfo, sig syscall.Signal) (succeeded, failed int) {
	sigName := sig.String()
	for _, info := range infos {
		err := syscall.Kill(info.PID, sig)
		if err != nil {
			fmt.Printf("✗ Failed to send %s to PID %d: %v\n", sigName, info.PID, err)
			failed++
		} else {
			fmt.Printf("✓ Sent %s to PID %d\n", sigName, info.PID)
			succeeded++
		}
	}
	return succeeded, failed
}

func main() {
	// CLI entrypoint - to be implemented in Task 6
	_ = bufio.NewScanner(os.Stdin)
	_ = syscall.SIGTERM
}
