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
//   - Voicing: a period of floor(8000 / Pitch) + 1 ticks, exactly periodic; an
//     effect's pulse runs ceil(8000 / rate) ticks, one shorter wherever
//     8000 / rate is whole; twelve rates (44 51 53 56 63 77 87 88 92 104 127
//     151) run one tick longer. The reset default is the manual's Pitch 88, one
//     of them (92 ticks on both units). At the start of
//     each period three formant sines restart at one phase under an envelope
//     that steps to full at once and falls linearly over about 7 ms.
//   - Noise: white noise through a two-pole resonance, a centre and width per
//     sound (the chip's hiss is Gaussian noise, not a thrown tone).
//   - Lengths are measured per sound, independent of context and Pitch, and
//     linear in Speed; Fast is x0.77 and Slow x1.22, stops unchanged. Inside a
//     run of sounds the voice runs on (no fades), each sound cut short a
//     little; only a run's edges ramp, 35 ms in and 30 ms out.
//   - Repeat: each copy lasts the sound's measured copy length; a stop's copy
//     is the stop again, closure and release; a glide's copy glides back.
//   - Effects pulse at Pitch plus an offset of their own; plain tones hold
//     pitch and level and end in a 55 ms fade; D0 to D11 are the standard DTMF
//     pairs. Under every effect's tones runs a faint noise bed that follows
//     each tone's change per tick, smoothed (up to about 1.5 kHz).
//   - The chip reads the command after a sound early, at that sound's release:
//     a Speed or Volume command read so changes the rest of the sound playing.
//   - Bend moves every oscillator by 44 Hz a step; Stress, Relax and code 18 do
//     nothing measurable.
//
// The tables (tables.go, generated) are about 2.7 KB: 72 allophones of 14
// bytes (kind, length, three formants with levels, a noise band, a Repeat copy
// length, a level), a second frame for the five voiced stops that have one,
// and the 55 effects as a byte stream of parts, each holding only the tones
// and fields it uses (fxDecode). The numbers were read off the recordings,
// then refined inside each sound's measured structure with two scores that
// compare the library to the chip the way two chips compare to each other.
package speakjet
