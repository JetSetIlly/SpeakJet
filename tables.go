package speakjet

import "math"

// part is one stage of a sound: the synthesizer glides from wherever it is to
// the target frame over glideMs and then holds it for the rest of ms
type part struct {
	ms      float64
	glideMs float64
	target  frame
}

// autoGlide in a part's glideMs means the default transition time for speech
// sounds (see msa.expand())
const autoGlide = -1

// sound is an allophone or sound effect. speech sounds take their pitch from
// the Pitch control, their duration from the Speed control, and are affected
// by Fast, Slow, Stress and Relax. sound effects use the pitch and timing in
// their parts as they are
type sound struct {
	name   string
	speech bool

	// how much longer the chip plays a speech sound than the manual's length,
	// which the parts add up to
	length float64

	parts []part
}

// the chip's lengths for speech sounds, measured from a recording of a real
// SpeakJet playing Alex Herbert's vox_test ROM (2004), compared sound by sound
// with this library playing the same codes. at the default speed the steady
// sounds (vowels, nasals, liquids, fricatives) last about 1.5 times the
// manual's length (EH 2.07, IH 1.6, NGE 1.43, and SE 1.67, TH 1.42, NE 1.14
// when slow), diphthongs about 1.2 times (EYIY 1.0, IHWW 1.5, and OHIY 1.2,
// AXRR 1.3 when fast), and voiceless stops what the manual gives (TT 1.0). at
// speed 127 the same steady and stop lengths and the speed curve give the
// recording's syllable length to within a few milliseconds
const (
	steadyLength    = 1.5
	diphthongLength = 1.2
	stopLength      = 1.0
)

// the duration of a sound at the default speed. for allophones this is the
// "Msec." column of the manual's allophone table
func (s sound) ms() float64 {
	var t float64
	for _, p := range s.parts {
		t += p.ms
	}
	return t
}

// formant targets. frequencies are approximate values for an adult male
// speaker, from published measurements of English vowels and consonants,
// except where two independent measurements of the real chip agree (see
// "measured on the chip" below)
type formants [3]float64

// measured on the chip: where two independent measurements of a sound's formant
// agree, it is set to their mean. they agree when they are within 10% of each
// other or within 88Hz (the spacing of the voice's harmonics at the default
// pitch, about as finely as a formant can be read from a recording). the
// measurements are the openFPGA-7800 AtariVox engine's, read from recordings
// of games speaking (docs/ATARIVOX.md, MIT licence), and those behind Stella's
// SpeakJet engine (feature/speakjet branch). formants where the two disagree,
// or that only one of them measured, keep their published values:
//
//	OW: F1 F2      AX: F1 F2 F3   RR: F1 F2 F3   UX: F1 F2   IH: F1 F2 F3
//	MM: F1 F3      EY: F1 F2 F3   EH: F1 F2      AW: F2

var (
	fIY = formants{270, 2290, 3010}
	fIH = formants{322, 1774, 2490}
	fEY = formants{508, 1820, 2425}
	fEH = formants{600, 1666, 2480}
	fAY = formants{660, 1720, 2410}
	fAX = formants{521, 1510, 2395}
	fUX = formants{613, 1328, 2390}
	fOH = formants{730, 1090, 2440}
	fAW = formants{590, 998, 2540}
	fOW = formants{512, 891, 2350}
	fUH = formants{440, 1020, 2240}
	fUW = formants{300, 870, 2240}
	fAH = formants{700, 1220, 2600}
	fER = formants{480, 1350, 1650}
	fRR = formants{439, 1318, 1720}
	fLE = formants{360, 1300, 2700}
	fLO = formants{400, 900, 2500}
	fWW = formants{300, 650, 2200}
	fYY = formants{280, 2200, 2900}

	// consonant loci: where the formants are at a consonant's place of
	// articulation. the following sound glides out of them
	lLabial      = formants{250, 900, 2100}
	lLabialBack  = formants{250, 750, 2100}
	lDental      = formants{300, 1400, 2600}
	lAlveolar    = formants{250, 1750, 2700}
	lAlveolarBak = formants{250, 1500, 2600}
	lPalatal     = formants{280, 1900, 2500}
	lVelar       = formants{250, 2300, 2800}
	lVelarBack   = formants{250, 1300, 2300}
	lGlottal     = formants{500, 1500, 2500}
)

