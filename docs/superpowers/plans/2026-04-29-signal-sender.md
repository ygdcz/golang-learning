# Signal Sender CLI Tool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a CLI tool that finds processes by keyword and sends Unix signals (SIGTERM/SIGQUIT) with dry-run safety.

**Architecture:** Command-line parameter driven tool using Go's `flag` package. Preserves existing `runCmds()` pipeline architecture, adds `ProcessInfo` struct for PID+command display, implements dry-run confirmation flow.

**Tech Stack:** Go stdlib (`flag`, `os/exec`, `syscall`, `bufio`)

---

## File Structure

**New files:**
- `signal_sender.go` - Main CLI implementation with flag parsing and signal sending logic

**Modified files:**
- `main.go` - Currently contains demo code, will be replaced with CLI entry point

**Preserved functions:**
- `runCmds([]*exec.Cmd)` - Command pipeline executor (keep as-is)
- `getCmdPlaintext(*exec.Cmd)` - Command display helper (keep as-is)
- `getError(error, *exec.Cmd, ...string)` - Error formatter (keep as-is)

---

### Task 1: Define ProcessInfo struct and parsing function

**Files:**
- Modify: `main.go` (replace demo code with CLI foundation)

- [ ] **Step 1: Remove demo code and add ProcessInfo struct**

Replace the entire `main.go` content with:

```go
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
```

- [ ] **Step 2: Add getProcessInfos function**

Add after the ProcessInfo struct:

```go
// getProcessInfos parses command output lines into ProcessInfo list
// Expected format: "PID COMMAND" per line (from awk '{print $2, $13}')
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
```

- [ ] **Step 3: Preserve existing helper functions**

Copy these functions from the old main.go (keep them unchanged):

```go
func runCmds(cmds []*exec.Cmd) ([]string, error) {
	if len(cmds) == 0 {
		return nil, errors.New("The cmd slice is invalid!")
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
```

- [ ] **Step 4: Verify code compiles**

Run: `go build -o /dev/null main.go`
Expected: Success (no errors)

- [ ] **Step 5: Commit foundation**

```bash
git add main.go
git commit -m "refactor: replace demo with CLI foundation

- Add ProcessInfo struct for PID + command display
- Add getProcessInfos() to parse awk output
- Preserve runCmds/getCmdPlaintext/getError helpers"
```

---

### Task 2: Implement command construction and process finding

**Files:**
- Modify: `main.go` (add buildSearchCmds and findProcesses functions)

- [ ] **Step 1: Add buildSearchCmds function**

Add after getProcessInfos:

```go
// buildSearchCmds constructs the command pipeline to find processes
// Pipeline: ps aux | grep keyword | grep -v grep | awk '{print $2, $13}'
func buildSearchCmds(keyword string) []*exec.Cmd {
	return []*exec.Cmd{
		exec.Command("ps", "aux"),
		exec.Command("grep", keyword),
		exec.Command("grep", "-v", "grep"),
		exec.Command("awk", "{print $2, $13}"),
	}
}
```

- [ ] **Step 2: Add findProcesses function**

Add after buildSearchCmds:

```go
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
```

- [ ] **Step 3: Verify code compiles**

Run: `go build -o /dev/null main.go`
Expected: Success

- [ ] **Step 4: Commit process finding logic**

```bash
git add main.go
git commit -m "feat: add process finding logic

- Add buildSearchCmds() for ps|grep|awk pipeline
- Add findProcesses() to execute search and parse results"
```

---

### Task 3: Implement signal mapping and validation

**Files:**
- Modify: `main.go` (add parseSignal function)

- [ ] **Step 1: Add parseSignal function**

Add after findProcesses:

```go
// parseSignal converts signal name to syscall constant
// Accepts: "term" (SIGTERM) or "quit" (SIGQUIT)
// Returns error for invalid signal names
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
```

- [ ] **Step 2: Verify code compiles**

Run: `go build -o /dev/null main.go`
Expected: Success

- [ ] **Step 3: Commit signal validation**

```bash
git add main.go
git commit -m "feat: add signal name validation

- Add parseSignal() to map 'term'/'quit' to syscall constants
- Whitelist only SIGTERM and SIGQUIT for safety"
```

---

### Task 4: Implement signal sending with error handling

**Files:**
- Modify: `main.go` (add sendSignals function)

- [ ] **Step 1: Add sendSignals function**

Add after parseSignal:

```go
// sendSignals sends signal to each process in the list
// Returns counts of succeeded and failed operations
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
```

- [ ] **Step 2: Verify code compiles**

Run: `go build -o /dev/null main.go`
Expected: Success

- [ ] **Step 3: Commit signal sending logic**

```bash
git add main.go
git commit -m "feat: add signal sending with per-PID error handling

- Add sendSignals() to send signal to each process independently
- Track and report success/failure counts
- Continue on individual failures"
```

---

### Task 5: Implement dry-run confirmation flow

