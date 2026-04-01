# cast

[Typecast](https://typecast.ai?utm_source=cast&utm_medium=github) Text-to-Speech CLI

## Installation

```bash
brew install neosapience/tap/cast
```

Or with Go:

```bash
go install github.com/neosapience/cast@latest
```

## Setup

Get your API key at [https://typecast.ai/developers/api/api-key](https://typecast.ai/developers/api/api-key?utm_source=cast&utm_medium=github)

```bash
cast login
# or
cast login <api_key>
```

> **Pricing & rate limits:** See [https://typecast.ai/developers/api/billing](https://typecast.ai/developers/api/billing?utm_source=cast&utm_medium=github) for plan details.

## Quick Start

```bash
# 1. Log in
cast login

# 2. Try it immediately with the default voice
cast "Hello, world!"

# 3. Interactively browse and pick a voice
cast voices pick
cast voices pick --gender female

# Or list voices non-interactively
cast voices list --gender female --use-case Audiobook

# 4. Set your preferred voice as default (or press S in pick mode)
cast config set voice-id tc_xxx
```

## Usage

### Text to Speech

```bash
# Play immediately
cast "Hello, world!"

# Use a specific voice
cast "Hello, world!" --voice-id tc_xxx

# Save to file
cast "Hello, world!" --out hello.wav
cast "Hello, world!" --out hello.mp3 --format mp3
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--voice-id` | Voice ID (run `cast voices get tc_60e5426de8b95f1d3000d7b5` to inspect the default) | `tc_60e5426de8b95f1d3000d7b5` |
| `--model` | Model (`ssfm-v30`, `ssfm-v21`) | `ssfm-v30` |
| `--language` | Language code (ISO 639-3) | auto-detected |
| `--emotion` | Emotion type: `smart` (ssfm-v30 only), `preset` | |
| `--emotion-preset` | Preset emotion (`normal`, `happy`, `sad`, `angry`, `whisper`, `toneup`, `tonedown`) — requires `--emotion preset` | |
| `--emotion-intensity` | Emotion intensity (0.0–2.0) — requires `--emotion preset` | 1.0 |
| `--prev-text` | Previous sentence for context (`--emotion smart` only) | |
| `--next-text` | Next sentence for context (`--emotion smart` only) | |
| `--volume` | Volume (0–200) | 100 |
| `--pitch` | Pitch in semitones (–12 to +12) | 0 |
| `--tempo` | Tempo multiplier (0.5–2.0) | 1.0 |
| `--format` | Output format (`wav`, `mp3`) | `wav` |
| `--seed` | Random seed for reproducible output | |
| `--out` | Save to file instead of playing | |

### Models

| Model | Languages | Emotions | Latency |
|-------|-----------|----------|---------|
| `ssfm-v30` | 37 | 7 presets + smart emotion | Standard |
| `ssfm-v21` | 27 | 4 presets (normal, happy, sad, angry) | Low |

```bash
# Use ssfm-v21 for lower latency
cast "Hello, world!" --model ssfm-v21
```

### Emotions

There are two emotion systems, selected with `--emotion`.

#### Smart emotion (`--emotion smart`) — ssfm-v30 only

AI infers the appropriate emotion from the text. Optionally provide surrounding sentences for better context.

```bash
cast "I just got promoted!" --emotion smart

cast "I just got promoted!" --emotion smart \
  --prev-text "I have been working so hard this year." \
  --next-text "Let's celebrate tonight!"
```

#### Preset emotion (`--emotion preset`)

Choose a specific emotion. Use `--emotion-preset` to pick one, and `--emotion-intensity` to control its strength (0.0–2.0, default 1.0).

| Model | Available presets |
|-------|-------------------|
| `ssfm-v30` | `normal`, `happy`, `sad`, `angry`, `whisper`, `toneup`, `tonedown` |
| `ssfm-v21` | `normal`, `happy`, `sad`, `angry` |

```bash
# ssfm-v30 (default model)
cast "Hello, world!" --emotion preset --emotion-preset happy
cast "Hello, world!" --emotion preset --emotion-preset happy --emotion-intensity 2.0
cast "Hello, world!" --emotion preset --emotion-preset whisper --emotion-intensity 0.5

# ssfm-v21
cast "Hello, world!" --model ssfm-v21 --emotion preset --emotion-preset sad
```

### Voices

```bash
# Interactive voice picker — browse, preview, and select
cast voices pick
cast voices pick --gender female --age young_adult
cast voices pick --text "Custom preview sentence"
```

In the picker, type to filter, press Enter to select a voice, then:
- **P** — Preview with current model/emotion preset (cycle with ←/→)
- **E** — Preview with smart emotion (ssfm-v30 presets only)
- **S** — Set as default voice
- **C** — Copy voice ID to clipboard
- **Enter** — Confirm and print voice ID to stdout
- **Esc** — Go back

```bash
# Voice tournament — head-to-head elimination to find your favorite
cast voices tournament
cast voices tournament --gender female --size 16
cast voices tournament --text "Custom preview sentence"

```

In the tournament, listen to two voices and pick a winner:
- **P** — Preview voice 1
- **Q** — Preview voice 2
- **1** — Pick voice 1
- **2** — Pick voice 2

```bash
# Pick a random voice (great for experimentation)
cast voices random
cast voices random --gender female --age young_adult
cast "Hello!" --voice-id $(cast voices random --model ssfm-v30 --gender female)
```

```bash
# List voices (table output by default)
cast voices list
cast voices list --gender female
cast voices list --age young_adult
cast voices list --model ssfm-v30
cast voices list --use-case Audiobook   # Announcer, Anime, Audiobook, Conversational, Documentary, E-learning, Rapper, Game, Tiktok/Reels, News, Podcast, Voicemail, Ads

# Get full JSON output
cast voices list --json

# Get voice details (shows available styles, emotions, and languages)
cast voices get <voice_id>

# Find a voice and save it as default
cast voices list --use-case Audiobook --gender female
cast config set voice-id tc_xxx
```

### Auth

```bash
cast login              # prompts for API key
cast login <api_key>    # pass directly
cast logout             # remove saved key
```

### Config

Set default values so you don't have to pass flags every time.

```bash
cast config set voice-id tc_xxx
cast config set model ssfm-v21
cast config set volume 120

cast config list          # show current config
cast config unset volume  # remove a value
```

Available keys: `voice-id`, `model`, `language`, `emotion`, `emotion-preset`, `emotion-intensity`, `volume`, `pitch`, `tempo`, `format`

## Recipes

**Read text from a file:**
```bash
cast "$(cat script.txt)"
```

**Pipe from another command:**
```bash
echo "System is ready." | cast
cast "$(curl -s https://example.com/status.txt)"
```

**Batch generate audio files:**
```bash
cast "Chapter one." --out ch1.wav
cast "Chapter two." --out ch2.wav
cast "Chapter three." --out ch3.wav
```

**Audiobook with emotion:**
```bash
# Narration in a calm tone
cast "It was a dark and stormy night." --emotion preset --emotion-preset normal --emotion-intensity 0.5 --out intro.wav

# Exciting moment
cast "She opened the letter and gasped." --emotion preset --emotion-preset happy --emotion-intensity 1.5 --out climax.wav

# Sad farewell
cast "He watched the train disappear into the fog." --emotion preset --emotion-preset sad --out farewell.wav
```

**Smart emotion for natural delivery:**
```bash
cast "I can't believe we actually made it!" --emotion smart \
  --prev-text "We've been working on this for three years." \
  --next-text "Let's celebrate tonight!"
```

**Reproducible output with a fixed seed:**
```bash
cast "Hello, world!" --seed 42 --out hello.wav
# Running again with the same seed produces identical audio
cast "Hello, world!" --seed 42 --out hello2.wav
```

**Use a different language:**
```bash
cast "Bonjour le monde" --language fra
cast "こんにちは" --language jpn
```

**Adjust delivery style:**
```bash
cast "Buy now, limited time offer!" --tempo 1.3 --pitch 2
cast "Relax and take a deep breath." --tempo 0.85 --volume 90
```

## Configuration Priority

Settings are resolved in the following order, from highest to lowest priority:

```
--flag  >  environment variable  >  ~/.typecast/config.yaml  >  built-in default
```

For example, if `model` is set to `ssfm-v21` in the config file:

```bash
cast "Hello" --model ssfm-v30         # uses ssfm-v30 (flag wins)
TYPECAST_MODEL=ssfm-v30 cast "Hello"  # uses ssfm-v30 (env wins over config)
cast "Hello"                          # uses ssfm-v21 (from config)
```

## Environment Variables

Any option can be set via environment variable using the `TYPECAST_` prefix:

| Variable | Flag equivalent |
|----------|----------------|
| `TYPECAST_API_KEY` | `--api-key` |
| `TYPECAST_VOICE_ID` | `--voice-id` |
| `TYPECAST_MODEL` | `--model` |
| `TYPECAST_LANGUAGE` | `--language` |
| `TYPECAST_EMOTION` | `--emotion` |
| `TYPECAST_EMOTION_PRESET` | `--emotion-preset` |
| `TYPECAST_EMOTION_INTENSITY` | `--emotion-intensity` |
| `TYPECAST_FORMAT` | `--format` |
| `TYPECAST_VOLUME` | `--volume` |
| `TYPECAST_PITCH` | `--pitch` |
| `TYPECAST_TEMPO` | `--tempo` |