// frame constructors

func voicedFrame(f formants, v1, v2, v3 float64) frame {
	return frame{
		freq:   [5]float64{f[0], f[1], f[2], 0, 0},
		vol:    [5]float64{v1, v2, v3, 0, 0},
		wave:   envSaw,
		voiced: true,
	}
}

func vowelFrame(f formants) frame  { return voicedFrame(f, 24, 18, 10) }
func nasalFrame(f formants) frame  { return voicedFrame(f, 22, 7, 4) }
func liquidFrame(f formants) frame { return voicedFrame(f, 22, 13, 7) }

// silence keeps the formant frequencies so that the next sound glides from them
func silentFrame(f formants) frame {
	return frame{freq: [5]float64{f[0], f[1], f[2], 0, 0}, wave: envSaw}
}

// noise from oscillators 4 and 5. oscillators 1 to 3 are silent but sit at the
// locus, so the next sound glides out of it
func noiseFrame(locus formants, n1, n2, v1, v2, dist float64) frame {
	return frame{
		freq:       [5]float64{locus[0], locus[1], locus[2], n1, n2},
		vol:        [5]float64{0, 0, 0, v1, v2},
		distortion: dist,
		wave:       envSaw,
	}
}

// voiced noise: a weak voiced murmur with half-enveloped noise
func voicedNoiseFrame(f formants, v1, n1, n2, nv1, nv2, dist float64) frame {
	return frame{
		freq:       [5]float64{f[0], f[1], f[2], n1, n2},
		vol:        [5]float64{v1, v1 * 0.4, v1 * 0.2, nv1, nv2},
		distortion: dist,
		wave:       envSaw,
		voiced:     true,
		halfEnv:    true,
	}
}

// hush turns the noise oscillators (4 and 5) of a frame down by db decibels
func hush(f frame, db float64) frame {
	g := math.Pow(10, -db/20)
	f.vol[3] *= g
	f.vol[4] *= g
	return f
}

// how much quieter the chip's noise is than the levels first chosen here.
// measured from the vox_test recording (see steadyLength) by comparing each
// noise sound's level against the vowel beside it, on the chip and here. on
// the chip hiss sits 16 to 23dB under the vowels; here it was level with them.
// the corrections: SE 23dB, CH (the SH frame) 16.4dB, TH 11.8dB, FF 13.7dB,
// stop bursts 19.5dB (TT 23dB, KE 16dB) and aspiration 14.5dB (TT). sounds the
// recording doesn't have take the mean of the fricatives that it does
const (
	seDB         = 23.0
	shDB         = 16.4
	thDB         = 11.8
	ffDB         = 13.7
	fricativeDB  = 17.0
	burstDB      = 19.5
	aspirationDB = 14.5
)

// sound constructors

func steady(name string, ms float64, f frame) sound {
	return sound{name: name, speech: true, length: steadyLength, parts: []part{{ms: ms, glideMs: autoGlide, target: f}}}
}

// a diphthong holds its first target, slides to its second and briefly holds it
func diphthong(name string, ms float64, a, b frame) sound {
	return sound{name: name, speech: true, length: diphthongLength, parts: []part{
		{ms: ms * 0.3, glideMs: autoGlide, target: a},
		{ms: ms * 0.55, glideMs: ms * 0.55, target: b},
		{ms: ms * 0.15, glideMs: 0, target: b},
	}}
}

// stop describes a plosive: a closure (silence, or a voice bar for voiced
// stops), a burst of noise at the release, aspiration for voiceless stops, and
// for voiced stops the onset of voicing at the locus
type stop struct {
	ms, closure, burst, aspiration float64
	voiced                         bool
	locus                          formants
	burstFreq                      [2]float64
	burstVol                       [2]float64
	aspFreq                        [2]float64
}

