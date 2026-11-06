***NOTE from @JetSetIlly**: I am hosting this under my name only because it is so closely related to `Gopher2600`.
The original impetus for the project was for the creation of an `AtariVox` emulation that would remove the need for `Festival`.
I challenged Andrew Davie to use `Claude` to create this new emulation from first principles using only the `SpeakJet` manual.
I have had no hand in the project except a small change to the API so that it could fit better with the needs of
`Gopher2600`.*

# SpeakJet

An emulation of the Magnevation SpeakJet speech and sound synthesizer, the
chip in the AtariVox peripheral for the Atari 2600, as a small Go package. The
Gopher2600 emulator uses it for its AtariVox.

It was written from scratch from the SpeakJet User's Manual and recordings of
two real AtariVoxes (an original AtariVox and an AtariVox+), and from nothing
else. It is a line-for-line port of the C library at
[andrew-davie/SpeakJet](https://github.com/andrew-davie/SpeakJet) and renders the same samples to within float rounding.

- Pure Go, no dependencies, no global state: any number of chips side by side.
- Takes the raw command stream and gives the Buffer Half Full and Speaking
  lines, so it drops straight into an emulator's AtariVox.
- Output: mono float32 samples at 48 kHz (`SampleRate`).
- Tables about 5 KB; the package adds about 50 KB to a stripped binary.

## Using it

```go
import "github.com/jetsetilly/speakjet"

sj := speakjet.NewSpeakJet()

// commands as the AtariVox receives them: a code and, for the commands that
// take one, its data byte (CommandSize says how many bytes a code takes)
sj.Write(21, 100) // Speed 100
sj.Write(183, 0)  // HO
sj.Write(135, 0)  // OH

// pull audio: mono float32 in -1..1 at speakjet.SampleRate (48000 Hz)
buf := make([]float32, 256)
for sj.Render(buf) {
	// play or resample buf
}
```

| | |
|---|---|
| `NewSpeakJet()` | A chip in its power-on state |
| `Write(code, data)` | One command into the 64-byte input buffer; false if it is too full to take it |
| `CommandSize(code)` | How many bytes a command takes, its data byte included |
| `Render(buf)` | Fill `buf` with samples; true while the chip is producing sound |
| `BufferHalfFull()` | The Buffer Half Full line: more than 32 bytes in use |
| `Speaking()` | The Speaking line |
| `Reset()` | The reset line: the buffer cleared, the controls back to their defaults |

The AtariVox's READY pin goes low while the SpeakJet's buffer is filling, so
drive it as `!sj.BufferHalfFull()`. The input buffer drains as you render, so
render in step with the emulated time to keep READY's timing right.

The output is at 48 kHz because the chip holds each of its own ~8 kHz samples
until the next, and the AtariVox's sound carries the resulting images up to
about 14 kHz; resample it as you would any other source. The output peaks at
about 0.5.

A chip isn't safe for concurrent use: write and render from one goroutine, or
lock around them.

## How close it is

Every sound is compared with recordings of both AtariVoxes by three scores,
lower is better: the whole-sound spectrum error and E (spectrum, time-frequency
picture and level over time together), both in rms dB on the takes of each
sound alone, and P, a perceptual score averaged over every recording of the
sound (1191 recordings in all). At 0.3.0:

```
              sounds   spectrum dB   E dB    P
allophones        72          2.62   4.31   0.61
effects           55          2.62   4.40   0.92
all sounds       127          2.62   4.35   0.74
every recording, item by item               0.89
```

The worst sounds are effects (254 M2, 218 A8, 210 A0, 235 C5, 219 A9). Known
and not yet modelled: Bend below zero, the linear Volume law, the pulse timbre
of 235 C5 and the first four robots, and the death sound in Man Goes Down. The
C library's README and `docs/technical-manual.md` list them in full.

## How it works

The package documentation (`go doc github.com/jetsetilly/speakjet`) summarises
what the chip was found to do. The C library's `docs/technical-manual.md`
describes the chip as measured and `docs/design.md` the model, measurement by
measurement; its `CHANGELOG.md` has the history.

## Version

0.3.0: the from-data rebuild (every table and most rules re-measured from the
chip), with the same API as before.
