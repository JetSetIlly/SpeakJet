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
// recording's syllable length to within a few milliseconds. the stops and
// affricates still use these; the other allophones now have their own lengths
// from a later recording (see allophoneLength)
const (
	steadyLength    = 1.5
	diphthongLength = 1.2
	stopLength      = 1.0
)

// how much longer the chip plays each allophone than the manual's length, at
// the default speed. 0 keeps the class's length (see steadyLength). fitted to
// the mean of two takes of the VoxRecord recording (see msa.speedFactor()),
// which agree to 2.5ms on average: each allophone was played on its own and its audible length on the chip, from where it rises out of
// silence to where it falls back, was matched here measured the same way. the
// chip plays the vowels IH to OH for about 140ms, the back vowels and RR 120 to
// 125ms, the nasals and liquids 90 to 110ms, the voiced fricatives 90 to 110ms
// and HE, HO, WH and FF about 75ms. the stops and affricates keep their
// class's length: where a stop's sound starts and ends on a recording says
// nothing about its silent closure
var allophoneLength = [...]float64{
	1.47, 1.62, 1.81, 1.77, 1.88, 1.80, 1.82, 1.79, 1.56,
	1.45, 1.42, 1.34, 1.14, 0.96, 1.03, 0.84, 0.92, 1.09,
	1.19, 1.10, 1.53, 1.12, 1.16, 1.21, 1.14, 1.14, 1.17,
	1.12, 1.06, 1.17, 1.20, 1.17, 1.06, 1.12, 1.26, 1.16,
	1.34, 0.00, 0.80, 1.11, 0.86, 1.04, 0.00, 0.00, 0.00,
	0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00,
	0.00, 1.00, 1.00, 0.99, 1.02, 2.14, 2.40, 1.12, 0.96,
	0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00,
}

