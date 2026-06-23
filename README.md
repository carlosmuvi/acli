# acli

An interactive terminal UI for managing Android emulators and watching logcat —
built to run side-by-side with Claude Code. Pick an emulator, launch/kill it,
open a live filterable logcat pane, and grab screenshots, all from the keyboard.

Every log line is also mirrored to a file under `.acli/logs/`, so an agent
working in an adjacent pane can `grep` the history that scrolls past on screen.

## Requirements

- **adb** (required — logcat depends on it). Install the Android SDK
  platform-tools (Android Studio, or via `sdkmanager "platform-tools"`). acli
  auto-discovers the SDK at `$ANDROID_HOME`, `$ANDROID_SDK_ROOT`, or the default
  location (`~/Library/Android/sdk` on macOS, `~/Android/Sdk` on Linux), so you
  usually don't need to set anything.
- **`android` CLI** (optional, preferred) — the agent-first CLI from
  https://developer.android.com/tools/agents. If present, it's used for emulator
  management and screenshots; otherwise acli falls back to `adb`/`emulator`.
- An AVD (create one in Android Studio's Device Manager or with
  `android emulator create`) or a USB-connected device.

On launch, **doctor** reports what's available and how to fix anything missing.
A missing `adb` aborts startup; everything else is a soft note.

## Install

Homebrew (once released):

```sh
brew install carlosmuvi/tap/acli
```

With Go:

```sh
go install github.com/carlosmuvi/acli@latest
```

Or build from source:

```sh
go build -o acli .
./acli
```

Check the build: `acli version`.

## Keys

**Emulator list**

| Key | Action |
| --- | --- |
| ↑/↓ (or j/k) | move |
| enter | launch (or open logcat if running) |
| c | cold boot |
| w | wipe + launch |
| K | kill |
| l | open logcat |
| s | screenshot → `.acli/shots/` |
| r | refresh |
| ? | doctor / help |
| q | quit |

**Logcat**

| Key | Action |
| --- | --- |
| f or / | filter by tag/message |
| L | cycle minimum level (V→D→I→W→E→F) |
| space | pause / resume autoscroll |
| x | clear buffer |
| esc | back to emulator list |

## Layout

| Path | Purpose |
| --- | --- |
| `internal/sdk` | locate `android`, `adb`, `emulator`, SDK root |
| `internal/doctor` | preflight health checks |
| `internal/android` | backend interface + `android`-CLI / adb impls + logcat stream |
| `internal/logmirror` | rotating per-device log files (`.acli/logs/`) |
| `internal/tui` | Bubble Tea UI: emulator list, logcat view, doctor overlay |

## Runtime data

- `.acli/logs/<device>.log` — mirrored logcat, rotated at ~10 MB, 5 generations kept.
- `.acli/shots/<device>-<time>.png` — screenshots.

Both are git-ignored.
