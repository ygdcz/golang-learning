# Signal Sender CLI

A small CLI tool to find processes by keyword and send a Unix signal to matching PIDs.

## What it does

- Searches running processes with a required `-keyword`
- Shows matched `PID` and command
- Sends a selected signal to all matches
- Uses confirmation by default (`-dry-run=true`) before signaling

## Build

```bash
go build -o signal_sender ./main.go
```

## Usage

```bash
./signal_sender -keyword <process-keyword> [flags]
```

### Example: default dry-run (safe default)

```bash
./signal_sender -keyword nginx
```

Behavior:
- Lists matched processes
- Shows the signal to send (`terminated` / SIGTERM by default)
- Prompts: `Type 'yes' to confirm`
- Sends signal only after confirmation

### Example: dry-run=false

```bash
./signal_sender -keyword nginx -dry-run=false
```

Behavior:
- Sends signal immediately to all matched processes

### Example: signal=quit

```bash
./signal_sender -keyword nginx -signal=quit
```

Behavior:
- Uses `SIGQUIT` instead of default `SIGTERM`
- Still asks for confirmation unless `-dry-run=false`

## Flags

- `-keyword` (required): process keyword to match
- `-signal` (optional, default: `term`): signal to send
  - allowed values: `term`, `quit`
- `-dry-run` (optional, default: `true`): confirmation mode before sending

## Safety notes

- Only `term` and `quit` are accepted for `-signal`
- `-dry-run` defaults to `true` to reduce accidental signaling
- With `-dry-run=true`, signals are only sent after explicit `yes` confirmation
