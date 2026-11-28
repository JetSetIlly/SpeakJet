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
- Tables about 7.2 KB; the package adds about 100 KB to a stripped binary.

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
three takes of each sound alone from one AtariVox in one session (VoxProbe 17), with a
fourth from the same unit in an earlier session. The score, X, is a
weighted mean of five views (level by third octave and 20 ms block, the whole
spectrum's shape, spectrograms at two resolutions, level over time), each
divided by what a real take scores on it when scored against the other three.
So **X = 1.0 is as close as one real recording of the chip is to the others**;
lower is closer. The render gets the recording's background noise added before
it is scored.

0.5.8 against 0.5.7 and 0.4.3, all three rescored on this measure (since 0.5.8 the target is one
unit; figures published before 0.5.8 are on older scales). No sound is further than in 0.5.7:

```
                      n   mean X                change on   median X              over 1.0
                          0.4.3  0.5.7  0.5.8   0.4.3  0.5.7   0.4.3  0.5.7  0.5.8     0.4.3  0.5.7  0.5.8
allophones           72    1.30   1.07   1.01    -22 %    -5 %    1.24   1.05   0.98       58     40     33
sound effects        55    2.10   1.50   1.45    -31 %    -4 %    2.05   1.43   1.39       55     55     54
all sounds          127    1.64   1.26   1.20    -27 %    -4 %    1.46   1.16   1.13      113     95     87
```

Connected speech, 44 phrase items recorded on both units, each scored against
the other unit's recording (lower is closer): one unit against the other 143.9
mean (138.3 median), 0.5.4 264 (185), 0.5.6 235.7 (175.3), 0.5.7 236.3 (173.7), 0.5.8 234.3 (173.7).

The furthest sounds are 208 R8, 238 C8, 207 R7, 239 C9, 204 R4 and 254 M2.
Known and not yet modelled: Bend below zero; the linear Volume law. The C
library's README, `docs/scores.md` (every sound's score) and
`docs/technical-manual.md` (section 13) list them in full.

## How it works

The package documentation (`go doc github.com/jetsetilly/speakjet`) summarises
what the chip was found to do. The C library's `docs/technical-manual.md`
describes the chip as measured and `docs/design.md` the model, measurement by
measurement; its `CHANGELOG.md` has the history.
