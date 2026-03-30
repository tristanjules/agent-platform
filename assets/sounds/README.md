# Notification Sounds

Place audio files here for DUSTY notifications.

## Required files

- `transmission.wav` — Played on incoming mesh transmission. Should be short (< 2 seconds), retro sci-fi style. WAV format, mono, 22050 Hz recommended.

## Generating transmission.wav

On the Pi, you can generate a simple beep tone using sox:

```bash
sox -n -r 22050 -c 1 transmission.wav synth 0.3 sine 880 fade 0 0.3 0.1
```

Or download a suitable retro radio sound effect in WAV format.