func init() {
	for i, l := range allophoneLength {
		if l > 0 {
			allophones[i].length = l
		}
	}
}

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
// recording doesn't have take the mean of the fricatives that it does. the
// voiceless fricatives now have these levels built into their frames (see
// frSE)
const (
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

// frames for the fricatives, reused by the affricates and TS.
//
// the noise of the voiceless fricatives (here and HE, HO, WH, FF and TH below)
// is shaped to the VoxRecord recording (see msa.speedFactor()): their spectra
// on the chip, which the recording has unclipped, were measured in third
// octaves from 250Hz to 4kHz in two takes (which agree to 2 to 4dB), averaged, and each frame's two noise oscillators fitted to
// that shape through the output filter here. the chip's hiss is peaked where
// these are broad: SE and SO at 3.2 to 4kHz, SH at 2 to 3kHz, HE and HO around
// 1 to 1.5kHz, WH at 0.6 to 0.9kHz. each keeps the overall level the vox_test
// recording gave it (see fricativeDB)
var (
	frSE = noiseFrame(lAlveolar, 3494, 3032, 15.2, 0.6, 9)
	frSO = noiseFrame(lAlveolarBak, 3471, 3049, 9.8, 3.2, 10)
	frSH = noiseFrame(lPalatal, 3275, 2532, 7.4, 4.6, 35)
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
	steady("HE", 70, noiseFrame(formants{400, 1900, 2600}, 1446, 3706, 1.3, 1.7, 141)),
	steady("HO", 70, noiseFrame(formants{600, 1000, 2400}, 1265, 3569, 1.6, 1.5, 155)),
	steady("WH", 70, noiseFrame(fWW, 3026, 780, 0.4, 1.6, 163)),
	steady("FF", 70, noiseFrame(lLabial, 3847, 3741, 8.9, 15.6, 21)),
	steady("SE", 40, frSE),
	steady("SO", 40, frSO),
	steady("SH", 50, frSH),
	steady("TH", 40, noiseFrame(lDental, 3218, 1576, 3.8, 1.0, 77)),
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

// effect constructors

// tone is a plain sine on oscillator 1 (and optionally 2) with no envelope
func tone(f1, f2 float64) frame {
	t := frame{freq: [5]float64{f1, f2, 0, 0, 0}, vol: [5]float64{28, 0, 0, 0, 0}}
	if f2 > 0 {
		t.vol[0] = 24
		t.vol[1] = 24
	}
	return t
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

// the manual gives the DTMF tones 95ms, but on the VoxRecord recording (see
// msa.speedFactor()) all twelve sound for 152 to 154ms
func dtmf(name string, low, high float64) sound {
	return effect(name, hold(150, 3, tone(low, high)), hold(3, 3, quiet(tone(low, high))))
}

// the manual describes the sound effects only by name and length. the others
// here are traced from the VoxRecord recording (see msa.speedFactor()), which
// plays each on its own: every 5ms the strongest frequency between 150Hz and
// 4kHz and the level were measured, and the trace reduced to the points
// below, between which the sound glides in a straight line. each point is
// where it is reached (in ms from the start of the sound), the frequency
// there, its level in dB (-60 is silence), what kind of sound it is, and for a
// buzz its pitch. a tone is played as a sine. noise (a flat spectrum) is
// played as noise around the frequency. a buzz (a pitch, found by
// autocorrelation, at a third of the strongest frequency or lower) is played
// voiced at that pitch with oscillator 1 at the frequency: A1, A2, A3 and M2
// buzz at 86 to 88Hz, the default speech pitch. the recording is clipped,
// which flattens the levels and hides the waveforms, and a sound with more
// than one tone at once keeps only its strongest. many are nothing like their
// names suggest, and many differ in length from the manual: A1 and A2 last
// nearly a second, and M0 is a 1.2 second tone around 1.8kHz that dies away
const (
	fxTone = iota
	fxNoise
	fxBuzz
)

type fxPoint struct {
	ms, freq, db, kind, pitch int
}

// the traced effects, by code
var fxPoints = map[int][]fxPoint{
	200: { // R0
		{5, 783, -13, fxTone, 0}, {10, 788, -6, fxTone, 0}, {25, 799, -1, fxTone, 0},
		{75, 798, -1, fxTone, 0}, {80, 401, -2, fxTone, 0}, {95, 401, -10, fxTone, 0},
		{105, 262, -18, fxTone, 0},
	},
	201: { // R1
		{5, 1086, -13, fxTone, 0}, {25, 1101, -1, fxTone, 0}, {85, 1094, -4, fxTone, 0},
		{90, 787, -8, fxTone, 0}, {95, 1086, -13, fxTone, 0}, {100, 1086, -24, fxTone, 0},
	},
	202: { // R2
		{5, 483, -13, fxTone, 0}, {10, 1279, -6, fxTone, 0}, {25, 1299, -1, fxTone, 0},
		{80, 1298, -3, fxTone, 0}, {85, 1114, -4, fxTone, 0}, {100, 1100, -16, fxTone, 0},
		{105, 238, -24, fxTone, 0},
	},
	203: { // R3
		{5, 372, -12, fxTone, 0}, {10, 1599, -6, fxTone, 0}, {25, 1599, -1, fxTone, 0},
		{80, 1598, -3, fxTone, 0}, {85, 690, -6, fxTone, 0}, {90, 1592, -10, fxTone, 0},
		{95, 681, -14, fxTone, 0}, {100, 210, -23, fxTone, 0},
	},
	204: { // R4
		{5, 251, -16, fxTone, 0}, {10, 1891, -7, fxTone, 0}, {20, 1897, -2, fxTone, 0},
		{80, 1898, -3, fxTone, 0}, {85, 727, -5, fxTone, 0}, {90, 700, -8, fxTone, 0},
		{95, 382, -11, fxTone, 0}, {100, 255, -22, fxTone, 0},
	},
	205: { // R5
		{5, 268, -13, fxTone, 0}, {10, 2184, -7, fxTone, 0}, {25, 2198, -1, fxTone, 0},
		{80, 2197, -3, fxTone, 0}, {85, 477, -5, fxTone, 0}, {100, 462, -15, fxTone, 0},
	},
	206: { // R6
		{5, 357, -18, fxTone, 0}, {10, 375, -8, fxTone, 0}, {15, 2498, -5, fxTone, 0},
		{80, 2498, -4, fxTone, 0}, {85, 725, -6, fxTone, 0}, {90, 716, -9, fxTone, 0},
		{95, 343, -14, fxTone, 0}, {100, 271, -17, fxTone, 0},
	},
	207: { // R7
		{5, 779, -14, fxTone, 0}, {10, 828, -8, fxTone, 0}, {15, 2794, -5, fxTone, 0},
		{75, 2797, -3, fxTone, 0}, {80, 673, -5, fxTone, 0}, {90, 653, -9, fxTone, 0},
		{95, 359, -14, fxTone, 0}, {100, 252, -16, fxTone, 0},
	},
	208: { // R8
		{5, 620, -17, fxTone, 0}, {10, 596, -8, fxTone, 0}, {20, 665, -4, fxTone, 0},
		{25, 665, -2, fxTone, 0}, {75, 593, -2, fxTone, 0}, {100, 589, -13, fxTone, 0},
	},
	209: { // R9
		{5, 247, -21, fxTone, 0}, {10, 624, -13, fxTone, 0}, {20, 646, -5, fxTone, 0},
		{40, 664, -2, fxTone, 0}, {75, 599, -2, fxTone, 0}, {80, 663, -3, fxTone, 0},
		{85, 572, -4, fxTone, 0}, {105, 564, -15, fxTone, 0},
	},
	210: { // A0
		{5, 1647, -21, fxBuzz, 1647}, {15, 1647, -10, fxBuzz, 1647}, {35, 1647, -8, fxBuzz, 1647},
		{40, 1737, -7, fxTone, 0}, {75, 1938, -7, fxTone, 0}, {80, 1956, -7, fxNoise, 0},
		{120, 2121, -8, fxNoise, 0}, {125, 237, -8, fxNoise, 0}, {160, 264, -5, fxNoise, 0},
		{165, 264, -4, fxTone, 0}, {285, 373, -1, fxTone, 0}, {310, 369, -2, fxTone, 0},
		{325, 395, -3, fxTone, 0}, {335, 381, -3, fxTone, 0}, {340, 401, -4, fxTone, 0},
		{345, 368, -5, fxTone, 0}, {360, 368, -14, fxTone, 0}, {365, 210, -21, fxTone, 0},
	},
	211: { // A1
		{5, 679, -10, fxTone, 0}, {30, 698, -7, fxTone, 0}, {35, 698, -8, fxNoise, 0},
		{40, 694, -9, fxNoise, 0}, {45, 444, -10, fxBuzz, 88}, {70, 436, -14, fxBuzz, 88},
		{75, 284, -11, fxBuzz, 88}, {110, 278, -7, fxBuzz, 88}, {115, 349, -7, fxBuzz, 88},
		{145, 359, -6, fxBuzz, 88}, {155, 437, -6, fxBuzz, 88}, {170, 463, -6, fxBuzz, 88},
		{175, 518, -8, fxBuzz, 88}, {205, 542, -9, fxBuzz, 88}, {210, 612, -8, fxBuzz, 88},
		{240, 626, -7, fxBuzz, 87}, {245, 702, -6, fxBuzz, 87}, {260, 712, -6, fxBuzz, 87},
		{265, 788, -7, fxBuzz, 87}, {325, 888, -7, fxBuzz, 87}, {340, 1038, -6, fxBuzz, 87},
		{375, 1059, -5, fxBuzz, 87}, {385, 1134, -5, fxBuzz, 87}, {525, 1229, -9, fxBuzz, 87},
		{535, 1316, -8, fxBuzz, 87}, {660, 1405, -10, fxBuzz, 87}, {665, 1743, -11, fxBuzz, 87},
		{680, 1751, -10, fxBuzz, 87}, {690, 1741, -10, fxBuzz, 87}, {695, 1486, -11, fxBuzz, 87},
		{770, 1479, -11, fxBuzz, 87}, {775, 525, -11, fxBuzz, 87}, {905, 511, -13, fxBuzz, 87},
		{910, 527, -12, fxBuzz, 87}, {915, 1654, -12, fxBuzz, 87}, {935, 1655, -12, fxBuzz, 87},
		{940, 1655, -13, fxNoise, 0}, {965, 1624, -17, fxNoise, 0}, {970, 260, -17, fxTone, 0},
		{975, 1592, -24, fxTone, 0},
	},
	212: { // A2
		{5, 235, -21, fxTone, 0}, {10, 260, -15, fxNoise, 0}, {25, 265, -12, fxNoise, 0},
		{30, 1648, -12, fxNoise, 0}, {35, 1648, -11, fxBuzz, 87}, {40, 1648, -12, fxBuzz, 87},
		{45, 521, -12, fxBuzz, 87}, {205, 526, -11, fxBuzz, 87}, {210, 1470, -12, fxBuzz, 87},
		{260, 1481, -10, fxBuzz, 87}, {265, 1801, -10, fxBuzz, 87}, {295, 1730, -10, fxBuzz, 87},
		{300, 1407, -11, fxBuzz, 87}, {530, 1215, -8, fxBuzz, 87}, {615, 1032, -7, fxBuzz, 87},
		{635, 862, -8, fxBuzz, 87}, {695, 780, -6, fxBuzz, 87}, {700, 694, -6, fxBuzz, 87},
		{745, 588, -7, fxBuzz, 86}, {755, 516, -8, fxBuzz, 86}, {785, 509, -7, fxBuzz, 86},
		{815, 338, -6, fxBuzz, 86}, {860, 327, -8, fxBuzz, 86}, {865, 770, -10, fxBuzz, 86},
		{870, 332, -9, fxBuzz, 86}, {875, 332, -10, fxBuzz, 86}, {880, 437, -11, fxBuzz, 86},
		{915, 442, -9, fxBuzz, 87}, {920, 689, -9, fxBuzz, 87}, {935, 690, -7, fxBuzz, 87},
		{940, 690, -7, fxBuzz, 346}, {945, 346, -8, fxNoise, 0}, {950, 697, -8, fxTone, 0},
		{965, 675, -13, fxTone, 0}, {975, 518, -24, fxTone, 0},
	},
	213: { // A3
		{5, 478, -15, fxBuzz, 478}, {15, 478, -7, fxBuzz, 478}, {20, 526, -4, fxBuzz, 88},
		{35, 526, -4, fxBuzz, 88}, {45, 550, -4, fxBuzz, 88}, {50, 598, -5, fxBuzz, 88},
		{65, 616, -5, fxBuzz, 88}, {95, 626, -5, fxBuzz, 88}, {100, 691, -4, fxBuzz, 88},
		{145, 713, -5, fxBuzz, 87}, {155, 787, -5, fxBuzz, 87}, {190, 788, -5, fxBuzz, 87},
		{205, 873, -4, fxBuzz, 87}, {240, 891, -4, fxBuzz, 87}, {245, 906, -5, fxBuzz, 853},
		{250, 959, -4, fxBuzz, 87}, {255, 962, -5, fxBuzz, 885}, {260, 901, -5, fxBuzz, 901},
		{285, 955, -4, fxBuzz, 955}, {290, 979, -5, fxBuzz, 87}, {295, 1043, -4, fxBuzz, 87},
		{370, 1151, -5, fxBuzz, 87}, {375, 1217, -4, fxBuzz, 87}, {490, 1402, -5, fxBuzz, 87},
		{495, 1410, -5, fxBuzz, 468}, {500, 1470, -5, fxTone, 0}, {525, 1464, -8, fxTone, 0},
		{530, 1399, -11, fxTone, 0}, {535, 304, -18, fxTone, 0}, {540, 258, -19, fxTone, 0},
	},
	214: { // A4
		{5, 266, -24, fxTone, 0}, {10, 270, -14, fxTone, 0}, {35, 267, -4, fxTone, 0},
		{95, 504, -1, fxTone, 0}, {130, 955, -1, fxTone, 0}, {145, 1097, -1, fxTone, 0},
		{155, 1400, -2, fxTone, 0}, {160, 1803, -2, fxTone, 0}, {170, 1975, -3, fxTone, 0},
		{175, 2382, -3, fxTone, 0}, {185, 2695, -4, fxTone, 0}, {190, 2720, -5, fxTone, 0},
		{195, 1327, -5, fxTone, 0}, {200, 1327, -5, fxTone, 0}, {205, 3897, -5, fxTone, 0},
		{235, 3901, -10, fxTone, 0}, {255, 3900, -20, fxTone, 0}, {260, 489, -24, fxTone, 0},
	},
	215: { // A5
		{5, 637, -8, fxBuzz, 637}, {55, 584, -6, fxBuzz, 646}, {60, 193, -6, fxBuzz, 193},
		{115, 260, -4, fxBuzz, 260}, {120, 504, -3, fxBuzz, 165}, {125, 496, -2, fxBuzz, 165},
		{130, 165, -3, fxBuzz, 165}, {135, 451, -4, fxBuzz, 451}, {150, 451, -5, fxBuzz, 451},
		{155, 220, -5, fxBuzz, 220}, {210, 387, -3, fxBuzz, 132}, {215, 132, -2, fxBuzz, 132},
		{220, 329, -2, fxBuzz, 329}, {225, 380, -2, fxTone, 0}, {275, 334, -1, fxTone, 0},
		{280, 329, -2, fxBuzz, 329}, {285, 334, -2, fxBuzz, 329}, {290, 132, -2, fxBuzz, 132},
		{300, 132, -3, fxBuzz, 132}, {305, 397, -3, fxBuzz, 132}, {350, 220, -5, fxBuzz, 220},
		{355, 451, -4, fxBuzz, 451}, {390, 473, -10, fxBuzz, 473}, {395, 501, -6, fxBuzz, 260},
		{400, 257, -6, fxBuzz, 257}, {415, 260, -5, fxBuzz, 260}, {430, 547, -6, fxBuzz, 278},
		{435, 553, -6, fxTone, 0}, {440, 559, -5, fxTone, 0}, {445, 189, -5, fxTone, 0},
		{450, 570, -5, fxTone, 0}, {455, 576, -5, fxTone, 0}, {460, 583, -6, fxBuzz, 194},
		{465, 589, -5, fxBuzz, 646}, {555, 629, -10, fxBuzz, 629}, {560, 629, -18, fxTone, 0},
	},
	216: { // A6
		{5, 255, -11, fxTone, 0}, {10, 273, -8, fxTone, 0}, {20, 278, -3, fxTone, 0},
		{30, 375, -2, fxTone, 0}, {45, 532, -1, fxTone, 0}, {70, 786, -1, fxTone, 0},
		{175, 801, -2, fxTone, 0}, {190, 796, -8, fxTone, 0}, {200, 796, -24, fxTone, 0},
	},
	217: { // A7
		{5, 1165, -13, fxTone, 0}, {20, 1155, -2, fxTone, 0}, {55, 1231, -1, fxTone, 0},
		{60, 419, -1, fxBuzz, 419}, {65, 419, -1, fxBuzz, 419}, {70, 1264, -1, fxBuzz, 419},
		{85, 1297, -2, fxBuzz, 430}, {90, 1306, -2, fxBuzz, 434}, {95, 438, -2, fxBuzz, 438},
		{215, 531, -1, fxBuzz, 531}, {220, 499, -1, fxBuzz, 531}, {225, 537, -1, fxBuzz, 537},
		{230, 506, -1, fxTone, 0}, {240, 511, -1, fxTone, 0}, {245, 555, -1, fxTone, 0},
		{250, 516, -1, fxTone, 0}, {280, 561, -1, fxTone, 0}, {285, 605, -1, fxTone, 0},
		{290, 576, -1, fxTone, 0}, {295, 584, -1, fxBuzz, 612}, {300, 620, -1, fxBuzz, 620},
		{380, 735, -1, fxBuzz, 735}, {550, 937, -2, fxBuzz, 937}, {570, 937, -5, fxBuzz, 937},
		{590, 937, -18, fxBuzz, 937}, {595, 1857, -23, fxTone, 0},
	},
	218: { // A8
		{5, 266, -21, fxTone, 0}, {15, 285, -9, fxTone, 0}, {20, 312, -7, fxTone, 0},
		{30, 312, -4, fxTone, 0}, {35, 346, -3, fxTone, 0}, {45, 346, -2, fxTone, 0},
		{55, 421, -2, fxTone, 0}, {60, 492, -1, fxBuzz, 492}, {110, 498, -3, fxBuzz, 498},
		{115, 417, -3, fxTone, 0}, {120, 417, -5, fxTone, 0}, {130, 302, -11, fxTone, 0},
		{135, 293, -17, fxTone, 0}, {140, 293, -60, fxTone, 0}, {145, 286, -24, fxTone, 0},
		{150, 286, -15, fxTone, 0}, {155, 360, -11, fxTone, 0}, {160, 410, -8, fxTone, 0},
		{170, 410, -6, fxTone, 0}, {175, 302, -5, fxTone, 0}, {185, 302, -2, fxTone, 0},
		{205, 456, -1, fxTone, 0}, {215, 497, -1, fxTone, 0}, {255, 499, -2, fxTone, 0},
		{270, 491, -9, fxTone, 0}, {275, 369, -17, fxTone, 0},
	},
	219: { // A9
		{5, 1291, -10, fxTone, 0}, {20, 1296, -2, fxTone, 0}, {60, 1288, -1, fxTone, 0},
		{80, 1194, -1, fxTone, 0}, {115, 951, -1, fxTone, 0}, {175, 580, -1, fxTone, 0},
		{205, 387, -1, fxTone, 0}, {220, 293, -2, fxTone, 0}, {225, 293, -4, fxTone, 0},
		{230, 314, -6, fxNoise, 0}, {235, 609, -7, fxNoise, 0}, {245, 609, -9, fxNoise, 0},
		{250, 326, -10, fxNoise, 0}, {260, 297, -12, fxNoise, 0}, {265, 100, -12, fxBuzz, 100},
		{280, 100, -13, fxBuzz, 100}, {285, 300, -13, fxBuzz, 100}, {290, 100, -14, fxBuzz, 100},
		{295, 300, -15, fxTone, 0}, {315, 277, -23, fxTone, 0}, {320, 231, -24, fxTone, 0},
	},
	220: { // B0
		{5, 2570, -8, fxTone, 0}, {20, 2593, -2, fxTone, 0}, {130, 2593, -2, fxTone, 0},
		{155, 2593, -6, fxTone, 0}, {175, 2588, -19, fxTone, 0}, {180, 345, -24, fxTone, 0},
	},
	221: { // B1
		{5, 1493, -14, fxTone, 0}, {10, 1393, -6, fxTone, 0}, {20, 1487, -2, fxTone, 0},
		{55, 2087, -1, fxTone, 0}, {105, 2949, -2, fxTone, 0}, {150, 3722, -6, fxTone, 0},
		{165, 3754, -6, fxTone, 0}, {230, 2376, -1, fxTone, 0}, {275, 1429, -1, fxTone, 0},
		{370, 1400, -2, fxTone, 0}, {395, 1393, -14, fxTone, 0}, {400, 507, -23, fxTone, 0},
	},
	222: { // B2
		{5, 1405, -22, fxTone, 0}, {10, 1397, -6, fxTone, 0}, {25, 1406, -1, fxTone, 0},
		{130, 1417, -1, fxTone, 0}, {150, 2477, -1, fxTone, 0}, {160, 2496, -1, fxTone, 0},
		{165, 838, -1, fxTone, 0}, {180, 838, -1, fxTone, 0}, {185, 1699, -1, fxTone, 0},
		{375, 1701, -2, fxTone, 0}, {395, 1700, -7, fxTone, 0}, {410, 1706, -24, fxTone, 0},
	},
	223: { // B3
		{5, 620, -8, fxTone, 0}, {20, 594, -2, fxTone, 0}, {110, 594, -2, fxTone, 0},
		{115, 629, -1, fxBuzz, 629}, {120, 703, -1, fxBuzz, 703}, {135, 703, -1, fxBuzz, 703},
		{140, 1397, -1, fxTone, 0}, {280, 1397, -1, fxTone, 0}, {300, 1397, -6, fxTone, 0},
		{310, 1396, -14, fxTone, 0}, {315, 242, -23, fxTone, 0},
	},
	224: { // B4
		{5, 1771, -8, fxTone, 0}, {20, 1818, -1, fxTone, 0}, {65, 1824, -1, fxTone, 0},
		{70, 612, -1, fxBuzz, 612}, {80, 612, -1, fxBuzz, 612}, {85, 919, -1, fxBuzz, 919},
		{115, 901, -1, fxBuzz, 901}, {120, 732, -1, fxTone, 0}, {140, 498, -1, fxTone, 0},
		{150, 380, -1, fxTone, 0}, {160, 380, -1, fxTone, 0}, {165, 404, -1, fxTone, 0},
		{180, 586, -1, fxTone, 0}, {195, 751, -1, fxTone, 0}, {200, 901, -1, fxBuzz, 901},
		{230, 937, -1, fxBuzz, 937}, {235, 612, -1, fxBuzz, 612}, {245, 612, -1, fxBuzz, 612},
		{250, 1824, -1, fxTone, 0}, {355, 1824, -3, fxTone, 0}, {370, 1825, -8, fxTone, 0},
		{385, 1837, -24, fxTone, 0},
	},
	225: { // B5
		{5, 947, -7, fxTone, 0}, {15, 947, -2, fxTone, 0}, {35, 764, -1, fxTone, 0},
		{60, 520, -1, fxTone, 0}, {80, 327, -2, fxTone, 0}, {90, 327, -5, fxTone, 0},
		{95, 290, -8, fxNoise, 0}, {105, 309, -13, fxNoise, 0}, {125, 312, -16, fxNoise, 0},
		{130, 313, -16, fxTone, 0}, {145, 313, -19, fxTone, 0}, {150, 3498, -20, fxTone, 0},
		{160, 3498, -22, fxTone, 0}, {165, 488, -24, fxTone, 0}, {170, 240, -24, fxTone, 0},
	},
	226: { // B6
		{5, 275, -13, fxTone, 0}, {10, 3487, -8, fxTone, 0}, {15, 1165, -6, fxBuzz, 1165},
		{25, 1165, -3, fxBuzz, 1165}, {30, 1769, -3, fxBuzz, 1769}, {70, 1769, -2, fxBuzz, 1769},
		{75, 885, -2, fxBuzz, 885}, {85, 885, -1, fxBuzz, 885}, {90, 1769, -2, fxBuzz, 1769},
		{100, 1769, -2, fxBuzz, 1769}, {105, 1797, -1, fxTone, 0}, {150, 1796, -3, fxTone, 0},
		{175, 1776, -21, fxTone, 0},
	},
	227: { // B7
		{5, 251, -14, fxTone, 0}, {10, 2900, -7, fxTone, 0}, {25, 2900, -2, fxTone, 0},
		{55, 2896, -5, fxTone, 0}, {60, 1405, -9, fxTone, 0}, {65, 1405, -16, fxTone, 0},
		{70, 1405, -9, fxTone, 0}, {75, 3897, -7, fxTone, 0}, {125, 3897, -6, fxTone, 0},
		{165, 3896, -18, fxTone, 0}, {170, 394, -22, fxTone, 0}, {175, 368, -24, fxTone, 0},
	},
	228: { // B8
		{5, 2054, -10, fxTone, 0}, {25, 2030, -1, fxTone, 0}, {100, 1592, -1, fxTone, 0},
		{105, 1599, -1, fxBuzz, 1592}, {235, 1291, -1, fxBuzz, 1291}, {240, 1298, -1, fxTone, 0},
		{285, 1299, -2, fxTone, 0}, {290, 1291, -2, fxBuzz, 1291}, {310, 1291, -5, fxBuzz, 1291},
		{325, 1291, -18, fxBuzz, 1291},
	},
	229: { // B9
		{5, 901, -14, fxBuzz, 901}, {25, 901, -2, fxBuzz, 901}, {70, 901, -4, fxBuzz, 901},
		{75, 882, -4, fxTone, 0}, {80, 775, -6, fxTone, 0}, {90, 759, -4, fxTone, 0},
		{95, 680, -5, fxTone, 0}, {100, 664, -4, fxTone, 0}, {110, 561, -3, fxTone, 0},
		{115, 561, -3, fxTone, 0}, {120, 522, -3, fxTone, 0}, {145, 523, -2, fxTone, 0},
		{150, 620, -3, fxTone, 0}, {155, 530, -3, fxTone, 0}, {160, 547, -3, fxTone, 0},
		{175, 703, -4, fxTone, 0}, {180, 712, -3, fxTone, 0}, {185, 797, -4, fxTone, 0},
		{190, 818, -4, fxTone, 0}, {195, 885, -3, fxBuzz, 885}, {270, 901, -2, fxBuzz, 901},
		{275, 1769, -3, fxBuzz, 1769}, {290, 1769, -10, fxBuzz, 1769}, {300, 1769, -21, fxBuzz, 1769},
	},
	230: { // C0
		{5, 211, -23, fxTone, 0}, {10, 1657, -23, fxTone, 0}, {15, 1691, -9, fxTone, 0},
		{20, 2201, -7, fxTone, 0}, {30, 2201, -4, fxTone, 0}, {35, 1965, -5, fxTone, 0},
		{40, 1953, -4, fxTone, 0}, {45, 1697, -4, fxTone, 0}, {50, 1692, -4, fxTone, 0},
		{55, 1433, -3, fxTone, 0}, {65, 1426, -3, fxTone, 0}, {70, 1168, -4, fxTone, 0},
		{75, 1162, -3, fxTone, 0}, {80, 911, -4, fxTone, 0}, {85, 905, -4, fxTone, 0},
		{90, 802, -3, fxTone, 0}, {100, 715, -3, fxTone, 0}, {105, 612, -4, fxTone, 0},
		{110, 589, -4, fxTone, 0}, {115, 439, -3, fxTone, 0}, {125, 412, -4, fxTone, 0},
		{130, 310, -6, fxTone, 0}, {140, 310, -8, fxTone, 0}, {145, 345, -14, fxTone, 0},
		{155, 363, -17, fxTone, 0}, {160, 345, -21, fxTone, 0}, {165, 286, -19, fxTone, 0},
		{185, 276, -21, fxTone, 0}, {195, 332, -21, fxTone, 0}, {200, 569, -23, fxTone, 0},
		{210, 569, -24, fxTone, 0}, {215, 317, -24, fxTone, 0},
	},
	231: { // C1
		{5, 749, -7, fxTone, 0}, {15, 912, -3, fxTone, 0}, {20, 1366, -2, fxTone, 0},
		{35, 1393, -1, fxTone, 0}, {40, 1005, -1, fxTone, 0}, {45, 964, -1, fxTone, 0},
		{50, 690, -1, fxNoise, 0}, {60, 690, -1, fxNoise, 0}, {65, 760, -1, fxNoise, 0},
		{70, 713, -1, fxNoise, 0}, {75, 713, -1, fxTone, 0}, {85, 703, -2, fxTone, 0},
		{90, 1342, -2, fxTone, 0}, {95, 1995, -1, fxTone, 0}, {110, 1978, -1, fxTone, 0},
		{120, 1763, -1, fxTone, 0}, {125, 1837, -1, fxTone, 0}, {130, 1707, -1, fxTone, 0},
		{160, 1936, -1, fxTone, 0}, {165, 1896, -1, fxTone, 0}, {170, 1837, -1, fxTone, 0},
		{175, 1352, -2, fxTone, 0}, {195, 1190, -2, fxTone, 0}, {220, 1140, -3, fxTone, 0},
		{235, 979, -4, fxTone, 0}, {240, 853, -5, fxBuzz, 853}, {260, 853, -9, fxBuzz, 853},
		{265, 2776, -10, fxTone, 0}, {290, 2785, -19, fxTone, 0}, {295, 343, -21, fxTone, 0},
	},
	232: { // C2
		{5, 464, -9, fxBuzz, 464}, {25, 464, -6, fxBuzz, 464}, {70, 586, -5, fxBuzz, 569},
		{125, 569, -9, fxBuzz, 569}, {130, 569, -11, fxBuzz, 569}, {135, 569, -23, fxBuzz, 569},
		{140, 569, -22, fxBuzz, 569}, {145, 3336, -20, fxTone, 0}, {175, 3338, -8, fxTone, 0},
		{180, 1647, -9, fxBuzz, 1647}, {205, 1687, -12, fxBuzz, 1647}, {210, 1647, -24, fxTone, 0},
	},
	233: { // C3
		{5, 368, -24, fxTone, 0}, {10, 3158, -21, fxTone, 0}, {25, 3176, -12, fxTone, 0},
		{55, 3161, -7, fxTone, 0}, {160, 3771, -10, fxTone, 0}, {215, 3778, -14, fxTone, 0},
		{220, 392, -17, fxTone, 0}, {225, 392, -24, fxTone, 0}, {235, 382, -24, fxTone, 0},
	},
	234: { // C4
		{5, 254, -22, fxTone, 0}, {10, 939, -21, fxTone, 0}, {20, 949, -7, fxTone, 0},
		{35, 961, -4, fxTone, 0}, {100, 957, -4, fxTone, 0}, {105, 955, -4, fxBuzz, 955},
		{125, 955, -4, fxBuzz, 955}, {130, 1914, -5, fxBuzz, 1911}, {145, 1915, -5, fxBuzz, 1911},
		{150, 975, -5, fxBuzz, 975}, {170, 995, -6, fxBuzz, 995}, {175, 2957, -7, fxTone, 0},
		{205, 2973, -7, fxTone, 0}, {230, 2963, -18, fxTone, 0}, {235, 526, -20, fxTone, 0},
	},
	235: { // C5
		{5, 885, -20, fxBuzz, 885}, {10, 885, -10, fxBuzz, 885}, {15, 2696, -7, fxBuzz, 885},
		{55, 2692, -3, fxBuzz, 885}, {60, 759, -3, fxBuzz, 869}, {65, 733, -2, fxTone, 0},
		{75, 694, -1, fxTone, 0}, {80, 713, -1, fxTone, 0}, {85, 647, -1, fxBuzz, 713},
		{90, 308, -1, fxBuzz, 308}, {150, 291, -2, fxBuzz, 291}, {155, 485, -2, fxTone, 0},
		{190, 371, -2, fxTone, 0}, {195, 312, -2, fxTone, 0}, {200, 303, -2, fxTone, 0},
		{215, 297, -2, fxTone, 0}, {230, 314, -5, fxTone, 0}, {235, 302, -6, fxTone, 0},
		{240, 518, -11, fxBuzz, 3981}, {245, 518, -13, fxBuzz, 3981}, {250, 521, -13, fxBuzz, 87},
		{325, 510, -14, fxBuzz, 87}, {330, 282, -14, fxBuzz, 87}, {335, 276, -16, fxTone, 0},
		{345, 237, -24, fxTone, 0},
	},
	236: { // C6
		{5, 2039, -13, fxTone, 0}, {65, 1933, -4, fxTone, 0}, {70, 1553, -4, fxTone, 0},
		{75, 1524, -4, fxTone, 0}, {80, 1281, -3, fxTone, 0}, {85, 1212, -3, fxTone, 0},
		{90, 991, -2, fxTone, 0}, {95, 983, -2, fxTone, 0}, {100, 903, -1, fxTone, 0},
		{110, 903, -2, fxTone, 0}, {115, 995, -2, fxBuzz, 995}, {150, 995, -3, fxBuzz, 995},
		{155, 905, -2, fxTone, 0}, {165, 912, -2, fxTone, 0}, {175, 1250, -2, fxTone, 0},
		{180, 1253, -2, fxTone, 0}, {185, 1480, -3, fxTone, 0}, {190, 1508, -3, fxTone, 0},
		{195, 1822, -4, fxTone, 0}, {210, 2040, -6, fxTone, 0}, {235, 2037, -6, fxTone, 0},
		{265, 2033, -15, fxTone, 0},
	},
	237: { // C7
		{5, 519, -10, fxTone, 0}, {15, 501, -7, fxTone, 0}, {55, 549, -4, fxTone, 0},
		{60, 555, -3, fxBuzz, 555}, {80, 569, -4, fxBuzz, 569}, {85, 2277, -4, fxTone, 0},
		{100, 2299, -5, fxTone, 0}, {105, 2114, -3, fxTone, 0}, {110, 1231, -4, fxTone, 0},
		{125, 1231, -3, fxTone, 0}, {130, 1677, -2, fxTone, 0}, {145, 1681, -2, fxTone, 0},
		{150, 1427, -2, fxTone, 0}, {155, 893, -1, fxTone, 0}, {165, 893, -1, fxTone, 0},
		{170, 1223, -1, fxTone, 0}, {175, 1271, -1, fxTone, 0}, {185, 1652, -1, fxTone, 0},
		{195, 1652, -1, fxTone, 0}, {200, 797, -1, fxTone, 0}, {215, 819, -1, fxTone, 0},
		{225, 1276, -1, fxTone, 0}, {240, 1299, -1, fxTone, 0}, {245, 568, -1, fxTone, 0},
		{250, 568, -1, fxTone, 0}, {255, 242, -1, fxBuzz, 242}, {265, 241, -1, fxBuzz, 241},
		{270, 960, -1, fxBuzz, 241}, {285, 965, -1, fxBuzz, 241}, {290, 462, -1, fxTone, 0},
		{315, 435, -1, fxTone, 0}, {320, 478, -1, fxTone, 0}, {325, 478, -1, fxTone, 0},
		{330, 483, -1, fxBuzz, 483}, {340, 496, -2, fxBuzz, 483}, {345, 483, -2, fxTone, 0},
		{355, 479, -3, fxTone, 0}, {360, 504, -4, fxTone, 0}, {380, 491, -13, fxTone, 0},
	},
	238: { // C8
		{5, 218, -15, fxTone, 0}, {10, 932, -7, fxTone, 0}, {15, 1295, -4, fxTone, 0},
		{20, 1727, -2, fxNoise, 0}, {40, 1702, -1, fxNoise, 0}, {45, 1191, -1, fxNoise, 0},
		{55, 2470, -1, fxNoise, 0}, {60, 1702, -1, fxNoise, 0}, {65, 1378, -1, fxNoise, 0},
		{70, 2619, -1, fxNoise, 0}, {75, 2469, -1, fxNoise, 0}, {80, 1378, -1, fxNoise, 0},
		{85, 1182, -1, fxNoise, 0}, {90, 1863, -1, fxNoise, 0}, {105, 1863, -8, fxNoise, 0},
		{110, 1863, -21, fxTone, 0}, {115, 479, -24, fxTone, 0},
	},
	239: { // C9
		{5, 927, -8, fxTone, 0}, {10, 944, -5, fxTone, 0}, {15, 1725, -3, fxNoise, 0},
		{25, 1725, -1, fxNoise, 0}, {30, 2311, -1, fxNoise, 0}, {35, 2290, -1, fxNoise, 0},
		{40, 1164, -1, fxNoise, 0}, {45, 1151, -1, fxNoise, 0}, {50, 1898, -1, fxNoise, 0},
		{55, 1582, -1, fxNoise, 0}, {60, 1037, -1, fxNoise, 0}, {65, 983, -1, fxNoise, 0},
		{70, 1396, -1, fxNoise, 0}, {80, 1396, -5, fxNoise, 0}, {85, 1396, -9, fxTone, 0},
		{90, 1396, -19, fxTone, 0}, {95, 332, -24, fxTone, 0},
	},
	252: { // M0
		{5, 1769, -9, fxTone, 0}, {25, 1789, -1, fxTone, 0}, {545, 1789, -2, fxTone, 0},
		{1200, 1764, -24, fxTone, 0},
	},
	254: { // M2
		{5, 292, -12, fxTone, 0}, {10, 531, -12, fxTone, 0}, {15, 457, -7, fxTone, 0},
		{25, 510, -5, fxTone, 0}, {35, 530, -5, fxTone, 0}, {60, 537, -4, fxTone, 0},
		{65, 595, -3, fxTone, 0}, {70, 615, -4, fxTone, 0}, {100, 617, -4, fxTone, 0},
		{105, 682, -5, fxTone, 0}, {110, 699, -4, fxTone, 0}, {140, 715, -4, fxTone, 0},
		{150, 790, -5, fxTone, 0}, {155, 758, -4, fxTone, 0}, {175, 824, -4, fxTone, 0},
		{180, 793, -4, fxTone, 0}, {190, 869, -4, fxTone, 0}, {220, 887, -5, fxTone, 0},
		{230, 965, -5, fxTone, 0}, {300, 940, -5, fxTone, 0}, {310, 862, -5, fxTone, 0},
		{340, 858, -4, fxTone, 0}, {345, 816, -5, fxTone, 0}, {350, 810, -4, fxBuzz, 810},
		{355, 777, -5, fxBuzz, 87}, {380, 772, -5, fxBuzz, 87}, {385, 766, -3, fxTone, 0},
		{395, 693, -3, fxTone, 0}, {420, 683, -4, fxTone, 0}, {430, 611, -3, fxTone, 0},
		{460, 592, -4, fxTone, 0}, {470, 512, -5, fxTone, 0}, {510, 499, -4, fxTone, 0},
		{525, 491, -10, fxTone, 0},
	},
}

func fxFrame(p fxPoint) frame {
	g := math.Pow(10, float64(p.db)/20.0)
	if p.db <= -60 {
		return quiet(tone(float64(p.freq), 0))
	}
	if p.kind == fxNoise {
		return noiseFrame(lGlottal, float64(p.freq)*0.85, float64(p.freq)*1.15, 24*g, 18*g, 200)
	}
	f := tone(float64(p.freq), 0)
	f.vol[0] *= g
	if p.kind == fxBuzz {
		f.pitch = float64(p.pitch)
		f.voiced = true
	}
	return f
}

// contour plays a traced effect: a quick start on the first point, then a
// glide to each of the others in turn, then 3ms to silence
func contour(name string, p []fxPoint) sound {
	s := effect(name, hold(float64(p[0].ms), 2, fxFrame(p[0])))
	for i := 1; i < len(p); i++ {
		ms := float64(p[i].ms - p[i-1].ms)
		s.parts = append(s.parts, hold(ms, ms, fxFrame(p[i]))...)
	}
	s.parts = append(s.parts, hold(3, 3, quiet(fxFrame(p[len(p)-1])))...)
	return s
}

// sound effects, codes 200 to 254
var effectNames = [...]string{
	"R0", "R1", "R2", "R3", "R4", "R5", "R6", "R7", "R8", "R9",
	"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9",
	"B0", "B1", "B2", "B3", "B4", "B5", "B6", "B7", "B8", "B9",
	"C0", "C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8", "C9",
	"D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9", "D10", "D11",
	"M0", "M1", "M2",
}

var effects = func() (e [len(effectNames)]sound) {
	dtmfs := [12][2]float64{{941, 1336}, {697, 1209}, {697, 1336}, {697, 1477}, {770, 1209}, {770, 1336},
		{770, 1477}, {852, 1209}, {852, 1336}, {852, 1477}, {941, 1209}, {941, 1477}}
	for i, name := range effectNames {
		code := 200 + i
		switch {
		case fxPoints[code] != nil:
			e[i] = contour(name, fxPoints[code])
		case code >= 240 && code <= 251:
			e[i] = dtmf(name, dtmfs[code-240][0], dtmfs[code-240][1])
		case code == 253:
			e[i] = m1()
		}
	}
	return
}()