func plosive(name string, st stop) sound {
	var close frame
	if st.voiced {
		close = voicedFrame(formants{180, st.locus[1], st.locus[2]}, 8, 0, 0)
	} else {
		close = silentFrame(st.locus)
	}

	burst := hush(noiseFrame(st.locus, st.burstFreq[0], st.burstFreq[1], st.burstVol[0], st.burstVol[1], 255), burstDB)
	if st.voiced {
		burst.vol[0] = 6
		burst.voiced = true
		burst.halfEnv = true
	}

	parts := []part{
		{ms: st.closure, glideMs: 6, target: close},
		{ms: st.burst, glideMs: 2, target: burst},
	}

	rest := st.ms - st.closure - st.burst
	if st.voiced {
		if rest > 0 {
			parts = append(parts, part{ms: rest, glideMs: rest, target: voicedFrame(st.locus, 14, 6, 3)})
		}
	} else if rest > 0 {
		asp := hush(noiseFrame(st.locus, st.aspFreq[0], st.aspFreq[1], 10, 8, 230), aspirationDB)
		parts = append(parts, part{ms: rest, glideMs: 4, target: asp})
	}

	return sound{name: name, speech: true, length: stopLength, parts: parts}
}

// an affricate is a stop released into a fricative
func affricate(name string, st stop, fric frame, fricMs float64) sound {
	s := plosive(name, st)
	s.parts = append(s.parts, part{ms: fricMs, glideMs: 5, target: fric})
	return s
}

// frames for the fricatives, reused by the affricates and TS
var (
	frSE = hush(noiseFrame(lAlveolar, 3800, 3990, 20, 16, 160), seDB)
	frSO = hush(noiseFrame(lAlveolarBak, 3500, 3900, 20, 14, 160), seDB)
	frSH = hush(noiseFrame(lPalatal, 2300, 3100, 22, 16, 200), shDB)
	frZH = hush(voicedNoiseFrame(formants{260, 1800, 2500}, 14, 2400, 3200, 14, 10, 200), fricativeDB)
)