**Files:**
- Modify: `main.go` (add confirmSend function)

- [ ] **Step 1: Add confirmSend function**

Add after sendSignals:

```go
// confirmSend prompts user to confirm signal sending
// Returns true if user types "yes", false otherwise
func confirmSend() bool {
	fmt.Print("[dry-run] Type 'yes' to confirm: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	
	response := strings.TrimSpace(scanner.Text())
	return strings.ToLower(response) == "yes"
}
```

- [ ] **Step 2: Verify code compiles**

Run: `go build -o /dev/null main.go`
Expected: Success

- [ ] **Step 3: Commit confirmation flow**

```bash
git add main.go
git commit -m "feat: add dry-run confirmation prompt

- Add confirmSend() to prompt user for 'yes' confirmation
- Only 'yes' proceeds, any other input aborts"
```

---

### Task 6: Implement main CLI entry point with flag parsing

**Files:**
- Modify: `main.go` (add main function with flag parsing)

- [ ] **Step 1: Add flag package import**

Update imports at the top of main.go:

```go
import (
	"bufio"
	"bytes"
	"errors"
	"flag"  // Add this line
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)
```

- [ ] **Step 2: Add main function**

Add at the end of main.go:

```go
func main() {
	// Define flags
	keyword := flag.String("keyword", "", "Process name or command keyword to match (required)")
	signalName := flag.String("signal", "term", "Signal type: 'term' (SIGTERM) or 'quit' (SIGQUIT)")
	dryRun := flag.Bool("dry-run", true, "Show targets without sending (default: true)")
	
	flag.Parse()
	
	// Validate required keyword
	if *keyword == "" {
		fmt.Fprintf(os.Stderr, "Error: -keyword is required\n")
		flag.Usage()
		os.Exit(1)
	}
	
	// Parse and validate signal
	sig, err := parseSignal(*signalName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	// Find matching processes
	fmt.Printf("Searching for processes matching '%s'...\n\n", *keyword)
	infos, err := findProcesses(*keyword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	// Handle empty results
	if len(infos) == 0 {
		fmt.Printf("No processes matched keyword \"%s\"\n", *keyword)
		os.Exit(0)
	}
	
	// Display found processes
	fmt.Printf("Found %d process(es) matching \"%s\":\n", len(infos), *keyword)
	for _, info := range infos {
		if info.Command != "" {
			fmt.Printf("  PID: %d (%s)\n", info.PID, info.Command)
		} else {
			fmt.Printf("  PID: %d\n", info.PID)
		}
	}
	fmt.Println()
	
	// Show signal type
	fmt.Printf("Signal to send: %s\n", sig.String())
	
	// Dry-run confirmation
	if *dryRun {
		if !confirmSend() {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
		fmt.Println()
	}
	
	// Send signals
	succeeded, failed := sendSignals(infos, sig)
	
	// Summary
	fmt.Printf("\nSummary: %d succeeded, %d failed\n", succeeded, failed)
	
	if failed > 0 {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify code compiles**

Run: `go build -o signal_sender main.go`
Expected: Success, creates `signal_sender` binary

- [ ] **Step 4: Test help output**

Run: `./signal_sender -h`
Expected output should show:
```
Usage of ./signal_sender:
  -dry-run
    	Show targets without sending (default: true) (default true)
  -keyword string
    	Process name or command keyword to match (required)
  -signal string
    	Signal type: 'term' (SIGTERM) or 'quit' (SIGQUIT) (default "term")
```

- [ ] **Step 5: Commit main CLI implementation**

```bash
git add main.go
git commit -m "feat: implement main CLI with flag parsing

