// Package speakjet is an emulation of the SpeakJet speech and sound
// synthesizer used in the AtariVox.
//
// It is written from the SpeakJet User's Manual (Magnevation, revision 1.1,
// 2004) and from general phonetics. Where two independent sets of
// measurements of the real chip agree, those replace guesses: the
// openFPGA-7800 AtariVox engine's, and those behind Stella's SpeakJet engine.
// They set some formants of nine allophones (see tables.go). No code or tables
// were taken from either.
//
// The manual documents the chip's synthesizer but not the MSA "sound
// component database" that drives it, so the package is in two parts:
//
// Synthesizer (synth.go). Follows the manual's description of the 5-channel
// synthesizer: oscillators 1 to 3 (0 to 3999Hz, volume 0 to 31) summed in a
// mixer limited to 63; oscillators 4 and 5 with a shared distortion control
// that spreads white noise around their frequency; a sixth oscillator that
// envelopes the others (saw, sine, triangle or square); a master volume of 0
// to 127; output at 8192 samples per second through a low pass filter (the
// chip's, measured: see outputFIR in synth.go).
// Choices not in the manual: oscillators 1 to 3 are sines whose phase restarts
// at each envelope period (so they ring like formants excited by a glottal
// pulse), and distortion is applied as random frequency modulation.
//
// MSA (msa.go, tables.go). The allophone, control and sound effect behaviour.
// Control ranges and defaults, pause behaviour, the 64 byte buffer and the
// Buffer Half Full status are from the manual. The oscillator settings
// for each allophone are derived from published formant measurements of
// English speech, except for the formants that the two sets of measurements
// of the chip agree on. Recordings of a real chip set how much longer than
// the manual's lengths speech sounds play, Fast, the level of the noise sounds
// and M1 (see steadyLength in tables.go); and the Speed curve, how pauses follow
// Speed, three pause lengths, each allophone's length, the shape of the
// fricatives' hiss and the other sound effects, which are traced from the
// recording (see allophoneLength and fxPoints in tables.go, and speedFactor in
// msa.go); and the output filter, five vowels' formants, each allophone's level,
// Bend, Stress, Relax and the pitch (see outputFIR in synth.go,
// allophoneGainDB in tables.go, and bendHz, voiceHz and expand in msa.go).
package speakjet
