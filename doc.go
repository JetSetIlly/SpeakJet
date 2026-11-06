// Package speakjet is an emulation of the SpeakJet speech and sound
// synthesizer used in the AtariVox.
//
// It was written from scratch from the SpeakJet User's Manual (Magnevation,
// revision 1.1, 2004) and from recordings of two AtariVoxes playing every
// sound and control, and from nothing else; the aim is the smallest model that
// matches those recordings. It is a line-for-line port of the C library
// (speakjet.c in github.com/andrew-davie/SpeakJet) and renders
// the same samples. That repository's docs/technical-manual.md describes the
// chip as measured, and docs/design.md the model.
//
// What the chip was found to do:
//
//   - It makes samples at about 8026 Hz (the two units: 8047 and 8014; the
//     manual says 8192), its arithmetic in 8000ths, and holds each sample until
//     the next; the AtariVox's output stage then filters the held signal as an
//     analogue circuit does: two real poles (about 1.1 and 2.4 kHz) and a real
//     zero at 14 kHz, solved exactly over each held sample. The output here is
//     at 48 kHz, so the hold's images are carried as the real thing's are.
//   - Voicing: a 16-bit phase counted up by floor(Pitch * 65536 / 8000) a tick,
//     a period of floor(65536 / step) + 1 ticks, exactly periodic; it gives the
//     twelve rates (44 51 53 56 63 77 87 88 92 104 127 151) that run one tick
//     longer than floor(8000 / Pitch) + 1. An effect's pulse runs
//     ceil(8000 / rate) ticks, plus that tick at those rates. The reset default
//     is the manual's Pitch 88 (92 ticks on both units). At the start of
//     each period three formant sines restart at one phase under an envelope
//     that steps to full at once and falls linearly over about 7 ms.
//   - Noise: white noise through a two-pole resonance, a centre and width per
//     sound (the chip's hiss is Gaussian noise, not a thrown tone); the
//     allophones' noise high-passed twice at 400 Hz, WH with a white floor,
//     SE SO SH starting on a burst of low noise.
//   - Lengths are measured per sound, independent of context and Pitch, and
//     linear in Speed; Fast is x2/3 and Slow x4/3 (Slow also on the voiced
//     stops); a sound straight after the same sound plays as a Repeat copy.
//     Inside a run of sounds the voice runs on (no fades), each sound cut short
//     a little; only a run's edges ramp, 35 ms in and 30 ms out (unvoiced
//     fricatives 3 ms, nasals in 25). Stops have their own closures, a voice
//     bar with weak upper formants, and louder onsets.
//   - Repeat: each copy lasts the sound's measured copy length; a stop's copy
//     is the stop again, closure and release; a glide's copy glides back.
//   - Effects pulse at Pitch plus an offset of their own; plain tones hold
//     pitch and level and end in a 55 ms fade, many starting on a 5 ms click;
//     D0 to D11 are the standard DTMF
//     pairs. Under every effect's tones runs a faint noise bed that follows
//     each tone's change per tick, smoothed (up to about 1.5 kHz).
//   - The chip reads the command after a sound early, at that sound's release:
//     a Speed or Volume command read so changes the rest of the sound playing.
//   - Bend moves every oscillator by 44 Hz a step; Stress, Relax and code 18 do
//     nothing measurable.
//
// The tables (tables.go, generated) are about 5 KB: 72 allophones of 26
// bytes (kind, length, three formants with levels, a noise band, a Repeat copy
// length, a level, and the sound's own structure: onset, closure, release
// level, white floor, attack and release, burst, bar, noise corner), a second
// frame for the voiced stops that have one, the 55 effects as a byte stream of
// parts, each holding only the tones and fields it uses (fxDecode), and a
// 4-byte structure row per effect (attack, release, starting click). The numbers were read off the recordings,
// then refined inside each sound's measured structure with two scores that
// compare the library to the chip the way two chips compare to each other.
package speakjet
