package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
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
// Pipeline: ps -eo pid,args | grep -F -- keyword
// Uses 'args' to match full command line (supports keywords with spaces)
func buildSearchCmds(keyword string) []*exec.Cmd {
	return []*exec.Cmd{
		exec.Command("ps", "-eo", "pid,args"),
		exec.Command("grep", "-F", "--", keyword),
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

	// Filter out current process to prevent self-termination
	currentPID := os.Getpid()
	var filtered []ProcessInfo
	for _, info := range infos {
		if info.PID != currentPID {
			filtered = append(filtered, info)
		}
	}

	return filtered, nil
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

func chooseTargets(infos []ProcessInfo) ([]ProcessInfo, bool) {
	if len(infos) == 1 {
		return infos, true
	}

	fmt.Println("Matched processes:")
	for i, info := range infos {
		fmt.Printf("  [%d] PID=%d COMMAND=%s\n", i+1, info.PID, info.Command)
	}
	fmt.Println("Select targets: comma-separated indexes (e.g. 1,3) or 'all'")
	fmt.Print("Your choice: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		}
		return nil, false
	}

	input := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if input == "all" {
		return infos, true
	}

	parts := strings.Split(input, ",")
	seen := make(map[int]bool)
	selected := make([]ProcessInfo, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(infos) {
			fmt.Fprintf(os.Stderr, "Invalid selection: %s\n", part)
			return nil, false
		}
		if seen[idx] {
			continue
		}
		seen[idx] = true
		selected = append(selected, infos[idx-1])
	}

	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, "No targets selected")
		return nil, false
	}

	return selected, true
}

func confirmSend() bool {
	fmt.Print("[dry-run] Type 'yes' to confirm: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		}
		return false
	}
	response := strings.TrimSpace(scanner.Text())
	return strings.ToLower(response) == "yes"
}

func main() {
	keyword := flag.String("keyword", "", "Process keyword to match (required)")
	signalName := flag.String("signal", "term", "Signal to send: term or quit")
	dryRun := flag.Bool("dry-run", true, "Dry run mode (ask for confirmation before sending)")
	flag.Parse()

	if strings.TrimSpace(*keyword) == "" {
		fmt.Fprintln(os.Stderr, "Error: -keyword is required")
		flag.Usage()
		os.Exit(1)
	}

	sig, err := parseSignal(*signalName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	infos, err := findProcesses(*keyword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(infos) == 0 {
		fmt.Printf("No processes matched keyword: %s\n", *keyword)
		os.Exit(0)
	}

	targets, ok := chooseTargets(infos)
	if !ok {
		fmt.Println("Aborted")
		os.Exit(0)
	}

	fmt.Printf("Signal to send: %s\n", sig.String())

	if *dryRun {
		if !confirmSend() {
			fmt.Println("Aborted")
			os.Exit(0)
		}
	}

	succeeded, failed := sendSignals(targets, sig)
	fmt.Printf("Summary: total=%d succeeded=%d failed=%d\n", len(targets), succeeded, failed)

	if failed > 0 {
		os.Exit(1)
	}
}