// allophones, codes 128 to 199. durations are from the manual
var allophones = [...]sound{
	steady("IY", 70, vowelFrame(fIY)),
	steady("IH", 70, vowelFrame(fIH)),
	steady("EY", 70, vowelFrame(fEY)),
	steady("EH", 70, vowelFrame(fEH)),
	steady("AY", 70, vowelFrame(fAY)),
	steady("AX", 70, vowelFrame(fAX)),
	steady("UX", 70, vowelFrame(fUX)),
	steady("OH", 70, vowelFrame(fOH)),
	steady("AW", 70, vowelFrame(fAW)),
	steady("OW", 70, vowelFrame(fOW)),
	steady("UH", 70, vowelFrame(fUH)),
	steady("UW", 70, vowelFrame(fUW)),
	steady("MM", 70, nasalFrame(formants{268, 1000, 2187})),
	steady("NE", 70, nasalFrame(formants{250, 1800, 2600})),
	steady("NO", 70, nasalFrame(formants{250, 1300, 2400})),
	steady("NGE", 70, nasalFrame(formants{250, 2100, 2700})),
	steady("NGO", 70, nasalFrame(formants{250, 1100, 2300})),
	steady("LE", 70, liquidFrame(fLE)),
	steady("LO", 70, liquidFrame(fLO)),
	steady("WW", 70, liquidFrame(fWW)),
	steady("RR", 70, liquidFrame(fRR)),
	diphthong("IYRR", 200, vowelFrame(fIY), liquidFrame(fER)),
	diphthong("EYRR", 200, vowelFrame(fEH), liquidFrame(fER)),
	diphthong("AXRR", 190, vowelFrame(formants{500, 1400, 1750}), liquidFrame(fER)),
	diphthong("AWRR", 200, vowelFrame(fOH), liquidFrame(fER)),
	diphthong("OWRR", 185, vowelFrame(fAW), liquidFrame(fER)),
	diphthong("EYIY", 165, vowelFrame(fEY), vowelFrame(fIY)),
	diphthong("OHIY", 200, vowelFrame(fOH), vowelFrame(fIY)),
	diphthong("OWIY", 225, vowelFrame(fAW), vowelFrame(fIY)),
	diphthong("OHIH", 185, vowelFrame(fOH), vowelFrame(fIH)),
	diphthong("IYEH", 170, vowelFrame(fYY), vowelFrame(fEH)),
	diphthong("EHLL", 140, vowelFrame(fAX), liquidFrame(fLO)),
	diphthong("IYUW", 180, vowelFrame(fYY), vowelFrame(fUW)),
	diphthong("AXUW", 170, vowelFrame(fAH), vowelFrame(fUW)),
	diphthong("IHWW", 170, vowelFrame(formants{330, 1300, 2300}), vowelFrame(fUW)),
	diphthong("AYWW", 200, vowelFrame(fAY), vowelFrame(fUH)),
	diphthong("OWWW", 131, vowelFrame(fOW), vowelFrame(formants{330, 700, 2250})),
	affricate("JH", stop{ms: 30, closure: 25, burst: 5, voiced: true, locus: lPalatal,
		burstFreq: [2]float64{2600, 3300}, burstVol: [2]float64{12, 8}}, frZH, 40),
	steady("VV", 70, hush(voicedNoiseFrame(formants{220, 1100, 2100}, 16, 3000, 3800, 8, 5, 255), fricativeDB)),
	steady("ZZ", 70, hush(voicedNoiseFrame(formants{240, 1500, 2500}, 14, 3700, 3950, 14, 10, 160), fricativeDB)),
	steady("ZH", 70, frZH),
	steady("DH", 70, hush(voicedNoiseFrame(formants{250, 1300, 2400}, 16, 3000, 3800, 7, 5, 255), fricativeDB)),
	plosive("BE", stop{ms: 45, closure: 25, burst: 8, voiced: true, locus: lLabial,
		burstFreq: [2]float64{700, 1200}, burstVol: [2]float64{10, 6}}),
	plosive("BO", stop{ms: 45, closure: 25, burst: 8, voiced: true, locus: lLabialBack,
		burstFreq: [2]float64{600, 1000}, burstVol: [2]float64{10, 6}}),
	plosive("EB", stop{ms: 10, closure: 5, burst: 5, voiced: true, locus: lLabial,
		burstFreq: [2]float64{700, 1200}, burstVol: [2]float64{8, 5}}),
	plosive("OB", stop{ms: 10, closure: 5, burst: 5, voiced: true, locus: lLabialBack,
		burstFreq: [2]float64{600, 1000}, burstVol: [2]float64{8, 5}}),
	plosive("DE", stop{ms: 45, closure: 25, burst: 8, voiced: true, locus: lAlveolar,
		burstFreq: [2]float64{3300, 3900}, burstVol: [2]float64{12, 8}}),
	plosive("DO", stop{ms: 45, closure: 25, burst: 8, voiced: true, locus: lAlveolarBak,
		burstFreq: [2]float64{3000, 3700}, burstVol: [2]float64{12, 8}}),
	plosive("ED", stop{ms: 10, closure: 5, burst: 5, voiced: true, locus: lAlveolar,
		burstFreq: [2]float64{3300, 3900}, burstVol: [2]float64{10, 6}}),
	plosive("OD", stop{ms: 10, closure: 5, burst: 5, voiced: true, locus: lAlveolarBak,
		burstFreq: [2]float64{3000, 3700}, burstVol: [2]float64{10, 6}}),
	plosive("GE", stop{ms: 55, closure: 30, burst: 10, voiced: true, locus: lVelar,
		burstFreq: [2]float64{2300, 2900}, burstVol: [2]float64{12, 8}}),
	plosive("GO", stop{ms: 55, closure: 30, burst: 10, voiced: true, locus: lVelarBack,
		burstFreq: [2]float64{1300, 1800}, burstVol: [2]float64{12, 8}}),
	plosive("EG", stop{ms: 55, closure: 35, burst: 10, voiced: true, locus: lVelar,
		burstFreq: [2]float64{2300, 2900}, burstVol: [2]float64{10, 6}}),
	plosive("OG", stop{ms: 55, closure: 35, burst: 10, voiced: true, locus: lVelarBack,
		burstFreq: [2]float64{1300, 1800}, burstVol: [2]float64{10, 6}}),
	affricate("CH", stop{ms: 30, closure: 25, burst: 5, locus: lPalatal,
		burstFreq: [2]float64{3000, 3600}, burstVol: [2]float64{16, 12}}, frSH, 40),
	steady("HE", 70, hush(noiseFrame(formants{400, 1900, 2600}, 1900, 2600, 12, 9, 255), fricativeDB)),
	steady("HO", 70, hush(noiseFrame(formants{600, 1000, 2400}, 1000, 2400, 12, 8, 255), fricativeDB)),
	steady("WH", 70, hush(noiseFrame(fWW, 700, 1300, 10, 7, 255), fricativeDB)),
	steady("FF", 70, hush(noiseFrame(lLabial, 2500, 3800, 6, 9, 255), ffDB)),
	steady("SE", 40, frSE),
	steady("SO", 40, frSO),
	steady("SH", 50, frSH),
	steady("TH", 40, hush(noiseFrame(lDental, 3000, 3900, 6, 6, 255), thDB)),
	plosive("TT", stop{ms: 50, closure: 30, burst: 10, locus: lAlveolar,
		burstFreq: [2]float64{3500, 3950}, burstVol: [2]float64{18, 14}, aspFreq: [2]float64{1800, 2600}}),
	plosive("TU", stop{ms: 70, closure: 30, burst: 8, locus: lAlveolar,
		burstFreq: [2]float64{3500, 3950}, burstVol: [2]float64{18, 14}, aspFreq: [2]float64{1800, 2600}}),
	affricate("TS", stop{ms: 38, closure: 30, burst: 8, locus: lAlveolar,
		burstFreq: [2]float64{3500, 3950}, burstVol: [2]float64{18, 14}}, frSE, 132),
	plosive("KE", stop{ms: 55, closure: 25, burst: 12, locus: lVelar,
		burstFreq: [2]float64{2300, 2800}, burstVol: [2]float64{18, 12}, aspFreq: [2]float64{2000, 2800}}),
	plosive("KO", stop{ms: 55, closure: 25, burst: 12, locus: lVelarBack,
		burstFreq: [2]float64{1300, 1700}, burstVol: [2]float64{18, 12}, aspFreq: [2]float64{1100, 2300}}),
	plosive("EK", stop{ms: 55, closure: 30, burst: 15, locus: lVelar,
		burstFreq: [2]float64{2300, 2800}, burstVol: [2]float64{16, 10}, aspFreq: [2]float64{2000, 2800}}),
	plosive("OK", stop{ms: 45, closure: 25, burst: 15, locus: lVelarBack,
		burstFreq: [2]float64{1300, 1700}, burstVol: [2]float64{16, 10}, aspFreq: [2]float64{1100, 2300}}),
	plosive("PE", stop{ms: 99, closure: 50, burst: 9, locus: lLabial,
		burstFreq: [2]float64{600, 1500}, burstVol: [2]float64{12, 9}, aspFreq: [2]float64{1200, 2400}}),
	plosive("PO", stop{ms: 99, closure: 50, burst: 9, locus: lLabialBack,
		burstFreq: [2]float64{500, 1200}, burstVol: [2]float64{12, 9}, aspFreq: [2]float64{1000, 2300}}),
}

