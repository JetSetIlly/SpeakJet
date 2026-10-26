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
// the default speed, fitted to the VoxRecord recording's third take in two
// steps. first each allophone was played on its own and its audible length on
// the chip, from where it rises out of silence to where it falls back, was
// matched here measured the same way. but strung together the chip gives
// sounds longer slots than that (MM IY MM: 320ms on the chip, 235 here), so
// then one multiplier for each class was fitted to 30 connected groups (AW MM
// at eight pitches, AW SE at eight bends, TT AW EYIY SE at five speeds, and
// nine syllables such as HE OH TT and MM IY MM): vowels 0.92, nasals 1.32,
// diphthongs 1.10, voiceless fricatives 1.14, stops and affricates 1.40 (on
// the class length). that takes the groups from 6% short to within 1% (16ms on
// average; 22ms when each is left out of the fit), at the cost of sounds
// played alone running 13ms long on average. the liquids and voiced
// fricatives, which no group has, keep their first-step lengths
var allophoneLength = [...]float64{
	1.19, 1.34, 1.62, 1.63, 1.68, 1.49, 1.64, 1.61, 1.48,
	1.29, 1.26, 1.24, 1.20, 1.15, 1.20, 1.15, 1.15, 1.04,
	1.14, 1.26, 1.51, 1.16, 1.28, 1.35, 1.22, 1.25, 1.21,
	1.18, 1.16, 1.21, 1.31, 1.23, 1.17, 1.25, 1.40, 1.22,
	1.47, 1.40, 0.84, 0.99, 0.88, 0.88, 1.40, 1.40, 1.40,
	1.40, 1.40, 1.40, 1.40, 1.40, 1.40, 1.40, 1.40, 1.40,
	1.40, 1.17, 1.15, 1.11, 1.09, 2.58, 2.60, 1.28, 1.09,
	1.40, 1.40, 1.40, 1.40, 1.40, 1.40, 1.40, 1.40, 1.40,
}

// how much louder (in dB) the chip plays each allophone than the oscillator
// volumes here give, against AW. fitted to the VoxRecord recording's third
// take, whose vowels aren't clipped: each allophone's level (where it is within
// 20dB of its loudest) was matched to the recording's, measured the same way,
// relative to AW. the vowels needed little; the nasals, liquids and hiss were
// too loud here (the hiss levels were set from an earlier, clipped recording,
// which made the vowels read quiet) and BE, BO, PE and PO too quiet
var allophoneGainDB = [...]float64{
	1.1, 1.5, 1.4, 0.6, 1.1, 1.0, 0.6, 0.8, 0.0,
	-0.8, -1.2, -1.4, -3.1, -3.7, -3.6, -3.8, -3.8, -2.4,
	-2.5, -1.6, 0.4, 0.4, 0.8, 0.7, 0.1, -0.6, 1.0,
	0.5, -0.5, 0.7, 0.6, -0.9, -0.6, 0.3, -1.1, -0.7,
	-2.3, -12.2, -7.8, 2.6, -1.5, -2.6, 7.2, 7.1, 4.8,
	4.5, 3.7, 3.8, -2.0, -2.3, 0.4, 0.4, 2.7, 3.0,
	-17.4, -30.5, -29.3, -28.0, -22.9, -12.5, -13.6, -17.3, -23.1,
	-2.6, -1.7, -15.6, -0.1, -0.1, -2.1, 0.4, 9.6, 8.6,
}

