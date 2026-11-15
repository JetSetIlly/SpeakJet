***NOTE from @JetSetIlly**: I am hosting this under my name only because it is so closely related to `Gopher2600`.
The original impetus for the project was for the creation of an `AtariVox` emulation that would remove the need for `Festival`.
I challenged Andrew Davie to use `Claude` to create this new emulation from first principles using only the `SpeakJet` manual.
I have had no hand in the project except a small change to the API so that it could fit better with the needs of
`Gopher2600`.*

# SpeakJet

An emulation of the Magnevation SpeakJet speech and sound synthesizer, the
chip in the AtariVox peripheral for the Atari 2600, as a small Go package. The
Gopher2600 emulator uses it for its AtariVox.

It is a line-for-line port of the C library at
[andrew-davie/SpeakJet](https://github.com/andrew-davie/SpeakJet), with the same version numbers (0.5.x), and
renders the same samples to within float rounding. Every rule in it is taken from recordings of two real
AtariVoxes (an original AtariVox and an AtariVox+) and checked against both.

- Pure Go, no dependencies, no global state: any number of chips side by side.
- Takes the raw command stream and gives the Buffer Half Full and Speaking
  lines, so it drops straight into an emulator's AtariVox.
- Output: mono float32 samples at 48 kHz (`SampleRate`).
- Tables about 7 KB; the package adds about 100 KB to a stripped binary.

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

Every allophone and sound effect is scored against the chip's own recordings:
four takes of each sound alone, two from each AtariVox. The score, X, compares
spectrograms at six resolutions (1.3 ms to 43 ms windows) with each take's
background noise taken off, and is calibrated on the chip itself: the four takes
are scored against each other and a sound's distance is divided by that spread.
**X = 1.0 means "as far from the chip as another recording of the chip"**; lower
is closer.

0.5.1 against 0.4.3, both scored the same way on the same recordings:

```
                                      0.4.3   0.5.1
allophones (72), mean X                1.07    0.93   57 closer, none further
sound effects (55), mean X             1.22    0.98   52 closer, 1 further (237 C7)
joins (88 recorded pairs and phrases)  1437     514   mean cost; 56 closer, 15 further
lengths 15 ms or more off (88 items)     24      14
```

The furthest sounds are effects (238 C8, 207 R7, 239 C9, 206 R6), then the
voiced stops 174 DE and 171 BO. Known and not yet modelled: a Pitch command
inside a phrase takes effect a few sounds late on the chip; the chip's level
dies away over about 100 ms after an unvoiced sound; a voiced effect's pulse
rings on between pulses; Bend below zero; the linear Volume law. The C
library's README, `docs/scores.md` (every sound's score) and
`docs/technical-manual.md` list them in full.

## How it works

The package documentation (`go doc github.com/jetsetilly/speakjet`) summarises
what the chip was found to do. The C library's `docs/technical-manual.md`
describes the chip as measured and `docs/design.md` the model, measurement by
measurement; its `CHANGELOG.md` has the history.
