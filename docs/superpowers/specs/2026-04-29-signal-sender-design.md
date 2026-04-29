# Signal Sender CLI Tool - Design Specification

**Date**: 2026-04-29\
**Author**: Claude (with user approval)\
**Status**: Approved

## Overview

A command-line tool that finds processes by keyword and sends Unix signals (SIGTERM or SIGQUIT) to them. Built for educational purposes to demonstrate Go's signal handling, command pipelining, and CLI design patterns.

## Design Decisions

### 1. Program Structure

**Architecture**: Command-line parameter driven tool

**Core Components**:

- `runCmds([]*exec.Cmd)`: Chains commands via pipes (existing, preserved)
- `getProcessInfos([]string)`: Parses output into `[]ProcessInfo{PID int, Command string}`
- `main()`: Orchestrates flag parsing, command construction, and signal sending

**Command-line Flags**:

- `-keyword` (string, required): Process name or command-line keyword to match
- `-signal` (string, default "term"): Signal type - "term" (SIGTERM) or "quit" (SIGQUIT)
- `-dry-run` (bool, default true): Show targets without sending (safety first)

**Execution Flow**:

1. Parse flags and validate inputs
2. Construct command pipeline: `ps aux | grep <keyword> | grep -v grep | awk '{print $2, $13}'`
3. Execute `runCmds()` to get output lines
4. Parse lines with `getPids()` to extract PID list
5. Display found PIDs and signal type
6. If dry-run mode: wait for user confirmation ("yes")
7. Send signal to each PID individually
8. Report success/failure summary

### 2. Safety Mechanisms

**Dry-run Mode** (default enabled):

- Display all target PIDs and signal type
- Prompt user to type "yes" to proceed
- Any other input (including Enter) aborts operation
- Prevents accidental signal sending

**Empty Result Handling**:

- If no processes match keyword: print clear message and exit
- No signal sending attempted
- Exit code 0 (not an error, just no matches)

**Per-PID Error Handling**:

- Each PID processed independently
- Single failure doesn't stop remaining PIDs
- Errors logged with reason (permission denied, process not found, etc.)
- Final summary shows: X succeeded, Y failed

**Signal Whitelist**:

- Only "term" and "quit" accepted
- Invalid signal values cause immediate error and exit
- Prevents accidental SIGKILL or other dangerous signals
- Error message suggests valid options

### 3. Command Pipeline & Output

**Command Construction**:

```go
cmds := []*exec.Cmd{
    exec.Command("ps", "aux"),
    exec.Command("grep", keyword),
    exec.Command("grep", "-v", "grep"),  // Exclude grep itself
    exec.Command("awk", "{print $2}"),   // Extract PID column
}
```

**Key Improvements from Original**:

- Removed `grep -v "go run"` filter (was causing false negatives)
- Kept `grep -v "grep"` to avoid matching the grep process itself
- Simplified to essential pipeline steps

**Output Format**:

When processes found:

```
Found 3 process(es) matching "nginx":
  PID: 1234 (nginx: master process)
  PID: 5678 (nginx: worker process)
  PID: 9012 (nginx: cache manager)

Signal to send: SIGTERM
[dry-run] Type 'yes' to confirm: _
```

After sending signals:

```
✓ Sent SIGTERM to PID 1234
✓ Sent SIGTERM to PID 5678
✗ Failed to send to PID 9012: operation not permitted

Summary: 2 succeeded, 1 failed
```

When no processes found:

```
No processes matched keyword "nonexistent"
```

## Usage Examples

**Dry-run (default)**:

```bash
go run main.go -keyword=nginx
# Shows PIDs, waits for confirmation
```

**Direct send** (skip confirmation):

```bash
go run main.go -keyword=nginx -dry-run=false
```

**Send SIGQUIT instead**:

```bash
go run main.go -keyword=myapp -signal=quit
```

## Non-Goals

- Multi-signal support (only TERM and QUIT)
- PID file input (only keyword matching)
- Remote process signaling (local only)
- Signal scheduling or retry logic
- Process monitoring after signal sent

## Educational Value

This design teaches:

1. **Command pipelining**: Chaining Unix commands in Go
2. **Flag-based CLI design**: Standard Go flag package patterns
3. **Error handling**: Per-item error handling in batch operations
4. **Safety patterns**: Dry-run mode, input validation, whitelisting
5. **Unix signals**: Practical signal sending with proper permissions

## Implementation Notes

- Preserve existing `runCmds()` function
- Replace `getPids()` with `getProcessInfos()` returning PID + command text
- Add `ProcessInfo` struct: `{ PID int; Command string }`
- Add `flag` package for parameter parsing
- Add `bufio.Scanner` for reading user confirmation
- Use `syscall.Kill()` for signal sending (not `os.Process.Signal()`)
- Map signal names to constants: "term" → `syscall.SIGTERM`, "quit" → `syscall.SIGQUIT`