// effect constructors. none of the sound effects except DTMF are described by
// the manual beyond their name and length, so these are approximations built
// from the same oscillators

// tone is a plain sine on oscillator 1 (and optionally 2) with no envelope
func tone(f1, f2 float64) frame {
	t := frame{freq: [5]float64{f1, f2, 0, 0, 0}, vol: [5]float64{28, 0, 0, 0, 0}}
	if f2 > 0 {
		t.vol[0] = 24
		t.vol[1] = 24
	}
	return t
}

// buzz is an enveloped (voiced) sound at a fixed pitch
func buzz(f formants, pitch float64, wave envelopeWave) frame {
	b := voicedFrame(f, 26, 18, 10)
	b.pitch = pitch
	b.wave = wave
	return b
}

func quiet(f frame) frame {
	f.vol = [5]float64{}
	return f
}

// fade turns every oscillator of a frame down by db decibels
func fade(f frame, db float64) frame {
	g := math.Pow(10, -db/20)
	for i := range f.vol {
		f.vol[i] *= g
	}
	return f
}

// a5 is measured from the vox_test recording (see steadyLength): two steady
// tones, the second 12dB under the first (set louder here, as the output
// filter takes 4dB off it), that glide down together by about 295Hz over
// 240ms and back up again. the ROM sets Bend 1 before it, and the recording
// has the tones at 486Hz and 2525Hz, gliding to 190Hz and 2232Hz and back to
// 478Hz and 2522Hz. the frequencies here are those divided by Bend 1's factor
// (see msa.bendFactor()), so that Bend 1 plays the recording. whether the
// chip bends its sound effects at all isn't known
func a5() sound {
	f := tone(612, 3181)
	f.vol[0], f.vol[1] = 28, 12
	down, up := f, f
	down.freq[0], down.freq[1] = 239, 2812
	up.freq[0], up.freq[1] = 602, 3178
	return effect("A5", hold(3, 3, f), hold(237, 237, down), hold(250, 250, up), hold(10, 10, quiet(up)))
}

