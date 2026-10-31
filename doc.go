// Package speakjet is an emulation of the SpeakJet speech and sound
// synthesizer used in the AtariVox.
//
// It was written from scratch from the SpeakJet User's Manual (Magnevation,
// revision 1.1, 2004) and from recordings of two AtariVoxes playing every
// sound and control, and from nothing else; the aim is the smallest model that
// matches those recordings. It is a line-for-line port of the C library
// (speakjet.c in github.com/andrew-davie/SpeakJet, branch rebuild) and renders
// the same samples. That repository's docs/technical-manual.md describes the
// chip as measured, and docs/design.md the model.
//
// What the chip was found to do:
//
//   - It makes samples at about 8026 Hz (the two units: 8039 and 8013; the
//     manual says 8192), its arithmetic in 8000ths, and holds each sample until
//     the next; the AtariVox then low-passes the held signal with two poles at
//     about 1609 Hz (Q 0.462). The output here is at 48 kHz, so the hold's
//     images are carried as the real thing's are.
//   - Voicing: a period of floor(rate / Pitch) + 1 ticks, exactly periodic. At
//     the start of each period three formant sines restart at one phase under
//     an envelope that steps to full at once and falls linearly over about
//     7 ms.
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
//     pairs.
//   - Bend moves every oscillator by 44 Hz a step; Stress, Relax and code 18 do
//     nothing measurable.
//
// The tables (tables.go, generated) are about 2.6 KB: 72 allophones of 14
// bytes (kind, length, three formants with levels, a noise band, a Repeat copy
// length, a level), a second frame for the five voiced stops that have one,
// and the 55 effects as a byte stream of parts, each holding only the tones
// and fields it uses (fxDecode). The numbers were read off the recordings,
// then refined inside each sound's measured structure with two scores that
// compare the library to the chip the way two chips compare to each other.
package speakjet