func init() {
	for i, l := range allophoneLength {
		if l > 0 {
			allophones[i].length = l
		}
		g := math.Pow(10, allophoneGainDB[i]/20)
		parts := make([]part, len(allophones[i].parts))
		copy(parts, allophones[i].parts)
		for j := range parts {
			for k := range 5 {
				parts[j].target.vol[k] *= g
			}
		}
		allophones[i].parts = parts
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
//
// the third take of the VoxRecord recording (see msa.speedFactor()), whose
// vowels aren't clipped, confirms AX, UX, EH, EY, IH, OW, UH, LO and RR to
// within about 5% (formants read by linear prediction, the same way on the
// recording and on this library's own output, so that the method's errors
// cancel). it corrects five: AW F1 (590 to 695Hz), UW F1 and F2 (300 to 380,
// 870 to 1065), AY F1 and F2 (660 to 710, 1720 to 1660), OH F1 and F2 (730 to
// 690, 1090 to 1150) and WW F1 and F2 (300 to 495, 650 to 765). IY and the
// nasals can't be read this way

var (
	fIY = formants{270, 2290, 3010}
	fIH = formants{322, 1774, 2490}
	fEY = formants{508, 1820, 2425}
	fEH = formants{600, 1666, 2480}
	fAY = formants{710, 1660, 2410}
	fAX = formants{521, 1510, 2395}
	fUX = formants{613, 1328, 2390}
	fOH = formants{690, 1150, 2440}
	fAW = formants{695, 998, 2540}
	fOW = formants{512, 891, 2350}
	fUH = formants{440, 1020, 2240}
	fUW = formants{380, 1065, 2240}
	fAH = formants{700, 1220, 2600}
	fER = formants{480, 1350, 1650}
	fRR = formants{439, 1318, 1720}
	fLE = formants{360, 1300, 2700}
	fLO = formants{400, 900, 2500}
	fWW = formants{495, 765, 2200}
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
// octaves from 250Hz to 4kHz in three takes (which agree to 1 to 4dB),
// averaged, and each frame's two noise oscillators fitted to that shape
// through the output filter here. the chip's hiss is peaked where these were
// broad: SE and SO at 3.2 to 4kHz, SH at 2 to 3kHz, HE and HO around 1 to
// 1.5kHz, WH at 0.6 to 0.9kHz. their levels are set by allophoneGainDB
var (
	frSE = noiseFrame(lAlveolar, 3673, 3365, 24.0, 11.5, 23)
	frSO = noiseFrame(lAlveolarBak, 3150, 3621, 14.3, 24.0, 21)
	frSH = noiseFrame(lPalatal, 3425, 2535, 21.0, 24.0, 41)
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
	steady("HE", 70, noiseFrame(formants{400, 1900, 2600}, 1468, 1865, 24.0, 20.9, 186)),
	steady("HO", 70, noiseFrame(formants{600, 1000, 2400}, 1952, 1257, 13.5, 24.0, 201)),
	steady("WH", 70, noiseFrame(fWW, 720, 3171, 24.0, 9.3, 207)),
	steady("FF", 70, noiseFrame(lLabial, 3482, 2087, 24.0, 6.2, 106)),
	steady("SE", 40, frSE),
	steady("SO", 40, frSO),
	steady("SH", 50, frSH),
	steady("TH", 40, noiseFrame(lDental, 2797, 1841, 24.0, 9.9, 107)),
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
// msa.speedFactor()) all twelve sound for 152 to 154ms, 2.4dB quieter against
// AW than tone() plays them
func dtmf(name string, low, high float64) sound {
	t := fade(tone(low, high), 2.4)
	return effect(name, hold(150, 3, t), hold(3, 3, quiet(t)))
}

// the manual describes the sound effects only by name and length. the others
// here are traced from the third take of the VoxRecord recording (see
// msa.speedFactor()), which plays each on its own and isn't clipped: every 5ms
// the level and the sound's kind were measured, and the trace reduced to the
// points below, between which the sound glides in a straight line. a tone is
// up to three pure sines (the chip's tones have no harmonics to speak of, 45dB
// and more down): the strongest peaks, within 15dB of the loudest, that aren't
// its harmonics (R0 to R9, for example, are three tones each). noise (a flat
// spectrum) is noise around the frequency. a buzz (a pitch between 40 and
// 500Hz found by autocorrelation, with six or more peaks) is voiced at that
// pitch with oscillators 1 to 3 on its three formants, read by linear
// prediction: A1, A2, A3 and M2 buzz at 87Hz, the default speech pitch, and M2
// ("WOW") is a vowel glide. each point is where it is reached (in ms from the
// start of the sound), its kind, the buzz's pitch, its level in dB (-60 is
// silence), the three frequencies (0 for none) and the second and third's
// levels against the first (-60 while a tone that comes and goes is absent, so
// that it fades rather than sweeping from 0Hz). the levels are the recording's with the output
// filter's loss at each frequency added back (see outputFIR), as the filter
// here takes it off again. many effects are nothing like their names suggest,
// and many differ in length from the manual: A1 and A2 last about a second,
// and M0 is a 1.2 second tone around 1.8kHz that dies away
const (
	fxTone = iota
	fxNoise
	fxBuzz
)

type fxPoint struct {
	ms, kind, pitch int
	db              float64
	freq            [3]int
	rel             [3]float64
}

// the traced effects, by code
var fxPoints = map[int][]fxPoint{
	200: { // R0
		{5, fxTone, 0, -26.1, [3]int{239, 332, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -26.1, [3]int{239, 332, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -10.3, [3]int{575, 332, 0}, [3]float64{0, 0.0, 0.0}},
		{30, fxTone, 0, -3.2, [3]int{787, 332, 0}, [3]float64{0, -6.4, 0.0}},
		{35, fxTone, 0, 0.2, [3]int{793, 332, 0}, [3]float64{0, -11.9, 0.0}},
		{40, fxTone, 0, 2.1, [3]int{799, 338, 0}, [3]float64{0, 0.1, 0.0}},
		{45, fxTone, 0, 3.4, [3]int{799, 414, 0}, [3]float64{0, -14.3, 0.0}},
		{70, fxTone, 0, 3.8, [3]int{799, 519, 0}, [3]float64{0, -10.7, 0.0}},
		{95, fxTone, 0, 0.8, [3]int{799, 577, 0}, [3]float64{0, -10.2, 0.0}},
		{100, fxTone, 0, -2.0, [3]int{799, 536, 0}, [3]float64{0, -6.6, 0.0}},
		{105, fxTone, 0, -7.7, [3]int{450, 507, 0}, [3]float64{0, -1.1, 0.0}},
		{110, fxTone, 0, -13.2, [3]int{373, 437, 0}, [3]float64{0, -0.0, 0.0}},
		{115, fxTone, 0, -20.6, [3]int{274, 437, 0}, [3]float64{0, -1.9, 0.0}},
		{120, fxTone, 0, -22.1, [3]int{245, 437, 0}, [3]float64{0, -5.0, 0.0}},
		{125, fxTone, 0, -26.1, [3]int{239, 478, 0}, [3]float64{0, -13.6, 0.0}},
		{135, fxTone, 0, -26.1, [3]int{239, 478, 0}, [3]float64{0, -14.3, 0.0}},
	},
	201: { // R1
		{5, fxTone, 0, -29.3, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{20, fxTone, 0, -29.3, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{25, fxTone, 0, -16.1, [3]int{690, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{30, fxTone, 0, -6.5, [3]int{797, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{35, fxTone, 0, -2.9, [3]int{1102, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{80, fxTone, 0, 0.6, [3]int{1102, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{100, fxTone, 0, -5.7, [3]int{1102, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{105, fxTone, 0, -12.1, [3]int{717, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{110, fxTone, 0, -21.7, [3]int{712, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{115, fxTone, 0, -26.6, [3]int{251, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{130, fxTone, 0, -29.3, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
	},
	202: { // R2
		{5, fxTone, 0, -29.4, [3]int{239, 991, 227}, [3]float64{0, -13.8, -60.0}},
		{20, fxTone, 0, -29.4, [3]int{239, 991, 227}, [3]float64{0, -14.7, -60.0}},
		{25, fxTone, 0, -13.6, [3]int{667, 735, 227}, [3]float64{0, -0.0, -60.0}},
		{30, fxTone, 0, -5.4, [3]int{1295, 735, 227}, [3]float64{0, -60.0, -60.0}},
		{35, fxTone, 0, -2.4, [3]int{1300, 735, 227}, [3]float64{0, -60.0, -60.0}},
		{60, fxTone, 0, 0.2, [3]int{1300, 735, 227}, [3]float64{0, -60.0, -60.0}},
		{65, fxTone, 0, -0.1, [3]int{1300, 426, 227}, [3]float64{0, -60.0, -60.0}},
		{95, fxTone, 0, -2.8, [3]int{1295, 426, 227}, [3]float64{0, -60.0, -60.0}},
		{100, fxTone, 0, -5.4, [3]int{1295, 426, 227}, [3]float64{0, -5.5, -7.8}},
		{105, fxTone, 0, -11.6, [3]int{1149, 461, 432}, [3]float64{0, -2.5, -6.0}},
		{110, fxTone, 0, -16.3, [3]int{1143, 461, 496}, [3]float64{0, -3.5, -10.1}},
		{115, fxTone, 0, -24.5, [3]int{1125, 461, 496}, [3]float64{0, -6.3, -12.4}},
		{120, fxTone, 0, -26.1, [3]int{1061, 496, 227}, [3]float64{0, -2.3, -3.7}},
		{125, fxTone, 0, -29.4, [3]int{239, 991, 227}, [3]float64{0, -14.9, -60.0}},
		{135, fxTone, 0, -29.4, [3]int{239, 991, 227}, [3]float64{0, -13.3, -60.0}},
	},
	203: { // R3
		{5, fxTone, 0, -32.3, [3]int{239, 262, 0}, [3]float64{0, -60.0, 0.0}},
		{10, fxTone, 0, -32.3, [3]int{239, 262, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -32.3, [3]int{239, 262, 0}, [3]float64{0, -14.2, 0.0}},
		{20, fxTone, 0, -27.5, [3]int{239, 636, 0}, [3]float64{0, -14.6, 0.0}},
		{25, fxTone, 0, -17.1, [3]int{641, 636, 0}, [3]float64{0, -3.1, 0.0}},
		{30, fxTone, 0, -8.2, [3]int{659, 636, 0}, [3]float64{0, -3.7, 0.0}},
		{35, fxTone, 0, -4.5, [3]int{1580, 636, 0}, [3]float64{0, -0.9, 0.0}},
		{40, fxTone, 0, -2.7, [3]int{1598, 665, 0}, [3]float64{0, -11.2, 0.0}},
		{65, fxTone, 0, -1.6, [3]int{1598, 711, 0}, [3]float64{0, -13.0, 0.0}},
		{70, fxTone, 0, -1.6, [3]int{1598, 373, 0}, [3]float64{0, -14.0, 0.0}},
		{90, fxTone, 0, -3.3, [3]int{1598, 350, 0}, [3]float64{0, -14.1, 0.0}},
		{95, fxTone, 0, -5.1, [3]int{1598, 280, 0}, [3]float64{0, -12.6, 0.0}},
		{100, fxTone, 0, -9.4, [3]int{722, 280, 0}, [3]float64{0, -60.0, 0.0}},
		{105, fxTone, 0, -15.2, [3]int{712, 280, 0}, [3]float64{0, -60.0, 0.0}},
		{110, fxTone, 0, -22.0, [3]int{691, 472, 0}, [3]float64{0, -60.0, 0.0}},
		{115, fxTone, 0, -25.9, [3]int{245, 472, 0}, [3]float64{0, -14.8, 0.0}},
		{120, fxTone, 0, -32.3, [3]int{245, 472, 0}, [3]float64{0, -60.0, 0.0}},
		{130, fxTone, 0, -32.3, [3]int{239, 472, 0}, [3]float64{0, -60.0, 0.0}},
	},
	204: { // R4
		{5, fxTone, 0, -33.4, [3]int{239, 542, 245}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -33.4, [3]int{245, 542, 245}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -17.2, [3]int{568, 542, 245}, [3]float64{0, 0.0, -60.0}},
		{30, fxTone, 0, -7.7, [3]int{1889, 571, 245}, [3]float64{0, -11.6, -60.0}},
		{55, fxTone, 0, -1.6, [3]int{1901, 501, 245}, [3]float64{0, -14.9, -60.0}},
		{60, fxTone, 0, -1.8, [3]int{1901, 472, 245}, [3]float64{0, -15.8, -60.0}},
		{65, fxTone, 0, -1.7, [3]int{1901, 321, 245}, [3]float64{0, -16.0, -60.0}},
		{90, fxTone, 0, -3.4, [3]int{1895, 338, 245}, [3]float64{0, -13.6, -60.0}},
		{95, fxTone, 0, -7.5, [3]int{623, 338, 245}, [3]float64{0, -60.0, -60.0}},
		{100, fxTone, 0, -10.6, [3]int{622, 338, 245}, [3]float64{0, -60.0, -60.0}},
		{105, fxTone, 0, -15.4, [3]int{622, 449, 245}, [3]float64{0, -60.0, -60.0}},
		{110, fxTone, 0, -19.3, [3]int{621, 449, 245}, [3]float64{0, -0.5, -5.7}},
		{115, fxTone, 0, -20.8, [3]int{496, 478, 286}, [3]float64{0, -2.9, -4.8}},
		{120, fxTone, 0, -28.4, [3]int{251, 490, 991}, [3]float64{0, -10.5, -13.3}},
		{130, fxTone, 0, -33.4, [3]int{239, 449, 991}, [3]float64{0, -12.5, -14.7}},
	},
	205: { // R5
		{5, fxTone, 0, -33.8, [3]int{239, 338, 799}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -34.4, [3]int{239, 338, 799}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -34.0, [3]int{239, 338, 799}, [3]float64{0, -0.0, -60.0}},
		{20, fxTone, 0, -33.6, [3]int{239, 327, 799}, [3]float64{0, -13.9, -60.0}},
		{25, fxTone, 0, -13.3, [3]int{2169, 327, 799}, [3]float64{0, -5.4, -11.1}},
		{30, fxTone, 0, -7.2, [3]int{2198, 338, 799}, [3]float64{0, -11.4, -14.6}},
		{45, fxTone, 0, -2.3, [3]int{2198, 379, 764}, [3]float64{0, -14.4, -15.9}},
		{90, fxTone, 0, -3.7, [3]int{2198, 379, 764}, [3]float64{0, -13.2, -16.8}},
		{95, fxTone, 0, -5.4, [3]int{2198, 373, 210}, [3]float64{0, -9.7, -14.6}},
		{100, fxTone, 0, -12.7, [3]int{636, 367, 210}, [3]float64{0, -1.3, -4.5}},
		{105, fxTone, 0, -16.7, [3]int{583, 344, 210}, [3]float64{0, 0.0, 0.1}},
		{110, fxTone, 0, -22.3, [3]int{420, 344, 618}, [3]float64{0, -4.5, -6.4}},
		{115, fxTone, 0, -27.4, [3]int{385, 344, 618}, [3]float64{0, -1.0, -1.7}},
		{120, fxTone, 0, -28.7, [3]int{362, 467, 840}, [3]float64{0, -3.6, -8.1}},
		{125, fxTone, 0, -33.9, [3]int{245, 467, 618}, [3]float64{0, -12.8, -0.1}},
		{130, fxTone, 0, -33.8, [3]int{239, 467, 618}, [3]float64{0, -12.0, -60.0}},
	},
	206: { // R6
		{5, fxTone, 0, -35.3, [3]int{233, 478, 729}, [3]float64{0, -12.1, -60.0}},
		{10, fxTone, 0, -35.0, [3]int{233, 496, 729}, [3]float64{0, -14.8, -60.0}},
		{15, fxTone, 0, -35.0, [3]int{239, 694, 729}, [3]float64{0, -14.6, -60.0}},
		{20, fxTone, 0, -23.6, [3]int{239, 694, 729}, [3]float64{0, -5.8, -60.0}},
		{25, fxTone, 0, -18.5, [3]int{484, 496, 729}, [3]float64{0, -1.7, -60.0}},
		{30, fxTone, 0, -13.9, [3]int{607, 367, 729}, [3]float64{0, 0.0, -0.0}},
		{35, fxTone, 0, -11.8, [3]int{615, 362, 717}, [3]float64{0, 0.1, -0.0}},
		{40, fxTone, 0, -3.6, [3]int{2496, 344, 676}, [3]float64{0, -13.5, -16.3}},
		{65, fxTone, 0, -2.9, [3]int{2496, 327, 641}, [3]float64{0, -15.7, -16.7}},
		{70, fxTone, 0, -2.8, [3]int{2496, 478, 700}, [3]float64{0, -15.1, -19.4}},
		{80, fxTone, 0, -2.9, [3]int{2496, 478, 700}, [3]float64{0, -15.6, -16.9}},
		{85, fxTone, 0, -10.3, [3]int{642, 478, 700}, [3]float64{0, -60.0, -60.0}},
		{100, fxTone, 0, -16.2, [3]int{641, 478, 700}, [3]float64{0, -60.0, -60.0}},
		{105, fxTone, 0, -20.4, [3]int{519, 478, 700}, [3]float64{0, -0.5, -60.0}},
		{110, fxTone, 0, -22.3, [3]int{484, 484, 700}, [3]float64{0, -0.5, -60.0}},
		{115, fxTone, 0, -29.2, [3]int{245, 706, 700}, [3]float64{0, -6.8, -60.0}},
		{120, fxTone, 0, -33.3, [3]int{239, 484, 700}, [3]float64{0, -11.5, -60.0}},
		{125, fxTone, 0, -34.8, [3]int{233, 478, 700}, [3]float64{0, -12.3, -60.0}},
	},
	207: { // R7
		{5, fxBuzz, 128, -41.0, [3]int{535, 923, 2003}, [3]float64{0, 29.9, 42.0}},
		{10, fxBuzz, 464, -41.4, [3]int{535, 968, 2004}, [3]float64{0, 28.7, 39.9}},
		{15, fxBuzz, 468, -40.8, [3]int{535, 976, 2190}, [3]float64{0, 4.0, -1.0}},
		{20, fxBuzz, 468, -26.5, [3]int{583, 976, 2210}, [3]float64{0, 1.5, -4.8}},
		{25, fxBuzz, 468, -21.6, [3]int{583, 976, 2238}, [3]float64{0, 1.2, -5.6}},
		{30, fxBuzz, 468, -19.3, [3]int{583, 1002, 2797}, [3]float64{0, -0.2, 37.5}},
		{50, fxBuzz, 468, -17.1, [3]int{569, 1071, 2798}, [3]float64{0, -3.1, 40.9}},
		{55, fxBuzz, 468, -17.1, [3]int{537, 1065, 2798}, [3]float64{0, 0.5, 41.6}},
		{60, fxBuzz, 279, -17.1, [3]int{513, 1065, 2798}, [3]float64{0, 3.4, 42.7}},
		{70, fxBuzz, 279, -17.1, [3]int{463, 1103, 2798}, [3]float64{0, 6.2, 43.0}},
		{80, fxBuzz, 279, -17.5, [3]int{463, 1159, 2798}, [3]float64{0, 6.0, 42.3}},
		{85, fxBuzz, 310, -18.3, [3]int{467, 1159, 2798}, [3]float64{0, 5.0, 40.0}},
		{95, fxBuzz, 310, -22.1, [3]int{477, 1145, 2799}, [3]float64{0, 3.0, 28.0}},
		{100, fxBuzz, 349, -25.1, [3]int{477, 1145, 2799}, [3]float64{0, 3.5, 19.9}},
		{105, fxBuzz, 187, -29.6, [3]int{477, 1145, 2802}, [3]float64{0, 5.5, 11.0}},
		{110, fxBuzz, 401, -33.4, [3]int{477, 1145, 2808}, [3]float64{0, 34.8, 28.8}},
		{115, fxBuzz, 127, -34.1, [3]int{477, 1145, 2897}, [3]float64{0, 33.0, 31.1}},
		{120, fxBuzz, 132, -41.3, [3]int{477, 1055, 2897}, [3]float64{0, 30.3, 39.3}},
		{125, fxBuzz, 127, -41.2, [3]int{477, 988, 2897}, [3]float64{0, 28.7, 43.8}},
	},
	208: { // R8
		{5, fxTone, 0, -34.7, [3]int{239, 3096, 1639}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -34.6, [3]int{239, 3096, 1639}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -21.6, [3]int{245, 3096, 1639}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -17.7, [3]int{672, 3096, 1639}, [3]float64{0, -60.0, -60.0}},
		{40, fxTone, 0, -11.5, [3]int{669, 3096, 1639}, [3]float64{0, -60.0, -60.0}},
		{45, fxTone, 0, -10.9, [3]int{665, 3096, 1639}, [3]float64{0, 9.1, -60.0}},
		{75, fxTone, 0, -10.7, [3]int{630, 3096, 1639}, [3]float64{0, 8.7, -60.0}},
		{80, fxTone, 0, -10.7, [3]int{618, 3096, 1639}, [3]float64{0, -3.5, -60.0}},
		{85, fxTone, 0, -10.9, [3]int{612, 3096, 1639}, [3]float64{0, 8.7, -60.0}},
		{90, fxTone, 0, -11.5, [3]int{587, 1825, 1639}, [3]float64{0, -1.4, -60.0}},
		{95, fxTone, 0, -12.7, [3]int{583, 397, 1639}, [3]float64{0, 0.0, 1.1}},
		{100, fxTone, 0, -14.7, [3]int{577, 397, 1639}, [3]float64{0, -12.3, -11.7}},
		{105, fxTone, 0, -17.7, [3]int{577, 373, 1125}, [3]float64{0, -11.7, -13.0}},
		{110, fxTone, 0, -21.9, [3]int{577, 397, 1125}, [3]float64{0, -13.3, 0.0}},
		{115, fxTone, 0, -25.8, [3]int{571, 397, 630}, [3]float64{0, -9.8, -14.4}},
		{120, fxTone, 0, -29.9, [3]int{245, 455, 630}, [3]float64{0, -9.4, -14.5}},
		{125, fxTone, 0, -34.2, [3]int{239, 175, 630}, [3]float64{0, 0.0, -0.1}},
		{130, fxTone, 0, -34.5, [3]int{239, 175, 630}, [3]float64{0, -60.0, -60.0}},
	},
	209: { // R9
		{5, fxTone, 0, -36.0, [3]int{239, 3400, 1650}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -35.8, [3]int{245, 3400, 1650}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -25.5, [3]int{268, 3400, 1650}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -20.0, [3]int{661, 3400, 1650}, [3]float64{0, -60.0, -60.0}},
		{40, fxTone, 0, -12.6, [3]int{665, 3400, 1650}, [3]float64{0, -60.0, -60.0}},
		{45, fxTone, 0, -12.0, [3]int{664, 3400, 1650}, [3]float64{0, 8.5, -11.5}},
		{50, fxTone, 0, -12.2, [3]int{664, 3400, 1650}, [3]float64{0, 8.6, -11.1}},
		{85, fxTone, 0, -12.2, [3]int{612, 3400, 1534}, [3]float64{0, 8.3, -11.2}},
		{90, fxTone, 0, -12.8, [3]int{597, 367, 1534}, [3]float64{0, 0.0, 0.7}},
		{95, fxTone, 0, -13.9, [3]int{597, 367, 1534}, [3]float64{0, 0.0, 0.7}},
		{100, fxTone, 0, -15.6, [3]int{588, 367, 1534}, [3]float64{0, -10.0, -60.0}},
		{110, fxTone, 0, -22.4, [3]int{560, 373, 1534}, [3]float64{0, -6.0, -60.0}},
		{115, fxTone, 0, -26.3, [3]int{525, 455, 1534}, [3]float64{0, -7.5, -60.0}},
		{120, fxTone, 0, -30.3, [3]int{239, 455, 1534}, [3]float64{0, -7.7, -60.0}},
		{125, fxTone, 0, -35.9, [3]int{239, 251, 1534}, [3]float64{0, -14.5, -60.0}},
		{130, fxTone, 0, -35.8, [3]int{222, 251, 1534}, [3]float64{0, -60.0, -60.0}},
	},
	210: { // A0
		{5, fxTone, 0, -28.3, [3]int{239, 1009, 472}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -28.3, [3]int{239, 1009, 472}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -8.1, [3]int{239, 1009, 472}, [3]float64{0, -60.0, -60.0}},
		{30, fxTone, 0, -7.1, [3]int{175, 1009, 472}, [3]float64{0, -0.1, -60.0}},
		{35, fxTone, 0, -6.9, [3]int{175, 1627, 472}, [3]float64{0, -9.7, -60.0}},
		{40, fxTone, 0, -3.2, [3]int{175, 1644, 472}, [3]float64{0, -3.0, -60.0}},
		{45, fxTone, 0, -4.1, [3]int{233, 1726, 472}, [3]float64{0, -7.2, -60.0}},
		{55, fxTone, 0, -1.7, [3]int{233, 1779, 472}, [3]float64{0, -3.5, -60.0}},
		{60, fxTone, 0, -3.8, [3]int{163, 1779, 472}, [3]float64{0, -10.0, -60.0}},
		{75, fxTone, 0, -3.0, [3]int{163, 1808, 472}, [3]float64{0, -11.2, -60.0}},
		{80, fxTone, 0, -1.3, [3]int{175, 1837, 472}, [3]float64{0, 1.9, -60.0}},
		{85, fxTone, 0, -2.6, [3]int{181, 1895, 472}, [3]float64{0, -10.8, -60.0}},
		{95, fxTone, 0, -2.2, [3]int{181, 1948, 472}, [3]float64{0, -11.0, -60.0}},
		{110, fxTone, 0, -1.1, [3]int{198, 1977, 472}, [3]float64{0, -12.0, -60.0}},
		{115, fxTone, 0, -0.9, [3]int{204, 1977, 472}, [3]float64{0, -60.0, -60.0}},
		{165, fxTone, 0, -0.9, [3]int{245, 1977, 472}, [3]float64{0, -60.0, -60.0}},
		{170, fxTone, 0, 0.5, [3]int{251, 157, 472}, [3]float64{0, -60.0, -60.0}},
		{220, fxTone, 0, 0.7, [3]int{297, 157, 472}, [3]float64{0, -60.0, -60.0}},
		{225, fxTone, 0, 0.7, [3]int{297, 157, 472}, [3]float64{0, 0.0, -60.0}},
		{230, fxTone, 0, 0.9, [3]int{309, 449, 472}, [3]float64{0, -11.9, -60.0}},
		{235, fxTone, 0, 0.9, [3]int{315, 157, 472}, [3]float64{0, -9.2, -60.0}},
		{240, fxTone, 0, 0.8, [3]int{315, 163, 472}, [3]float64{0, -11.8, -0.0}},
		{245, fxTone, 0, 0.8, [3]int{321, 163, 478}, [3]float64{0, -10.1, -11.7}},
		{315, fxTone, 0, 1.1, [3]int{373, 192, 560}, [3]float64{0, -13.5, -14.4}},
		{320, fxTone, 0, 0.4, [3]int{373, 496, 187}, [3]float64{0, -12.5, -13.6}},
		{330, fxTone, 0, -1.6, [3]int{356, 496, 157}, [3]float64{0, -4.9, -10.6}},
		{335, fxTone, 0, -2.6, [3]int{356, 157, 157}, [3]float64{0, -10.0, -60.0}},
		{340, fxTone, 0, -4.1, [3]int{391, 157, 157}, [3]float64{0, -60.0, -60.0}},
		{345, fxTone, 0, -5.2, [3]int{391, 157, 157}, [3]float64{0, -60.0, -60.0}},
		{350, fxTone, 0, -7.8, [3]int{391, 192, 157}, [3]float64{0, -60.0, -60.0}},
		{355, fxTone, 0, -9.7, [3]int{373, 192, 157}, [3]float64{0, -60.0, -60.0}},
		{360, fxTone, 0, -10.9, [3]int{362, 192, 157}, [3]float64{0, 0.0, -60.0}},
		{365, fxTone, 0, -15.8, [3]int{362, 566, 157}, [3]float64{0, -10.2, -60.0}},
		{370, fxTone, 0, -17.4, [3]int{367, 589, 157}, [3]float64{0, -11.3, -60.0}},
		{375, fxTone, 0, -25.4, [3]int{367, 636, 157}, [3]float64{0, -12.5, -60.0}},
		{380, fxTone, 0, -26.0, [3]int{362, 991, 157}, [3]float64{0, -13.3, -60.0}},
		{385, fxTone, 0, -28.3, [3]int{245, 991, 157}, [3]float64{0, -13.7, -60.0}},
		{395, fxTone, 0, -28.3, [3]int{239, 991, 157}, [3]float64{0, -13.0, -60.0}},
	},
	211: { // A1
		{5, fxBuzz, 127, -22.8, [3]int{790, 2001, 2980}, [3]float64{0, 10.4, 17.1}},
		{10, fxBuzz, 125, -22.8, [3]int{790, 2002, 2981}, [3]float64{0, 10.4, 16.9}},
		{15, fxBuzz, 142, -22.8, [3]int{790, 2133, 3053}, [3]float64{0, -4.0, 2.1}},
		{35, fxBuzz, 88, 2.1, [3]int{790, 2133, 3053}, [3]float64{0, -4.0, 2.1}},
		{45, fxBuzz, 87, 4.3, [3]int{790, 2133, 3053}, [3]float64{0, -4.0, 2.1}},
		{50, fxBuzz, 87, 3.8, [3]int{763, 2130, 3053}, [3]float64{0, -20.0, -32.4}},
		{55, fxBuzz, 87, 4.2, [3]int{763, 2074, 3053}, [3]float64{0, -19.6, -31.9}},
		{60, fxBuzz, 87, 5.5, [3]int{763, 2074, 3101}, [3]float64{0, -37.9, -14.2}},
		{65, fxBuzz, 87, 5.1, [3]int{772, 1484, 3101}, [3]float64{0, -13.7, -12.6}},
		{85, fxBuzz, 88, 6.1, [3]int{825, 1520, 3071}, [3]float64{0, -8.7, -7.9}},
		{105, fxBuzz, 88, 7.2, [3]int{833, 1563, 3170}, [3]float64{0, -8.2, -5.1}},
		{110, fxBuzz, 87, 5.5, [3]int{833, 1867, 3346}, [3]float64{0, -9.5, -5.2}},
		{155, fxBuzz, 87, 4.4, [3]int{875, 1933, 3295}, [3]float64{0, -20.4, -22.0}},
		{160, fxBuzz, 87, 5.9, [3]int{880, 1933, 2545}, [3]float64{0, -19.4, -17.7}},
		{185, fxBuzz, 88, 5.8, [3]int{905, 1980, 2478}, [3]float64{0, -17.9, -16.4}},
		{190, fxBuzz, 87, 4.4, [3]int{908, 2009, 2478}, [3]float64{0, -17.8, -37.7}},
		{235, fxBuzz, 87, 3.8, [3]int{941, 2008, 2478}, [3]float64{0, -15.1, -32.8}},
		{245, fxBuzz, 87, 4.5, [3]int{975, 2331, 2478}, [3]float64{0, -13.2, -32.7}},
		{250, fxBuzz, 87, 5.2, [3]int{981, 2342, 3056}, [3]float64{0, -13.2, -12.2}},
		{265, fxBuzz, 87, 5.3, [3]int{1004, 2369, 3097}, [3]float64{0, -11.4, -9.2}},
		{270, fxBuzz, 87, 3.9, [3]int{1004, 2388, 3113}, [3]float64{0, -11.5, -9.5}},
		{275, fxBuzz, 87, 5.3, [3]int{1004, 2451, 3113}, [3]float64{0, -10.9, -26.0}},
		{325, fxBuzz, 87, 3.4, [3]int{979, 2564, 3113}, [3]float64{0, -10.3, -26.3}},
		{340, fxBuzz, 87, 4.1, [3]int{967, 3007, 3113}, [3]float64{0, -9.3, -29.4}},
		{345, fxBuzz, 87, 5.2, [3]int{982, 3042, 2232}, [3]float64{0, -9.8, -16.6}},
		{435, fxBuzz, 87, 5.4, [3]int{1173, 3549, 2555}, [3]float64{0, -11.6, -17.6}},
		{475, fxBuzz, 87, 3.0, [3]int{1194, 3515, 2589}, [3]float64{0, -14.0, -18.6}},
		{480, fxBuzz, 87, 4.4, [3]int{1192, 3066, 2589}, [3]float64{0, -13.0, -31.1}},
		{530, fxBuzz, 87, 3.8, [3]int{1142, 2997, 2589}, [3]float64{0, -13.0, -29.5}},
		{540, fxBuzz, 87, 4.3, [3]int{1229, 2392, 2589}, [3]float64{0, -17.1, -34.2}},
		{840, fxBuzz, 87, 2.4, [3]int{1236, 2392, 2589}, [3]float64{0, -17.5, -34.6}},
		{845, fxBuzz, 87, 3.7, [3]int{1566, 2584, 3295}, [3]float64{0, 5.1, -16.1}},
		{950, fxBuzz, 87, 4.2, [3]int{1671, 2883, 3355}, [3]float64{0, 8.0, -10.8}},
		{975, fxBuzz, 89, -3.2, [3]int{1673, 2895, 3397}, [3]float64{0, 7.8, -12.6}},
		{980, fxBuzz, 77, -12.1, [3]int{1661, 2891, 3535}, [3]float64{0, 5.6, -8.9}},
		{990, fxBuzz, 82, -18.5, [3]int{1652, 2879, 3535}, [3]float64{0, 3.8, -23.9}},
		{995, fxBuzz, 119, -21.6, [3]int{1652, 2879, 3535}, [3]float64{0, 3.8, -23.9}},
		{1005, fxBuzz, 126, -21.7, [3]int{1647, 2876, 3535}, [3]float64{0, 7.6, -22.9}},
	},
	212: { // A2
		{5, fxBuzz, 128, -32.2, [3]int{165, 1996, 1996}, [3]float64{0, -0.0, -35.8}},
		{10, fxBuzz, 127, -32.2, [3]int{165, 1996, 1996}, [3]float64{0, -0.1, -36.0}},
		{15, fxBuzz, 153, -32.2, [3]int{165, 1996, 1996}, [3]float64{0, 0.3, -35.9}},
		{20, fxBuzz, 183, -32.2, [3]int{165, 1698, 1455}, [3]float64{0, 36.6, 28.5}},
		{25, fxBuzz, 92, -12.5, [3]int{165, 1693, 1455}, [3]float64{0, 40.4, 29.7}},
		{30, fxBuzz, 87, -9.5, [3]int{184, 1689, 1455}, [3]float64{0, 15.7, 3.9}},
		{35, fxBuzz, 87, -8.2, [3]int{186, 1685, 1476}, [3]float64{0, 15.4, 3.1}},
		{45, fxBuzz, 87, -7.6, [3]int{186, 1673, 1497}, [3]float64{0, 15.1, 4.0}},
		{55, fxBuzz, 87, -7.4, [3]int{170, 1658, 1504}, [3]float64{0, 15.7, 5.0}},
		{70, fxBuzz, 87, -7.9, [3]int{162, 1641, 1515}, [3]float64{0, 15.4, 6.5}},
		{75, fxBuzz, 87, -6.3, [3]int{162, 1639, 1497}, [3]float64{0, 42.4, 31.7}},
		{80, fxBuzz, 87, -7.8, [3]int{162, 1635, 1497}, [3]float64{0, 42.7, 32.0}},
		{85, fxBuzz, 87, -6.3, [3]int{162, 1631, 1497}, [3]float64{0, 42.6, 0.5}},
		{490, fxBuzz, 87, -6.6, [3]int{162, 1584, 1497}, [3]float64{0, 43.2, 0.5}},
		{495, fxBuzz, 87, -5.7, [3]int{162, 1374, 1239}, [3]float64{0, 39.9, 36.9}},
		{510, fxBuzz, 87, -5.1, [3]int{162, 1334, 1254}, [3]float64{0, 39.3, 40.2}},
		{530, fxBuzz, 87, -4.7, [3]int{162, 1334, 1254}, [3]float64{0, 39.3, 40.2}},
		{535, fxBuzz, 87, -5.3, [3]int{162, 1212, 1531}, [3]float64{0, 42.7, 29.1}},
		{580, fxBuzz, 87, -3.8, [3]int{162, 1130, 1491}, [3]float64{0, 45.9, 27.4}},
		{585, fxBuzz, 87, -4.8, [3]int{162, 1124, 1491}, [3]float64{0, 46.2, 0.5}},
		{620, fxBuzz, 87, -4.6, [3]int{162, 1048, 1491}, [3]float64{0, 44.3, 0.5}},
		{625, fxBuzz, 87, -3.9, [3]int{162, 1030, 1304}, [3]float64{0, 43.0, 29.8}},
		{690, fxBuzz, 87, -4.6, [3]int{162, 938, 1172}, [3]float64{0, 38.2, 31.4}},
		{695, fxBuzz, 87, -5.0, [3]int{162, 919, 1040}, [3]float64{0, 37.9, 37.7}},
		{750, fxBuzz, 87, -4.1, [3]int{162, 525, 963}, [3]float64{0, 33.8, 39.1}},
		{770, fxBuzz, 87, -4.4, [3]int{162, 525, 963}, [3]float64{0, 33.8, 39.1}},
		{775, fxBuzz, 87, -4.9, [3]int{162, 467, 944}, [3]float64{0, 34.3, 39.5}},
		{815, fxBuzz, 87, -4.5, [3]int{162, 389, 897}, [3]float64{0, 38.1, 45.2}},
		{850, fxBuzz, 87, -3.5, [3]int{162, 317, 865}, [3]float64{0, 33.5, 42.8}},
		{860, fxBuzz, 87, -4.0, [3]int{162, 250, 857}, [3]float64{0, 32.0, 41.0}},
		{865, fxBuzz, 87, -3.1, [3]int{162, 250, 853}, [3]float64{0, -0.0, 39.4}},
		{895, fxBuzz, 87, -3.5, [3]int{162, 250, 845}, [3]float64{0, -0.0, 34.4}},
		{955, fxBuzz, 87, -5.5, [3]int{162, 250, 757}, [3]float64{0, -0.0, 42.8}},
		{980, fxBuzz, 82, -14.2, [3]int{162, 250, 731}, [3]float64{0, -0.0, 50.7}},
		{985, fxBuzz, 183, -23.9, [3]int{162, 250, 731}, [3]float64{0, -0.0, 50.7}},
		{990, fxBuzz, 181, -32.2, [3]int{162, 250, 731}, [3]float64{0, -0.0, 50.7}},
		{995, fxBuzz, 71, -32.2, [3]int{160, 250, 731}, [3]float64{0, -39.9, -12.7}},
		{1000, fxBuzz, 127, -32.2, [3]int{149, 250, 731}, [3]float64{0, -39.7, -10.4}},
	},
	213: { // A3
		{5, fxBuzz, 128, -32.6, [3]int{576, 977, 1996}, [3]float64{0, 27.2, 36.5}},
		{10, fxBuzz, 128, -32.6, [3]int{576, 980, 2002}, [3]float64{0, 27.3, 37.5}},
		{15, fxBuzz, 468, -32.5, [3]int{576, 990, 2010}, [3]float64{0, 26.3, 38.0}},
		{20, fxBuzz, 468, -32.6, [3]int{576, 990, 2010}, [3]float64{0, 26.3, 38.0}},
		{25, fxBuzz, 473, -6.5, [3]int{577, 990, 2386}, [3]float64{0, -35.0, -12.1}},
		{30, fxBuzz, 478, -5.5, [3]int{577, 978, 2386}, [3]float64{0, -1.6, -13.1}},
		{35, fxBuzz, 87, -3.9, [3]int{577, 978, 2386}, [3]float64{0, -1.0, -13.9}},
		{80, fxBuzz, 87, -4.6, [3]int{578, 1065, 2476}, [3]float64{0, 1.0, -14.0}},
		{120, fxBuzz, 87, -3.5, [3]int{626, 1164, 2491}, [3]float64{0, 0.2, -15.2}},
		{125, fxBuzz, 87, -5.5, [3]int{632, 1177, 1852}, [3]float64{0, 0.1, 3.6}},
		{135, fxBuzz, 87, -4.9, [3]int{645, 1199, 1852}, [3]float64{0, -0.1, 2.7}},
		{365, fxBuzz, 87, -9.1, [3]int{1121, 1915, 2760}, [3]float64{0, 3.0, 6.1}},
		{520, fxBuzz, 87, -8.7, [3]int{1452, 2357, 3417}, [3]float64{0, 4.1, 6.0}},
		{525, fxBuzz, 483, -10.9, [3]int{1462, 2370, 3440}, [3]float64{0, 4.3, 6.1}},
		{545, fxBuzz, 487, -14.2, [3]int{1465, 2382, 3447}, [3]float64{0, 5.2, 6.6}},
		{550, fxBuzz, 487, -26.0, [3]int{1459, 2382, 3431}, [3]float64{0, 4.0, 6.5}},
		{555, fxBuzz, 487, -28.6, [3]int{1304, 2379, 3423}, [3]float64{0, -21.7, 6.5}},
		{560, fxBuzz, 125, -30.0, [3]int{1283, 2184, 3423}, [3]float64{0, 1.5, -11.8}},
		{565, fxBuzz, 142, -32.6, [3]int{1067, 1987, 2980}, [3]float64{0, 7.8, 15.7}},
		{570, fxBuzz, 135, -32.6, [3]int{1011, 1986, 2975}, [3]float64{0, 9.0, 16.6}},
	},
	214: { // A4
		{5, fxTone, 0, -21.0, [3]int{239, 467, 420}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -21.0, [3]int{239, 467, 420}, [3]float64{0, -13.9, -60.0}},
		{15, fxTone, 0, -21.0, [3]int{239, 991, 420}, [3]float64{0, -0.1, -60.0}},
		{20, fxTone, 0, -21.0, [3]int{239, 467, 420}, [3]float64{0, -15.0, -60.0}},
		{25, fxTone, 0, -21.0, [3]int{239, 467, 420}, [3]float64{0, -13.9, -60.0}},
		{30, fxTone, 0, -0.2, [3]int{239, 624, 420}, [3]float64{0, -2.1, -60.0}},
		{35, fxTone, 0, 4.2, [3]int{222, 624, 420}, [3]float64{0, -8.7, -60.0}},
		{40, fxTone, 0, 6.2, [3]int{222, 461, 420}, [3]float64{0, -12.1, -60.0}},
		{45, fxTone, 0, 7.7, [3]int{222, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{75, fxTone, 0, 8.5, [3]int{338, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, 8.9, [3]int{496, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{120, fxTone, 0, 8.8, [3]int{536, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{125, fxTone, 0, 8.5, [3]int{601, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{145, fxTone, 0, 7.2, [3]int{846, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{170, fxTone, 0, 4.3, [3]int{1166, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{175, fxTone, 0, 4.4, [3]int{1411, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{195, fxTone, 0, 4.2, [3]int{2169, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{200, fxTone, 0, 4.9, [3]int{2362, 210, 420}, [3]float64{0, -5.6, -60.0}},
		{205, fxTone, 0, 5.6, [3]int{2461, 210, 420}, [3]float64{0, -9.2, -60.0}},
		{210, fxTone, 0, 4.4, [3]int{2840, 472, 420}, [3]float64{0, -14.1, -60.0}},
		{215, fxTone, 0, 6.0, [3]int{3254, 472, 420}, [3]float64{0, -24.3, -60.0}},
		{220, fxTone, 0, 10.0, [3]int{3860, 472, 420}, [3]float64{0, -19.3, -60.0}},
		{225, fxTone, 0, 6.4, [3]int{3895, 472, 420}, [3]float64{0, -60.0, -60.0}},
		{230, fxTone, 0, 5.1, [3]int{3901, 472, 420}, [3]float64{0, -60.0, -60.0}},
		{255, fxTone, 0, -0.4, [3]int{3901, 496, 420}, [3]float64{0, -60.0, -60.0}},
		{260, fxTone, 0, -1.9, [3]int{3901, 496, 420}, [3]float64{0, -31.3, -60.0}},
		{265, fxTone, 0, -3.4, [3]int{3901, 606, 420}, [3]float64{0, -30.3, -31.8}},
		{270, fxTone, 0, -3.4, [3]int{3901, 624, 706}, [3]float64{0, -28.5, -30.1}},
		{275, fxTone, 0, -21.0, [3]int{490, 3901, 706}, [3]float64{0, 16.0, -1.9}},
		{280, fxTone, 0, -21.0, [3]int{379, 3901, 647}, [3]float64{0, 11.0, -8.3}},
		{285, fxTone, 0, -21.0, [3]int{239, 3901, 420}, [3]float64{0, 7.8, -11.7}},
		{290, fxTone, 0, -21.0, [3]int{239, 997, 420}, [3]float64{0, -14.8, -60.0}},
		{295, fxTone, 0, -21.0, [3]int{239, 991, 420}, [3]float64{0, -14.5, -60.0}},
	},
	215: { // A5
		{5, fxTone, 0, -25.9, [3]int{239, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -25.9, [3]int{239, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -8.6, [3]int{636, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{35, fxTone, 0, 0.2, [3]int{630, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{130, fxTone, 0, 2.8, [3]int{507, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{135, fxTone, 0, 2.8, [3]int{169, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{140, fxTone, 0, 2.7, [3]int{169, 484, 0}, [3]float64{0, -0.0, 0.0}},
		{190, fxTone, 0, 2.7, [3]int{222, 432, 0}, [3]float64{0, -0.0, 0.0}},
		{215, fxTone, 0, 3.2, [3]int{257, 408, 0}, [3]float64{0, -0.5, 0.0}},
		{220, fxTone, 0, 2.8, [3]int{262, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{245, fxTone, 0, 3.1, [3]int{292, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{250, fxTone, 0, 3.6, [3]int{327, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{260, fxTone, 0, 4.1, [3]int{332, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{280, fxTone, 0, 1.5, [3]int{327, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{285, fxTone, 0, 2.7, [3]int{303, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{325, fxTone, 0, 3.0, [3]int{262, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{330, fxTone, 0, 3.1, [3]int{251, 408, 0}, [3]float64{0, -0.5, 0.0}},
		{400, fxTone, 0, 2.7, [3]int{169, 490, 0}, [3]float64{0, -0.7, 0.0}},
		{405, fxTone, 0, 2.6, [3]int{169, 490, 0}, [3]float64{0, -15.0, 0.0}},
		{410, fxTone, 0, 2.6, [3]int{169, 490, 0}, [3]float64{0, -1.0, 0.0}},
		{415, fxTone, 0, 2.7, [3]int{507, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{510, fxTone, 0, 0.4, [3]int{624, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{545, fxTone, 0, -4.3, [3]int{624, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{570, fxTone, 0, -17.2, [3]int{618, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{575, fxTone, 0, -22.3, [3]int{577, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{580, fxTone, 0, -25.9, [3]int{245, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{585, fxTone, 0, -25.9, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
	},
	216: { // A6
		{5, fxTone, 0, -26.1, [3]int{239, 991, 0}, [3]float64{0, -60.0, 0.0}},
		{10, fxTone, 0, -26.1, [3]int{239, 991, 0}, [3]float64{0, -13.1, 0.0}},
		{20, fxTone, 0, -26.1, [3]int{239, 991, 0}, [3]float64{0, -14.7, 0.0}},
		{25, fxTone, 0, -20.5, [3]int{239, 700, 0}, [3]float64{0, -11.5, 0.0}},
		{30, fxTone, 0, -2.5, [3]int{257, 501, 0}, [3]float64{0, -12.8, 0.0}},
		{35, fxTone, 0, 1.1, [3]int{262, 501, 0}, [3]float64{0, -1.8, 0.0}},
		{40, fxTone, 0, 2.7, [3]int{303, 665, 0}, [3]float64{0, -5.4, 0.0}},
		{45, fxTone, 0, 3.8, [3]int{303, 665, 0}, [3]float64{0, -5.9, 0.0}},
		{65, fxTone, 0, 2.9, [3]int{513, 1230, 0}, [3]float64{0, -9.8, 0.0}},
		{85, fxTone, 0, 1.3, [3]int{729, 1796, 0}, [3]float64{0, -11.8, 0.0}},
		{95, fxTone, 0, 0.8, [3]int{799, 1994, 0}, [3]float64{0, -9.2, 0.0}},
		{110, fxTone, 0, 0.5, [3]int{799, 2000, 0}, [3]float64{0, -11.6, 0.0}},
		{115, fxTone, 0, 0.5, [3]int{799, 2000, 0}, [3]float64{0, -60.0, 0.0}},
		{180, fxTone, 0, -3.9, [3]int{799, 2000, 0}, [3]float64{0, -60.0, 0.0}},
		{200, fxTone, 0, -11.2, [3]int{799, 2000, 0}, [3]float64{0, -60.0, 0.0}},
		{215, fxTone, 0, -23.6, [3]int{799, 2000, 0}, [3]float64{0, -60.0, 0.0}},
		{220, fxTone, 0, -26.1, [3]int{245, 2000, 0}, [3]float64{0, -60.0, 0.0}},
		{230, fxTone, 0, -26.1, [3]int{227, 2000, 0}, [3]float64{0, -60.0, 0.0}},
	},
	217: { // A7
		{5, fxTone, 0, -28.1, [3]int{239, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -28.1, [3]int{239, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -9.8, [3]int{1131, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{40, fxTone, 0, -3.9, [3]int{1160, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{55, fxTone, 0, -4.3, [3]int{1195, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{60, fxTone, 0, -4.5, [3]int{1207, 373, 770}, [3]float64{0, -13.3, -14.9}},
		{115, fxTone, 0, -3.8, [3]int{1306, 408, 886}, [3]float64{0, -0.4, -3.1}},
		{120, fxTone, 0, -3.8, [3]int{420, 1335, 898}, [3]float64{0, -0.3, -2.6}},
		{130, fxTone, 0, -3.4, [3]int{420, 1335, 921}, [3]float64{0, -2.1, -2.6}},
		{135, fxTone, 0, -3.2, [3]int{426, 951, 1370}, [3]float64{0, -2.8, -2.6}},
		{215, fxTone, 0, 0.3, [3]int{484, 1108, 1528}, [3]float64{0, -3.5, -13.8}},
		{220, fxTone, 0, 0.4, [3]int{484, 1120, 1528}, [3]float64{0, -3.7, -60.0}},
		{460, fxTone, 0, 0.2, [3]int{805, 1644, 1528}, [3]float64{0, -4.2, -60.0}},
		{465, fxTone, 0, 0.1, [3]int{811, 1644, 1528}, [3]float64{0, 1.1, -60.0}},
		{470, fxTone, 0, 0.1, [3]int{822, 1644, 1528}, [3]float64{0, -4.4, -60.0}},
		{475, fxTone, 0, 0.0, [3]int{828, 1644, 1528}, [3]float64{0, -60.0, -60.0}},
		{550, fxTone, 0, -6.6, [3]int{927, 1825, 1528}, [3]float64{0, -60.0, -60.0}},
		{555, fxTone, 0, -7.6, [3]int{933, 1825, 1528}, [3]float64{0, -2.3, -60.0}},
		{580, fxTone, 0, -14.8, [3]int{962, 1849, 1528}, [3]float64{0, 0.5, -60.0}},
		{585, fxTone, 0, -14.8, [3]int{1860, 968, 1528}, [3]float64{0, -2.5, -60.0}},
		{595, fxTone, 0, -20.3, [3]int{1860, 927, 1528}, [3]float64{0, -13.9, -60.0}},
		{600, fxTone, 0, -23.6, [3]int{1860, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{605, fxTone, 0, -28.1, [3]int{1249, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{610, fxTone, 0, -28.2, [3]int{1090, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{615, fxTone, 0, -28.1, [3]int{245, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{625, fxTone, 0, -28.1, [3]int{239, 927, 1528}, [3]float64{0, -60.0, -60.0}},
	},
	218: { // A8
		{5, fxTone, 0, -27.1, [3]int{239, 997, 0}, [3]float64{0, -13.8, 0.0}},
		{20, fxTone, 0, -27.1, [3]int{245, 997, 0}, [3]float64{0, -13.6, 0.0}},
		{25, fxTone, 0, -27.1, [3]int{245, 455, 0}, [3]float64{0, -5.4, 0.0}},
		{30, fxTone, 0, -14.4, [3]int{245, 426, 0}, [3]float64{0, -10.5, 0.0}},
		{35, fxTone, 0, -12.0, [3]int{227, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{50, fxTone, 0, -3.1, [3]int{233, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{65, fxTone, 0, 0.5, [3]int{344, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{80, fxTone, 0, 2.6, [3]int{449, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{85, fxTone, 0, 2.9, [3]int{484, 665, 0}, [3]float64{0, -60.0, 0.0}},
		{95, fxTone, 0, 2.0, [3]int{501, 665, 0}, [3]float64{0, -60.0, 0.0}},
		{125, fxTone, 0, -5.7, [3]int{490, 665, 0}, [3]float64{0, -60.0, 0.0}},
		{130, fxTone, 0, -7.1, [3]int{449, 665, 0}, [3]float64{0, -4.8, 0.0}},
		{140, fxTone, 0, -11.0, [3]int{303, 665, 0}, [3]float64{0, -2.9, 0.0}},
		{145, fxTone, 0, -13.7, [3]int{303, 542, 0}, [3]float64{0, -5.1, 0.0}},
		{150, fxTone, 0, -17.4, [3]int{303, 542, 0}, [3]float64{0, -1.2, 0.0}},
		{155, fxTone, 0, -23.1, [3]int{239, 542, 0}, [3]float64{0, -6.7, 0.0}},
		{160, fxTone, 0, -27.1, [3]int{239, 443, 0}, [3]float64{0, -2.9, 0.0}},
		{165, fxTone, 0, -27.1, [3]int{338, 548, 0}, [3]float64{0, -13.7, 0.0}},
		{170, fxTone, 0, -25.4, [3]int{338, 548, 0}, [3]float64{0, -7.5, 0.0}},
		{175, fxTone, 0, -18.6, [3]int{338, 391, 0}, [3]float64{0, -11.3, 0.0}},
		{180, fxTone, 0, -13.4, [3]int{338, 391, 0}, [3]float64{0, -4.8, 0.0}},
		{185, fxTone, 0, -9.6, [3]int{338, 548, 0}, [3]float64{0, -11.8, 0.0}},
		{190, fxTone, 0, -6.9, [3]int{251, 700, 0}, [3]float64{0, -6.2, 0.0}},
		{195, fxTone, 0, -4.8, [3]int{251, 863, 0}, [3]float64{0, -7.0, 0.0}},
		{200, fxTone, 0, -3.2, [3]int{251, 863, 0}, [3]float64{0, -8.2, 0.0}},
		{205, fxTone, 0, -1.9, [3]int{286, 863, 0}, [3]float64{0, -8.8, 0.0}},
		{210, fxTone, 0, -0.7, [3]int{327, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{225, fxTone, 0, 1.4, [3]int{443, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{235, fxTone, 0, 1.4, [3]int{496, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{280, fxTone, 0, -10.4, [3]int{496, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{295, fxTone, 0, -23.6, [3]int{490, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{300, fxTone, 0, -27.1, [3]int{251, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{310, fxTone, 0, -27.1, [3]int{239, 863, 0}, [3]float64{0, -60.0, 0.0}},
	},
	219: { // A9
		{5, fxTone, 0, -28.2, [3]int{233, 2443, 828}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -28.2, [3]int{239, 2443, 828}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -17.3, [3]int{245, 2443, 828}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -10.2, [3]int{1295, 2443, 828}, [3]float64{0, -60.0, -60.0}},
		{40, fxTone, 0, -5.1, [3]int{1295, 2443, 828}, [3]float64{0, -60.0, -60.0}},
		{85, fxTone, 0, -4.8, [3]int{1248, 2443, 828}, [3]float64{0, -60.0, -60.0}},
		{90, fxTone, 0, -4.7, [3]int{1219, 2443, 828}, [3]float64{0, -1.5, -60.0}},
		{140, fxTone, 0, -2.4, [3]int{904, 2169, 828}, [3]float64{0, -3.6, -60.0}},
		{180, fxTone, 0, -0.5, [3]int{653, 1901, 828}, [3]float64{0, -5.3, -60.0}},
		{185, fxTone, 0, -0.3, [3]int{618, 1831, 828}, [3]float64{0, 2.0, -60.0}},
		{195, fxTone, 0, 0.1, [3]int{560, 1761, 828}, [3]float64{0, -5.9, -60.0}},
		{220, fxTone, 0, 1.2, [3]int{402, 1627, 828}, [3]float64{0, -6.2, -60.0}},
		{245, fxTone, 0, 1.7, [3]int{245, 1458, 828}, [3]float64{0, -6.1, -60.0}},
		{255, fxTone, 0, 1.6, [3]int{245, 1388, 828}, [3]float64{0, -5.8, -60.0}},
		{260, fxTone, 0, 1.6, [3]int{1306, 216, 828}, [3]float64{0, -0.1, -60.0}},
		{265, fxTone, 0, 1.1, [3]int{1306, 216, 828}, [3]float64{0, -0.1, -60.0}},
		{270, fxTone, 0, 0.8, [3]int{1306, 216, 828}, [3]float64{0, -11.3, -60.0}},
		{275, fxTone, 0, 0.3, [3]int{1265, 216, 828}, [3]float64{0, -3.1, -7.2}},
		{280, fxTone, 0, -0.3, [3]int{216, 478, 828}, [3]float64{0, -11.0, -13.2}},
		{285, fxTone, 0, -1.0, [3]int{216, 513, 921}, [3]float64{0, -6.6, -13.1}},
		{290, fxTone, 0, -1.9, [3]int{216, 531, 921}, [3]float64{0, -9.3, -0.1}},
		{295, fxTone, 0, -3.0, [3]int{216, 513, 764}, [3]float64{0, -7.1, -11.8}},
		{300, fxTone, 0, -4.1, [3]int{216, 624, 764}, [3]float64{0, -4.6, -6.4}},
		{310, fxTone, 0, -7.0, [3]int{222, 630, 764}, [3]float64{0, -7.7, -8.9}},
		{315, fxTone, 0, -8.8, [3]int{222, 601, 770}, [3]float64{0, -1.4, -1.5}},
		{320, fxTone, 0, -11.1, [3]int{356, 630, 863}, [3]float64{0, -2.8, -5.8}},
		{325, fxTone, 0, -14.2, [3]int{356, 601, 770}, [3]float64{0, -1.7, -10.7}},
		{330, fxTone, 0, -18.1, [3]int{257, 490, 863}, [3]float64{0, -1.0, -9.0}},
		{335, fxTone, 0, -24.4, [3]int{257, 490, 863}, [3]float64{0, -3.3, -5.5}},
		{340, fxTone, 0, -28.2, [3]int{239, 490, 863}, [3]float64{0, -10.0, -11.6}},
		{345, fxTone, 0, -28.2, [3]int{239, 478, 682}, [3]float64{0, -13.2, -0.1}},
		{350, fxTone, 0, -28.2, [3]int{233, 478, 682}, [3]float64{0, -14.8, -60.0}},
	},
	220: { // B0
		{5, fxTone, 0, -33.7, [3]int{239, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -34.0, [3]int{239, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -20.1, [3]int{245, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -7.3, [3]int{2583, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{40, fxTone, 0, -2.7, [3]int{2595, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{135, fxTone, 0, -2.9, [3]int{2595, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{185, fxTone, 0, -15.6, [3]int{2595, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{190, fxTone, 0, -18.3, [3]int{2595, 227, 0}, [3]float64{0, -13.9, 0.0}},
		{195, fxTone, 0, -20.8, [3]int{2595, 455, 0}, [3]float64{0, -11.8, 0.0}},
		{200, fxTone, 0, -29.9, [3]int{245, 461, 0}, [3]float64{0, -11.9, 0.0}},
		{210, fxTone, 0, -33.4, [3]int{233, 455, 0}, [3]float64{0, -12.9, 0.0}},
	},
	221: { // B1
		{5, fxTone, 0, -28.8, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{20, fxTone, 0, -28.8, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{25, fxTone, 0, -6.0, [3]int{1376, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{40, fxTone, 0, 0.7, [3]int{1516, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{95, fxTone, 0, -1.1, [3]int{2473, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{170, fxTone, 0, -2.5, [3]int{3767, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{185, fxTone, 0, -2.4, [3]int{3709, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{230, fxTone, 0, -1.4, [3]int{2752, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{260, fxTone, 0, -0.8, [3]int{2111, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{295, fxTone, 0, 1.5, [3]int{1411, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{375, fxTone, 0, -7.6, [3]int{1400, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{410, fxTone, 0, -23.3, [3]int{1400, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{415, fxTone, 0, -28.8, [3]int{257, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{430, fxTone, 0, -28.8, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
	},
	222: { // B2
		{5, fxTone, 0, -27.1, [3]int{239, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -27.1, [3]int{239, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -6.1, [3]int{1376, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{40, fxTone, 0, 3.1, [3]int{1405, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{140, fxTone, 0, 3.0, [3]int{1405, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{145, fxTone, 0, 2.5, [3]int{1429, 198, 0}, [3]float64{0, -10.4, 0.0}},
		{150, fxTone, 0, 1.6, [3]int{1429, 198, 0}, [3]float64{0, -3.8, 0.0}},
		{160, fxTone, 0, 1.4, [3]int{1977, 198, 0}, [3]float64{0, -2.8, 0.0}},
		{165, fxTone, 0, 4.4, [3]int{2484, 198, 0}, [3]float64{0, -11.6, 0.0}},
		{175, fxTone, 0, 4.0, [3]int{2484, 198, 0}, [3]float64{0, -13.1, 0.0}},
		{180, fxTone, 0, 0.0, [3]int{1709, 198, 0}, [3]float64{0, -1.3, 0.0}},
		{190, fxTone, 0, 2.0, [3]int{1697, 198, 0}, [3]float64{0, -10.5, 0.0}},
		{230, fxTone, 0, 2.6, [3]int{1697, 204, 0}, [3]float64{0, -11.6, 0.0}},
		{255, fxTone, 0, 2.6, [3]int{1697, 397, 0}, [3]float64{0, -12.3, 0.0}},
		{280, fxTone, 0, 2.6, [3]int{1697, 595, 0}, [3]float64{0, -12.8, 0.0}},
		{355, fxTone, 0, -2.7, [3]int{1703, 601, 0}, [3]float64{0, -13.2, 0.0}},
		{400, fxTone, 0, -13.5, [3]int{1703, 601, 0}, [3]float64{0, -16.1, 0.0}},
		{405, fxTone, 0, -16.0, [3]int{1703, 472, 0}, [3]float64{0, -1.3, 0.0}},
		{410, fxTone, 0, -19.0, [3]int{1703, 472, 0}, [3]float64{0, -15.8, 0.0}},
		{415, fxTone, 0, -23.2, [3]int{1703, 472, 0}, [3]float64{0, -14.9, 0.0}},
		{420, fxTone, 0, -25.8, [3]int{1703, 478, 0}, [3]float64{0, -3.3, 0.0}},
		{425, fxTone, 0, -27.1, [3]int{356, 478, 0}, [3]float64{0, -10.6, 0.0}},
		{430, fxTone, 0, -27.1, [3]int{239, 478, 0}, [3]float64{0, -4.3, 0.0}},
		{435, fxTone, 0, -27.1, [3]int{233, 478, 0}, [3]float64{0, -60.0, 0.0}},
	},
	223: { // B3
		{5, fxTone, 0, -30.9, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -30.9, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -20.9, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -13.9, [3]int{595, 490, 0}, [3]float64{0, 0.0, 0.0}},
		{30, fxTone, 0, -11.4, [3]int{595, 3096, 0}, [3]float64{0, 9.4, 0.0}},
		{130, fxTone, 0, -7.2, [3]int{595, 3096, 0}, [3]float64{0, 8.2, 0.0}},
		{135, fxTone, 0, -4.5, [3]int{595, 1423, 0}, [3]float64{0, -5.2, 0.0}},
		{140, fxTone, 0, -2.5, [3]int{1394, 601, 0}, [3]float64{0, -5.9, 0.0}},
		{255, fxTone, 0, -1.0, [3]int{1394, 601, 0}, [3]float64{0, -9.9, 0.0}},
		{310, fxTone, 0, -12.3, [3]int{1400, 601, 0}, [3]float64{0, -11.6, 0.0}},
		{325, fxTone, 0, -21.3, [3]int{1394, 589, 0}, [3]float64{0, -15.1, 0.0}},
		{330, fxTone, 0, -26.6, [3]int{1394, 589, 0}, [3]float64{0, -6.9, 0.0}},
		{335, fxTone, 0, -30.4, [3]int{478, 991, 0}, [3]float64{0, -4.5, 0.0}},
		{340, fxTone, 0, -30.9, [3]int{245, 991, 0}, [3]float64{0, -12.9, 0.0}},
		{345, fxTone, 0, -30.9, [3]int{239, 991, 0}, [3]float64{0, -13.8, 0.0}},
	},
	224: { // B4
		{5, fxTone, 0, -30.9, [3]int{239, 1551, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -30.9, [3]int{239, 1551, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -9.4, [3]int{1819, 1551, 0}, [3]float64{0, -60.0, 0.0}},
		{35, fxTone, 0, -4.1, [3]int{1831, 1551, 0}, [3]float64{0, -60.0, 0.0}},
		{55, fxTone, 0, -3.2, [3]int{1825, 1551, 0}, [3]float64{0, -60.0, 0.0}},
		{60, fxTone, 0, -2.5, [3]int{1825, 1551, 0}, [3]float64{0, -6.6, 0.0}},
		{70, fxTone, 0, -2.2, [3]int{1825, 1551, 0}, [3]float64{0, -5.6, 0.0}},
		{120, fxTone, 0, -1.0, [3]int{1825, 962, 0}, [3]float64{0, -2.8, 0.0}},
		{130, fxTone, 0, -0.5, [3]int{1825, 962, 0}, [3]float64{0, -2.1, 0.0}},
		{135, fxTone, 0, -2.3, [3]int{781, 1825, 0}, [3]float64{0, 1.7, 0.0}},
		{155, fxTone, 0, -1.4, [3]int{548, 1825, 0}, [3]float64{0, 0.4, 0.0}},
		{170, fxTone, 0, -1.0, [3]int{373, 1825, 0}, [3]float64{0, -0.4, 0.0}},
		{180, fxTone, 0, -0.9, [3]int{373, 1825, 0}, [3]float64{0, -0.8, 0.0}},
		{185, fxTone, 0, -1.0, [3]int{402, 1825, 0}, [3]float64{0, -0.4, 0.0}},
		{200, fxTone, 0, -1.5, [3]int{577, 1825, 0}, [3]float64{0, 0.5, 0.0}},
		{220, fxTone, 0, -2.3, [3]int{811, 1825, 0}, [3]float64{0, 1.8, 0.0}},
		{225, fxTone, 0, -0.6, [3]int{1825, 986, 0}, [3]float64{0, -2.2, 0.0}},
		{235, fxTone, 0, -1.0, [3]int{1825, 986, 0}, [3]float64{0, -2.9, 0.0}},
		{280, fxTone, 0, -2.4, [3]int{1825, 1510, 0}, [3]float64{0, -5.4, 0.0}},
		{290, fxTone, 0, -2.4, [3]int{1825, 1510, 0}, [3]float64{0, -6.2, 0.0}},
		{295, fxTone, 0, -3.0, [3]int{1825, 1510, 0}, [3]float64{0, -60.0, 0.0}},
		{305, fxTone, 0, -4.9, [3]int{1819, 1510, 0}, [3]float64{0, -60.0, 0.0}},
		{310, fxTone, 0, -5.4, [3]int{1825, 1994, 0}, [3]float64{0, -60.0, 0.0}},
		{320, fxTone, 0, -3.3, [3]int{1825, 1994, 0}, [3]float64{0, -60.0, 0.0}},
		{325, fxTone, 0, -3.2, [3]int{1825, 1994, 0}, [3]float64{0, -4.9, 0.0}},
		{360, fxTone, 0, -8.6, [3]int{1825, 1994, 0}, [3]float64{0, -5.2, 0.0}},
		{390, fxTone, 0, -19.7, [3]int{1825, 1994, 0}, [3]float64{0, -6.8, 0.0}},
		{395, fxTone, 0, -23.7, [3]int{1825, 991, 0}, [3]float64{0, -2.0, 0.0}},
		{400, fxTone, 0, -28.4, [3]int{1825, 991, 0}, [3]float64{0, -2.9, 0.0}},
		{405, fxTone, 0, -30.9, [3]int{239, 991, 0}, [3]float64{0, -14.2, 0.0}},
		{415, fxTone, 0, -30.9, [3]int{239, 991, 0}, [3]float64{0, -13.8, 0.0}},
	},
	225: { // B5
		{5, fxTone, 0, -25.2, [3]int{233, 490, 204}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -25.2, [3]int{239, 490, 204}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -25.2, [3]int{239, 490, 204}, [3]float64{0, -0.0, -60.0}},
		{20, fxTone, 0, -25.2, [3]int{239, 490, 204}, [3]float64{0, -0.0, -60.0}},
		{25, fxTone, 0, -4.5, [3]int{951, 501, 204}, [3]float64{0, -8.9, -60.0}},
		{30, fxTone, 0, -1.4, [3]int{951, 781, 204}, [3]float64{0, -12.9, -60.0}},
		{35, fxTone, 0, 0.5, [3]int{951, 3493, 204}, [3]float64{0, -0.2, -60.0}},
		{50, fxTone, 0, 2.5, [3]int{811, 3493, 204}, [3]float64{0, -0.4, -60.0}},
		{55, fxTone, 0, 2.8, [3]int{764, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{85, fxTone, 0, 4.2, [3]int{472, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{95, fxTone, 0, 4.6, [3]int{379, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{105, fxTone, 0, 4.8, [3]int{280, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{110, fxTone, 0, 4.7, [3]int{280, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, 4.3, [3]int{280, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{120, fxTone, 0, 18.6, [3]int{3493, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{130, fxTone, 0, 16.7, [3]int{3499, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{155, fxTone, 0, 12.1, [3]int{3499, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{160, fxTone, 0, 11.4, [3]int{3499, 175, 204}, [3]float64{0, -14.5, -60.0}},
		{165, fxTone, 0, 9.8, [3]int{3499, 175, 204}, [3]float64{0, -29.1, -60.0}},
		{170, fxTone, 0, 7.5, [3]int{3499, 181, 204}, [3]float64{0, -26.1, -29.0}},
		{175, fxTone, 0, 6.1, [3]int{3499, 233, 472}, [3]float64{0, -22.0, -24.8}},
		{180, fxTone, 0, 2.0, [3]int{3499, 461, 472}, [3]float64{0, -20.1, -20.5}},
		{185, fxTone, 0, -15.1, [3]int{338, 461, 472}, [3]float64{0, -4.7, -5.0}},
		{190, fxTone, 0, -18.6, [3]int{239, 531, 204}, [3]float64{0, -1.6, -4.1}},
		{195, fxTone, 0, -25.2, [3]int{239, 986, 204}, [3]float64{0, -0.1, -60.0}},
		{200, fxTone, 0, -25.2, [3]int{239, 531, 204}, [3]float64{0, -13.0, -60.0}},
		{205, fxTone, 0, -25.2, [3]int{233, 531, 204}, [3]float64{0, -60.0, -60.0}},
	},
	226: { // B6
		{5, fxTone, 0, -33.8, [3]int{239, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -33.4, [3]int{239, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -22.5, [3]int{239, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -3.7, [3]int{3481, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{30, fxTone, 0, -1.5, [3]int{3493, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{40, fxTone, 0, 0.9, [3]int{3493, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{45, fxTone, 0, -8.1, [3]int{1802, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{70, fxTone, 0, -5.5, [3]int{1796, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{75, fxTone, 0, 5.3, [3]int{3493, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{85, fxTone, 0, 5.2, [3]int{3493, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{90, fxTone, 0, -5.6, [3]int{1796, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{140, fxTone, 0, -5.0, [3]int{1796, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{180, fxTone, 0, -18.6, [3]int{1796, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{185, fxTone, 0, -22.4, [3]int{1796, 367, 0}, [3]float64{0, -13.0, 0.0}},
		{190, fxTone, 0, -28.3, [3]int{251, 467, 0}, [3]float64{0, -2.1, 0.0}},
		{195, fxTone, 0, -31.3, [3]int{239, 991, 0}, [3]float64{0, -13.9, 0.0}},
		{200, fxTone, 0, -33.5, [3]int{239, 991, 0}, [3]float64{0, -14.1, 0.0}},
		{205, fxTone, 0, -33.5, [3]int{239, 467, 0}, [3]float64{0, -14.9, 0.0}},
	},
	227: { // B7
		{5, fxTone, 0, -32.2, [3]int{239, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -32.0, [3]int{239, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -22.2, [3]int{239, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -3.3, [3]int{2892, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{40, fxTone, 0, 2.0, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{60, fxTone, 0, 1.7, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{70, fxTone, 0, -1.6, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{80, fxTone, 0, -11.1, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{85, fxTone, 0, -0.0, [3]int{3895, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{125, fxTone, 0, 3.9, [3]int{3895, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{160, fxTone, 0, -2.1, [3]int{3895, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{165, fxTone, 0, -3.6, [3]int{3895, 268, 0}, [3]float64{0, -31.7, 0.0}},
		{170, fxTone, 0, -5.4, [3]int{3895, 379, 0}, [3]float64{0, -17.6, 0.0}},
		{175, fxTone, 0, -24.9, [3]int{422, 414, 0}, [3]float64{0, -4.2, 0.0}},
		{180, fxTone, 0, -26.4, [3]int{391, 997, 0}, [3]float64{0, -0.4, 0.0}},
		{185, fxTone, 0, -28.8, [3]int{373, 997, 0}, [3]float64{0, -4.3, 0.0}},
		{190, fxTone, 0, -29.4, [3]int{274, 1003, 0}, [3]float64{0, -10.1, 0.0}},
		{195, fxTone, 0, -30.2, [3]int{239, 997, 0}, [3]float64{0, -0.1, 0.0}},
		{200, fxTone, 0, -31.8, [3]int{239, 997, 0}, [3]float64{0, -60.0, 0.0}},
	},
	228: { // B8
		{5, fxTone, 0, -33.9, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{15, fxTone, 0, -33.9, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{20, fxTone, 0, -17.4, [3]int{245, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{25, fxTone, 0, -9.6, [3]int{2053, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{35, fxTone, 0, -6.0, [3]int{2053, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{150, fxTone, 0, -4.3, [3]int{1481, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{220, fxTone, 0, -3.8, [3]int{1324, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{290, fxTone, 0, -5.6, [3]int{1300, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{320, fxTone, 0, -13.2, [3]int{1300, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{340, fxTone, 0, -28.2, [3]int{1271, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{345, fxTone, 0, -33.7, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{355, fxTone, 0, -33.9, [3]int{227, 0, 0}, [3]float64{0, 0.0, 0.0}},
	},
	229: { // B9
		{5, fxTone, 0, -34.8, [3]int{233, 898, 2700}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -28.8, [3]int{245, 898, 2700}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -15.6, [3]int{894, 898, 2700}, [3]float64{0, -60.0, -60.0}},
		{30, fxTone, 0, -12.2, [3]int{896, 898, 2700}, [3]float64{0, -0.0, -60.0}},
		{35, fxTone, 0, -8.3, [3]int{1802, 898, 2700}, [3]float64{0, -4.4, 1.4}},
		{80, fxTone, 0, -7.7, [3]int{1802, 904, 2700}, [3]float64{0, -3.9, 1.2}},
		{85, fxTone, 0, -10.9, [3]int{900, 268, 2700}, [3]float64{0, 0.1, -60.0}},
		{95, fxTone, 0, -15.3, [3]int{787, 268, 2700}, [3]float64{0, -12.6, -60.0}},
		{100, fxTone, 0, -12.7, [3]int{776, 408, 2700}, [3]float64{0, -11.9, -60.0}},
		{115, fxTone, 0, -8.9, [3]int{676, 362, 2700}, [3]float64{0, -11.6, -60.0}},
		{120, fxTone, 0, -9.4, [3]int{589, 344, 2700}, [3]float64{0, -13.8, -60.0}},
		{125, fxTone, 0, -6.4, [3]int{563, 344, 2700}, [3]float64{0, -60.0, -60.0}},
		{140, fxTone, 0, -5.3, [3]int{524, 344, 2700}, [3]float64{0, -60.0, -60.0}},
		{145, fxBuzz, 87, -5.4, [3]int{524, 344, 268}, [3]float64{0, -60.0, -60.0}},
		{150, fxBuzz, 87, -4.9, [3]int{525, 344, 268}, [3]float64{0, -60.0, -60.0}},
		{155, fxTone, 0, -6.2, [3]int{525, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{160, fxBuzz, 87, -4.9, [3]int{525, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{165, fxTone, 0, -6.9, [3]int{525, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{175, fxTone, 0, -6.3, [3]int{543, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, -6.1, [3]int{590, 286, 268}, [3]float64{0, 0.1, -60.0}},
		{185, fxTone, 0, -5.9, [3]int{636, 414, 268}, [3]float64{0, -10.9, -60.0}},
		{195, fxTone, 0, -6.4, [3]int{717, 437, 268}, [3]float64{0, -11.1, -60.0}},
		{200, fxTone, 0, -8.6, [3]int{764, 507, 268}, [3]float64{0, 0.1, 0.1}},
		{210, fxTone, 0, -7.0, [3]int{892, 531, 268}, [3]float64{0, -10.9, -14.9}},
		{215, fxTone, 0, -7.2, [3]int{892, 898, 601}, [3]float64{0, -0.0, 0.1}},
		{220, fxTone, 0, -5.0, [3]int{1802, 898, 3493}, [3]float64{0, -3.0, 3.0}},
		{270, fxTone, 0, -7.7, [3]int{1802, 904, 3493}, [3]float64{0, -6.7, 2.8}},
		{275, fxTone, 0, -8.4, [3]int{1802, 3493, 898}, [3]float64{0, 2.8, -12.0}},
		{285, fxTone, 0, -10.7, [3]int{1802, 3493, 892}, [3]float64{0, 2.8, -1.9}},
		{290, fxTone, 0, -12.1, [3]int{1802, 3493, 881}, [3]float64{0, 2.8, -16.3}},
		{295, fxTone, 0, -14.2, [3]int{1802, 3493, 881}, [3]float64{0, 2.7, -60.0}},
		{300, fxTone, 0, -16.3, [3]int{1802, 3493, 881}, [3]float64{0, 2.8, -60.0}},
		{305, fxTone, 0, -21.5, [3]int{932, 1802, 881}, [3]float64{0, 1.9, -60.0}},
		{310, fxTone, 0, -25.0, [3]int{884, 986, 881}, [3]float64{0, 0.0, -60.0}},
		{315, fxTone, 0, -28.7, [3]int{257, 986, 881}, [3]float64{0, -60.0, -60.0}},
		{320, fxTone, 0, -31.2, [3]int{245, 467, 881}, [3]float64{0, -13.3, -60.0}},
		{325, fxTone, 0, -34.8, [3]int{239, 467, 881}, [3]float64{0, -60.0, -60.0}},
		{330, fxTone, 0, -34.8, [3]int{239, 467, 881}, [3]float64{0, -60.0, -60.0}},
	},
	230: { // C0
		{5, fxTone, 0, -29.6, [3]int{239, 991, 1545}, [3]float64{0, -14.9, -60.0}},
		{15, fxTone, 0, -29.6, [3]int{239, 991, 1545}, [3]float64{0, -14.9, -60.0}},
		{20, fxTone, 0, -29.6, [3]int{239, 991, 1545}, [3]float64{0, -0.1, -60.0}},
		{25, fxTone, 0, -7.4, [3]int{2315, 991, 1545}, [3]float64{0, -5.4, -60.0}},
		{30, fxTone, 0, -7.1, [3]int{2315, 2058, 1545}, [3]float64{0, -14.2, -60.0}},
		{35, fxTone, 0, -4.0, [3]int{2315, 2058, 1545}, [3]float64{0, -16.1, -60.0}},
		{40, fxTone, 0, -5.3, [3]int{2058, 2058, 1545}, [3]float64{0, -2.2, -60.0}},
		{45, fxTone, 0, -5.9, [3]int{2053, 2058, 1545}, [3]float64{0, 0.0, -2.6}},
		{55, fxTone, 0, -7.2, [3]int{1790, 1767, 2327}, [3]float64{0, -13.5, -10.6}},
		{70, fxTone, 0, -4.3, [3]int{1528, 1767, 2327}, [3]float64{0, -11.8, -9.0}},
		{75, fxTone, 0, -4.2, [3]int{1265, 1767, 2327}, [3]float64{0, -3.5, -4.9}},
		{80, fxTone, 0, -4.6, [3]int{1265, 2298, 1277}, [3]float64{0, -6.9, -14.6}},
		{85, fxTone, 0, -2.7, [3]int{1260, 1516, 2327}, [3]float64{0, -0.6, -2.9}},
		{90, fxTone, 0, -4.6, [3]int{1009, 1015, 2239}, [3]float64{0, -11.7, -9.3}},
		{95, fxTone, 0, -1.8, [3]int{991, 1003, 2239}, [3]float64{0, -8.1, -5.9}},
		{100, fxTone, 0, -2.5, [3]int{781, 764, 2239}, [3]float64{0, -10.2, -8.0}},
		{105, fxTone, 0, -1.5, [3]int{764, 577, 2239}, [3]float64{0, -13.7, 4.8}},
		{110, fxTone, 0, -0.9, [3]int{723, 402, 245}, [3]float64{0, -12.3, -13.3}},
		{115, fxTone, 0, -3.0, [3]int{647, 309, 245}, [3]float64{0, 0.1, -60.0}},
		{120, fxTone, 0, -0.4, [3]int{630, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{125, fxTone, 0, -2.3, [3]int{519, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{130, fxTone, 0, -0.7, [3]int{501, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{135, fxTone, 0, -0.1, [3]int{362, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{140, fxTone, 0, -2.6, [3]int{344, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{145, fxTone, 0, 0.4, [3]int{338, 455, 245}, [3]float64{0, -60.0, -60.0}},
		{150, fxTone, 0, -1.2, [3]int{198, 455, 583}, [3]float64{0, -60.0, -60.0}},
		{165, fxTone, 0, -8.3, [3]int{198, 455, 583}, [3]float64{0, -60.0, -60.0}},
		{170, fxTone, 0, -9.4, [3]int{297, 455, 583}, [3]float64{0, -6.3, -60.0}},
		{175, fxTone, 0, -9.0, [3]int{327, 595, 583}, [3]float64{0, -0.1, -60.0}},
		{180, fxTone, 0, -10.2, [3]int{297, 455, 583}, [3]float64{0, -6.0, -14.7}},
		{185, fxTone, 0, -13.1, [3]int{327, 595, 787}, [3]float64{0, -2.7, -6.1}},
		{190, fxTone, 0, -12.2, [3]int{327, 595, 898}, [3]float64{0, -6.4, -9.9}},
		{200, fxTone, 0, -16.6, [3]int{251, 484, 787}, [3]float64{0, -8.2, -13.8}},
		{205, fxTone, 0, -15.9, [3]int{251, 566, 881}, [3]float64{0, -9.3, -12.0}},
		{210, fxTone, 0, -17.7, [3]int{251, 484, 881}, [3]float64{0, -0.0, -0.1}},
		{215, fxTone, 0, -20.5, [3]int{216, 490, 881}, [3]float64{0, -8.8, -60.0}},
		{220, fxTone, 0, -20.3, [3]int{216, 490, 881}, [3]float64{0, -0.0, -60.0}},
		{225, fxTone, 0, -23.9, [3]int{268, 490, 881}, [3]float64{0, -10.7, -60.0}},
		{240, fxTone, 0, -29.6, [3]int{239, 461, 881}, [3]float64{0, -0.0, -60.0}},
		{245, fxTone, 0, -29.6, [3]int{239, 461, 881}, [3]float64{0, -60.0, -60.0}},
	},
	231: { // C1
		{5, fxTone, 0, -28.7, [3]int{239, 449, 426}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -28.7, [3]int{239, 449, 426}, [3]float64{0, -13.5, -60.0}},
		{15, fxTone, 0, -28.7, [3]int{245, 449, 426}, [3]float64{0, -14.1, -60.0}},
		{20, fxTone, 0, -24.4, [3]int{653, 478, 426}, [3]float64{0, -1.8, -14.5}},
		{25, fxTone, 0, -17.8, [3]int{671, 676, 426}, [3]float64{0, -0.0, 0.1}},
		{30, fxTone, 0, -9.4, [3]int{1096, 676, 682}, [3]float64{0, -1.5, -8.1}},
		{40, fxTone, 0, -4.5, [3]int{1155, 834, 682}, [3]float64{0, -1.1, -7.9}},
		{45, fxTone, 0, -3.2, [3]int{1155, 1388, 933}, [3]float64{0, -3.6, -6.0}},
		{55, fxTone, 0, -1.4, [3]int{1423, 1400, 933}, [3]float64{0, -4.9, -8.5}},
		{65, fxTone, 0, -0.7, [3]int{986, 332, 1609}, [3]float64{0, -3.9, -3.6}},
		{70, fxTone, 0, 0.2, [3]int{741, 332, 1609}, [3]float64{0, -2.9, -4.0}},
		{80, fxTone, 0, 1.2, [3]int{741, 350, 1522}, [3]float64{0, -9.0, -13.9}},
		{95, fxTone, 0, -1.6, [3]int{834, 420, 869}, [3]float64{0, -0.5, -2.4}},
		{100, fxTone, 0, -2.4, [3]int{1283, 1347, 869}, [3]float64{0, -0.1, -13.0}},
		{105, fxTone, 0, -2.5, [3]int{1341, 1347, 846}, [3]float64{0, 0.0, -0.3}},
		{110, fxTone, 0, 0.5, [3]int{2029, 1265, 840}, [3]float64{0, -4.4, -10.4}},
		{120, fxTone, 0, -0.1, [3]int{1989, 1598, 840}, [3]float64{0, -4.2, -8.0}},
		{125, fxTone, 0, -0.4, [3]int{1989, 1598, 408}, [3]float64{0, -4.5, -9.8}},
		{140, fxTone, 0, -3.5, [3]int{1528, 1895, 385}, [3]float64{0, -2.8, -5.9}},
		{145, fxTone, 0, -4.2, [3]int{1528, 1895, 1353}, [3]float64{0, 0.4, -2.3}},
		{185, fxTone, 0, -6.4, [3]int{1446, 1994, 1219}, [3]float64{0, 1.5, -6.9}},
		{190, fxTone, 0, -7.3, [3]int{1405, 1994, 1954}, [3]float64{0, 1.2, -4.9}},
		{195, fxTone, 0, -8.7, [3]int{1359, 1994, 1160}, [3]float64{0, -2.2, -5.9}},
		{225, fxTone, 0, -12.2, [3]int{1114, 1749, 1353}, [3]float64{0, -0.0, -2.4}},
		{235, fxTone, 0, -13.5, [3]int{1079, 986, 1674}, [3]float64{0, -0.6, -2.1}},
		{245, fxTone, 0, -15.0, [3]int{1073, 986, 1662}, [3]float64{0, -3.1, -2.0}},
		{265, fxTone, 0, -18.3, [3]int{834, 2507, 1662}, [3]float64{0, 2.5, -4.3}},
		{270, fxTone, 0, -18.7, [3]int{787, 2682, 1528}, [3]float64{0, 4.0, -3.7}},
		{280, fxTone, 0, -22.3, [3]int{741, 2688, 239}, [3]float64{0, 3.0, -6.2}},
		{285, fxTone, 0, -23.0, [3]int{741, 2688, 227}, [3]float64{0, 7.4, -4.0}},
		{290, fxTone, 0, -16.5, [3]int{2805, 630, 227}, [3]float64{0, -10.1, -13.6}},
		{295, fxTone, 0, -16.6, [3]int{2805, 222, 192}, [3]float64{0, -9.2, -9.2}},
		{300, fxTone, 0, -18.2, [3]int{2805, 222, 192}, [3]float64{0, -14.5, -17.8}},
		{305, fxTone, 0, -27.5, [3]int{519, 222, 490}, [3]float64{0, 0.0, 0.0}},
		{310, fxTone, 0, -28.7, [3]int{490, 280, 799}, [3]float64{0, -3.2, -4.5}},
		{315, fxTone, 0, -28.7, [3]int{245, 472, 886}, [3]float64{0, -2.0, -4.9}},
		{325, fxTone, 0, -28.7, [3]int{239, 472, 799}, [3]float64{0, -11.2, -12.6}},
		{330, fxTone, 0, -28.7, [3]int{227, 472, 799}, [3]float64{0, -14.7, -60.0}},
	},
	232: { // C2
		{5, fxTone, 0, -33.9, [3]int{233, 216, 974}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -33.9, [3]int{239, 216, 974}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -33.9, [3]int{239, 216, 974}, [3]float64{0, 0.0, -60.0}},
		{25, fxTone, 0, -29.0, [3]int{455, 216, 974}, [3]float64{0, -2.1, -60.0}},
		{30, fxTone, 0, -29.0, [3]int{478, 910, 974}, [3]float64{0, -5.1, -0.1}},
		{35, fxTone, 0, -11.7, [3]int{499, 904, 974}, [3]float64{0, -0.1, -0.1}},
		{40, fxTone, 0, -11.3, [3]int{501, 904, 974}, [3]float64{0, -2.9, -60.0}},
		{45, fxTone, 0, -11.3, [3]int{507, 904, 169}, [3]float64{0, -2.8, -11.0}},
		{50, fxTone, 0, -5.7, [3]int{507, 910, 2210}, [3]float64{0, -0.1, 4.5}},
		{55, fxTone, 0, -6.3, [3]int{507, 916, 2198}, [3]float64{0, -2.7, -2.8}},
		{60, fxTone, 0, -6.4, [3]int{507, 921, 175}, [3]float64{0, -2.6, -10.3}},
		{65, fxTone, 0, -4.0, [3]int{519, 933, 2012}, [3]float64{0, -2.9, -7.5}},
		{75, fxTone, 0, -6.5, [3]int{525, 986, 2006}, [3]float64{0, -3.2, -6.4}},
		{80, fxTone, 0, -4.2, [3]int{531, 1009, 239}, [3]float64{0, -4.3, -9.4}},
		{85, fxTone, 0, -5.9, [3]int{543, 1020, 1814}, [3]float64{0, -0.0, 1.9}},
		{90, fxTone, 0, -6.8, [3]int{548, 986, 239}, [3]float64{0, -8.3, -10.9}},
		{95, fxTone, 0, -4.8, [3]int{549, 986, 239}, [3]float64{0, -60.0, -60.0}},
		{100, fxTone, 0, -6.5, [3]int{566, 1114, 239}, [3]float64{0, -0.0, 0.1}},
		{105, fxTone, 0, -7.8, [3]int{566, 1114, 239}, [3]float64{0, -60.0, -60.0}},
		{110, fxTone, 0, -6.5, [3]int{566, 1155, 245}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, -8.8, [3]int{566, 1155, 245}, [3]float64{0, 0.0, 0.1}},
		{135, fxTone, 0, -14.6, [3]int{560, 1172, 245}, [3]float64{0, -3.5, -9.9}},
		{140, fxTone, 0, -14.4, [3]int{554, 1172, 344}, [3]float64{0, -7.0, -12.4}},
		{165, fxTone, 0, -30.6, [3]int{449, 1155, 251}, [3]float64{0, -14.3, 0.0}},
		{170, fxTone, 0, -30.1, [3]int{449, 1155, 251}, [3]float64{0, -9.1, -60.0}},
		{175, fxTone, 0, -27.6, [3]int{449, 175, 251}, [3]float64{0, 0.0, -60.0}},
		{180, fxTone, 0, -14.2, [3]int{3312, 175, 251}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, -12.9, [3]int{3336, 175, 251}, [3]float64{0, -60.0, -60.0}},
		{190, fxTone, 0, -8.8, [3]int{3336, 175, 251}, [3]float64{0, -13.1, -60.0}},
		{195, fxTone, 0, -9.3, [3]int{3336, 175, 618}, [3]float64{0, -26.6, -60.0}},
		{200, fxTone, 0, -9.1, [3]int{3336, 175, 618}, [3]float64{0, -60.0, -60.0}},
		{205, fxTone, 0, -4.6, [3]int{3341, 192, 618}, [3]float64{0, -13.2, -60.0}},
		{215, fxTone, 0, -5.9, [3]int{3341, 216, 618}, [3]float64{0, -26.5, -60.0}},
		{220, fxTone, 0, -5.7, [3]int{3341, 297, 618}, [3]float64{0, -17.5, -22.8}},
		{225, fxTone, 0, -25.6, [3]int{1674, 315, 618}, [3]float64{0, -6.6, -10.5}},
		{230, fxTone, 0, -26.2, [3]int{1668, 315, 618}, [3]float64{0, -7.2, -10.7}},
		{235, fxTone, 0, -27.3, [3]int{262, 472, 618}, [3]float64{0, -2.9, -9.2}},
		{240, fxTone, 0, -32.9, [3]int{245, 472, 484}, [3]float64{0, -14.6, -0.0}},
		{245, fxTone, 0, -33.9, [3]int{239, 472, 484}, [3]float64{0, -12.0, -60.0}},
		{250, fxTone, 0, -33.9, [3]int{233, 472, 484}, [3]float64{0, -60.0, -60.0}},
	},
	233: { // C3
		{5, fxTone, 0, -34.1, [3]int{222, 490, 414}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -33.0, [3]int{239, 490, 414}, [3]float64{0, -12.0, -60.0}},
		{15, fxTone, 0, -32.3, [3]int{239, 490, 414}, [3]float64{0, -13.2, -60.0}},
		{20, fxTone, 0, -28.2, [3]int{268, 490, 414}, [3]float64{0, -7.4, -60.0}},
		{25, fxTone, 0, -14.7, [3]int{3143, 216, 414}, [3]float64{0, -11.7, -60.0}},
		{30, fxTone, 0, -13.6, [3]int{3161, 175, 414}, [3]float64{0, -19.1, -60.0}},
		{40, fxTone, 0, -8.3, [3]int{3166, 187, 414}, [3]float64{0, -11.8, -60.0}},
		{45, fxTone, 0, -8.5, [3]int{3172, 175, 414}, [3]float64{0, -25.7, -60.0}},
		{50, fxTone, 0, -6.2, [3]int{3166, 175, 414}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, -1.5, [3]int{3376, 175, 414}, [3]float64{0, -60.0, -60.0}},
		{120, fxTone, 0, -1.4, [3]int{3470, 204, 414}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, -1.0, [3]int{3785, 204, 414}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, -0.9, [3]int{3785, 204, 414}, [3]float64{0, -16.8, -60.0}},
		{190, fxTone, 0, -0.9, [3]int{3785, 204, 414}, [3]float64{0, -16.8, -60.0}},
		{195, fxTone, 0, -0.9, [3]int{3785, 204, 414}, [3]float64{0, -26.4, -60.0}},
		{200, fxTone, 0, -4.8, [3]int{3785, 338, 414}, [3]float64{0, -18.9, -60.0}},
		{205, fxTone, 0, -4.6, [3]int{3773, 338, 414}, [3]float64{0, -30.2, -60.0}},
		{210, fxTone, 0, -4.7, [3]int{3773, 356, 414}, [3]float64{0, -31.1, -16.8}},
		{215, fxTone, 0, -4.7, [3]int{3773, 356, 414}, [3]float64{0, -27.2, -31.4}},
		{220, fxTone, 0, -24.2, [3]int{233, 3779, 571}, [3]float64{0, 10.3, -8.6}},
		{225, fxTone, 0, -25.6, [3]int{233, 3779, 741}, [3]float64{0, 16.5, -10.4}},
		{230, fxTone, 0, -25.8, [3]int{233, 3779, 571}, [3]float64{0, 16.7, -8.9}},
		{235, fxTone, 0, -25.6, [3]int{239, 3755, 571}, [3]float64{0, 10.1, -8.0}},
		{240, fxTone, 0, -32.6, [3]int{239, 997, 571}, [3]float64{0, -13.4, -60.0}},
		{250, fxTone, 0, -33.9, [3]int{239, 991, 571}, [3]float64{0, -13.5, -60.0}},
	},
	234: { // C4
		{5, fxTone, 0, -32.3, [3]int{239, 991, 0}, [3]float64{0, -14.7, 0.0}},
		{20, fxTone, 0, -32.3, [3]int{239, 991, 0}, [3]float64{0, -14.9, 0.0}},
		{25, fxTone, 0, -23.0, [3]int{927, 402, 0}, [3]float64{0, -8.5, 0.0}},
		{35, fxTone, 0, -15.8, [3]int{962, 350, 0}, [3]float64{0, -13.7, 0.0}},
		{45, fxTone, 0, -12.8, [3]int{962, 338, 0}, [3]float64{0, 0.1, 0.0}},
		{50, fxTone, 0, -7.5, [3]int{962, 735, 0}, [3]float64{0, -7.4, 0.0}},
		{60, fxTone, 0, -5.8, [3]int{962, 735, 0}, [3]float64{0, -14.8, 0.0}},
		{70, fxTone, 0, -5.5, [3]int{962, 735, 0}, [3]float64{0, 0.0, 0.0}},
		{75, fxTone, 0, -2.7, [3]int{956, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{85, fxTone, 0, -2.8, [3]int{956, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{90, fxTone, 0, -2.6, [3]int{839, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{95, fxTone, 0, -3.3, [3]int{839, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{100, fxBuzz, 87, -2.5, [3]int{839, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{105, fxTone, 0, -4.1, [3]int{839, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{110, fxTone, 0, -2.4, [3]int{956, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{115, fxTone, 0, -4.2, [3]int{956, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{120, fxTone, 0, -4.3, [3]int{956, 257, 0}, [3]float64{0, 0.1, 0.0}},
		{125, fxTone, 0, -5.1, [3]int{956, 962, 0}, [3]float64{0, -4.1, 0.0}},
		{130, fxTone, 0, -6.4, [3]int{1913, 986, 0}, [3]float64{0, -2.5, 0.0}},
		{135, fxTone, 0, -6.1, [3]int{1913, 1330, 0}, [3]float64{0, -10.2, 0.0}},
		{140, fxTone, 0, -8.0, [3]int{1913, 986, 0}, [3]float64{0, -17.1, 0.0}},
		{145, fxTone, 0, -7.1, [3]int{1913, 986, 0}, [3]float64{0, -10.8, 0.0}},
		{150, fxNoise, 0, -8.7, [3]int{1913, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{155, fxNoise, 0, -9.3, [3]int{974, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{160, fxNoise, 0, -11.2, [3]int{974, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{165, fxNoise, 0, -9.1, [3]int{1913, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{175, fxNoise, 0, -12.9, [3]int{1914, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{180, fxNoise, 0, -4.6, [3]int{2957, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{185, fxNoise, 0, -7.4, [3]int{2957, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{190, fxTone, 0, -6.1, [3]int{2957, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{205, fxTone, 0, -6.4, [3]int{2962, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{210, fxTone, 0, -7.0, [3]int{2980, 222, 0}, [3]float64{0, -25.3, 0.0}},
		{215, fxTone, 0, -6.8, [3]int{2986, 222, 0}, [3]float64{0, -10.5, 0.0}},
		{220, fxTone, 0, -10.6, [3]int{2986, 222, 0}, [3]float64{0, -24.2, 0.0}},
		{230, fxTone, 0, -12.1, [3]int{2986, 227, 0}, [3]float64{0, -10.5, 0.0}},
		{235, fxTone, 0, -13.5, [3]int{2986, 239, 0}, [3]float64{0, -21.4, 0.0}},
		{240, fxTone, 0, -24.2, [3]int{239, 688, 0}, [3]float64{0, -4.7, 0.0}},
		{245, fxTone, 0, -30.7, [3]int{239, 997, 0}, [3]float64{0, -12.9, 0.0}},
		{255, fxTone, 0, -32.3, [3]int{239, 997, 0}, [3]float64{0, -15.0, 0.0}},
		{265, fxTone, 0, -32.3, [3]int{239, 478, 0}, [3]float64{0, -12.0, 0.0}},
	},
	235: { // C5
		{5, fxTone, 0, -28.7, [3]int{239, 467, 297}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -28.7, [3]int{239, 467, 297}, [3]float64{0, -0.0, -60.0}},
		{15, fxTone, 0, -28.7, [3]int{239, 665, 297}, [3]float64{0, -14.5, -60.0}},
		{20, fxTone, 0, -28.7, [3]int{239, 665, 297}, [3]float64{0, -2.1, -60.0}},
		{25, fxTone, 0, -24.8, [3]int{309, 764, 297}, [3]float64{0, -10.9, -60.0}},
		{30, fxTone, 0, -12.4, [3]int{2694, 764, 297}, [3]float64{0, -11.0, -60.0}},
		{40, fxTone, 0, -9.1, [3]int{2700, 764, 297}, [3]float64{0, -10.6, -60.0}},
		{70, fxTone, 0, -8.6, [3]int{2694, 764, 297}, [3]float64{0, -10.2, -60.0}},
		{75, fxTone, 0, -15.1, [3]int{770, 764, 297}, [3]float64{0, 0.0, -60.0}},
		{80, fxTone, 0, -12.5, [3]int{746, 892, 297}, [3]float64{0, -10.8, -60.0}},
		{110, fxTone, 0, -1.8, [3]int{618, 898, 297}, [3]float64{0, -3.3, -60.0}},
		{115, fxTone, 0, -1.2, [3]int{606, 898, 297}, [3]float64{0, -3.3, -13.8}},
		{125, fxTone, 0, -0.8, [3]int{595, 898, 297}, [3]float64{0, -4.0, -5.3}},
		{130, fxTone, 0, -0.4, [3]int{595, 297, 898}, [3]float64{0, -2.8, -4.9}},
		{145, fxTone, 0, -0.1, [3]int{595, 297, 892}, [3]float64{0, -0.1, -14.9}},
		{150, fxTone, 0, -0.0, [3]int{297, 297, 892}, [3]float64{0, 0.0, -60.0}},
		{160, fxTone, 0, -0.1, [3]int{297, 297, 892}, [3]float64{0, -0.3, -60.0}},
		{165, fxTone, 0, -0.1, [3]int{496, 297, 892}, [3]float64{0, -0.5, -60.0}},
		{175, fxTone, 0, -0.1, [3]int{496, 297, 892}, [3]float64{0, -0.7, -60.0}},
		{185, fxTone, 0, -0.2, [3]int{432, 297, 892}, [3]float64{0, -0.0, -60.0}},
		{190, fxTone, 0, -0.1, [3]int{338, 192, 892}, [3]float64{0, -14.9, -60.0}},
		{195, fxTone, 0, -0.3, [3]int{379, 181, 892}, [3]float64{0, 0.0, -60.0}},
		{200, fxTone, 0, -0.8, [3]int{338, 181, 892}, [3]float64{0, -60.0, -60.0}},
		{205, fxTone, 0, -0.3, [3]int{309, 181, 892}, [3]float64{0, -60.0, -60.0}},
		{215, fxTone, 0, 0.2, [3]int{297, 181, 892}, [3]float64{0, -60.0, -60.0}},
		{240, fxTone, 0, -3.0, [3]int{297, 181, 892}, [3]float64{0, -60.0, -60.0}},
		{245, fxTone, 0, -3.1, [3]int{268, 181, 892}, [3]float64{0, -60.0, -60.0}},
		{250, fxTone, 0, -3.2, [3]int{181, 181, 892}, [3]float64{0, -60.0, -60.0}},
		{270, fxTone, 0, -5.6, [3]int{175, 181, 892}, [3]float64{0, -60.0, -60.0}},
		{275, fxTone, 0, -6.0, [3]int{175, 362, 892}, [3]float64{0, -60.0, -60.0}},
		{345, fxTone, 0, -7.4, [3]int{175, 362, 892}, [3]float64{0, -60.0, -60.0}},
		{350, fxTone, 0, -9.8, [3]int{175, 362, 892}, [3]float64{0, -0.0, -60.0}},
		{355, fxTone, 0, -10.9, [3]int{175, 373, 892}, [3]float64{0, -13.6, -60.0}},
		{360, fxTone, 0, -27.4, [3]int{187, 612, 892}, [3]float64{0, -11.8, -60.0}},
		{365, fxTone, 0, -28.4, [3]int{198, 612, 892}, [3]float64{0, -9.6, -60.0}},
		{370, fxTone, 0, -28.7, [3]int{239, 991, 892}, [3]float64{0, -5.0, -60.0}},
		{375, fxTone, 0, -28.7, [3]int{239, 991, 892}, [3]float64{0, -14.5, -60.0}},
		{380, fxTone, 0, -28.7, [3]int{239, 991, 892}, [3]float64{0, -13.8, -60.0}},
	},
	236: { // C6
		{5, fxTone, 0, -31.9, [3]int{233, 478, 583}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -31.9, [3]int{233, 478, 583}, [3]float64{0, -0.0, -60.0}},
		{15, fxTone, 0, -31.9, [3]int{239, 980, 583}, [3]float64{0, -13.4, -0.1}},
		{20, fxTone, 0, -31.3, [3]int{239, 980, 583}, [3]float64{0, -11.9, -12.9}},
		{25, fxTone, 0, -26.2, [3]int{2012, 991, 595}, [3]float64{0, -14.7, -15.1}},
		{30, fxTone, 0, -25.8, [3]int{2029, 980, 997}, [3]float64{0, -10.5, -15.7}},
		{35, fxTone, 0, -17.1, [3]int{2041, 227, 595}, [3]float64{0, -7.5, -9.8}},
		{40, fxTone, 0, -16.7, [3]int{2029, 227, 1569}, [3]float64{0, -14.8, -17.3}},
		{55, fxTone, 0, -11.9, [3]int{2023, 227, 1569}, [3]float64{0, -3.1, -60.0}},
		{65, fxTone, 0, -7.9, [3]int{2023, 227, 1569}, [3]float64{0, -60.0, -60.0}},
		{70, fxTone, 0, -8.6, [3]int{2023, 1475, 1125}, [3]float64{0, -60.0, -60.0}},
		{75, fxTone, 0, -8.9, [3]int{2023, 1475, 1125}, [3]float64{0, -2.6, -60.0}},
		{80, fxTone, 0, -5.1, [3]int{2023, 1475, 1125}, [3]float64{0, -60.0, -60.0}},
		{90, fxTone, 0, -5.6, [3]int{1738, 1475, 1125}, [3]float64{0, -0.9, -1.5}},
		{100, fxTone, 0, -4.3, [3]int{1376, 1143, 1125}, [3]float64{0, -14.2, -14.8}},
		{110, fxTone, 0, -2.6, [3]int{1067, 1143, 2799}, [3]float64{0, -9.8, -2.4}},
		{130, fxTone, 0, -5.1, [3]int{904, 647, 641}, [3]float64{0, 0.0, 0.0}},
		{135, fxTone, 0, -6.0, [3]int{904, 1942, 641}, [3]float64{0, 2.7, 0.0}},
		{140, fxTone, 0, -6.5, [3]int{904, 1942, 910}, [3]float64{0, 1.8, -7.0}},
		{145, fxTone, 0, -5.6, [3]int{1930, 1942, 910}, [3]float64{0, -1.8, -6.2}},
		{150, fxTone, 0, -5.5, [3]int{1936, 1942, 1679}, [3]float64{0, -9.1, -13.1}},
		{160, fxTone, 0, -5.7, [3]int{1930, 898, 910}, [3]float64{0, -5.1, -8.5}},
		{165, fxTone, 0, -6.7, [3]int{906, 898, 647}, [3]float64{0, -1.4, -10.3}},
		{175, fxTone, 0, -5.5, [3]int{898, 898, 647}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, -3.0, [3]int{904, 641, 2782}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, -2.5, [3]int{904, 641, 2782}, [3]float64{0, -13.7, -5.0}},
		{200, fxTone, 0, -3.7, [3]int{1225, 1096, 2898}, [3]float64{0, -0.5, 7.4}},
		{205, fxTone, 0, -4.1, [3]int{1330, 1073, 1382}, [3]float64{0, -10.7, -12.7}},
		{240, fxTone, 0, -9.7, [3]int{2029, 1096, 1195}, [3]float64{0, -60.0, -60.0}},
		{245, fxTone, 0, -8.8, [3]int{2035, 338, 338}, [3]float64{0, -60.0, -60.0}},
		{265, fxTone, 0, -13.4, [3]int{2035, 338, 338}, [3]float64{0, -60.0, -60.0}},
		{280, fxTone, 0, -19.8, [3]int{2035, 571, 531}, [3]float64{0, -7.5, -11.0}},
		{285, fxTone, 0, -31.9, [3]int{239, 571, 531}, [3]float64{0, -8.0, -10.3}},
		{295, fxTone, 0, -31.9, [3]int{239, 764, 566}, [3]float64{0, -14.6, -0.1}},
		{300, fxTone, 0, -31.9, [3]int{239, 764, 566}, [3]float64{0, -10.4, -60.0}},
		{305, fxTone, 0, -31.9, [3]int{239, 478, 566}, [3]float64{0, -13.0, -60.0}},
	},
	237: { // C7
		{5, fxTone, 0, -28.4, [3]int{239, 472, 332}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -28.4, [3]int{239, 472, 332}, [3]float64{0, -12.2, -60.0}},
		{20, fxTone, 0, -28.4, [3]int{245, 472, 332}, [3]float64{0, -12.2, -60.0}},
		{25, fxTone, 0, -26.0, [3]int{490, 262, 332}, [3]float64{0, -4.9, -60.0}},
		{35, fxTone, 0, -14.6, [3]int{513, 210, 332}, [3]float64{0, 0.0, -60.0}},
		{45, fxTone, 0, -13.3, [3]int{519, 210, 332}, [3]float64{0, -60.0, -60.0}},
		{50, fxTone, 0, -7.3, [3]int{519, 210, 332}, [3]float64{0, -9.2, -60.0}},
		{65, fxTone, 0, -3.8, [3]int{519, 309, 332}, [3]float64{0, -6.1, -60.0}},
		{70, fxTone, 0, -5.0, [3]int{519, 548, 332}, [3]float64{0, -0.0, -60.0}},
		{75, fxTone, 0, -2.6, [3]int{525, 630, 332}, [3]float64{0, -8.5, -14.1}},
		{90, fxTone, 0, -3.9, [3]int{1382, 939, 1073}, [3]float64{0, -5.6, -6.6}},
		{110, fxTone, 0, -7.3, [3]int{2035, 1458, 1726}, [3]float64{0, -8.6, -9.5}},
		{120, fxTone, 0, -4.4, [3]int{2035, 1458, 1306}, [3]float64{0, -7.0, -7.9}},
		{130, fxTone, 0, -4.3, [3]int{945, 729, 711}, [3]float64{0, -0.3, -1.3}},
		{140, fxTone, 0, -4.3, [3]int{945, 729, 811}, [3]float64{0, -11.8, -14.8}},
		{155, fxTone, 0, -4.5, [3]int{1860, 1499, 1580}, [3]float64{0, -3.4, -13.9}},
		{165, fxTone, 0, -4.2, [3]int{1860, 2035, 1213}, [3]float64{0, -3.0, -9.5}},
		{175, fxTone, 0, -3.1, [3]int{758, 1230, 997}, [3]float64{0, -2.0, -2.7}},
		{185, fxTone, 0, -1.8, [3]int{758, 986, 904}, [3]float64{0, -3.0, -5.8}},
		{200, fxTone, 0, -3.9, [3]int{1580, 1277, 1330}, [3]float64{0, -1.2, -9.3}},
		{215, fxTone, 0, -2.1, [3]int{1668, 1271, 1487}, [3]float64{0, -2.0, -3.5}},
		{220, fxTone, 0, -1.7, [3]int{571, 1230, 968}, [3]float64{0, -1.3, -2.9}},
		{230, fxTone, 0, 0.0, [3]int{571, 793, 776}, [3]float64{0, -0.2, -9.3}},
		{240, fxTone, 0, -0.7, [3]int{1015, 974, 781}, [3]float64{0, -3.3, -10.7}},
		{260, fxTone, 0, -1.1, [3]int{1260, 1020, 1032}, [3]float64{0, -3.4, -6.4}},
		{270, fxTone, 0, 0.8, [3]int{706, 606, 589}, [3]float64{0, -5.3, -11.0}},
		{280, fxTone, 0, 1.3, [3]int{706, 688, 443}, [3]float64{0, -3.8, -10.8}},
		{285, fxTone, 0, 0.9, [3]int{706, 688, 227}, [3]float64{0, -9.1, -11.2}},
		{300, fxTone, 0, -0.3, [3]int{945, 951, 694}, [3]float64{0, -5.3, -12.3}},
		{310, fxTone, 0, 0.5, [3]int{676, 951, 694}, [3]float64{0, -0.3, -3.9}},
		{315, fxTone, 0, 1.0, [3]int{641, 659, 239}, [3]float64{0, -3.1, -8.9}},
		{335, fxTone, 0, 0.6, [3]int{426, 601, 204}, [3]float64{0, -9.5, -12.3}},
		{360, fxTone, 0, -4.4, [3]int{484, 321, 157}, [3]float64{0, -9.4, -14.5}},
		{365, fxTone, 0, -5.7, [3]int{484, 292, 157}, [3]float64{0, -10.4, -60.0}},
		{405, fxTone, 0, -27.2, [3]int{484, 227, 157}, [3]float64{0, -4.2, -60.0}},
		{410, fxTone, 0, -28.4, [3]int{239, 227, 157}, [3]float64{0, -13.8, -60.0}},
		{420, fxTone, 0, -28.4, [3]int{239, 227, 157}, [3]float64{0, -60.0, -60.0}},
	},
	238: { // C8
		{5, fxTone, 0, -31.5, [3]int{239, 280, 402}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -31.5, [3]int{239, 280, 402}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -31.5, [3]int{239, 280, 402}, [3]float64{0, -13.1, -60.0}},
		{20, fxTone, 0, -19.6, [3]int{951, 991, 402}, [3]float64{0, -1.2, -1.3}},
		{25, fxTone, 0, -13.8, [3]int{974, 991, 402}, [3]float64{0, -2.2, -4.6}},
		{30, fxTone, 0, -11.4, [3]int{1335, 280, 402}, [3]float64{0, -0.5, -1.2}},
		{35, fxTone, 0, -8.1, [3]int{1335, 1195, 402}, [3]float64{0, -0.2, -0.2}},
		{40, fxTone, 0, -2.8, [3]int{1335, 886, 729}, [3]float64{0, -0.3, -0.3}},
		{45, fxTone, 0, -1.9, [3]int{1184, 886, 729}, [3]float64{0, -60.0, -60.0}},
		{50, fxTone, 0, -1.9, [3]int{1184, 886, 729}, [3]float64{0, -60.0, -60.0}},
		{55, fxNoise, 0, -1.7, [3]int{1172, 886, 729}, [3]float64{0, -0.1, -0.1}},
		{60, fxNoise, 0, -1.5, [3]int{1178, 886, 729}, [3]float64{0, -0.1, -0.1}},
		{65, fxTone, 0, -1.8, [3]int{1178, 886, 729}, [3]float64{0, -60.0, -60.0}},
		{70, fxTone, 0, -0.6, [3]int{1691, 886, 1370}, [3]float64{0, -60.0, -60.0}},
		{75, fxNoise, 0, -1.7, [3]int{1178, 886, 1370}, [3]float64{0, -0.1, 0.3}},
		{80, fxNoise, 0, -1.6, [3]int{1178, 886, 676}, [3]float64{0, -0.1, -0.1}},
		{85, fxTone, 0, -0.6, [3]int{1691, 886, 676}, [3]float64{0, -60.0, -60.0}},
		{90, fxTone, 0, -1.7, [3]int{1306, 892, 676}, [3]float64{0, -60.0, -60.0}},
		{95, fxTone, 0, -1.6, [3]int{1178, 892, 676}, [3]float64{0, -0.1, -0.1}},
		{100, fxTone, 0, -1.9, [3]int{1178, 892, 531}, [3]float64{0, -0.1, -0.0}},
		{105, fxTone, 0, -2.4, [3]int{1190, 986, 531}, [3]float64{0, -0.9, -2.3}},
		{110, fxTone, 0, -2.8, [3]int{840, 986, 531}, [3]float64{0, -0.5, -1.4}},
		{115, fxTone, 0, -4.4, [3]int{840, 1125, 531}, [3]float64{0, -2.4, -2.9}},
		{120, fxTone, 0, -18.3, [3]int{793, 986, 373}, [3]float64{0, -0.0, 0.1}},
		{125, fxTone, 0, -31.1, [3]int{251, 986, 373}, [3]float64{0, -2.7, -60.0}},
		{130, fxTone, 0, -31.5, [3]int{245, 536, 373}, [3]float64{0, -12.9, -60.0}},
		{135, fxTone, 0, -31.5, [3]int{245, 536, 373}, [3]float64{0, -14.6, -60.0}},
		{140, fxTone, 0, -31.5, [3]int{239, 478, 373}, [3]float64{0, -11.2, -60.0}},
		{145, fxTone, 0, -31.5, [3]int{239, 478, 373}, [3]float64{0, -60.0, -60.0}},
	},
	239: { // C9
		{5, fxTone, 0, -32.0, [3]int{239, 997, 1114}, [3]float64{0, -14.5, -60.0}},
		{15, fxTone, 0, -32.0, [3]int{239, 997, 1114}, [3]float64{0, -14.9, -60.0}},
		{20, fxTone, 0, -30.4, [3]int{239, 997, 1114}, [3]float64{0, -0.1, -60.0}},
		{25, fxTone, 0, -18.4, [3]int{898, 362, 1114}, [3]float64{0, -4.5, -60.0}},
		{30, fxTone, 0, -15.0, [3]int{1003, 362, 1114}, [3]float64{0, 0.1, -1.2}},
		{35, fxTone, 0, -11.4, [3]int{1709, 362, 1114}, [3]float64{0, -1.3, -1.4}},
		{40, fxTone, 0, -7.1, [3]int{1709, 1155, 1114}, [3]float64{0, -1.8, -2.6}},
		{45, fxTone, 0, -2.9, [3]int{1709, 1143, 595}, [3]float64{0, -1.3, -1.4}},
		{50, fxTone, 0, -2.5, [3]int{1709, 1155, 1353}, [3]float64{0, -2.3, -4.5}},
		{55, fxTone, 0, -3.7, [3]int{1102, 1155, 746}, [3]float64{0, -0.1, -3.4}},
		{60, fxTone, 0, -3.1, [3]int{869, 1155, 1254}, [3]float64{0, -0.5, -1.2}},
		{65, fxTone, 0, -3.1, [3]int{869, 1143, 1254}, [3]float64{0, -0.3, -1.2}},
		{70, fxTone, 0, -3.0, [3]int{1102, 1475, 1260}, [3]float64{0, -0.5, -3.8}},
		{75, fxTone, 0, -2.8, [3]int{980, 1073, 1260}, [3]float64{0, -1.4, -2.7}},
		{80, fxTone, 0, -2.1, [3]int{799, 1067, 1260}, [3]float64{0, -0.7, -1.6}},
		{85, fxTone, 0, -3.0, [3]int{799, 991, 1260}, [3]float64{0, -0.8, -1.2}},
		{90, fxTone, 0, -3.5, [3]int{980, 991, 1283}, [3]float64{0, -1.6, -6.2}},
		{95, fxTone, 0, -4.3, [3]int{980, 980, 1143}, [3]float64{0, -2.0, -7.9}},
		{100, fxTone, 0, -8.8, [3]int{694, 991, 525}, [3]float64{0, -0.3, -2.9}},
		{105, fxTone, 0, -25.4, [3]int{694, 991, 525}, [3]float64{0, -0.0, 0.0}},
		{110, fxTone, 0, -32.0, [3]int{239, 397, 525}, [3]float64{0, -0.8, -7.7}},
		{115, fxTone, 0, -32.0, [3]int{239, 397, 525}, [3]float64{0, -13.5, -14.8}},
		{120, fxTone, 0, -32.0, [3]int{222, 397, 525}, [3]float64{0, -0.0, -0.0}},
		{125, fxTone, 0, -32.0, [3]int{222, 397, 525}, [3]float64{0, -60.0, -60.0}},
	},
	252: { // M0
		{5, fxTone, 0, -30.5, [3]int{239, 986, 1627}, [3]float64{0, -13.9, -60.0}},
		{15, fxTone, 0, -30.5, [3]int{239, 991, 1627}, [3]float64{0, -14.8, -60.0}},
		{25, fxTone, 0, -2.4, [3]int{1755, 1609, 1627}, [3]float64{0, -11.4, -60.0}},
		{35, fxTone, 0, 1.1, [3]int{1790, 1609, 1627}, [3]float64{0, -12.9, -60.0}},
		{60, fxTone, 0, 0.8, [3]int{1814, 1574, 1627}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, -2.1, [3]int{1819, 1598, 1627}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, -2.1, [3]int{1819, 1598, 1627}, [3]float64{0, -1.0, -60.0}},
		{210, fxTone, 0, -2.9, [3]int{1784, 1574, 1627}, [3]float64{0, -0.9, -60.0}},
		{215, fxTone, 0, -3.1, [3]int{1784, 1574, 1627}, [3]float64{0, -60.0, -60.0}},
		{930, fxTone, 0, -24.3, [3]int{1796, 1580, 1627}, [3]float64{0, -14.7, -60.0}},
		{935, fxTone, 0, -24.8, [3]int{1796, 251, 1627}, [3]float64{0, -16.4, -60.0}},
		{950, fxTone, 0, -24.7, [3]int{1819, 222, 1627}, [3]float64{0, -15.6, -60.0}},
		{980, fxTone, 0, -24.7, [3]int{1773, 251, 1627}, [3]float64{0, -60.0, -60.0}},
		{985, fxTone, 0, -24.8, [3]int{1796, 251, 1627}, [3]float64{0, -11.5, -60.0}},
		{990, fxTone, 0, -25.0, [3]int{1825, 251, 1627}, [3]float64{0, -12.3, -0.9}},
		{995, fxTone, 0, -25.7, [3]int{1825, 1475, 1627}, [3]float64{0, -13.8, -13.4}},
		{1000, fxTone, 0, -26.1, [3]int{1825, 1475, 1627}, [3]float64{0, -11.2, -15.2}},
		{1005, fxTone, 0, -26.6, [3]int{1814, 251, 1627}, [3]float64{0, -14.1, -13.7}},
		{1010, fxTone, 0, -26.9, [3]int{1805, 245, 478}, [3]float64{0, -12.5, -14.4}},
		{1030, fxTone, 0, -26.9, [3]int{1800, 245, 478}, [3]float64{0, -60.0, -60.0}},
		{1035, fxTone, 0, -27.1, [3]int{1795, 245, 478}, [3]float64{0, -1.7, -60.0}},
		{1060, fxTone, 0, -26.8, [3]int{1808, 233, 583}, [3]float64{0, -10.6, -14.1}},
		{1075, fxTone, 0, -26.7, [3]int{1831, 227, 1679}, [3]float64{0, -11.5, -12.2}},
		{1085, fxTone, 0, -26.9, [3]int{1819, 513, 1679}, [3]float64{0, -13.5, -15.3}},
		{1110, fxTone, 0, -27.1, [3]int{1796, 292, 443}, [3]float64{0, -14.4, -16.5}},
		{1130, fxTone, 0, -28.9, [3]int{1779, 612, 840}, [3]float64{0, -8.9, -12.9}},
		{1135, fxBuzz, 128, -30.6, [3]int{957, 1781, 2639}, [3]float64{0, 26.7, 11.7}},
		{1160, fxBuzz, 128, -30.6, [3]int{1014, 1784, 2636}, [3]float64{0, 24.6, 10.0}},
		{1165, fxTone, 0, -28.8, [3]int{1790, 239, 1586}, [3]float64{0, -9.2, -13.0}},
		{1210, fxTone, 0, -28.8, [3]int{1802, 251, 461}, [3]float64{0, -4.6, -9.0}},
		{1215, fxTone, 0, -30.5, [3]int{239, 496, 443}, [3]float64{0, -5.8, -7.1}},
		{1220, fxTone, 0, -30.5, [3]int{239, 991, 443}, [3]float64{0, -12.5, -14.5}},
		{1230, fxTone, 0, -30.5, [3]int{233, 496, 397}, [3]float64{0, -2.2, -60.0}},
	},
	254: { // M2
		{5, fxBuzz, 128, -33.0, [3]int{540, 1687, 2856}, [3]float64{0, 37.6, 43.5}},
		{10, fxBuzz, 177, -33.4, [3]int{540, 1707, 2858}, [3]float64{0, 38.1, 44.6}},
		{15, fxBuzz, 498, -32.9, [3]int{540, 1707, 2866}, [3]float64{0, 4.2, 8.3}},
		{20, fxBuzz, 498, -10.7, [3]int{539, 1703, 2866}, [3]float64{0, 2.2, 6.6}},
		{25, fxBuzz, 85, -8.8, [3]int{530, 1703, 2866}, [3]float64{0, 2.2, 6.6}},
		{45, fxBuzz, 87, -4.0, [3]int{502, 1724, 2919}, [3]float64{0, 2.9, 7.1}},
		{155, fxBuzz, 87, -6.8, [3]int{735, 1954, 3166}, [3]float64{0, 3.2, 7.2}},
		{285, fxBuzz, 87, -6.3, [3]int{987, 2198, 3417}, [3]float64{0, 5.9, 9.2}},
		{410, fxBuzz, 87, -5.1, [3]int{714, 1934, 3145}, [3]float64{0, 2.8, 6.9}},
		{525, fxBuzz, 87, -6.1, [3]int{488, 1710, 2903}, [3]float64{0, 2.8, 7.5}},
		{530, fxBuzz, 103, -6.4, [3]int{488, 1710, 2903}, [3]float64{0, 2.5, 7.2}},
		{535, fxBuzz, 492, -13.3, [3]int{489, 1710, 2902}, [3]float64{0, 2.4, 7.2}},
		{540, fxBuzz, 83, -13.5, [3]int{489, 1710, 2902}, [3]float64{0, 2.7, 7.2}},
		{545, fxBuzz, 492, -21.9, [3]int{491, 1712, 2903}, [3]float64{0, 4.7, 9.4}},
		{550, fxBuzz, 492, -30.7, [3]int{491, 1710, 2901}, [3]float64{0, 5.4, 9.9}},
		{555, fxBuzz, 492, -33.1, [3]int{491, 1707, 2899}, [3]float64{0, 38.2, 43.8}},
	},
}

func fxFrame(p fxPoint) frame {
	g := math.Pow(10, p.db/20.0)
	var f frame
	if p.kind == fxNoise {
		f = noiseFrame(lGlottal, float64(p.freq[0])*0.85, float64(p.freq[0])*1.15, 24*g, 18*g, 200)
	} else {
		base := 28.0
		if p.freq[1] > 0 {
			base = 24
		}
		for i := range 3 {
			f.freq[i] = float64(p.freq[i])
			if p.freq[i] > 0 {
				f.vol[i] = base * g * math.Pow(10, p.rel[i]/20.0)
			}
		}
		if p.kind == fxBuzz {
			f.pitch = float64(p.pitch)
			f.voiced = true
		}
	}
	if p.db <= -60 {
		return quiet(f)
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