// m1 is measured from the vox_test recording (see steadyLength): 30ms of noise
// with most of its energy under 2kHz, then noise around 1 to 2kHz that dies
// away by 30dB, fastest at the start and the end
func m1() sound {
	bang := noiseFrame(lGlottal, 500, 1400, 31, 26, 255)
	hiss := noiseFrame(lGlottal, 1000, 1700, 31, 26, 160)
	return effect("M1",
		hold(30, 2, bang),
		hold(20, 20, fade(hiss, 5)),
		hold(50, 50, fade(hiss, 12)),
		hold(100, 100, fade(hiss, 15.5)),
		hold(25, 25, fade(hiss, 19.5)),
		hold(25, 25, quiet(hiss)))
}

// beeps plays a sequence of tones of the given frequencies, on for on ms and
// off for off ms each
func beeps(name string, on, off float64, freqs ...float64) sound {
	s := sound{name: name}
	for _, f := range freqs {
		s.parts = append(s.parts,
			part{ms: on, glideMs: 3, target: tone(f, 0)},
			part{ms: off, glideMs: 3, target: quiet(tone(f, 0))})
	}
	return s
}

// sweep glides a tone from one frequency to another
func sweep(ms, from, to float64) []part {
	return []part{
		{ms: 3, glideMs: 3, target: tone(from, 0)},
		{ms: ms - 3, glideMs: ms - 3, target: tone(to, 0)},
	}
}

func effect(name string, parts ...[]part) sound {
	s := sound{name: name}
	for _, p := range parts {
		s.parts = append(s.parts, p...)
	}
	return s
}

func hold(ms, glide float64, f frame) []part {
	return []part{{ms: ms, glideMs: glide, target: f}}
}

// decay starts f and fades it to silence over ms
func decay(ms float64, f frame) []part {
	return []part{
		{ms: 2, glideMs: 2, target: f},
		{ms: ms - 2, glideMs: ms - 2, target: quiet(f)},
	}
}

func dtmf(name string, low, high float64) sound {
	return effect(name, hold(92, 3, tone(low, high)), hold(3, 3, quiet(tone(low, high))))
}

func robot(name string, n float64) sound {
	f := buzz(formants{300 + 60*n, 900 + 120*n, 2000 + 50*n}, 60+15*n, envSquare)
	return effect(name, hold(70, 8, f), hold(10, 8, quiet(f)))
}

