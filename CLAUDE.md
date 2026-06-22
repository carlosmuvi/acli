# CLAUDE.md

Guidance for Claude Code working in this repository.

## What this is

`acli` is an interactive **Go + Bubble Tea** TUI for managing Android emulators
and watching logcat, meant to run alongside Claude Code. See `README.md` for
user-facing docs and keybindings.

## Commands

```sh
go build ./...        # build
go test ./...         # all tests (no Android SDK or TTY required)
go vet ./...          # vet
gofmt -l .            # must print nothing; CI fails on unformatted files
go run .              # run (exits via doctor if adb is missing)
```

CI (`.github/workflows/ci.yml`) runs gofmt-check, vet, test, and build on every
push and PR. Releases are cut by tagging `vX.Y.Z`, which triggers GoReleaser
(`.goreleaser.yaml`) to publish binaries and a Homebrew cask to
`carlosmuvi/homebrew-tap`.

## Architecture

```
main.go                 discover tooling -> doctor (hard-fail on missing adb) -> launch TUI
internal/sdk            locate `android`, `adb`, `emulator`, SDK root
internal/doctor         preflight health checks (adb is the only hard requirement)
internal/android        Backend interface + impls, logcat stream, device parsing
internal/logmirror      rotating per-device log files under .acli/logs/
internal/tui            Bubble Tea UI: root model, emulator list, logcat view, doctor overlay
```

### Backend is hybrid (`internal/android`)
- `Backend` interface (`backend.go`) abstracts emulator mgmt + screenshots.
  `Select()` prefers the agent-first `android` CLI (`backend_cli.go`) when
  present, else raw `adb`/`emulator` (`backend_adb.go`).
- **Logcat is NOT part of the Backend interface** — the `android` CLI has no
  logcat, so it always streams via `adb logcat` (`logcat.go`).
- Screenshots always go through `adb -s <serial>` (serial-accurate); the
  `android` CLI's `screen capture` has no device selector.

## Conventions & gotchas

- **Never name a Go file `*_<GOOS>.go`** (e.g. `backend_android.go`). Go treats
  the suffix as a build constraint, so `android`/`linux`/`darwin` suffixes
  silently exclude the file from other platforms. The android-CLI backend lives
  in `backend_cli.go` for this reason.
- **The published `android` CLI docs are unreliable.** Verify flags against the
  actual binary (`android <cmd> --help`). Known truths: cold boot is `--cold`
  (no `--cold-boot`); wipe-data/headless aren't supported (delegate to the
  `emulator` binary); `screen capture` has no `--device`.
- **TUI list state:** the emulator view keeps `avds` and `devices` as
  independent source-of-truth lists (`setAVDs`/`setDevices` in `emulators.go`).
  Do not reconstruct one from the current rows — that previously let transient,
  model-named devices leak in as phantom rows during the kill/shutdown race.
- **Emulator name resolution:** running emulators are matched to their AVD via
  `adb -s <serial> emu avd name`, attempted even while offline/booting so a
  booting emulator collapses onto its AVD row.

## Testing the TUI without a device

The model is tested by driving it through `Update`/`View` with injected messages
(`internal/tui/model_test.go`) — no TTY or SDK needed. When a real run is needed
without a device, fake `adb`/`android` shell scripts on `PATH` plus a pty that
answers the terminal background/cursor queries work (Bubble Tea stalls before
first render otherwise). Prefer the message-driven approach for regressions.
