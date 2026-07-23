#!/usr/bin/env python3
"""Generate a deterministic asciinema v2 cast that *types* the Sailwright
quick-start commands, for embedding on the website via asciinema-player.

The cast is generated (not recorded), so it is short, clean, and byte-for-byte
reproducible: running it twice yields an identical file, which lets CI verify the
committed cast is in sync (see .github/workflows/generate-demo.yml).

Input:  demo/quickstart.demo  (see that file for the tiny line syntax)
Output: demo/quickstart.cast  (asciinema v2; one JSON header line + event lines)

Usage:
    python3 demo/generate_cast.py [--output demo/quickstart.cast]
                                  [--cols 100] [--rows 28] [--speed 1.0]
"""

import argparse
import json
import os
import random
import sys

# --- Look & feel ------------------------------------------------------------

# ANSI SGR helpers (asciinema-player renders these).
RESET = "\x1b[0m"
GREEN = "\x1b[1;32m"
BLUE = "\x1b[1;34m"
BOLD = "\x1b[1m"
DIM = "\x1b[2m"

# Coloured shell prompt: user@host:path$
PROMPT = f"{GREEN}chris@sailwright{RESET}:{BLUE}~/dev-alchemy{RESET}$ "

# Fixed header timestamp keeps the output deterministic across runs.
FIXED_TIMESTAMP = 1700000000
SEED = 20240501

# Base timings (seconds), scaled by --speed.
CHAR_DELAY = 0.055        # per typed character
CHAR_JITTER = 0.035       # +/- randomness on typing, seeded (still deterministic)
ENTER_DELAY = 0.28        # after pressing Enter, before output appears
LINE_DELAY = 0.16         # between output lines
POST_OUTPUT_PAUSE = 0.35  # small beat after a command's output
START_DELAY = 0.4         # before the very first prompt


def colorize_output(line: str) -> str:
    """Add subtle colour to representative output lines."""
    if line.startswith("✅"):
        return f"{GREEN}{line}{RESET}"
    if line.startswith("🔧"):
        return f"{BOLD}{line}{RESET}"
    if line.startswith("Command finished successfully"):
        return f"{DIM}{line}{RESET}"
    return line


# --- Parsing ----------------------------------------------------------------

def parse_demo(path):
    """Parse the .demo file into (leading_pause, [steps]).

    Each step: {"cmd": str, "outputs": [str, ...], "pause_after": float}.
    """
    leading_pause = 0.0
    steps = []
    current = None

    with open(path, encoding="utf-8") as fh:
        for raw in fh:
            line = raw.rstrip("\n")
            stripped = line.strip()

            if stripped.startswith("#"):
                continue
            if stripped.startswith("pause "):
                try:
                    value = float(stripped.split(None, 1)[1])
                except (IndexError, ValueError):
                    value = 0.0
                if current is None:
                    leading_pause += value
                else:
                    current["pause_after"] += value
                continue
            if line.startswith("$ "):
                current = {"cmd": line[2:], "outputs": [], "pause_after": 0.0}
                steps.append(current)
                continue
            if stripped == "":
                # Blank line inside a command becomes a blank output line.
                if current is not None:
                    current["outputs"].append("")
                continue
            # Anything else is an output line (tolerate an optional "> " prefix).
            text = line[2:] if line.startswith("> ") else (line[1:] if line == ">" else line)
            if current is not None:
                current["outputs"].append(text)

    return leading_pause, steps


# --- Cast generation --------------------------------------------------------

def generate(leading_pause, steps, cols, rows, speed):
    rng = random.Random(SEED)
    events = []
    t = 0.0

    def emit(data):
        # asciinema v2 event: [time, "o", data]
        events.append([round(t, 6), "o", data])

    char_delay = CHAR_DELAY / speed
    char_jitter = CHAR_JITTER / speed
    enter_delay = ENTER_DELAY / speed
    line_delay = LINE_DELAY / speed

    t += START_DELAY / speed
    emit(PROMPT)

    for i, step in enumerate(steps):
        if i == 0 and leading_pause:
            t += leading_pause / speed

        # Type the command, one character per event.
        for ch in step["cmd"]:
            t += max(0.01, char_delay + rng.uniform(-char_jitter, char_jitter))
            emit(ch)

        # Enter.
        t += enter_delay
        emit("\r\n")

        # Output lines.
        for out in step["outputs"]:
            t += line_delay
            emit(colorize_output(out) + "\r\n")

        # Beat, then the next prompt.
        t += (POST_OUTPUT_PAUSE / speed) + (step["pause_after"] / speed)
        emit(PROMPT)

    header = {
        "version": 2,
        "width": cols,
        "height": rows,
        "timestamp": FIXED_TIMESTAMP,
        "env": {"SHELL": "/bin/bash", "TERM": "xterm-256color"},
    }
    return header, events


def write_cast(path, header, events):
    lines = [json.dumps(header, ensure_ascii=False)]
    lines += [json.dumps(evt, ensure_ascii=False) for evt in events]
    data = "\n".join(lines) + "\n"
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(data)


def main(argv=None):
    here = os.path.dirname(os.path.abspath(__file__))
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", default=os.path.join(here, "quickstart.demo"))
    parser.add_argument("--output", default=os.path.join(here, "quickstart.cast"))
    parser.add_argument("--cols", type=int, default=100)
    parser.add_argument("--rows", type=int, default=28)
    parser.add_argument("--speed", type=float, default=1.0,
                        help="scale all delays (>1 = faster)")
    args = parser.parse_args(argv)

    if args.speed <= 0:
        parser.error("--speed must be > 0")

    leading_pause, steps = parse_demo(args.input)
    if not steps:
        print(f"error: no commands found in {args.input}", file=sys.stderr)
        return 1

    header, events = generate(leading_pause, steps, args.cols, args.rows, args.speed)
    write_cast(args.output, header, events)

    duration = events[-1][0] if events else 0.0
    print(f"Wrote {args.output}: {len(steps)} commands, "
          f"{len(events)} events, ~{duration:.1f}s playback.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