// sound effects, codes 200 to 254. lengths are from the manual
var effects = [...]sound{
	robot("R0", 0), robot("R1", 1), robot("R2", 2), robot("R3", 3), robot("R4", 4),
	robot("R5", 5), robot("R6", 6), robot("R7", 7), robot("R8", 8), robot("R9", 9),

	effect("A0", sweep(297, 600, 1400), hold(3, 3, quiet(tone(1400, 0)))),
	effect("A1", hold(50, 3, tone(1000, 0)), hold(48, 1, tone(700, 0)), hold(3, 3, quiet(tone(700, 0)))),
	effect("A2", sweep(99, 1400, 700), hold(3, 3, quiet(tone(700, 0)))),
	effect("A3", hold(537, 5, func() frame { f := tone(900, 1100); f.pitch = 10; f.wave = envSquare; f.voiced = true; return f }()),
		hold(3, 3, quiet(tone(900, 1100)))),
	effect("A4", sweep(265, 500, 1200), sweep(262, 1200, 500), hold(3, 3, quiet(tone(500, 0)))),
	a5(),
	effect("A6", sweep(132, 2000, 400), hold(3, 3, quiet(tone(400, 0)))),
	effect("A7", sweep(200, 400, 1600), sweep(200, 400, 1600), sweep(197, 400, 1600), hold(3, 3, quiet(tone(1600, 0)))),
	beeps("A8", 100, 50, 1600, 1600),
	effect("A9", hold(245, 5, func() frame { f := tone(400, 800); f.pitch = 60; f.wave = envSquare; f.voiced = true; return f }()),
		hold(5, 5, quiet(tone(400, 800)))),

	beeps("B0", 150, 50, 1000),
	beeps("B1", 60, 30, 1200, 1200, 1200),
	beeps("B2", 70, 70.0/3, 800, 1000, 1200),
	beeps("B3", 70, 50.0/3, 1200, 1000, 800),
	beeps("B4", 290, 10, 2000),
	beeps("B5", 90, 10, 880),
	beeps("B6", 94, 10, 1320),
	beeps("B7", 90, 10, 660),
	beeps("B8", 120, 15, 1500, 1000),
	effect("B9", sweep(259, 500, 2500), hold(3, 3, quiet(tone(2500, 0)))),

	effect("C0", sweep(78, 2000, 3500), sweep(79, 2000, 3500), hold(3, 3, quiet(tone(3500, 0)))),
	effect("C1", hold(290, 15, buzz(formants{400, 900, 2000}, 30, envSaw)), hold(10, 10, quiet(buzz(formants{400, 900, 2000}, 30, envSaw)))),
	effect("C2", hold(172, 15, buzz(formants{300, 700, 1800}, 25, envSaw)), hold(10, 10, quiet(buzz(formants{300, 700, 1800}, 25, envSaw)))),
	effect("C3", sweep(117, 3000, 2200), hold(3, 3, quiet(tone(2200, 0)))),
	effect("C4", hold(170, 5, func() frame { f := tone(2200, 0); f.pitch = 20; f.wave = envSine; f.voiced = true; return f }()),
		hold(5, 5, quiet(tone(2200, 0)))),
	effect("C5", hold(340, 20, func() frame {
		f := buzz(formants{500, 1000, 2300}, 45, envSaw)
		f.freq[3], f.freq[4], f.vol[3], f.vol[4], f.distortion, f.halfEnv = 800, 1600, 10, 8, 255, true
		return f
	}()), hold(10, 10, quiet(buzz(formants{500, 1000, 2300}, 45, envSaw)))),
	effect("C6", hold(155, 5, func() frame { f := tone(3600, 0); f.pitch = 40; f.wave = envSquare; f.voiced = true; return f }()),
		hold(5, 5, quiet(tone(3600, 0)))),
	effect("C7",
		hold(90, 20, buzz(fIY, 300, envSaw)),
		hold(90, 90, buzz(fAY, 500, envSaw)),
		hold(70, 70, buzz(fOW, 250, envSaw)),
		hold(10, 10, quiet(buzz(fOW, 250, envSaw)))),
	effect("C8", decay(95, noiseFrame(lGlottal, 2000, 3000, 24, 16, 255))),
	effect("C9", decay(75, noiseFrame(lGlottal, 800, 1400, 28, 20, 255))),

	dtmf("D0", 941, 1336), dtmf("D1", 697, 1209), dtmf("D2", 697, 1336), dtmf("D3", 697, 1477),
	dtmf("D4", 770, 1209), dtmf("D5", 770, 1336), dtmf("D6", 770, 1477), dtmf("D7", 852, 1209),
	dtmf("D8", 852, 1336), dtmf("D9", 852, 1477), dtmf("D10", 941, 1209), dtmf("D11", 941, 1477),

	effect("M0", decay(125, tone(1250, 0))),
	m1(),
	effect("M2",
		hold(80, 15, buzz(fWW, 100, envSaw)),
		hold(200, 200, buzz(fAH, 150, envSaw)),
		hold(230, 230, buzz(fUW, 90, envSaw)),
		hold(20, 20, quiet(buzz(fUW, 90, envSaw)))),
}