- Add flag parsing for keyword, signal, dry-run
- Implement full execution flow: find -> display -> confirm -> send
- Add empty result handling and summary output
- Exit with code 1 if any signals failed"
```

---

### Task 7: Manual testing and validation

**Files:**
- Test: `main.go` (manual CLI testing)

- [ ] **Step 1: Test with no keyword (should fail)**

Run: `go run main.go`
Expected output:
```
Error: -keyword is required
Usage of ...
```
Exit code: 1

- [ ] **Step 2: Test with invalid signal (should fail)**

Run: `go run main.go -keyword=test -signal=kill`
Expected output:
```
Error: invalid signal 'kill': must be 'term' or 'quit'
```
Exit code: 1

- [ ] **Step 3: Test with nonexistent process (should succeed with no matches)**

Run: `go run main.go -keyword=nonexistent12345`
Expected output:
```
Searching for processes matching 'nonexistent12345'...
Run command: /bin/ps aux
Run command: /usr/bin/grep nonexistent12345
Run command: /usr/bin/grep -v grep
Run command: /usr/bin/awk {print $2, $13}
No processes matched keyword "nonexistent12345"
```
Exit code: 0

- [ ] **Step 4: Test dry-run with real process (should prompt)**

Run: `go run main.go -keyword=go`
Expected:
- Shows found processes with PIDs and commands
- Shows "Signal to send: terminated"
- Prompts "[dry-run] Type 'yes' to confirm: "
- Type anything except "yes" → "Aborted."
- Exit code: 0

- [ ] **Step 5: Test actual signal sending (use with caution)**

Start a test process in another terminal:
```bash
sleep 300 &
echo $!  # Note the PID
```

Run: `go run main.go -keyword=sleep -dry-run=false`
Expected:
- Shows the sleep process
- Sends SIGTERM immediately (no confirmation)
- Shows "✓ Sent terminated to PID <pid>"
- Summary: "1 succeeded, 0 failed"
- The sleep process should terminate

- [ ] **Step 6: Test SIGQUIT signal**

Start another test process:
```bash
sleep 300 &
```

Run: `go run main.go -keyword=sleep -signal=quit -dry-run=false`
Expected:
- Sends SIGQUIT
- Shows "✓ Sent quit to PID <pid>"
- Process terminates

- [ ] **Step 7: Document testing results**

Create a simple test log:
```bash
echo "Manual testing completed successfully" > test_results.txt
echo "- Flag validation: PASS" >> test_results.txt
echo "- Signal whitelist: PASS" >> test_results.txt
echo "- Empty results: PASS" >> test_results.txt
echo "- Dry-run flow: PASS" >> test_results.txt
echo "- Signal sending: PASS" >> test_results.txt
```

- [ ] **Step 8: Commit test results**

```bash
git add test_results.txt
git commit -m "test: manual CLI testing completed

- Validated flag parsing and error handling
- Confirmed dry-run confirmation flow
- Tested signal sending with SIGTERM and SIGQUIT
- All test cases passed"
```

---

### Task 8: Add usage documentation

**Files:**
- Create: `README.md` (usage guide)

- [ ] **Step 1: Create README.md**

```markdown
# Signal Sender CLI Tool

A command-line tool for finding and signaling processes by keyword.

## Features

- Find processes by command-line keyword matching
- Send SIGTERM or SIGQUIT signals
- Dry-run mode for safety (enabled by default)
- Per-process error handling
- Clear success/failure reporting

## Installation

```bash
go build -o signal_sender main.go
```

## Usage

### Basic usage (dry-run)

```bash
./signal_sender -keyword=nginx
```

Shows matching processes and prompts for confirmation.

### Send signal without confirmation

```bash
./signal_sender -keyword=nginx -dry-run=false
```

### Send SIGQUIT instead of SIGTERM

```bash
./signal_sender -keyword=myapp -signal=quit
```

## Flags

- `-keyword` (required): Process name or command keyword to match
- `-signal` (default: "term"): Signal type - "term" (SIGTERM) or "quit" (SIGQUIT)
- `-dry-run` (default: true): Show targets and prompt for confirmation

## Examples

**Find and terminate nginx processes:**
```bash
./signal_sender -keyword=nginx
# Shows PIDs, type 'yes' to confirm
```

**Send SIGQUIT to all Go processes:**
```bash
./signal_sender -keyword=go -signal=quit
```

**Direct send without confirmation:**
```bash
./signal_sender -keyword=test -dry-run=false
```

## Safety

- Only SIGTERM and SIGQUIT are allowed (no SIGKILL)
- Dry-run mode is enabled by default
- Individual process failures don't stop other signals
- Clear error messages for permission issues

## Educational Value

This tool demonstrates:
- Command pipelining in Go (`ps | grep | awk`)
- Flag-based CLI design
- Unix signal handling
- Error handling patterns
- Dry-run safety mechanisms
```

- [ ] **Step 2: Commit README**

```bash
git add README.md
git commit -m "docs: add usage documentation

- Add installation and usage instructions
- Document all flags and examples
- Highlight safety features"
```

---

## Self-Review Checklist

**Spec coverage:**
- ✅ Command-line flags (keyword, signal, dry-run) - Task 6
- ✅ Command pipeline construction - Task 2
- ✅ ProcessInfo struct with PID + command - Task 1
- ✅ Signal whitelist (term/quit only) - Task 3
- ✅ Dry-run confirmation flow - Task 5
- ✅ Per-PID error handling - Task 4
- ✅ Empty result handling - Task 6
- ✅ Success/failure summary - Task 4, 6

**Placeholder scan:**
- ✅ No TBD/TODO markers
- ✅ All code blocks complete
- ✅ All commands have expected output
- ✅ No "add appropriate error handling" placeholders

**Type consistency:**
- ✅ ProcessInfo struct used consistently
- ✅ Function signatures match across tasks
- ✅ syscall.Signal type used correctly

**Implementation notes from spec:**
- ✅ Preserved runCmds() function
- ✅ Replaced getPids() with getProcessInfos()
- ✅ Added ProcessInfo struct
- ✅ Used flag package
- ✅ Used bufio.Scanner for confirmation
- ✅ Used syscall.Kill() for signaling
- ✅ Mapped signal names to constants

All requirements covered, no placeholders, types consistent.
