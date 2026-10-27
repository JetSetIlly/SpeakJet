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

// how much longer than the manual's length the chip plays the vowels, nasals,
// liquids and diphthongs (codes 128 to 164): a fixed time, rather than a
// multiple of the length. repeating a sound on the chip (the VoxProbe
// recording) adds just the manual's length for each extra copy (AW 70ms; IY,
// UW and MM six times are one of them and five times 70ms, to within 6ms), so
// the first copy is the manual's length and a fixed time more. 28ms matches
// the VoxRecord recording's third take nearly as well as a length fitted to
// each of the 37 sounds did: sounds played alone within 13ms on average (13ms
// before), the 30 connected groups within 15ms (13ms). it is stretched by
// Speed with the rest of the sound
const transitionMs = 28.0

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
// fricatives, which no group has, keep their first-step lengths.
//
// the stops, the affricates and the voiceless fricatives SE, SO, SH, FF and TH
// that were fitted as whole sounds (see frSE) have their lengths from that fit
// instead, matched to each played alone, with TT and SE also fitted to the
// connected groups. that takes the groups to within 2% (12ms on average).
// TS is 190ms, measured from both units
//
// the vowels, nasals, liquids and diphthongs (codes 128 to 164) are 0 here:
// they are the manual's length and transitionMs more
var allophoneLength = [...]float64{
	0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00,
	0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00,
	0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00,
	0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00, 0.00,
	0.00, 0.8686, 0.84, 0.99, 0.88, 0.88, 1.63, 1.40, 2.67,
	3.27, 1.40, 1.40, 3.74, 3.16, 1.40, 1.40, 1.40, 1.40,
	2.4090, 1.17, 1.15, 1.11, 1.49, 2.80, 2.23, 1.17, 1.77,
	2.07, 2.78, 1.117647, 2.44, 1.40, 2.40, 2.46, 1.40, 1.40,
}

// how much louder (in dB) the chip plays each allophone than the oscillator
// volumes here give, against AW. fitted to the VoxRecord recording's third
// take, whose vowels barely clip: each allophone's level (where it is within
// 20dB of its loudest) was matched to the recording's, measured the same way,
// relative to AW. the vowels needed little; the nasals, liquids and hiss were
// too loud here (the hiss levels were set from an earlier, clipped recording,
// which made the vowels read quiet) and BE, BO, PE and PO too quiet
var allophoneGainDB = [...]float64{
	1.1, 2.0, 2.0, 1.2, 2.0, 1.7, 1.3, 0.8, 0.0,
	-1.8, -1.9, -1.3, -3.9, -4.6, -4.5, -4.5, -4.7, -4.2,
	-3.2, -4.2, 0.4, 0.3, 0.8, 0.2, -0.5, -1.1, 1.2,
	0.5, -0.7, 0.9, 0.4, -0.9, -1.0, -0.2, -1.6, -0.6,
	-3.0, -11.3, -2.1, 1.7, -2.2, -3.3, 5.9, 6.3, -2.6,
	2.7, 2.9, 3.0, -4.6, -8.6, -0.4, -0.5, 1.7, 2.1,
	-15.5, -32.3, -31.1, -29.8, -26.3, -15.9, -17.6, -16.6, -26.8,
	-2.1, -3.3, -26.4, 1.5, -3.0, 2.5, -3.8, 6.4, 5.9,
}

// VF-BEGIN
// some voiced sounds are played as the chip plays them, frame by frame. at the
// default pitch each voicing pulse of the VoxRecord recording stands alone (the
// fade is over long before the next pulse), so each pulse of every voiced sound
// alone was fitted as three formant sines with the chip's fade and output
// filter (see voiceFade), on both units, and the two units' pulses paired and
// averaged. each point is a pulse: its time after the one before, the glide to
// it, the three frequencies and their volumes (an absolute level: AW's first
// formant is 24). a formant too weak to measure takes the frequency it has
// where it is strongest. these are the sounds whose recordings this brings
// closer on both units: the diphthongs, which move between their formants in
// steps rather than one straight line, the voiced fricatives, which on the chip
// are mostly voicing, and voiced stops such as BO and GE, which are a voice bar
// and then a short vowel-like release with no noise burst to speak of. their
// length is the one measured, not stretched (see allophoneLength)
type vfPoint struct {
	ms, glide float64
	freq      formants
	vol       [3]float64
}

// the allophones played from the chip's frames, by code
var vfPoints = map[int][]vfPoint{
	129: { // IH
		{1.00, 1.00, formants{352, 1750, 2524}, [3]float64{10.64, 7.00, 4.52}},
		{11.18, 11.18, formants{351, 1767, 2486}, [3]float64{21.55, 16.20, 9.46}},
		{11.54, 11.54, formants{347, 1772, 2475}, [3]float64{23.65, 18.11, 10.93}},
		{11.46, 11.46, formants{348, 1771, 2471}, [3]float64{23.66, 18.01, 11.14}},
		{11.44, 11.44, formants{347, 1771, 2471}, [3]float64{23.64, 18.06, 11.16}},
		{11.46, 11.46, formants{347, 1766, 2483}, [3]float64{23.65, 18.16, 10.79}},
		{11.41, 11.41, formants{347, 1772, 2474}, [3]float64{23.59, 18.04, 10.94}},
		{11.46, 11.46, formants{346, 1770, 2471}, [3]float64{22.77, 17.49, 10.08}},
		{11.56, 11.56, formants{346, 1772, 2477}, [3]float64{15.41, 12.37, 6.42}},
		{12.69, 12.69, formants{348, 1751, 2497}, [3]float64{5.84, 5.68, 2.93}},
		{11.50, 11.50, formants{348, 1751, 2497}, [3]float64{5.84, 5.68, 2.93}},
	},
	142: { // NO
		{1.00, 1.00, formants{251, 1149, 2596}, [3]float64{0.81, 0.23, 0.25}},
		{12.39, 12.39, formants{254, 1116, 2604}, [3]float64{7.53, 1.69, 3.23}},
		{11.41, 11.41, formants{251, 1142, 2601}, [3]float64{9.26, 2.04, 4.02}},
		{11.44, 11.44, formants{253, 1154, 2596}, [3]float64{9.10, 2.01, 4.29}},
		{11.44, 11.44, formants{251, 1146, 2620}, [3]float64{9.21, 2.01, 4.02}},
		{11.44, 11.44, formants{253, 1149, 2590}, [3]float64{9.21, 2.08, 3.99}},
		{11.46, 11.46, formants{253, 1154, 2599}, [3]float64{9.20, 2.04, 4.05}},
		{11.52, 11.52, formants{260, 1157, 2555}, [3]float64{6.99, 1.31, 2.92}},
		{11.50, 11.50, formants{260, 1157, 2555}, [3]float64{6.99, 1.31, 2.92}},
	},
	146: { // LO
		{1.00, 1.00, formants{442, 875, 2394}, [3]float64{4.79, 5.02, 0.42}},
		{12.53, 12.53, formants{446, 875, 2465}, [3]float64{9.33, 10.33, 1.45}},
		{11.46, 11.46, formants{446, 875, 2458}, [3]float64{9.32, 10.31, 1.50}},
		{11.44, 11.44, formants{447, 876, 2459}, [3]float64{9.34, 10.37, 1.51}},
		{11.44, 11.44, formants{447, 875, 2463}, [3]float64{9.28, 10.33, 1.46}},
		{11.44, 11.44, formants{447, 876, 2394}, [3]float64{9.30, 10.39, 1.70}},
		{11.46, 11.46, formants{447, 874, 2390}, [3]float64{9.30, 10.28, 1.61}},
		{11.39, 11.39, formants{443, 869, 2450}, [3]float64{5.54, 6.05, 0.56}},
		{11.50, 11.50, formants{443, 869, 2450}, [3]float64{5.54, 6.05, 0.56}},
	},
	149: { // IYRR
		{1.00, 1.00, formants{358, 1336, 2487}, [3]float64{3.97, 2.40, 2.83}},
		{12.15, 12.15, formants{234, 2208, 2552}, [3]float64{12.00, 8.04, 6.25}},
		{11.56, 11.56, formants{248, 2199, 2522}, [3]float64{14.79, 10.39, 8.21}},
		{11.50, 11.50, formants{251, 2123, 2494}, [3]float64{15.21, 9.36, 8.26}},
		{11.50, 11.50, formants{262, 2061, 2441}, [3]float64{15.28, 8.68, 8.73}},
		{11.48, 11.48, formants{256, 2048, 2441}, [3]float64{16.38, 8.91, 7.61}},
		{11.07, 11.07, formants{279, 1936, 2295}, [3]float64{16.90, 9.59, 8.69}},
		{11.47, 11.47, formants{289, 1906, 2259}, [3]float64{16.36, 10.86, 6.08}},
		{11.49, 11.49, formants{311, 1860, 2226}, [3]float64{16.03, 10.48, 6.62}},
		{11.56, 11.56, formants{324, 1790, 2259}, [3]float64{15.30, 11.65, 3.39}},
		{10.02, 10.02, formants{351, 1712, 2441}, [3]float64{16.56, 12.69, 2.11}},
		{11.46, 11.46, formants{358, 1692, 2412}, [3]float64{18.58, 14.54, 2.78}},
		{12.40, 12.40, formants{369, 1646, 2441}, [3]float64{16.97, 13.38, 1.59}},
		{11.48, 11.48, formants{375, 1591, 2441}, [3]float64{16.78, 13.13, 1.71}},
		{11.60, 11.60, formants{396, 1510, 2441}, [3]float64{16.18, 12.84, 1.04}},
		{12.36, 12.36, formants{398, 1471, 2503}, [3]float64{13.98, 11.22, 2.27}},
		{11.62, 11.62, formants{417, 1396, 2334}, [3]float64{11.07, 8.89, 2.40}},
		{11.55, 11.55, formants{439, 1330, 2441}, [3]float64{10.25, 9.06, 1.20}},
		{9.13, 9.13, formants{425, 1336, 2441}, [3]float64{17.90, 15.13, 1.72}},
		{11.73, 11.73, formants{429, 1322, 2441}, [3]float64{10.24, 9.55, 1.29}},
		{11.50, 11.50, formants{429, 1322, 2441}, [3]float64{10.24, 9.55, 1.29}},
	},
	151: { // AXRR
		{1.00, 1.00, formants{532, 1502, 2368}, [3]float64{12.05, 7.97, 5.86}},
		{11.89, 11.89, formants{530, 1501, 2374}, [3]float64{23.40, 15.20, 10.74}},
		{11.78, 11.78, formants{525, 1500, 2347}, [3]float64{26.63, 16.67, 12.24}},
		{11.48, 11.48, formants{520, 1490, 2295}, [3]float64{26.49, 16.89, 12.04}},
		{11.40, 11.40, formants{514, 1474, 2250}, [3]float64{26.24, 16.54, 10.32}},
		{11.49, 11.49, formants{507, 1468, 2426}, [3]float64{25.00, 16.61, 6.10}},
		{11.54, 11.54, formants{502, 1461, 2230}, [3]float64{24.25, 16.01, 10.16}},
		{11.45, 11.45, formants{499, 1446, 2138}, [3]float64{23.77, 16.42, 8.41}},
		{11.48, 11.48, formants{493, 1437, 2084}, [3]float64{23.02, 16.27, 7.66}},
		{11.45, 11.45, formants{486, 1427, 2042}, [3]float64{23.00, 16.12, 6.56}},
		{11.46, 11.46, formants{483, 1414, 1959}, [3]float64{21.75, 15.30, 6.74}},
		{11.48, 11.48, formants{476, 1408, 1933}, [3]float64{21.41, 15.09, 6.00}},
		{11.46, 11.46, formants{468, 1384, 1943}, [3]float64{21.36, 14.98, 5.50}},
		{11.50, 11.50, formants{463, 1377, 2347}, [3]float64{19.81, 15.16, 2.10}},
		{11.46, 11.46, formants{456, 1373, 2347}, [3]float64{19.87, 15.25, 2.11}},
		{11.44, 11.44, formants{449, 1358, 2347}, [3]float64{19.37, 14.87, 2.31}},
		{11.44, 11.44, formants{447, 1355, 2347}, [3]float64{18.55, 15.28, 2.04}},
		{10.89, 10.89, formants{435, 1340, 2347}, [3]float64{19.49, 16.07, 0.90}},
		{11.48, 11.48, formants{433, 1334, 2347}, [3]float64{15.41, 13.03, 0.82}},
		{12.06, 12.06, formants{436, 1324, 2347}, [3]float64{7.40, 6.72, 0.73}},
		{11.50, 11.50, formants{436, 1324, 2347}, [3]float64{7.40, 6.72, 0.73}},
	},
	152: { // AWRR
		{1.00, 1.00, formants{726, 930, 2530}, [3]float64{7.32, 6.79, 2.55}},
		{12.36, 12.36, formants{708, 964, 2486}, [3]float64{19.98, 17.80, 5.78}},
		{11.40, 11.40, formants{694, 981, 2460}, [3]float64{22.75, 21.26, 6.22}},
		{11.48, 11.48, formants{676, 1003, 2397}, [3]float64{22.60, 20.86, 6.42}},
		{11.42, 11.42, formants{662, 1025, 2350}, [3]float64{22.83, 20.99, 6.31}},
		{11.42, 11.42, formants{646, 1047, 2345}, [3]float64{22.62, 21.16, 6.17}},
		{11.40, 11.40, formants{631, 1066, 2247}, [3]float64{23.13, 20.87, 6.14}},
		{11.54, 11.54, formants{615, 1090, 2207}, [3]float64{22.91, 20.36, 5.74}},
		{11.36, 11.36, formants{599, 1110, 2164}, [3]float64{23.02, 20.37, 5.53}},
		{12.73, 12.73, formants{590, 1127, 2130}, [3]float64{20.81, 17.55, 5.24}},
		{11.42, 11.42, formants{582, 1139, 2148}, [3]float64{17.76, 15.72, 4.41}},
		{11.44, 11.44, formants{563, 1178, 2094}, [3]float64{17.64, 15.89, 3.77}},
		{11.44, 11.44, formants{547, 1195, 1971}, [3]float64{17.72, 15.93, 4.21}},
		{11.41, 11.41, formants{529, 1233, 2229}, [3]float64{18.15, 15.80, 4.08}},
		{11.44, 11.44, formants{511, 1247, 2153}, [3]float64{18.10, 15.97, 2.93}},
		{11.48, 11.48, formants{492, 1283, 2168}, [3]float64{18.16, 15.73, 4.00}},
		{11.39, 11.39, formants{470, 1305, 2388}, [3]float64{18.57, 15.85, 2.19}},
		{12.54, 12.54, formants{453, 1308, 2382}, [3]float64{13.33, 10.73, 4.80}},
		{11.42, 11.42, formants{448, 1321, 2405}, [3]float64{11.59, 9.75, 2.80}},
		{10.25, 10.25, formants{446, 1318, 2397}, [3]float64{9.51, 9.62, 0.97}},
		{11.50, 11.50, formants{446, 1318, 2397}, [3]float64{9.51, 9.62, 0.97}},
	},
	153: { // OWRR
		{1.00, 1.00, formants{519, 910, 2400}, [3]float64{1.06, 0.89, 0.85}},
		{12.89, 12.89, formants{562, 865, 2400}, [3]float64{6.87, 7.72, 0.86}},
		{9.68, 9.68, formants{529, 883, 2427}, [3]float64{16.52, 20.16, 4.58}},
		{11.48, 11.48, formants{522, 910, 2391}, [3]float64{18.00, 21.92, 5.28}},
		{11.49, 11.49, formants{519, 937, 2315}, [3]float64{18.26, 21.88, 5.42}},
		{13.48, 13.48, formants{524, 950, 2400}, [3]float64{11.75, 13.56, 7.44}},
		{10.74, 10.74, formants{520, 993, 2313}, [3]float64{14.28, 16.08, 2.91}},
		{11.34, 11.34, formants{517, 1024, 2400}, [3]float64{14.71, 16.57, 1.08}},
		{11.43, 11.43, formants{508, 1050, 2511}, [3]float64{14.71, 16.30, 1.96}},
		{11.44, 11.44, formants{497, 1083, 2555}, [3]float64{14.91, 15.93, 3.20}},
		{11.29, 11.29, formants{486, 1113, 2619}, [3]float64{15.74, 16.09, 3.04}},
		{11.38, 11.38, formants{484, 1147, 2584}, [3]float64{17.47, 16.34, 3.30}},
		{11.56, 11.56, formants{479, 1170, 2583}, [3]float64{17.39, 16.17, 2.47}},
		{11.37, 11.37, formants{473, 1205, 2528}, [3]float64{17.05, 15.93, 1.88}},
		{11.44, 11.44, formants{464, 1236, 2379}, [3]float64{17.32, 15.97, 2.06}},
		{11.49, 11.49, formants{456, 1269, 2518}, [3]float64{17.10, 15.56, 2.35}},
		{11.25, 11.25, formants{451, 1305, 2583}, [3]float64{17.41, 15.90, 1.92}},
		{12.62, 12.62, formants{450, 1322, 2482}, [3]float64{12.43, 11.20, 4.01}},
		{9.59, 9.59, formants{449, 1316, 2400}, [3]float64{11.05, 9.98, 1.70}},
		{13.29, 13.29, formants{450, 910, 2400}, [3]float64{5.79, 5.13, 0.38}},
		{11.50, 11.50, formants{450, 910, 2400}, [3]float64{5.79, 5.13, 0.38}},
	},
	156: { // OWIY
		{1.00, 1.00, formants{367, 1006, 2461}, [3]float64{0.43, 0.48, 0.48}},
		{13.15, 13.15, formants{551, 871, 2461}, [3]float64{5.79, 6.16, 1.14}},
		{12.23, 12.23, formants{552, 904, 2543}, [3]float64{10.17, 12.25, 4.25}},
		{11.38, 11.38, formants{534, 917, 2503}, [3]float64{11.71, 13.00, 7.05}},
		{10.49, 10.49, formants{518, 1006, 2427}, [3]float64{14.37, 16.32, 4.82}},
		{11.50, 11.50, formants{498, 1081, 2428}, [3]float64{14.59, 15.80, 5.49}},
		{11.38, 11.38, formants{478, 1156, 2393}, [3]float64{15.07, 15.69, 5.67}},
		{11.38, 11.38, formants{457, 1231, 2424}, [3]float64{16.43, 15.58, 3.73}},
		{10.81, 10.81, formants{436, 1308, 2429}, [3]float64{14.99, 13.52, 7.32}},
		{11.43, 11.43, formants{418, 1380, 2456}, [3]float64{18.27, 16.19, 7.37}},
		{12.54, 12.54, formants{417, 1443, 2457}, [3]float64{14.70, 12.58, 5.85}},
		{11.27, 11.27, formants{390, 1537, 2517}, [3]float64{15.50, 12.45, 6.37}},
		{11.38, 11.38, formants{380, 1597, 2595}, [3]float64{16.72, 12.66, 4.93}},
		{10.52, 10.52, formants{367, 1665, 2473}, [3]float64{18.91, 15.11, 8.64}},
		{11.42, 11.42, formants{357, 1724, 2461}, [3]float64{18.40, 14.76, 9.17}},
		{12.86, 12.86, formants{336, 1809, 2522}, [3]float64{16.10, 13.08, 7.15}},
		{11.21, 11.21, formants{311, 1893, 2554}, [3]float64{16.15, 10.91, 6.30}},
		{11.31, 11.31, formants{294, 1946, 2548}, [3]float64{16.52, 10.38, 7.09}},
		{11.87, 11.87, formants{260, 2042, 2536}, [3]float64{16.66, 9.22, 7.40}},
		{11.43, 11.43, formants{243, 2092, 2561}, [3]float64{16.16, 8.68, 8.00}},
		{11.39, 11.39, formants{245, 2198, 2536}, [3]float64{15.07, 9.97, 8.38}},
		{11.52, 11.52, formants{232, 2231, 2561}, [3]float64{14.13, 8.52, 7.00}},
		{12.68, 12.68, formants{249, 2232, 2545}, [3]float64{7.17, 5.18, 3.47}},
		{11.50, 11.50, formants{249, 2232, 2545}, [3]float64{7.17, 5.18, 3.47}},
	},
	158: { // IYEH
		{1.00, 1.00, formants{595, 1732, 2592}, [3]float64{1.13, 0.41, 0.39}},
		{13.69, 13.69, formants{595, 2137, 2553}, [3]float64{5.36, 3.89, 3.71}},
		{10.56, 10.56, formants{263, 2190, 2496}, [3]float64{13.73, 10.61, 7.87}},
		{12.67, 12.67, formants{255, 2169, 2526}, [3]float64{16.77, 10.04, 8.61}},
		{11.47, 11.47, formants{262, 2146, 2592}, [3]float64{17.51, 10.52, 8.76}},
		{11.04, 11.04, formants{310, 2081, 2510}, [3]float64{17.74, 10.01, 9.51}},
		{11.46, 11.46, formants{347, 2055, 2518}, [3]float64{19.26, 10.71, 10.32}},
		{11.41, 11.41, formants{377, 1999, 2592}, [3]float64{19.86, 9.44, 11.70}},
		{11.46, 11.46, formants{410, 1944, 2592}, [3]float64{19.88, 9.37, 10.56}},
		{11.05, 11.05, formants{433, 1944, 2461}, [3]float64{22.43, 12.88, 9.16}},
		{11.50, 11.50, formants{464, 1901, 2444}, [3]float64{23.21, 13.24, 8.47}},
		{11.41, 11.41, formants{494, 1840, 2477}, [3]float64{23.16, 12.84, 8.85}},
		{11.46, 11.46, formants{524, 1793, 2512}, [3]float64{22.79, 12.85, 7.46}},
		{11.38, 11.38, formants{549, 1771, 2577}, [3]float64{24.62, 11.59, 9.68}},
		{11.47, 11.47, formants{575, 1726, 2484}, [3]float64{24.11, 11.74, 9.96}},
		{11.12, 11.12, formants{595, 1732, 2392}, [3]float64{26.14, 15.40, 11.58}},
		{11.33, 11.33, formants{613, 1687, 2356}, [3]float64{25.88, 15.32, 10.01}},
		{11.97, 11.97, formants{613, 1696, 2335}, [3]float64{18.88, 11.07, 7.06}},
		{13.22, 13.22, formants{610, 1692, 2342}, [3]float64{9.04, 5.22, 3.41}},
		{11.50, 11.50, formants{610, 1692, 2342}, [3]float64{9.04, 5.22, 3.41}},
	},
	159: { // EHLL
		{1.00, 1.00, formants{617, 1681, 2416}, [3]float64{0.57, 0.38, 0.19}},
		{10.47, 10.47, formants{617, 1681, 2416}, [3]float64{4.82, 3.19, 2.38}},
		{11.83, 11.83, formants{618, 1688, 2357}, [3]float64{17.73, 10.48, 7.59}},
		{12.37, 12.37, formants{617, 1681, 2353}, [3]float64{25.21, 14.61, 10.75}},
		{11.51, 11.51, formants{601, 1607, 2408}, [3]float64{24.98, 14.31, 10.23}},
		{11.47, 11.47, formants{587, 1550, 2416}, [3]float64{24.03, 13.82, 10.94}},
		{12.49, 12.49, formants{579, 1469, 2454}, [3]float64{19.73, 12.32, 7.48}},
		{10.43, 10.43, formants{558, 1414, 2503}, [3]float64{21.30, 13.43, 8.51}},
		{11.58, 11.58, formants{545, 1361, 2551}, [3]float64{19.05, 13.16, 6.49}},
		{11.56, 11.56, formants{532, 1288, 2466}, [3]float64{17.32, 12.10, 3.41}},
		{11.43, 11.43, formants{515, 1227, 2485}, [3]float64{15.96, 12.00, 4.87}},
		{11.25, 11.25, formants{498, 1158, 2630}, [3]float64{14.83, 11.29, 3.62}},
		{11.70, 11.70, formants{481, 1096, 2580}, [3]float64{12.90, 11.26, 3.74}},
		{11.59, 11.59, formants{471, 1037, 2633}, [3]float64{11.05, 10.59, 3.05}},
		{11.53, 11.53, formants{457, 971, 2416}, [3]float64{9.68, 10.34, 1.19}},
		{11.47, 11.47, formants{617, 965, 2416}, [3]float64{5.21, 6.13, 0.49}},
		{11.50, 11.50, formants{617, 965, 2416}, [3]float64{5.21, 6.13, 0.49}},
	},
	160: { // IYUW
		{1.00, 1.00, formants{214, 1118, 2504}, [3]float64{4.23, 2.83, 2.45}},
		{12.44, 12.44, formants{208, 2176, 2548}, [3]float64{14.05, 7.71, 5.76}},
		{11.08, 11.08, formants{214, 2208, 2507}, [3]float64{18.49, 9.82, 8.01}},
		{11.49, 11.49, formants{242, 2081, 2548}, [3]float64{15.07, 9.33, 7.62}},
		{11.49, 11.49, formants{260, 2045, 2552}, [3]float64{14.67, 8.20, 8.25}},
		{11.02, 11.02, formants{256, 1926, 2439}, [3]float64{16.79, 10.54, 7.65}},
		{11.49, 11.49, formants{265, 1837, 2504}, [3]float64{15.25, 11.07, 6.06}},
		{10.23, 10.23, formants{305, 1767, 2368}, [3]float64{15.78, 14.49, 7.55}},
		{11.51, 11.51, formants{303, 1680, 2360}, [3]float64{16.44, 13.51, 7.94}},
		{13.04, 13.04, formants{289, 1598, 2419}, [3]float64{13.28, 9.14, 3.37}},
		{9.99, 9.99, formants{311, 1523, 2345}, [3]float64{15.92, 14.61, 5.33}},
		{11.47, 11.47, formants{319, 1447, 2269}, [3]float64{15.83, 14.54, 5.28}},
		{12.52, 12.52, formants{311, 1370, 2225}, [3]float64{15.07, 12.61, 5.51}},
		{11.55, 11.55, formants{320, 1273, 2336}, [3]float64{12.66, 10.78, 2.81}},
		{10.26, 10.26, formants{333, 1205, 2198}, [3]float64{13.66, 12.58, 3.89}},
		{11.54, 11.54, formants{333, 1118, 2321}, [3]float64{15.55, 14.63, 3.30}},
		{11.46, 11.46, formants{336, 1064, 2342}, [3]float64{14.98, 14.53, 3.10}},
		{12.12, 12.12, formants{326, 1062, 2552}, [3]float64{8.86, 8.49, 0.65}},
		{11.50, 11.50, formants{326, 1062, 2552}, [3]float64{8.86, 8.49, 0.65}},
	},
	161: { // AXUW
		{1.00, 1.00, formants{675, 1207, 2378}, [3]float64{2.26, 2.07, 1.53}},
		{10.99, 10.99, formants{531, 1504, 2358}, [3]float64{13.51, 8.71, 6.50}},
		{11.64, 11.64, formants{530, 1500, 2376}, [3]float64{24.66, 15.78, 11.67}},
		{12.77, 12.77, formants{546, 1465, 2378}, [3]float64{22.59, 14.49, 11.76}},
		{11.49, 11.49, formants{570, 1437, 2385}, [3]float64{22.55, 15.03, 11.33}},
		{11.47, 11.47, formants{589, 1397, 2409}, [3]float64{22.13, 14.82, 10.42}},
		{10.37, 10.37, formants{605, 1357, 2379}, [3]float64{25.60, 16.82, 11.24}},
		{11.46, 11.46, formants{624, 1314, 2389}, [3]float64{24.82, 16.60, 10.32}},
		{11.47, 11.47, formants{645, 1280, 2407}, [3]float64{24.58, 16.61, 10.70}},
		{11.51, 11.51, formants{663, 1238, 2411}, [3]float64{23.91, 16.86, 10.89}},
		{11.48, 11.48, formants{675, 1207, 2415}, [3]float64{26.24, 18.88, 11.19}},
		{11.88, 11.88, formants{699, 1180, 2450}, [3]float64{20.31, 15.66, 8.64}},
		{11.51, 11.51, formants{683, 1137, 2432}, [3]float64{23.12, 17.14, 8.71}},
		{11.64, 11.64, formants{594, 1120, 2423}, [3]float64{18.48, 14.54, 5.84}},
		{10.82, 10.82, formants{515, 1090, 2341}, [3]float64{18.61, 15.28, 3.74}},
		{11.56, 11.56, formants{407, 1080, 2511}, [3]float64{16.19, 13.59, 3.08}},
		{11.54, 11.54, formants{343, 1061, 2378}, [3]float64{14.28, 13.74, 1.20}},
		{11.58, 11.58, formants{317, 1059, 2378}, [3]float64{8.03, 7.35, 0.48}},
		{11.50, 11.50, formants{317, 1059, 2378}, [3]float64{8.03, 7.35, 0.48}},
	},
	162: { // IHWW
		{1.00, 1.00, formants{351, 1664, 2435}, [3]float64{4.35, 2.15, 1.70}},
		{11.70, 11.70, formants{354, 1762, 2499}, [3]float64{14.62, 10.95, 6.09}},
		{11.81, 11.81, formants{348, 1732, 2484}, [3]float64{19.41, 14.16, 8.04}},
		{11.44, 11.44, formants{351, 1713, 2435}, [3]float64{22.44, 17.04, 9.58}},
		{11.46, 11.46, formants{355, 1664, 2434}, [3]float64{21.72, 17.43, 9.12}},
		{11.42, 11.42, formants{358, 1595, 2422}, [3]float64{21.35, 16.35, 9.22}},
		{11.43, 11.43, formants{362, 1542, 2378}, [3]float64{20.12, 16.58, 8.63}},
		{12.44, 12.44, formants{369, 1463, 2484}, [3]float64{17.12, 13.04, 3.95}},
		{11.50, 11.50, formants{376, 1416, 2416}, [3]float64{15.99, 12.82, 4.15}},
		{11.50, 11.50, formants{381, 1354, 2439}, [3]float64{15.07, 12.86, 2.93}},
		{11.55, 11.55, formants{386, 1290, 2404}, [3]float64{14.17, 11.95, 3.29}},
		{10.43, 10.43, formants{380, 1255, 2283}, [3]float64{16.49, 15.57, 5.75}},
		{12.06, 12.06, formants{383, 1187, 2443}, [3]float64{15.61, 14.27, 3.15}},
		{11.45, 11.45, formants{393, 1129, 2435}, [3]float64{14.04, 13.58, 2.39}},
		{11.51, 11.51, formants{398, 1064, 2228}, [3]float64{13.16, 13.37, 3.76}},
		{11.47, 11.47, formants{408, 1013, 2435}, [3]float64{12.30, 13.21, 1.68}},
		{11.57, 11.57, formants{413, 953, 2435}, [3]float64{11.25, 13.31, 1.71}},
		{11.54, 11.54, formants{426, 898, 2435}, [3]float64{10.46, 12.38, 1.98}},
		{12.33, 12.33, formants{415, 844, 2448}, [3]float64{8.18, 10.63, 2.46}},
		{11.49, 11.49, formants{430, 811, 2435}, [3]float64{6.70, 9.13, 2.27}},
		{12.16, 12.16, formants{351, 802, 2435}, [3]float64{3.25, 4.83, 1.14}},
		{11.50, 11.50, formants{351, 802, 2435}, [3]float64{3.25, 4.83, 1.14}},
	},
	163: { // AYWW
		{1.00, 1.00, formants{679, 1567, 2175}, [3]float64{1.02, 0.71, 0.70}},
		{11.71, 11.71, formants{716, 1647, 2175}, [3]float64{10.26, 4.78, 1.39}},
		{12.58, 12.58, formants{712, 1671, 2607}, [3]float64{19.40, 9.73, 2.08}},
		{11.67, 11.67, formants{707, 1657, 2377}, [3]float64{26.18, 12.68, 4.52}},
		{11.55, 11.55, formants{698, 1609, 2251}, [3]float64{26.21, 13.45, 6.23}},
		{10.67, 10.67, formants{679, 1567, 2483}, [3]float64{28.31, 15.13, 5.04}},
		{11.44, 11.44, formants{661, 1517, 2467}, [3]float64{26.91, 14.71, 5.48}},
		{11.47, 11.47, formants{649, 1452, 2464}, [3]float64{25.75, 14.36, 4.56}},
		{11.44, 11.44, formants{633, 1397, 2110}, [3]float64{24.43, 14.07, 4.74}},
		{11.55, 11.55, formants{618, 1350, 2115}, [3]float64{23.02, 14.13, 5.69}},
		{11.46, 11.46, formants{602, 1293, 2175}, [3]float64{21.54, 13.92, 6.41}},
		{11.51, 11.51, formants{589, 1234, 2157}, [3]float64{19.90, 14.05, 4.93}},
		{11.46, 11.46, formants{573, 1175, 2170}, [3]float64{19.05, 12.90, 5.33}},
		{11.54, 11.54, formants{560, 1116, 2228}, [3]float64{17.16, 13.35, 4.35}},
		{11.47, 11.47, formants{539, 1067, 2705}, [3]float64{16.03, 13.33, 2.33}},
		{11.58, 11.58, formants{527, 1015, 2710}, [3]float64{14.00, 13.41, 1.93}},
		{11.55, 11.55, formants{515, 952, 2620}, [3]float64{12.43, 12.51, 2.33}},
		{12.22, 12.22, formants{490, 924, 2524}, [3]float64{9.27, 10.25, 4.21}},
		{11.63, 11.63, formants{470, 870, 2538}, [3]float64{7.84, 10.14, 3.79}},
		{11.53, 11.53, formants{459, 814, 2459}, [3]float64{7.25, 9.89, 2.54}},
		{11.77, 11.77, formants{679, 791, 2175}, [3]float64{4.58, 6.03, 1.26}},
		{11.50, 11.50, formants{679, 791, 2175}, [3]float64{4.58, 6.03, 1.26}},
	},
	164: { // OWWW
		{1.00, 1.00, formants{564, 862, 2501}, [3]float64{6.51, 7.02, 0.82}},
		{11.98, 11.98, formants{549, 905, 2529}, [3]float64{10.07, 11.94, 4.16}},
		{11.48, 11.48, formants{561, 891, 2482}, [3]float64{10.26, 13.15, 2.52}},
		{11.46, 11.46, formants{542, 892, 2501}, [3]float64{10.05, 12.72, 4.18}},
		{11.47, 11.47, formants{548, 866, 2387}, [3]float64{9.95, 12.89, 2.26}},
		{11.43, 11.43, formants{519, 869, 2602}, [3]float64{8.82, 12.02, 2.70}},
		{11.46, 11.46, formants{518, 844, 2556}, [3]float64{9.07, 11.88, 3.30}},
		{11.46, 11.46, formants{520, 832, 2506}, [3]float64{9.18, 11.78, 3.43}},
		{11.46, 11.46, formants{485, 827, 2543}, [3]float64{8.68, 11.40, 2.06}},
		{11.48, 11.48, formants{501, 837, 2529}, [3]float64{8.22, 11.09, 1.07}},
		{11.46, 11.46, formants{487, 824, 2595}, [3]float64{8.10, 11.03, 3.34}},
		{11.46, 11.46, formants{476, 823, 2586}, [3]float64{8.17, 11.09, 3.09}},
		{11.48, 11.48, formants{468, 818, 2535}, [3]float64{7.15, 10.09, 2.54}},
		{11.47, 11.47, formants{466, 826, 2584}, [3]float64{7.05, 9.64, 3.75}},
		{11.44, 11.44, formants{457, 811, 2496}, [3]float64{7.20, 9.73, 2.52}},
		{11.43, 11.43, formants{452, 793, 2494}, [3]float64{6.41, 8.73, 2.24}},
		{12.41, 12.41, formants{561, 789, 2501}, [3]float64{2.56, 3.98, 1.02}},
		{11.50, 11.50, formants{561, 789, 2501}, [3]float64{2.56, 3.98, 1.02}},
	},
	167: { // ZZ
		{1.00, 1.00, formants{271, 1499, 2383}, [3]float64{0.40, 0.24, 0.21}},
		{11.45, 11.45, formants{298, 1491, 2394}, [3]float64{4.81, 3.49, 3.70}},
		{11.39, 11.39, formants{274, 1499, 2392}, [3]float64{10.03, 6.86, 7.25}},
		{11.26, 11.26, formants{270, 1498, 2397}, [3]float64{10.21, 6.59, 7.12}},
		{11.17, 11.17, formants{271, 1495, 2391}, [3]float64{10.54, 6.61, 7.17}},
		{11.14, 11.14, formants{273, 1500, 2394}, [3]float64{10.23, 6.74, 7.19}},
		{11.23, 11.23, formants{270, 1499, 2383}, [3]float64{10.36, 6.60, 7.31}},
		{11.21, 11.21, formants{274, 1498, 2392}, [3]float64{10.26, 6.74, 7.06}},
		{11.42, 11.42, formants{276, 1508, 2395}, [3]float64{8.23, 5.19, 5.43}},
		{11.50, 11.50, formants{276, 1508, 2395}, [3]float64{8.23, 5.19, 5.43}},
	},
	168: { // ZH
		{1.00, 1.00, formants{434, 1676, 2563}, [3]float64{6.69, 4.59, 1.88}},
		{10.94, 10.94, formants{441, 1673, 2540}, [3]float64{8.29, 5.79, 2.75}},
		{10.87, 10.87, formants{435, 1682, 2447}, [3]float64{8.14, 5.29, 3.14}},
		{11.05, 11.05, formants{437, 1671, 2488}, [3]float64{7.92, 5.50, 2.21}},
		{11.80, 11.80, formants{434, 1662, 2551}, [3]float64{6.39, 3.57, 2.98}},
		{10.94, 10.94, formants{439, 1677, 2500}, [3]float64{6.58, 4.44, 2.37}},
		{11.50, 11.50, formants{439, 1677, 2500}, [3]float64{6.58, 4.44, 2.37}},
	},
	169: { // DH
		{1.00, 1.00, formants{282, 1589, 2352}, [3]float64{3.90, 3.82, 1.28}},
		{11.36, 11.36, formants{294, 1581, 2464}, [3]float64{6.18, 5.96, 2.72}},
		{11.67, 11.67, formants{272, 1578, 2420}, [3]float64{6.64, 5.96, 2.79}},
		{10.94, 10.94, formants{269, 1586, 2397}, [3]float64{6.64, 5.89, 2.13}},
		{11.87, 11.87, formants{280, 1592, 2379}, [3]float64{5.99, 5.85, 1.88}},
		{10.96, 10.96, formants{281, 1590, 2372}, [3]float64{6.12, 5.78, 2.37}},
		{10.91, 10.91, formants{265, 1590, 2493}, [3]float64{5.74, 5.06, 1.14}},
		{11.12, 11.12, formants{246, 1598, 2331}, [3]float64{2.44, 2.02, 0.76}},
		{11.50, 11.50, formants{246, 1598, 2331}, [3]float64{2.44, 2.02, 0.76}},
	},
	171: { // BO
		{1.00, 1.00, formants{624, 1406, 2392}, [3]float64{1.19, 0.41, 0.17}},
		{11.12, 11.12, formants{274, 1406, 2392}, [3]float64{6.89, 0.33, 0.61}},
		{11.37, 11.37, formants{271, 1406, 2392}, [3]float64{7.40, 0.29, 0.72}},
		{11.30, 11.30, formants{410, 1406, 2392}, [3]float64{8.53, 1.49, 1.65}},
		{11.87, 11.87, formants{624, 1406, 2392}, [3]float64{20.14, 14.98, 7.74}},
		{11.44, 11.44, formants{624, 1405, 2394}, [3]float64{17.24, 12.95, 6.59}},
		{12.09, 12.09, formants{622, 1400, 2395}, [3]float64{8.18, 6.33, 3.09}},
		{11.50, 11.50, formants{622, 1400, 2395}, [3]float64{8.18, 6.33, 3.09}},
	},
	175: { // DO
		{1.00, 1.00, formants{440, 1486, 2485}, [3]float64{0.35, 0.29, 0.43}},
		{11.33, 11.33, formants{263, 1486, 2485}, [3]float64{4.58, 0.28, 0.59}},
		{10.16, 10.16, formants{272, 1486, 2485}, [3]float64{7.37, 0.36, 0.55}},
		{11.47, 11.47, formants{274, 1486, 2485}, [3]float64{7.52, 0.38, 0.72}},
		{12.34, 12.34, formants{440, 1486, 2485}, [3]float64{10.49, 7.15, 4.09}},
		{12.41, 12.41, formants{439, 1486, 2514}, [3]float64{7.92, 5.46, 2.77}},
		{11.77, 11.77, formants{440, 1486, 2485}, [3]float64{1.67, 1.00, 0.26}},
		{11.50, 11.50, formants{440, 1486, 2485}, [3]float64{1.67, 1.00, 0.26}},
	},
	176: { // ED
		{1.00, 1.00, formants{254, 1972, 2404}, [3]float64{2.56, 1.69, 1.06}},
		{11.53, 11.53, formants{239, 1951, 2635}, [3]float64{3.99, 1.83, 1.51}},
		{12.84, 12.84, formants{205, 1951, 2635}, [3]float64{1.99, 0.41, 0.18}},
		{11.50, 11.50, formants{205, 1951, 2635}, [3]float64{1.99, 0.41, 0.18}},
	},
	177: { // OD
		{1.00, 1.00, formants{265, 1504, 2470}, [3]float64{3.32, 3.27, 1.48}},
		{10.81, 10.81, formants{260, 1522, 2447}, [3]float64{3.12, 2.00, 1.06}},
		{12.92, 12.92, formants{273, 1504, 2470}, [3]float64{1.57, 0.48, 0.19}},
		{11.50, 11.50, formants{273, 1504, 2470}, [3]float64{1.57, 0.48, 0.19}},
	},
	178: { // GE
		{1.00, 1.00, formants{254, 1306, 2523}, [3]float64{3.01, 0.37, 0.49}},
		{10.53, 10.53, formants{265, 1063, 2551}, [3]float64{6.37, 0.29, 0.69}},
		{11.46, 11.46, formants{265, 1307, 2508}, [3]float64{6.36, 0.35, 0.61}},
		{11.41, 11.41, formants{267, 1960, 2582}, [3]float64{5.72, 0.35, 0.91}},
		{11.50, 11.50, formants{267, 1960, 2582}, [3]float64{5.72, 0.35, 0.91}},
	},
	179: { // GO
		{1.00, 1.00, formants{250, 2012, 2386}, [3]float64{5.66, 0.19, 0.54}},
		{11.45, 11.45, formants{258, 2012, 2480}, [3]float64{6.87, 1.03, 0.53}},
		{11.43, 11.43, formants{264, 1436, 2570}, [3]float64{6.73, 0.55, 0.82}},
		{11.47, 11.47, formants{282, 1718, 2551}, [3]float64{2.94, 0.62, 0.37}},
		{11.50, 11.50, formants{282, 1718, 2551}, [3]float64{2.94, 0.62, 0.37}},
	},
	180: { // EG
		{1.00, 1.00, formants{264, 2117, 2480}, [3]float64{8.05, 5.10, 1.17}},
		{11.45, 11.45, formants{258, 2124, 2467}, [3]float64{5.51, 3.38, 0.63}},
		{11.50, 11.50, formants{258, 2124, 2467}, [3]float64{5.51, 3.38, 0.63}},
	},
	181: { // OG
		{1.00, 1.00, formants{279, 1314, 2503}, [3]float64{1.06, 0.51, 0.38}},
		{10.62, 10.62, formants{279, 1314, 2423}, [3]float64{7.49, 6.23, 0.95}},
		{11.44, 11.44, formants{262, 1319, 2480}, [3]float64{5.45, 3.96, 0.82}},
		{11.50, 11.50, formants{262, 1319, 2480}, [3]float64{5.45, 3.96, 0.82}},
	},
}

// framed plays an allophone from the chip's frames, at their measured length
func framed(name string, p []vfPoint) sound {
	s := sound{name: name, speech: true, length: 1}
	for _, q := range p {
		s.parts = append(s.parts, part{ms: q.ms, glideMs: q.glide, target: voicedFrame(q.freq, q.vol[0], q.vol[1], q.vol[2])})
	}
	return s
}

// VF-END
func init() {
	for i, l := range allophoneLength {
		if p := vfPoints[128+i]; p != nil {
			allophones[i] = framed(allophones[i].name, p)
			for j := range allophones[i].parts {
				allophones[i].parts[j].target.onsetMs = onsetFor(128 + i)
			}
			continue
		}
		if l > 0 {
			allophones[i].length = l
		} else {
			// the manual's length and transitionMs more
			ms := allophones[i].ms()
			allophones[i].length = (ms + transitionMs) / ms
		}
		g := math.Pow(10, allophoneGainDB[i]/20)
		parts := make([]part, len(allophones[i].parts))
		copy(parts, allophones[i].parts)
		for j := range parts {
			for k := range 5 {
				parts[j].target.vol[k] *= g
			}
			// the vowels, nasals, liquids and diphthongs (see onsetMs)
			parts[j].target.onsetMs = onsetFor(128 + i)
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
// speaker, from published measurements of English vowels and consonants, except
// where two independent measurements of the real chip agree.
//
// measured on the chip: where two independent measurements of a sound's formant
// agree, it is set to their mean. they agree when they are within 10% of each
// other or within 88Hz (the spacing of the voice's harmonics at the default
// pitch, about as finely as a formant can be read from a recording). the
// measurements are the openFPGA-7800 AtariVox engine's, read from recordings of
// games speaking (docs/ATARIVOX.md, MIT licence), and those behind Stella's
// SpeakJet engine (feature/speakjet branch). formants where the two disagree,
// or that only one of them measured, keep their published values:
//
// OW: F1 F2 AX: F1 F2 F3 RR: F1 F2 F3 UX: F1 F2 IH: F1 F2 F3 MM: F1 F3 EY: F1
// F2 F3 EH: F1 F2 AW: F2
//
// the third take of the VoxRecord recording (see msa_speed_factor()), whose
// vowels barely clip, confirms AX, UX, EH, EY, IH, OW, UH, LO and RR to
// within about 5% (formants read by linear prediction, the same way on the
// recording and on this library's own output, so that the method's errors
// cancel). it corrects five: AW F1 (590 to 695Hz), UW F1 and F2 (300 to 380,
// 870 to 1065), AY F1 and F2 (660 to 710, 1720 to 1660), OH F1 and F2 (730 to
// 690, 1090 to 1150) and WW F1 and F2 (300 to 495, 650 to 765). IY can't be
// read this way. the nasals (MM among them) are fitted to its spectra instead
// (see allophone()).
//
// the second VoxProbe recording then measured the oscillators directly: at
// pitch 40 each voicing pulse stands alone, a few milliseconds of the three
// formant sines starting together, so their frequencies can be fitted to the
// waveform itself. IY, IH, EY, EH, AY (but its weak F3), AX, UX, OH, AW, OW,
// UH, UW, LE, WW, RR and MM take those frequencies (NE and NO were worse with
// them on both recordings, and keep theirs). the biggest changes: IY F3 3010 to
// 2539Hz, LE F2 1300 to 970, WW F1 495 to 385, MM F1 396 to 257. with them the
// sustained sounds of the first VoxProbe recording (not used to measure them)
// go from 4.5 to 3.7dB of the chip's spectra.
//
// the third VoxProbe recording measures them again, on two units, with the
// chip's own fade and output filter (see voiceFade): IY, IH, EY, EH, AY, AX,
// UX, OH, AW, OW, UH, UW, LE, WW, RR, MM, NE and NO are those, the mean of both
// units (unit B's clock is 0.29% slow), which agree within a few hertz. F1 and
// F2 of all 18 are whole multiples of about 88.4Hz less 3.5Hz, within 2.2Hz RMS
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
// vowels barely clip, confirms AX, UX, EH, EY, IH, OW, UH, LO and RR to
// within about 5% (formants read by linear prediction, the same way on the
// recording and on this library's own output, so that the method's errors
// cancel). it corrects five: AW F1 (590 to 695Hz), UW F1 and F2 (300 to 380,
// 870 to 1065), AY F1 and F2 (660 to 710, 1720 to 1660), OH F1 and F2 (730 to
// 690, 1090 to 1150) and WW F1 and F2 (300 to 495, 650 to 765). IY can't be
// read this way. the nasals (MM among them) are fitted to its spectra instead
// (see the allophones).
//
// the second VoxProbe recording then measured the oscillators directly: at
// pitch 40 each voicing pulse stands alone, a few milliseconds of the three
// formant sines starting together, so their frequencies can be fitted to the
// waveform itself. IY, IH, EY, EH, AY (but its weak F3), AX, UX, OH, AW, OW,
// UH, UW, LE, WW, RR and MM take those frequencies (NE and NO were worse with
// them on both recordings, and keep theirs). the biggest changes: IY F3 3010 to
// 2539Hz, LE F2 1300 to 970, WW F1 495 to 385, MM F1 396 to 257. with them the
// sustained sounds of the first VoxProbe recording (not used to measure them)
// go from 4.5 to 3.7dB of the chip's spectra

var (
	fIY = formants{256, 2208, 2551}
	fIH = formants{350, 1762, 2479}
	fEY = formants{531, 1764, 2383}
	fEH = formants{617, 1676, 2384}
	fAY = formants{703, 1679, 2216}
	fAX = formants{530, 1503, 2383}
	fUX = formants{617, 1324, 2379}
	fOH = formants{704, 1147, 2379}
	fAW = formants{706, 970, 2464}
	fOW = formants{526, 881, 2353}
	fUH = formants{435, 1055, 2275}
	fUW = formants{349, 1055, 2094}
	fAH = formants{700, 1220, 2600}
	fER = formants{480, 1350, 1650}
	fRR = formants{442, 1325, 1680}
	fLE = formants{438, 968, 2635}
	fLO = formants{400, 900, 2500}
	fWW = formants{438, 790, 2058}

	// the start of AWRR, OWRR and OWIY keeps the earlier OH and AW: the third
	// take's corrections to OH and AW (see above) put those three 1 to 3dB
	// further from the chip's spectra, where the diphthongs that start from OH
	// (OHIY, OHIH) came closer
	fOHr = formants{730, 1090, 2440}
	fAWr = formants{590, 998, 2540}
	fYY  = formants{280, 2200, 2900}

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
		wave:   envVoice,
		voiced: true,
	}
}

// the chip doesn't restart all three formant sines at the same phase: fitted to
// the single pulses of the third VoxProbe recording with the chip's fade and
// output filter (see voiceFade), each sound's second and third formants start
// at their own phase against the first, the same on both units (IH's 0.13 and
// 0.17 of a cycle ahead, MM's 0.25 and 0.27, IY's 0.92 and 0.93). they shape
// the dips between the formants: restarting all three together, as first
// assumed here, leaves those dips up to 10dB too deep (IY, MM, NE) and the
// single pulses 4.5dB from the chip's rather than 2.0. formantPhases gives a
// voiced frame those phases. but measured at pitch 40, they don't all hold at
// speaking pitch: against the VoxRecord recording at the default pitch
// (background removed, unit B and unit A's unclipped sounds) and the first
// VoxProbe recording at pitches 88 and 60 (which nothing here was fitted to),
// IY, EY, EH, AY, AX, UX, OH, AW and MM match better without theirs (IY 85 to
// 91% and 3.2 to 2.0dB, and 3.4 to 2.2dB on VoxProbe), so those restart
// together; OW, UH, UW, NGE, NGO, NE, LE, WW and RR keep theirs
func formantPhases(f frame, p2, p3 float64) frame {
	f.phase0[1] = p2
	f.phase0[2] = p3
	return f
}

func vowelFrame(f formants) frame  { return voicedFrame(f, 24, 18, 10) }
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
		wave:       envVoice,
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

// the chip starts SE, SO and SH with a short low thump before the hiss, about
// 10ms of a tone near 400 to 600Hz (in both unclipped takes of the VoxRecord
// recording). onsetMs of the hiss frame with that tone added, then the hiss
// alone
func thumped(name string, onsetMs, hz, vol, ms float64, f frame) sound {
	o := f
	o.freq[0] = hz
	o.vol[0] = vol
	return sound{name: name, speech: true, length: steadyLength, parts: []part{
		{ms: onsetMs, glideMs: autoGlide, target: o},
		{ms: ms, glideMs: 3, target: f},
	}}
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

	// the noise widths (distortion) of the burst and the aspiration, the
	// aspiration's volumes, and for voiced stops the voice bar's volume during
	// the closure and a scale on the voicing after the release. zero means the
	// default (see withDefaults())
	burstDist, aspDist       float64
	aspVol                   [2]float64
	closureVoice, onsetVoice float64
}

// withDefaults fills in the fields of a stop that weren't given
func (st stop) withDefaults() stop {
	if st.burstDist == 0 {
		st.burstDist = 255
	}
	if st.aspDist == 0 {
		st.aspDist = 230
	}
	if st.aspVol == [2]float64{} {
		st.aspVol = [2]float64{10, 8}
	}
	if st.closureVoice == 0 {
		st.closureVoice = 8
	}
	if st.onsetVoice == 0 {
		st.onsetVoice = 1
	}
	return st
}

func plosive(name string, st stop) sound {
	st = st.withDefaults()

	var close frame
	if st.voiced {
		close = voicedFrame(formants{180, st.locus[1], st.locus[2]}, st.closureVoice, 0, 0)
	} else {
		close = silentFrame(st.locus)
	}

	burst := hush(noiseFrame(st.locus, st.burstFreq[0], st.burstFreq[1], st.burstVol[0], st.burstVol[1], st.burstDist), burstDB)
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
			v := st.onsetVoice
			parts = append(parts, part{ms: rest, glideMs: rest, target: voicedFrame(st.locus, 14*v, 6*v, 3*v)})
		}
	} else if rest > 0 && st.aspFreq[0]+st.aspFreq[1] == 0 {
		// no aspiration: silence. noise oscillators at 0Hz would hold whatever
		// phase the burst left them at, a step the dc blocker turns into a thump
		parts = append(parts, part{ms: rest, glideMs: 4, target: silentFrame(st.locus)})
	} else if rest > 0 {
		asp := hush(noiseFrame(st.locus, st.aspFreq[0], st.aspFreq[1], st.aspVol[0], st.aspVol[1], st.aspDist), aspirationDB)
		parts = append(parts, part{ms: rest, glideMs: 4, target: asp})
	}

	return sound{name: name, speech: true, length: stopLength, parts: parts}
}

// on both units (the VoxRecord recording) TS is hiss that starts at its
// loudest, peaked at 3.5kHz, and dies away: 6dB down 40ms on, 14dB by 120ms,
// spreading out to a flat spectrum with a little left at 3.5kHz. the fit as a
// stop had it the other way round, a long broad burst into a short hiss at the
// end. the hiss here is that fit's, then the same oscillators spread wide
// (distortion) and moved up, gliding down to 20dB under it. against both units
// this matches 37% (13% before) where they match each other 35%, its spectrum
// 3.7dB off (3.5 before, 3.5 between the two). the parts are 50, 10, 40 and
// 90ms (with the glides) at the manual's 170ms: see allophoneLength
func ts() sound {
	hiss := noiseFrame(lAlveolar, 3282.5, 3250.4, 8.04, 19.69, 48.6)
	return sound{name: "TS", speech: true, length: stopLength, parts: []part{
		{ms: 44.7368, glideMs: 5.3684, target: silentFrame(lAlveolar)},
		{ms: 8.9474, glideMs: 1.7895, target: hiss},
		{ms: 35.7895, glideMs: 35.7895, target: hush(hiss, 6)},
		{ms: 80.5263, glideMs: 80.5263, target: hush(noiseFrame(lAlveolar, 3900, 3800, 11.95, 25.78, 100), 20)},
	}}
}

// an affricate is a stop released into a fricative
func affricate(name string, st stop, fric frame, fricMs float64) sound {
	s := plosive(name, st)
	s.parts = append(s.parts, part{ms: fricMs, glideMs: 5, target: fric})
	return s
}

// frames for the fricatives SE, SO and SH. the noise of the voiceless
// fricatives (here and HE, HO, WH, FF and TH below) is shaped to the VoxRecord
// recording (see msa.speedFactor()). HE, HO and WH have their spectra on the
// chip, measured in third octaves from 250Hz to 4kHz in three takes, fitted
// through the output filter here. SE, SO, SH, FF and TH, like the stops and
// affricates (see the stops below), were fitted further, as whole sounds: each
// was played here from a set of values, which a search (differential evolution)
// moved to bring its spectrogram closest to the chip's in the first and third
// takes together, and its length to the chip's. two takes rather than one,
// because on the hiss sounds the chip's own takes match each other only 30 to
// 70%, so fitting one fits that take's noise. the chip's hiss is narrow bands,
// peaked where these were broad: SE and SO at 3 to 4kHz, SH at 2.5 to  3.2kHz,
// HE and HO around 1 to 1.5kHz, WH at 0.6 to 0.9kHz. their levels are set by
// allophoneGainDB. all of them were then fitted again to both units at once
// (see stop)
var (
	frSE = noiseFrame(lAlveolar, 3629.7, 2609.1, 25.87, 5.78, 33.4)
	frSO = noiseFrame(lAlveolarBak, 2953.5, 3580, 17.5, 30.41, 27.3)
	frSH = noiseFrame(lPalatal, 3605.7, 2502.8, 17.72, 22.59, 44.4)
	frZH = hush(voicedNoiseFrame(formants{260, 1800, 2500}, 14, 2400, 3200, 14, 10, 200), fricativeDB)
)

// a stop's noise widths, aspiration volumes and voicing default to the first
// ones chosen here (see stop.withDefaults()). the stops that give them (BE, EB,
// OB, ED, OD, TT, TU, KE, EK and OK) and the affricates (CH, JH and TS) were
// fitted as whole sounds to the VoxRecord recording (see frSE). the chip's
// bursts and aspiration are narrower noise than first chosen here (TT and TU
// hiss at 2.6 to 3.9kHz, which the first values spread from 1 to 2kHz), its
// voiced stops' loci differ from the published ones, and EB, OB, ED and OD last
// 25 to 35ms on the chip, not 10. a closure that is silent can't be measured
// from a sound played alone, so those keep the length they had (35 to 42ms, 7ms
// for ED and OD), and TT's comes from the connected groups (see
// allophoneLength). the other stops' burst and aspiration frequencies were
// scaled together (by 0.5 to 1.15) to bring each spectrum closest to the
// recording's third take; most of the chip's bursts are lower than first chosen
// here (PE and PO by half).
//
// all of them but BE, BO, DO, ED, OD, GE, GO, EG and OG (which are played from
// the chip's frames: see vfPoints) and TT were then fitted again, with the
// output filter measured from the voicing pulses (see chipRate), to both
// units' VoxRecord recordings at once: the spectrogram, the spectrum and the
// level over time, each unit's level compared down to its own background, so
// that a part the chip doesn't play costs as much as one it plays wrongly (the
// first fit had put faint tails and tones at the background's level, where they
// cost nothing). each keeps the length it had (see allophoneLength), which
// comes from the connected groups; fitted to the sounds alone the lengths came
// out shorter, and connected speech went from 21 to 60ms out. TT keeps its
// values, which the connected groups fix: its refit made groups ending in TT
// 100ms short. with these KE went from 54 to 74% spectrogram match and 11.3 to
// 4.2dB, KO 55 to 81%, EK 71 to 88%, OK 68 to 86%, DE 44 to 86%, JH 49 to 80%
// and CH 65 to 75%
// allophones, codes 128 to 199. durations are from the manual
var allophones = [...]sound{
	// the vowels' and liquids' three volumes are the chip's: at pitch 40 on the
	// third VoxProbe recording each voicing pulse is the three formant sines
	// alone, and fitting them through the chip's fade and output filter (see
	// voiceFade) gives each one's amplitude, the mean of both units. some of
	// these sounds are instead played from the chip's own frames, pulse by pulse
	// (see vfPoints)
	steady("IY", 70, voicedFrame(fIY, 24.0, 15.6, 9.8)),
	steady("IH", 70, formantPhases(voicedFrame(fIH, 24.0, 16.7, 10.0), 0.132, 0.175)),
	steady("EY", 70, voicedFrame(fEY, 24.0, 14.0, 7.7)),
	steady("EH", 70, voicedFrame(fEH, 24.0, 13.4, 9.3)),
	steady("AY", 70, voicedFrame(fAY, 24.0, 11.3, 6.4)),
	steady("AX", 70, voicedFrame(fAX, 24.0, 14.8, 8.8)),
	steady("UX", 70, voicedFrame(fUX, 24.0, 15.4, 9.6)),
	steady("OH", 70, voicedFrame(fOH, 24.0, 17.9, 9.1)),
	steady("AW", 70, voicedFrame(fAW, 24.0, 22.3, 6.6)),
	steady("OW", 70, formantPhases(voicedFrame(fOW, 20.1, 24.0, 4.9), 0.954, 0.982)),
	steady("UH", 70, formantPhases(voicedFrame(fUH, 22.5, 24.0, 6.0), 0.924, 0.928)),
	steady("UW", 70, formantPhases(voicedFrame(fUW, 24.0, 22.5, 3.9), 0.955, 0.032)),
	// the nasals' resonances and volumes are fitted to the VoxRecord
	// recording's third take (the middle of each, in sixth octaves from 200Hz
	// to 3.9kHz): besides the low murmur the chip's nasals have strong
	// resonances near 1.4 to 1.6kHz and 2.2 to 2.6kHz that the published values
	// here lacked
	steady("MM", 70, voicedFrame(formants{266, 1410, 2252}, 24.0, 6.3, 1.9)),
	steady("NE", 70, formantPhases(voicedFrame(formants{262, 1572, 2582}, 24.0, 3.6, 7.6), 0.032, 0.965)),
	steady("NO", 70, formantPhases(voicedFrame(formants{269, 1169, 2590}, 24.0, 4.6, 7.3), 0.962, 0.031)),
	// NGE and NGO had no measurement (VoxProbe 3 doesn't play them), but on the
	// VoxRecord recording both units agree on every harmonic of them to within
	// 0.5dB RMS. their frequencies, volumes and phases are fitted to those
	// harmonics, played here: NGE 87.0% to 97.0% and 3.93 to 1.11dB, NGO 93.1%
	// to 96.2% and 2.98 to 1.83dB. on the chip NGE's second and third formants
	// are one peak near 2.3kHz, where they had been 2.1 and 2.7kHz
	steady("NGE", 70, formantPhases(voicedFrame(formants{368.334, 2266.3, 2386.423}, 22.0, 6.845, 2.093), 0.225, 0.909)),
	steady("NGO", 70, formantPhases(voicedFrame(formants{374.718, 1491.004, 1938.775}, 22.0, 4.018, 9.051), 0.868, 0.994)),
	steady("LE", 70, formantPhases(voicedFrame(fLE, 23.3, 24.0, 4.3), 0.945, 0.930)),
	steady("LO", 70, liquidFrame(fLO)),
	steady("WW", 70, formantPhases(voicedFrame(fWW, 17.7, 24.0, 1.8), 0.952, 0.056)),
	steady("RR", 70, formantPhases(voicedFrame(fRR, 24.0, 19.5, 4.5), 0.993, 0.005)),
	// the diphthongs' vowel ends take their vowel's volumes (above)
	diphthong("IYRR", 200, voicedFrame(fIY, 24.0, 15.6, 9.8), liquidFrame(fER)),
	diphthong("EYRR", 200, voicedFrame(fEH, 24.0, 13.4, 9.3), liquidFrame(fER)),
	diphthong("AXRR", 190, vowelFrame(formants{500, 1400, 1750}), liquidFrame(fER)),
	diphthong("AWRR", 200, vowelFrame(fOHr), liquidFrame(fER)),
	diphthong("OWRR", 185, vowelFrame(fAWr), liquidFrame(fER)),
	diphthong("EYIY", 165, voicedFrame(fEY, 24.0, 14.0, 7.7), voicedFrame(fIY, 24.0, 15.6, 9.8)),
	diphthong("OHIY", 200, voicedFrame(fOH, 24.0, 17.9, 9.1), voicedFrame(fIY, 24.0, 15.6, 9.8)),
	diphthong("OWIY", 225, vowelFrame(fAWr), voicedFrame(fIY, 24.0, 15.6, 9.8)),
	diphthong("OHIH", 185, voicedFrame(fOH, 24.0, 17.9, 9.1), voicedFrame(fIH, 24.0, 16.7, 10.0)),
	diphthong("IYEH", 170, vowelFrame(fYY), voicedFrame(fEH, 24.0, 13.4, 9.3)),
	diphthong("EHLL", 140, voicedFrame(fAX, 24.0, 14.8, 8.8), liquidFrame(fLO)),
	diphthong("IYUW", 180, vowelFrame(fYY), voicedFrame(fUW, 24.0, 22.5, 3.9)),
	diphthong("AXUW", 170, vowelFrame(fAH), voicedFrame(fUW, 24.0, 22.5, 3.9)),
	diphthong("IHWW", 170, vowelFrame(formants{330, 1300, 2300}), voicedFrame(fUW, 24.0, 22.5, 3.9)),
	diphthong("AYWW", 200, voicedFrame(fAY, 24.0, 11.3, 6.4), voicedFrame(fUH, 22.5, 24.0, 6.0)),
	diphthong("OWWW", 131, voicedFrame(fOW, 20.1, 24.0, 4.9), vowelFrame(formants{330, 700, 2250})),
	affricate("JH", stop{ms: 23.6008, closure: 14.6703, burst: 8.8361, voiced: true, locus: lPalatal, burstFreq: [2]float64{4086.8, 1843.5}, burstVol: [2]float64{20.47, 25.65}},
		voicedNoiseFrame(formants{158, 1974, 1842}, 21.23, 1984.3, 3081.6, 26.87, 18.82, 47.7), 46.3992),
	steady("VV", 70, voicedNoiseFrame(formants{492, 2083, 3156}, 3.6, 2253, 1368, 1.4, 5.62, 78)),
	steady("ZZ", 70, hush(voicedNoiseFrame(formants{240, 1500, 2500}, 14, 3700, 3950, 14, 10, 160), fricativeDB)),
	steady("ZH", 70, frZH),
	steady("DH", 70, hush(voicedNoiseFrame(formants{250, 1300, 2400}, 16, 3000, 3800, 7, 5, 255), fricativeDB)),
	plosive("BE", stop{ms: 45, closure: 31.6, burst: 10.8, voiced: true, locus: formants{363, 1741, 2562},
		burstFreq: [2]float64{2149, 3662}, burstVol: [2]float64{22, 16.4},
		burstDist: 124, closureVoice: 10, onsetVoice: 0.84}),
	plosive("BO", stop{ms: 45, closure: 25, burst: 8, voiced: true, locus: lLabialBack,
		burstFreq: [2]float64{600, 1000}, burstVol: [2]float64{10, 6}}),
	plosive("EB", stop{ms: 10, closure: 2.85, burst: 7.15, voiced: true, locus: formants{345, 918, 2893}, burstFreq: [2]float64{1733.4, 1870.5}, burstVol: [2]float64{15.96, 21.89}, burstDist: 99.2, closureVoice: 25.63, onsetVoice: 0.444}),
	// OB and PO keep the values they had before the refit through the new
	// filter, which matched both units worse once the recordings' background was
	// taken out (OB 66 to 75%, PO 48 to 64% and 4.9 to 3.0dB on unit B)
	plosive("OB", stop{ms: 10, closure: 1, burst: 0.2, voiced: true, locus: formants{416, 1414, 2909},
		burstFreq: [2]float64{835, 3504}, burstVol: [2]float64{29.1, 22.8},
		burstDist: 201, closureVoice: 2.2, onsetVoice: 0.25}),
	plosive("DE", stop{ms: 45, closure: 21.26, burst: 23.74, voiced: true, locus: lAlveolar, burstFreq: [2]float64{2535.3, 1885.7}, burstVol: [2]float64{29.96, 28.29}, burstDist: 57.5, closureVoice: 12.27, onsetVoice: 0.941}),
	plosive("DO", stop{ms: 45, closure: 25, burst: 8, voiced: true, locus: lAlveolarBak,
		burstFreq: [2]float64{1800, 2220}, burstVol: [2]float64{12, 8}}),
	plosive("ED", stop{ms: 10, closure: 1.87, burst: 0.77, voiced: true, locus: formants{281, 2404, 2711},
		burstFreq: [2]float64{3411, 3749}, burstVol: [2]float64{16.6, 20.2},
		burstDist: 189, closureVoice: 0.3, onsetVoice: 0.44}),
	plosive("OD", stop{ms: 10, closure: 2.21, burst: 0.46, voiced: true, locus: formants{278, 1462, 2581},
		burstFreq: [2]float64{2076, 3335}, burstVol: [2]float64{15.2, 24.3},
		burstDist: 208, closureVoice: 0.6, onsetVoice: 0.93}),
	plosive("GE", stop{ms: 55, closure: 30, burst: 10, voiced: true, locus: lVelar,
		burstFreq: [2]float64{1610, 2030}, burstVol: [2]float64{12, 8}}),
	plosive("GO", stop{ms: 55, closure: 30, burst: 10, voiced: true, locus: lVelarBack,
		burstFreq: [2]float64{650, 900}, burstVol: [2]float64{12, 8}}),
	plosive("EG", stop{ms: 55, closure: 35, burst: 10, voiced: true, locus: lVelar,
		burstFreq: [2]float64{2300, 2900}, burstVol: [2]float64{10, 6}}),
	plosive("OG", stop{ms: 55, closure: 35, burst: 10, voiced: true, locus: lVelarBack,
		burstFreq: [2]float64{910, 1260}, burstVol: [2]float64{10, 6}}),
	affricate("CH", stop{ms: 49.4544, closure: 23.4763, burst: 3.4515, locus: lPalatal, burstFreq: [2]float64{2623.8, 2225.2}, burstVol: [2]float64{1.57, 2.79}},
		noiseFrame(lPalatal, 2260.1, 3154.6, 10.14, 24.25, 42.6), 20.5456),
	steady("HE", 70, noiseFrame(formants{400, 1900, 2600}, 1546.1, 1464.8, 27.07, 7.24, 192.3)),
	steady("HO", 70, noiseFrame(formants{600, 1000, 2400}, 1368.4, 1277.8, 23.85, 10.37, 165.2)),
	steady("WH", 70, noiseFrame(fWW, 821.4, 731.8, 16.48, 19.02, 209.7)),
	steady("FF", 70, noiseFrame(lLabial, 2823.1, 3165.5, 15.64, 16.67, 107.3)),
	thumped("SE", 4.36, 414.1, 15.84, 35.64, frSE),
	thumped("SO", 4.53, 436.9, 15.26, 35.47, frSO),
	thumped("SH", 9.13, 363.2, 11.07, 40.87, frSH),
	steady("TH", 40, noiseFrame(lDental, 2170.3, 2087.4, 26.95, 26.05, 143.4)),
	plosive("TT", stop{ms: 50, closure: 31.8, burst: 7.11, locus: lAlveolar,
		burstFreq: [2]float64{2689, 3257}, burstVol: [2]float64{15.9, 22.1}, aspFreq: [2]float64{3188, 3038},
		burstDist: 73, aspDist: 19, aspVol: [2]float64{3.7, 17.4}}),
	plosive("TU", stop{ms: 70, closure: 43.56, burst: 26.34, locus: lAlveolar, burstFreq: [2]float64{3361.7, 2605.1}, burstVol: [2]float64{24.92, 19.05}, aspFreq: [2]float64{2420.8, 2928}, burstDist: 45.6, aspDist: 56.5, aspVol: [2]float64{7.51, 5.36}}),
	ts(),
	plosive("KE", stop{ms: 55, closure: 15.33, burst: 6.09, locus: lVelar, burstFreq: [2]float64{1641, 2125.3}, burstVol: [2]float64{21.74, 25.19}, aspFreq: [2]float64{3713.1, 4001.3}, burstDist: 62.9, aspDist: 159, aspVol: [2]float64{0.39, 0.78}}),
	plosive("KO", stop{ms: 55, closure: 43.36, burst: 4.21, locus: lVelarBack, burstFreq: [2]float64{1064, 1788.5}, burstVol: [2]float64{18.12, 16.08}, aspFreq: [2]float64{1591.2, 1262}, burstDist: 185.3, aspDist: 79.5, aspVol: [2]float64{5.49, 7.86}}),
	plosive("EK", stop{ms: 55, closure: 43.95, burst: 8.79, locus: lVelar, burstFreq: [2]float64{2933.4, 3338}, burstVol: [2]float64{0.86, 0.06}, aspFreq: [2]float64{3538.4, 1860.3}, burstDist: 56.9, aspDist: 108.4, aspVol: [2]float64{4.54, 25.1}}),
	plosive("OK", stop{ms: 45, closure: 11.86, burst: 31.21, locus: lVelarBack, burstFreq: [2]float64{3171.3, 812.5}, burstVol: [2]float64{1.72, 0.23}, aspFreq: [2]float64{1975, 1050.1}, burstDist: 58.1, aspDist: 112.5, aspVol: [2]float64{26.71, 7.48}}),
	plosive("PE", stop{ms: 99, closure: 59.57, burst: 31.25, locus: lLabial, burstFreq: [2]float64{410.4, 848.4}, burstVol: [2]float64{19.62, 14.17}, aspFreq: [2]float64{1315.3, 2497.7}, burstDist: 227.1, aspDist: 88.6, aspVol: [2]float64{9.82, 30.48}}),
	plosive("PO", stop{ms: 99, closure: 50, burst: 9, locus: lLabialBack,
		burstFreq: [2]float64{300, 720}, burstVol: [2]float64{12, 9}, aspFreq: [2]float64{600, 1380}}),
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
// away by 30dB, fastest at the start and the end. on both units of the
// VoxRecord recording the hiss holds within 3dB of the bang for 80ms, is 8dB
// down by 130ms, 13 by 200ms, 17 by 250ms and 27 by 290ms, and stops at 300ms:
// these fades, where it had been faded sooner and stopped at 250ms
func m1() sound {
	bang := noiseFrame(lGlottal, 500, 1400, 31, 26, 255)
	hiss := noiseFrame(lGlottal, 1000, 1700, 31, 26, 160)
	return effect("M1",
		hold(30, 2, bang),
		hold(20, 20, fade(hiss, 1.5)),
		hold(30, 30, fade(hiss, 2)),
		hold(50, 50, fade(hiss, 7)),
		hold(70, 70, fade(hiss, 12)),
		hold(50, 50, fade(hiss, 16)),
		hold(40, 40, fade(hiss, 26)),
		hold(10, 10, quiet(hiss)))
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
// msa_speed_factor()), which plays each on its own and clips little: every 5ms
// the level and the sound's kind were measured, and the trace reduced to the
// points below, between which the sound glides in a straight line. a tone is up
// to three pure sines (the chip's tones have no harmonics to speak of, 45dB and
// more down): the strongest peaks, within 15dB of the loudest, that aren't its
// harmonics (R0 to R9, for example, are three tones each). noise (a flat
// spectrum) is noise around the frequency. a buzz (a pitch between 40 and 500Hz
// found by autocorrelation, with six or more peaks) is voiced at that pitch
// with oscillators 1 to 3 on its three formants, read by linear prediction: A1,
// A2, A3 and M2 buzz at 87Hz, the default speech pitch, and M2 ("WOW") is a
// vowel glide. each point is where it is reached (in ms from the start of the
// sound), its kind, the buzz's pitch, its level in dB (-60 is silence), the
// three frequencies (0 for none) and the second and third's levels against the
// first (-60 while a tone that comes and goes is absent, so that it fades
// rather than sweeping from 0Hz). the levels are the recording's with the
// output filter's loss at each frequency added back (see chipRate), as the
// filter here takes it off again. many effects are nothing like their names
// suggest, and many differ in length from the manual: A1 and A2 last about a
// second, and M0 is a 1.2 second tone around 1.8kHz that dies away. the points
// at the start and end of an effect that were no louder than the recording's
// background (within 6dB of the first point) are silent: the trace had taken
// that background for part of the effect, making each one 20 to 50ms too long.
// the tone effects are traced again in fx2Points
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
		{5, fxTone, 0, -60.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{25, fxTone, 0, -9.1, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{30, fxTone, 0, -2.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{35, fxTone, 0, 1.4, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{40, fxTone, 0, 3.3, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{45, fxTone, 0, 4.6, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{70, fxTone, 0, 5.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{95, fxTone, 0, 2.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{100, fxTone, 0, -0.8, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{105, fxTone, 0, -6.5, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{110, fxTone, 0, -12.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{115, fxTone, 0, -60.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{120, fxTone, 0, -60.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{125, fxTone, 0, -60.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
		{135, fxTone, 0, -60.0, [3]int{799, 511, 0}, [3]float64{0, -15.1, -60.0}},
	},
	201: { // R1
		{5, fxTone, 0, -60.0, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{25, fxTone, 0, -14.8, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{30, fxTone, 0, -5.2, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{35, fxTone, 0, -1.6, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{80, fxTone, 0, 1.9, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{100, fxTone, 0, -4.4, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{105, fxTone, 0, -10.8, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{110, fxTone, 0, -20.4, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{115, fxTone, 0, -60.0, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
		{130, fxTone, 0, -60.0, [3]int{1100, 731, 0}, [3]float64{0, -19.3, -60.0}},
	},
	202: { // R2
		{5, fxTone, 0, -60.0, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{25, fxTone, 0, -13.7, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{30, fxTone, 0, -5.5, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{35, fxTone, 0, -2.5, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{60, fxTone, 0, 0.1, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{65, fxTone, 0, -0.2, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{95, fxTone, 0, -2.9, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{100, fxTone, 0, -5.5, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{105, fxTone, 0, -11.7, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{110, fxTone, 0, -16.4, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{115, fxTone, 0, -60.0, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{120, fxTone, 0, -60.0, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{125, fxTone, 0, -60.0, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
		{135, fxTone, 0, -60.0, [3]int{1299, 531, 0}, [3]float64{0, -21.0, -60.0}},
	},
	203: { // R3
		{5, fxTone, 0, -60.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{10, fxTone, 0, -60.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{15, fxTone, 0, -60.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{20, fxTone, 0, -60.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{25, fxTone, 0, -17.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{30, fxTone, 0, -8.1, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{35, fxTone, 0, -4.4, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{40, fxTone, 0, -2.6, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{65, fxTone, 0, -1.5, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{70, fxTone, 0, -1.5, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{90, fxTone, 0, -3.2, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{95, fxTone, 0, -5.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{100, fxTone, 0, -9.3, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{105, fxTone, 0, -15.1, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{110, fxTone, 0, -21.9, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{115, fxTone, 0, -25.8, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{120, fxTone, 0, -60.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
		{130, fxTone, 0, -60.0, [3]int{1599, 726, 360}, [3]float64{0, -14.1, -15.2}},
	},
	204: { // R4
		{5, fxTone, 0, -60.0, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{20, fxTone, 0, -60.0, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{25, fxTone, 0, -17.0, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{30, fxTone, 0, -7.5, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{55, fxTone, 0, -1.4, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{60, fxTone, 0, -1.6, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{65, fxTone, 0, -1.5, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{90, fxTone, 0, -3.2, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{95, fxTone, 0, -7.3, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{100, fxTone, 0, -10.4, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{105, fxTone, 0, -15.2, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{110, fxTone, 0, -19.1, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{115, fxTone, 0, -20.6, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{120, fxTone, 0, -60.0, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
		{130, fxTone, 0, -60.0, [3]int{1898, 305, 561}, [3]float64{0, -18.5, -23.3}},
	},
	205: { // R5
		{5, fxTone, 0, -60.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{10, fxTone, 0, -60.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{15, fxTone, 0, -60.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{20, fxTone, 0, -60.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{25, fxTone, 0, -13.1, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{30, fxTone, 0, -7.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{45, fxTone, 0, -2.1, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{90, fxTone, 0, -3.5, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{95, fxTone, 0, -5.2, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{100, fxTone, 0, -12.5, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{105, fxTone, 0, -16.5, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{110, fxTone, 0, -22.1, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{115, fxTone, 0, -27.2, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{120, fxTone, 0, -60.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{125, fxTone, 0, -60.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
		{130, fxTone, 0, -60.0, [3]int{2198, 381, 2667}, [3]float64{0, -14.3, -18.3}},
	},
	206: { // R6
		{5, fxTone, 0, -60.0, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{10, fxTone, 0, -60.0, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{15, fxTone, 0, -60.0, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{20, fxTone, 0, -21.5, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{25, fxTone, 0, -16.4, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{30, fxTone, 0, -11.8, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{35, fxTone, 0, -9.7, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{40, fxTone, 0, -1.5, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{65, fxTone, 0, -0.8, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{70, fxTone, 0, -0.7, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{80, fxTone, 0, -0.8, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{85, fxTone, 0, -8.2, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{100, fxTone, 0, -14.1, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{105, fxTone, 0, -18.3, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{110, fxTone, 0, -20.2, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{115, fxTone, 0, -27.1, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{120, fxTone, 0, -60.0, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
		{125, fxTone, 0, -60.0, [3]int{2498, 314, 546}, [3]float64{0, -19.0, -22.2}},
	},
	207: { // R7
		{5, fxTone, 0, -60.0, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{10, fxTone, 0, -60.0, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{15, fxTone, 0, -60.0, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{20, fxTone, 0, -9.9, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{25, fxTone, 0, -5.0, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{30, fxTone, 0, -2.7, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{50, fxTone, 0, -0.5, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{55, fxTone, 0, -0.5, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{60, fxTone, 0, -0.5, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{70, fxTone, 0, -0.5, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{80, fxTone, 0, -0.9, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{85, fxTone, 0, -1.7, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{95, fxTone, 0, -5.5, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{100, fxTone, 0, -8.5, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{105, fxTone, 0, -13.0, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{110, fxTone, 0, -16.8, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{115, fxTone, 0, -17.5, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{120, fxTone, 0, -60.0, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
		{125, fxTone, 0, -60.0, [3]int{2797, 253, 1055}, [3]float64{0, -17.8, -24.6}},
	},
	208: { // R8
		{5, fxTone, 0, -60.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{15, fxTone, 0, -60.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{20, fxTone, 0, -22.7, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{25, fxTone, 0, -18.8, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{40, fxTone, 0, -12.6, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{45, fxTone, 0, -12.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{75, fxTone, 0, -11.8, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{80, fxTone, 0, -11.8, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{85, fxTone, 0, -12.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{90, fxTone, 0, -12.6, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{95, fxTone, 0, -13.8, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{100, fxTone, 0, -15.8, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{105, fxTone, 0, -18.8, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{110, fxTone, 0, -23.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{115, fxTone, 0, -26.9, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{120, fxTone, 0, -60.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{125, fxTone, 0, -60.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
		{130, fxTone, 0, -60.0, [3]int{643, 3097, 321}, [3]float64{0, 10.7, -16.6}},
	},
	209: { // R9
		{5, fxTone, 0, -60.0, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{15, fxTone, 0, -60.0, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{20, fxTone, 0, -24.8, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{25, fxTone, 0, -19.3, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{40, fxTone, 0, -11.9, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{45, fxTone, 0, -11.3, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{50, fxTone, 0, -11.5, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{85, fxTone, 0, -11.5, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{90, fxTone, 0, -12.1, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{95, fxTone, 0, -13.2, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{100, fxTone, 0, -14.9, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{110, fxTone, 0, -21.7, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{115, fxTone, 0, -25.6, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{120, fxTone, 0, -60.0, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{125, fxTone, 0, -60.0, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
		{130, fxTone, 0, -60.0, [3]int{640, 3397, 1592}, [3]float64{0, 10.1, -14.2}},
	},
	210: { // A0
		{5, fxTone, 0, -60.0, [3]int{239, 1009, 472}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 1009, 472}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -8.0, [3]int{239, 1009, 472}, [3]float64{0, -60.0, -60.0}},
		{30, fxTone, 0, -7.0, [3]int{175, 1009, 472}, [3]float64{0, -0.1, -60.0}},
		{35, fxTone, 0, -6.8, [3]int{175, 1627, 472}, [3]float64{0, -9.7, -60.0}},
		{40, fxTone, 0, -3.1, [3]int{175, 1644, 472}, [3]float64{0, -3.0, -60.0}},
		{45, fxTone, 0, -4.0, [3]int{233, 1726, 472}, [3]float64{0, -7.2, -60.0}},
		{55, fxTone, 0, -1.6, [3]int{233, 1779, 472}, [3]float64{0, -3.5, -60.0}},
		{60, fxTone, 0, -3.7, [3]int{163, 1779, 472}, [3]float64{0, -10.0, -60.0}},
		{75, fxTone, 0, -2.9, [3]int{163, 1808, 472}, [3]float64{0, -11.2, -60.0}},
		{80, fxTone, 0, -1.2, [3]int{175, 1837, 472}, [3]float64{0, 1.9, -60.0}},
		{85, fxTone, 0, -2.5, [3]int{181, 1895, 472}, [3]float64{0, -10.8, -60.0}},
		{95, fxTone, 0, -2.1, [3]int{181, 1948, 472}, [3]float64{0, -11.0, -60.0}},
		{110, fxTone, 0, -1.0, [3]int{198, 1977, 472}, [3]float64{0, -12.0, -60.0}},
		{115, fxTone, 0, -0.8, [3]int{204, 1977, 472}, [3]float64{0, -60.0, -60.0}},
		{165, fxTone, 0, -0.8, [3]int{245, 1977, 472}, [3]float64{0, -60.0, -60.0}},
		{170, fxTone, 0, 0.6, [3]int{251, 157, 472}, [3]float64{0, -60.0, -60.0}},
		{220, fxTone, 0, 0.8, [3]int{297, 157, 472}, [3]float64{0, -60.0, -60.0}},
		{225, fxTone, 0, 0.8, [3]int{297, 157, 472}, [3]float64{0, 0.0, -60.0}},
		{230, fxTone, 0, 1.0, [3]int{309, 449, 472}, [3]float64{0, -11.9, -60.0}},
		{235, fxTone, 0, 1.0, [3]int{315, 157, 472}, [3]float64{0, -9.2, -60.0}},
		{240, fxTone, 0, 0.9, [3]int{315, 163, 472}, [3]float64{0, -11.8, -0.0}},
		{245, fxTone, 0, 0.9, [3]int{321, 163, 478}, [3]float64{0, -10.1, -11.7}},
		{315, fxTone, 0, 1.2, [3]int{373, 192, 560}, [3]float64{0, -13.5, -14.4}},
		{320, fxTone, 0, 0.5, [3]int{373, 496, 187}, [3]float64{0, -12.5, -13.6}},
		{330, fxTone, 0, -1.5, [3]int{356, 496, 157}, [3]float64{0, -4.9, -10.6}},
		{335, fxTone, 0, -2.5, [3]int{356, 157, 157}, [3]float64{0, -10.0, -60.0}},
		{340, fxTone, 0, -4.0, [3]int{391, 157, 157}, [3]float64{0, -60.0, -60.0}},
		{345, fxTone, 0, -5.1, [3]int{391, 157, 157}, [3]float64{0, -60.0, -60.0}},
		{350, fxTone, 0, -7.7, [3]int{391, 192, 157}, [3]float64{0, -60.0, -60.0}},
		{355, fxTone, 0, -9.6, [3]int{373, 192, 157}, [3]float64{0, -60.0, -60.0}},
		{360, fxTone, 0, -10.8, [3]int{362, 192, 157}, [3]float64{0, 0.0, -60.0}},
		{365, fxTone, 0, -15.7, [3]int{362, 566, 157}, [3]float64{0, -10.2, -60.0}},
		{370, fxTone, 0, -17.3, [3]int{367, 589, 157}, [3]float64{0, -11.3, -60.0}},
		{375, fxTone, 0, -60.0, [3]int{367, 636, 157}, [3]float64{0, -12.5, -60.0}},
		{380, fxTone, 0, -60.0, [3]int{362, 991, 157}, [3]float64{0, -13.3, -60.0}},
		{385, fxTone, 0, -60.0, [3]int{245, 991, 157}, [3]float64{0, -13.7, -60.0}},
		{395, fxTone, 0, -60.0, [3]int{239, 991, 157}, [3]float64{0, -13.0, -60.0}},
	},
	211: { // A1
		{5, fxBuzz, 127, -60.0, [3]int{790, 2001, 2980}, [3]float64{0, 10.4, 17.1}},
		{10, fxBuzz, 125, -60.0, [3]int{790, 2002, 2981}, [3]float64{0, 10.4, 16.9}},
		{15, fxBuzz, 142, -60.0, [3]int{790, 2133, 3053}, [3]float64{0, -4.0, 2.1}},
		{35, fxBuzz, 88, 4.5, [3]int{790, 2133, 3053}, [3]float64{0, -4.0, 2.1}},
		{45, fxBuzz, 87, 6.7, [3]int{790, 2133, 3053}, [3]float64{0, -4.0, 2.1}},
		{50, fxBuzz, 87, 6.2, [3]int{763, 2130, 3053}, [3]float64{0, -20.0, -32.4}},
		{55, fxBuzz, 87, 6.6, [3]int{763, 2074, 3053}, [3]float64{0, -19.6, -31.9}},
		{60, fxBuzz, 87, 7.9, [3]int{763, 2074, 3101}, [3]float64{0, -37.9, -14.2}},
		{65, fxBuzz, 87, 7.5, [3]int{772, 1484, 3101}, [3]float64{0, -13.7, -12.6}},
		{85, fxBuzz, 88, 8.5, [3]int{825, 1520, 3071}, [3]float64{0, -8.7, -7.9}},
		{105, fxBuzz, 88, 9.6, [3]int{833, 1563, 3170}, [3]float64{0, -8.2, -5.1}},
		{110, fxBuzz, 87, 7.9, [3]int{833, 1867, 3346}, [3]float64{0, -9.5, -5.2}},
		{155, fxBuzz, 87, 6.8, [3]int{875, 1933, 3295}, [3]float64{0, -20.4, -22.0}},
		{160, fxBuzz, 87, 8.3, [3]int{880, 1933, 2545}, [3]float64{0, -19.4, -17.7}},
		{185, fxBuzz, 88, 8.2, [3]int{905, 1980, 2478}, [3]float64{0, -17.9, -16.4}},
		{190, fxBuzz, 87, 6.8, [3]int{908, 2009, 2478}, [3]float64{0, -17.8, -37.7}},
		{235, fxBuzz, 87, 6.2, [3]int{941, 2008, 2478}, [3]float64{0, -15.1, -32.8}},
		{245, fxBuzz, 87, 6.9, [3]int{975, 2331, 2478}, [3]float64{0, -13.2, -32.7}},
		{250, fxBuzz, 87, 7.6, [3]int{981, 2342, 3056}, [3]float64{0, -13.2, -12.2}},
		{265, fxBuzz, 87, 7.7, [3]int{1004, 2369, 3097}, [3]float64{0, -11.4, -9.2}},
		{270, fxBuzz, 87, 6.3, [3]int{1004, 2388, 3113}, [3]float64{0, -11.5, -9.5}},
		{275, fxBuzz, 87, 7.7, [3]int{1004, 2451, 3113}, [3]float64{0, -10.9, -26.0}},
		{325, fxBuzz, 87, 5.8, [3]int{979, 2564, 3113}, [3]float64{0, -10.3, -26.3}},
		{340, fxBuzz, 87, 6.5, [3]int{967, 3007, 3113}, [3]float64{0, -9.3, -29.4}},
		{345, fxBuzz, 87, 7.6, [3]int{982, 3042, 2232}, [3]float64{0, -9.8, -16.6}},
		{435, fxBuzz, 87, 7.8, [3]int{1173, 3549, 2555}, [3]float64{0, -11.6, -17.6}},
		{475, fxBuzz, 87, 5.4, [3]int{1194, 3515, 2589}, [3]float64{0, -14.0, -18.6}},
		{480, fxBuzz, 87, 6.8, [3]int{1192, 3066, 2589}, [3]float64{0, -13.0, -31.1}},
		{530, fxBuzz, 87, 6.2, [3]int{1142, 2997, 2589}, [3]float64{0, -13.0, -29.5}},
		{540, fxBuzz, 87, 6.7, [3]int{1229, 2392, 2589}, [3]float64{0, -17.1, -34.2}},
		{840, fxBuzz, 87, 4.8, [3]int{1236, 2392, 2589}, [3]float64{0, -17.5, -34.6}},
		{845, fxBuzz, 87, 6.1, [3]int{1566, 2584, 3295}, [3]float64{0, 5.1, -16.1}},
		{950, fxBuzz, 87, 6.6, [3]int{1671, 2883, 3355}, [3]float64{0, 8.0, -10.8}},
		{975, fxBuzz, 89, -0.8, [3]int{1673, 2895, 3397}, [3]float64{0, 7.8, -12.6}},
		{980, fxBuzz, 77, -9.7, [3]int{1661, 2891, 3535}, [3]float64{0, 5.6, -8.9}},
		{990, fxBuzz, 82, -60.0, [3]int{1652, 2879, 3535}, [3]float64{0, 3.8, -23.9}},
		{995, fxBuzz, 119, -60.0, [3]int{1652, 2879, 3535}, [3]float64{0, 3.8, -23.9}},
		{1005, fxBuzz, 126, -60.0, [3]int{1647, 2876, 3535}, [3]float64{0, 7.6, -22.9}},
	},
	212: { // A2
		{5, fxBuzz, 128, -60.0, [3]int{165, 1996, 1996}, [3]float64{0, -0.0, -35.8}},
		{10, fxBuzz, 127, -60.0, [3]int{165, 1996, 1996}, [3]float64{0, -0.1, -36.0}},
		{15, fxBuzz, 153, -60.0, [3]int{165, 1996, 1996}, [3]float64{0, 0.3, -35.9}},
		{20, fxBuzz, 183, -60.0, [3]int{165, 1698, 1455}, [3]float64{0, 36.6, 28.5}},
		{25, fxBuzz, 92, -14.1, [3]int{165, 1693, 1455}, [3]float64{0, 40.4, 29.7}},
		{30, fxBuzz, 87, -11.1, [3]int{184, 1689, 1455}, [3]float64{0, 15.7, 3.9}},
		{35, fxBuzz, 87, -9.8, [3]int{186, 1685, 1476}, [3]float64{0, 15.4, 3.1}},
		{45, fxBuzz, 87, -9.2, [3]int{186, 1673, 1497}, [3]float64{0, 15.1, 4.0}},
		{55, fxBuzz, 87, -9.0, [3]int{170, 1658, 1504}, [3]float64{0, 15.7, 5.0}},
		{70, fxBuzz, 87, -9.5, [3]int{162, 1641, 1515}, [3]float64{0, 15.4, 6.5}},
		{75, fxBuzz, 87, -7.9, [3]int{162, 1639, 1497}, [3]float64{0, 42.4, 31.7}},
		{80, fxBuzz, 87, -9.4, [3]int{162, 1635, 1497}, [3]float64{0, 42.7, 32.0}},
		{85, fxBuzz, 87, -7.9, [3]int{162, 1631, 1497}, [3]float64{0, 42.6, 0.5}},
		{490, fxBuzz, 87, -8.2, [3]int{162, 1584, 1497}, [3]float64{0, 43.2, 0.5}},
		{495, fxBuzz, 87, -7.3, [3]int{162, 1374, 1239}, [3]float64{0, 39.9, 36.9}},
		{510, fxBuzz, 87, -6.7, [3]int{162, 1334, 1254}, [3]float64{0, 39.3, 40.2}},
		{530, fxBuzz, 87, -6.3, [3]int{162, 1334, 1254}, [3]float64{0, 39.3, 40.2}},
		{535, fxBuzz, 87, -6.9, [3]int{162, 1212, 1531}, [3]float64{0, 42.7, 29.1}},
		{580, fxBuzz, 87, -5.4, [3]int{162, 1130, 1491}, [3]float64{0, 45.9, 27.4}},
		{585, fxBuzz, 87, -6.4, [3]int{162, 1124, 1491}, [3]float64{0, 46.2, 0.5}},
		{620, fxBuzz, 87, -6.2, [3]int{162, 1048, 1491}, [3]float64{0, 44.3, 0.5}},
		{625, fxBuzz, 87, -5.5, [3]int{162, 1030, 1304}, [3]float64{0, 43.0, 29.8}},
		{690, fxBuzz, 87, -6.2, [3]int{162, 938, 1172}, [3]float64{0, 38.2, 31.4}},
		{695, fxBuzz, 87, -6.6, [3]int{162, 919, 1040}, [3]float64{0, 37.9, 37.7}},
		{750, fxBuzz, 87, -5.7, [3]int{162, 525, 963}, [3]float64{0, 33.8, 39.1}},
		{770, fxBuzz, 87, -6.0, [3]int{162, 525, 963}, [3]float64{0, 33.8, 39.1}},
		{775, fxBuzz, 87, -6.5, [3]int{162, 467, 944}, [3]float64{0, 34.3, 39.5}},
		{815, fxBuzz, 87, -6.1, [3]int{162, 389, 897}, [3]float64{0, 38.1, 45.2}},
		{850, fxBuzz, 87, -5.1, [3]int{162, 317, 865}, [3]float64{0, 33.5, 42.8}},
		{860, fxBuzz, 87, -5.6, [3]int{162, 250, 857}, [3]float64{0, 32.0, 41.0}},
		{865, fxBuzz, 87, -4.7, [3]int{162, 250, 853}, [3]float64{0, -0.0, 39.4}},
		{895, fxBuzz, 87, -5.1, [3]int{162, 250, 845}, [3]float64{0, -0.0, 34.4}},
		{955, fxBuzz, 87, -7.1, [3]int{162, 250, 757}, [3]float64{0, -0.0, 42.8}},
		{980, fxBuzz, 82, -15.8, [3]int{162, 250, 731}, [3]float64{0, -0.0, 50.7}},
		{985, fxBuzz, 183, -25.5, [3]int{162, 250, 731}, [3]float64{0, -0.0, 50.7}},
		{990, fxBuzz, 181, -60.0, [3]int{162, 250, 731}, [3]float64{0, -0.0, 50.7}},
		{995, fxBuzz, 71, -60.0, [3]int{160, 250, 731}, [3]float64{0, -39.9, -12.7}},
		{1000, fxBuzz, 127, -60.0, [3]int{149, 250, 731}, [3]float64{0, -39.7, -10.4}},
	},
	213: { // A3
		{5, fxBuzz, 128, -60.0, [3]int{576, 977, 1996}, [3]float64{0, 27.2, 36.5}},
		{10, fxBuzz, 128, -60.0, [3]int{576, 980, 2002}, [3]float64{0, 27.3, 37.5}},
		{15, fxBuzz, 468, -60.0, [3]int{576, 990, 2010}, [3]float64{0, 26.3, 38.0}},
		{20, fxBuzz, 468, -60.0, [3]int{576, 990, 2010}, [3]float64{0, 26.3, 38.0}},
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
		{555, fxBuzz, 487, -60.0, [3]int{1304, 2379, 3423}, [3]float64{0, -21.7, 6.5}},
		{560, fxBuzz, 125, -60.0, [3]int{1283, 2184, 3423}, [3]float64{0, 1.5, -11.8}},
		{565, fxBuzz, 142, -60.0, [3]int{1067, 1987, 2980}, [3]float64{0, 7.8, 15.7}},
		{570, fxBuzz, 135, -60.0, [3]int{1011, 1986, 2975}, [3]float64{0, 9.0, 16.6}},
	},
	214: { // A4
		{5, fxTone, 0, -60.0, [3]int{239, 467, 420}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 467, 420}, [3]float64{0, -13.9, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 991, 420}, [3]float64{0, -0.1, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 467, 420}, [3]float64{0, -15.0, -60.0}},
		{25, fxTone, 0, -60.0, [3]int{239, 467, 420}, [3]float64{0, -13.9, -60.0}},
		{30, fxTone, 0, 5.2, [3]int{239, 624, 420}, [3]float64{0, -2.1, -60.0}},
		{35, fxTone, 0, 9.6, [3]int{222, 624, 420}, [3]float64{0, -8.7, -60.0}},
		{40, fxTone, 0, 11.6, [3]int{222, 461, 420}, [3]float64{0, -12.1, -60.0}},
		{45, fxTone, 0, 13.1, [3]int{222, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{75, fxTone, 0, 13.9, [3]int{338, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, 14.3, [3]int{496, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{120, fxTone, 0, 14.2, [3]int{536, 461, 420}, [3]float64{0, -60.0, -60.0}},
		{125, fxTone, 0, 13.9, [3]int{601, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{145, fxTone, 0, 12.6, [3]int{846, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{170, fxTone, 0, 9.7, [3]int{1166, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{175, fxTone, 0, 9.8, [3]int{1411, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{195, fxTone, 0, 9.6, [3]int{2169, 210, 420}, [3]float64{0, -60.0, -60.0}},
		{200, fxTone, 0, 10.3, [3]int{2362, 210, 420}, [3]float64{0, -5.6, -60.0}},
		{205, fxTone, 0, 11.0, [3]int{2461, 210, 420}, [3]float64{0, -9.2, -60.0}},
		{210, fxTone, 0, 9.8, [3]int{2840, 472, 420}, [3]float64{0, -14.1, -60.0}},
		{215, fxTone, 0, 11.4, [3]int{3254, 472, 420}, [3]float64{0, -24.3, -60.0}},
		{220, fxTone, 0, 15.4, [3]int{3860, 472, 420}, [3]float64{0, -19.3, -60.0}},
		{225, fxTone, 0, 11.8, [3]int{3895, 472, 420}, [3]float64{0, -60.0, -60.0}},
		{230, fxTone, 0, 10.5, [3]int{3901, 472, 420}, [3]float64{0, -60.0, -60.0}},
		{255, fxTone, 0, 5.0, [3]int{3901, 496, 420}, [3]float64{0, -60.0, -60.0}},
		{260, fxTone, 0, 3.5, [3]int{3901, 496, 420}, [3]float64{0, -31.3, -60.0}},
		{265, fxTone, 0, 2.0, [3]int{3901, 606, 420}, [3]float64{0, -30.3, -31.8}},
		{270, fxTone, 0, 2.0, [3]int{3901, 624, 706}, [3]float64{0, -28.5, -30.1}},
		{275, fxTone, 0, -60.0, [3]int{490, 3901, 706}, [3]float64{0, 16.0, -1.9}},
		{280, fxTone, 0, -60.0, [3]int{379, 3901, 647}, [3]float64{0, 11.0, -8.3}},
		{285, fxTone, 0, -60.0, [3]int{239, 3901, 420}, [3]float64{0, 7.8, -11.7}},
		{290, fxTone, 0, -60.0, [3]int{239, 997, 420}, [3]float64{0, -14.8, -60.0}},
		{295, fxTone, 0, -60.0, [3]int{239, 991, 420}, [3]float64{0, -14.5, -60.0}},
	},
	215: { // A5
		{5, fxTone, 0, -60.0, [3]int{239, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -8.3, [3]int{636, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{35, fxTone, 0, 0.5, [3]int{630, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{130, fxTone, 0, 3.1, [3]int{507, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{135, fxTone, 0, 3.1, [3]int{169, 484, 0}, [3]float64{0, -60.0, 0.0}},
		{140, fxTone, 0, 3.0, [3]int{169, 484, 0}, [3]float64{0, -0.0, 0.0}},
		{190, fxTone, 0, 3.0, [3]int{222, 432, 0}, [3]float64{0, -0.0, 0.0}},
		{215, fxTone, 0, 3.5, [3]int{257, 408, 0}, [3]float64{0, -0.5, 0.0}},
		{220, fxTone, 0, 3.1, [3]int{262, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{245, fxTone, 0, 3.4, [3]int{292, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{250, fxTone, 0, 3.9, [3]int{327, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{260, fxTone, 0, 4.4, [3]int{332, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{280, fxTone, 0, 1.8, [3]int{327, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{285, fxTone, 0, 3.0, [3]int{303, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{325, fxTone, 0, 3.3, [3]int{262, 408, 0}, [3]float64{0, -60.0, 0.0}},
		{330, fxTone, 0, 3.4, [3]int{251, 408, 0}, [3]float64{0, -0.5, 0.0}},
		{400, fxTone, 0, 3.0, [3]int{169, 490, 0}, [3]float64{0, -0.7, 0.0}},
		{405, fxTone, 0, 2.9, [3]int{169, 490, 0}, [3]float64{0, -15.0, 0.0}},
		{410, fxTone, 0, 2.9, [3]int{169, 490, 0}, [3]float64{0, -1.0, 0.0}},
		{415, fxTone, 0, 3.0, [3]int{507, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{510, fxTone, 0, 0.7, [3]int{624, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{545, fxTone, 0, -4.0, [3]int{624, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{570, fxTone, 0, -16.9, [3]int{618, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{575, fxTone, 0, -60.0, [3]int{577, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{580, fxTone, 0, -60.0, [3]int{245, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{585, fxTone, 0, -60.0, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
	},
	216: { // A6
		{5, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -60.0, 0.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -13.1, 0.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -14.7, 0.0}},
		{25, fxTone, 0, -60.0, [3]int{239, 700, 0}, [3]float64{0, -11.5, 0.0}},
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
		{215, fxTone, 0, -60.0, [3]int{799, 2000, 0}, [3]float64{0, -60.0, 0.0}},
		{220, fxTone, 0, -60.0, [3]int{245, 2000, 0}, [3]float64{0, -60.0, 0.0}},
		{230, fxTone, 0, -60.0, [3]int{227, 2000, 0}, [3]float64{0, -60.0, 0.0}},
	},
	217: { // A7
		{5, fxTone, 0, -60.0, [3]int{239, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -9.7, [3]int{1131, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{40, fxTone, 0, -3.8, [3]int{1160, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{55, fxTone, 0, -4.2, [3]int{1195, 373, 770}, [3]float64{0, -60.0, -60.0}},
		{60, fxTone, 0, -4.4, [3]int{1207, 373, 770}, [3]float64{0, -13.3, -14.9}},
		{115, fxTone, 0, -3.7, [3]int{1306, 408, 886}, [3]float64{0, -0.4, -3.1}},
		{120, fxTone, 0, -3.7, [3]int{420, 1335, 898}, [3]float64{0, -0.3, -2.6}},
		{130, fxTone, 0, -3.3, [3]int{420, 1335, 921}, [3]float64{0, -2.1, -2.6}},
		{135, fxTone, 0, -3.1, [3]int{426, 951, 1370}, [3]float64{0, -2.8, -2.6}},
		{215, fxTone, 0, 0.4, [3]int{484, 1108, 1528}, [3]float64{0, -3.5, -13.8}},
		{220, fxTone, 0, 0.5, [3]int{484, 1120, 1528}, [3]float64{0, -3.7, -60.0}},
		{460, fxTone, 0, 0.3, [3]int{805, 1644, 1528}, [3]float64{0, -4.2, -60.0}},
		{465, fxTone, 0, 0.2, [3]int{811, 1644, 1528}, [3]float64{0, 1.1, -60.0}},
		{470, fxTone, 0, 0.2, [3]int{822, 1644, 1528}, [3]float64{0, -4.4, -60.0}},
		{475, fxTone, 0, 0.1, [3]int{828, 1644, 1528}, [3]float64{0, -60.0, -60.0}},
		{550, fxTone, 0, -6.5, [3]int{927, 1825, 1528}, [3]float64{0, -60.0, -60.0}},
		{555, fxTone, 0, -7.5, [3]int{933, 1825, 1528}, [3]float64{0, -2.3, -60.0}},
		{580, fxTone, 0, -14.7, [3]int{962, 1849, 1528}, [3]float64{0, 0.5, -60.0}},
		{585, fxTone, 0, -14.7, [3]int{1860, 968, 1528}, [3]float64{0, -2.5, -60.0}},
		{595, fxTone, 0, -20.2, [3]int{1860, 927, 1528}, [3]float64{0, -13.9, -60.0}},
		{600, fxTone, 0, -60.0, [3]int{1860, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{605, fxTone, 0, -60.0, [3]int{1249, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{610, fxTone, 0, -60.0, [3]int{1090, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{615, fxTone, 0, -60.0, [3]int{245, 927, 1528}, [3]float64{0, -60.0, -60.0}},
		{625, fxTone, 0, -60.0, [3]int{239, 927, 1528}, [3]float64{0, -60.0, -60.0}},
	},
	218: { // A8
		{5, fxTone, 0, -60.0, [3]int{239, 997, 0}, [3]float64{0, -13.8, 0.0}},
		{20, fxTone, 0, -60.0, [3]int{245, 997, 0}, [3]float64{0, -13.6, 0.0}},
		{25, fxTone, 0, -60.0, [3]int{245, 455, 0}, [3]float64{0, -5.4, 0.0}},
		{30, fxTone, 0, -14.3, [3]int{245, 426, 0}, [3]float64{0, -10.5, 0.0}},
		{35, fxTone, 0, -11.9, [3]int{227, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{50, fxTone, 0, -3.0, [3]int{233, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{65, fxTone, 0, 0.6, [3]int{344, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{80, fxTone, 0, 2.7, [3]int{449, 426, 0}, [3]float64{0, -60.0, 0.0}},
		{85, fxTone, 0, 3.0, [3]int{484, 665, 0}, [3]float64{0, -60.0, 0.0}},
		{95, fxTone, 0, 2.1, [3]int{501, 665, 0}, [3]float64{0, -60.0, 0.0}},
		{125, fxTone, 0, -5.6, [3]int{490, 665, 0}, [3]float64{0, -60.0, 0.0}},
		{130, fxTone, 0, -7.0, [3]int{449, 665, 0}, [3]float64{0, -4.8, 0.0}},
		{140, fxTone, 0, -10.9, [3]int{303, 665, 0}, [3]float64{0, -2.9, 0.0}},
		{145, fxTone, 0, -13.6, [3]int{303, 542, 0}, [3]float64{0, -5.1, 0.0}},
		{150, fxTone, 0, -17.3, [3]int{303, 542, 0}, [3]float64{0, -1.2, 0.0}},
		{155, fxTone, 0, -23.0, [3]int{239, 542, 0}, [3]float64{0, -6.7, 0.0}},
		{160, fxTone, 0, -27.0, [3]int{239, 443, 0}, [3]float64{0, -2.9, 0.0}},
		{165, fxTone, 0, -27.0, [3]int{338, 548, 0}, [3]float64{0, -13.7, 0.0}},
		{170, fxTone, 0, -25.3, [3]int{338, 548, 0}, [3]float64{0, -7.5, 0.0}},
		{175, fxTone, 0, -18.5, [3]int{338, 391, 0}, [3]float64{0, -11.3, 0.0}},
		{180, fxTone, 0, -13.3, [3]int{338, 391, 0}, [3]float64{0, -4.8, 0.0}},
		{185, fxTone, 0, -9.5, [3]int{338, 548, 0}, [3]float64{0, -11.8, 0.0}},
		{190, fxTone, 0, -6.8, [3]int{251, 700, 0}, [3]float64{0, -6.2, 0.0}},
		{195, fxTone, 0, -4.7, [3]int{251, 863, 0}, [3]float64{0, -7.0, 0.0}},
		{200, fxTone, 0, -3.1, [3]int{251, 863, 0}, [3]float64{0, -8.2, 0.0}},
		{205, fxTone, 0, -1.8, [3]int{286, 863, 0}, [3]float64{0, -8.8, 0.0}},
		{210, fxTone, 0, -0.6, [3]int{327, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{225, fxTone, 0, 1.5, [3]int{443, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{235, fxTone, 0, 1.5, [3]int{496, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{280, fxTone, 0, -10.3, [3]int{496, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{295, fxTone, 0, -60.0, [3]int{490, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{300, fxTone, 0, -60.0, [3]int{251, 863, 0}, [3]float64{0, -60.0, 0.0}},
		{310, fxTone, 0, -60.0, [3]int{239, 863, 0}, [3]float64{0, -60.0, 0.0}},
	},
	219: { // A9
		{5, fxTone, 0, -60.0, [3]int{233, 2443, 828}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 2443, 828}, [3]float64{0, -60.0, -60.0}},
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
		{335, fxTone, 0, -60.0, [3]int{257, 490, 863}, [3]float64{0, -3.3, -5.5}},
		{340, fxTone, 0, -60.0, [3]int{239, 490, 863}, [3]float64{0, -10.0, -11.6}},
		{345, fxTone, 0, -60.0, [3]int{239, 478, 682}, [3]float64{0, -13.2, -0.1}},
		{350, fxTone, 0, -60.0, [3]int{233, 478, 682}, [3]float64{0, -14.8, -60.0}},
	},
	220: { // B0
		{5, fxTone, 0, -60.0, [3]int{239, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -20.1, [3]int{245, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -7.3, [3]int{2583, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{40, fxTone, 0, -2.7, [3]int{2595, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{135, fxTone, 0, -2.9, [3]int{2595, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{185, fxTone, 0, -15.6, [3]int{2595, 227, 0}, [3]float64{0, -60.0, 0.0}},
		{190, fxTone, 0, -18.3, [3]int{2595, 227, 0}, [3]float64{0, -13.9, 0.0}},
		{195, fxTone, 0, -20.8, [3]int{2595, 455, 0}, [3]float64{0, -11.8, 0.0}},
		{200, fxTone, 0, -60.0, [3]int{245, 461, 0}, [3]float64{0, -11.9, 0.0}},
		{210, fxTone, 0, -60.0, [3]int{233, 455, 0}, [3]float64{0, -12.9, 0.0}},
	},
	221: { // B1
		{5, fxTone, 0, -60.0, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{25, fxTone, 0, -6.0, [3]int{1376, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{40, fxTone, 0, 0.7, [3]int{1516, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{95, fxTone, 0, -1.1, [3]int{2473, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{170, fxTone, 0, -2.5, [3]int{3767, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{185, fxTone, 0, -2.4, [3]int{3709, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{230, fxTone, 0, -1.4, [3]int{2752, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{260, fxTone, 0, -0.8, [3]int{2111, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{295, fxTone, 0, 1.5, [3]int{1411, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{375, fxTone, 0, -7.6, [3]int{1400, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{410, fxTone, 0, -60.0, [3]int{1400, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{415, fxTone, 0, -60.0, [3]int{257, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{430, fxTone, 0, -60.0, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
	},
	222: { // B2
		{5, fxTone, 0, -60.0, [3]int{239, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -5.8, [3]int{1376, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{40, fxTone, 0, 3.4, [3]int{1405, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{140, fxTone, 0, 3.3, [3]int{1405, 198, 0}, [3]float64{0, -60.0, 0.0}},
		{145, fxTone, 0, 2.8, [3]int{1429, 198, 0}, [3]float64{0, -10.4, 0.0}},
		{150, fxTone, 0, 1.9, [3]int{1429, 198, 0}, [3]float64{0, -3.8, 0.0}},
		{160, fxTone, 0, 1.7, [3]int{1977, 198, 0}, [3]float64{0, -2.8, 0.0}},
		{165, fxTone, 0, 4.7, [3]int{2484, 198, 0}, [3]float64{0, -11.6, 0.0}},
		{175, fxTone, 0, 4.3, [3]int{2484, 198, 0}, [3]float64{0, -13.1, 0.0}},
		{180, fxTone, 0, 0.3, [3]int{1709, 198, 0}, [3]float64{0, -1.3, 0.0}},
		{190, fxTone, 0, 2.3, [3]int{1697, 198, 0}, [3]float64{0, -10.5, 0.0}},
		{230, fxTone, 0, 2.9, [3]int{1697, 204, 0}, [3]float64{0, -11.6, 0.0}},
		{255, fxTone, 0, 2.9, [3]int{1697, 397, 0}, [3]float64{0, -12.3, 0.0}},
		{280, fxTone, 0, 2.9, [3]int{1697, 595, 0}, [3]float64{0, -12.8, 0.0}},
		{355, fxTone, 0, -2.4, [3]int{1703, 601, 0}, [3]float64{0, -13.2, 0.0}},
		{400, fxTone, 0, -13.2, [3]int{1703, 601, 0}, [3]float64{0, -16.1, 0.0}},
		{405, fxTone, 0, -15.7, [3]int{1703, 472, 0}, [3]float64{0, -1.3, 0.0}},
		{410, fxTone, 0, -18.7, [3]int{1703, 472, 0}, [3]float64{0, -15.8, 0.0}},
		{415, fxTone, 0, -60.0, [3]int{1703, 472, 0}, [3]float64{0, -14.9, 0.0}},
		{420, fxTone, 0, -60.0, [3]int{1703, 478, 0}, [3]float64{0, -3.3, 0.0}},
		{425, fxTone, 0, -60.0, [3]int{356, 478, 0}, [3]float64{0, -10.6, 0.0}},
		{430, fxTone, 0, -60.0, [3]int{239, 478, 0}, [3]float64{0, -4.3, 0.0}},
		{435, fxTone, 0, -60.0, [3]int{233, 478, 0}, [3]float64{0, -60.0, 0.0}},
	},
	223: { // B3
		{5, fxTone, 0, -60.0, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -20.8, [3]int{239, 490, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -13.8, [3]int{595, 490, 0}, [3]float64{0, 0.0, 0.0}},
		{30, fxTone, 0, -11.3, [3]int{595, 3096, 0}, [3]float64{0, 9.4, 0.0}},
		{130, fxTone, 0, -7.1, [3]int{595, 3096, 0}, [3]float64{0, 8.2, 0.0}},
		{135, fxTone, 0, -4.4, [3]int{595, 1423, 0}, [3]float64{0, -5.2, 0.0}},
		{140, fxTone, 0, -2.4, [3]int{1394, 601, 0}, [3]float64{0, -5.9, 0.0}},
		{255, fxTone, 0, -0.9, [3]int{1394, 601, 0}, [3]float64{0, -9.9, 0.0}},
		{310, fxTone, 0, -12.2, [3]int{1400, 601, 0}, [3]float64{0, -11.6, 0.0}},
		{325, fxTone, 0, -21.2, [3]int{1394, 589, 0}, [3]float64{0, -15.1, 0.0}},
		{330, fxTone, 0, -60.0, [3]int{1394, 589, 0}, [3]float64{0, -6.9, 0.0}},
		{335, fxTone, 0, -60.0, [3]int{478, 991, 0}, [3]float64{0, -4.5, 0.0}},
		{340, fxTone, 0, -60.0, [3]int{245, 991, 0}, [3]float64{0, -12.9, 0.0}},
		{345, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -13.8, 0.0}},
	},
	224: { // B4
		{5, fxTone, 0, -60.0, [3]int{239, 1551, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 1551, 0}, [3]float64{0, -60.0, 0.0}},
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
		{400, fxTone, 0, -60.0, [3]int{1825, 991, 0}, [3]float64{0, -2.9, 0.0}},
		{405, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -14.2, 0.0}},
		{415, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -13.8, 0.0}},
	},
	225: { // B5
		{5, fxTone, 0, -60.0, [3]int{233, 490, 204}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 490, 204}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 490, 204}, [3]float64{0, -0.0, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 490, 204}, [3]float64{0, -0.0, -60.0}},
		{25, fxTone, 0, -3.3, [3]int{951, 501, 204}, [3]float64{0, -8.9, -60.0}},
		{30, fxTone, 0, -0.2, [3]int{951, 781, 204}, [3]float64{0, -12.9, -60.0}},
		{35, fxTone, 0, 1.7, [3]int{951, 3493, 204}, [3]float64{0, -0.2, -60.0}},
		{50, fxTone, 0, 3.7, [3]int{811, 3493, 204}, [3]float64{0, -0.4, -60.0}},
		{55, fxTone, 0, 4.0, [3]int{764, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{85, fxTone, 0, 5.4, [3]int{472, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{95, fxTone, 0, 5.8, [3]int{379, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{105, fxTone, 0, 6.0, [3]int{280, 3493, 204}, [3]float64{0, -60.0, -60.0}},
		{110, fxTone, 0, 5.9, [3]int{280, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, 5.5, [3]int{280, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{120, fxTone, 0, 19.8, [3]int{3493, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{130, fxTone, 0, 17.9, [3]int{3499, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{155, fxTone, 0, 13.3, [3]int{3499, 175, 204}, [3]float64{0, -60.0, -60.0}},
		{160, fxTone, 0, 12.6, [3]int{3499, 175, 204}, [3]float64{0, -14.5, -60.0}},
		{165, fxTone, 0, 11.0, [3]int{3499, 175, 204}, [3]float64{0, -29.1, -60.0}},
		{170, fxTone, 0, 8.7, [3]int{3499, 181, 204}, [3]float64{0, -26.1, -29.0}},
		{175, fxTone, 0, 7.3, [3]int{3499, 233, 472}, [3]float64{0, -22.0, -24.8}},
		{180, fxTone, 0, 3.2, [3]int{3499, 461, 472}, [3]float64{0, -20.1, -20.5}},
		{185, fxTone, 0, -13.9, [3]int{338, 461, 472}, [3]float64{0, -4.7, -5.0}},
		{190, fxTone, 0, -17.4, [3]int{239, 531, 204}, [3]float64{0, -1.6, -4.1}},
		{195, fxTone, 0, -60.0, [3]int{239, 986, 204}, [3]float64{0, -0.1, -60.0}},
		{200, fxTone, 0, -60.0, [3]int{239, 531, 204}, [3]float64{0, -13.0, -60.0}},
		{205, fxTone, 0, -60.0, [3]int{233, 531, 204}, [3]float64{0, -60.0, -60.0}},
	},
	226: { // B6
		{5, fxTone, 0, -60.0, [3]int{239, 367, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 367, 0}, [3]float64{0, -60.0, 0.0}},
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
		{190, fxTone, 0, -60.0, [3]int{251, 467, 0}, [3]float64{0, -2.1, 0.0}},
		{195, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -13.9, 0.0}},
		{200, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -14.1, 0.0}},
		{205, fxTone, 0, -60.0, [3]int{239, 467, 0}, [3]float64{0, -14.9, 0.0}},
	},
	227: { // B7
		{5, fxTone, 0, -60.0, [3]int{239, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{20, fxTone, 0, -23.5, [3]int{239, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{25, fxTone, 0, -4.6, [3]int{2892, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{40, fxTone, 0, 0.7, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{60, fxTone, 0, 0.4, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{70, fxTone, 0, -2.9, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{80, fxTone, 0, -12.4, [3]int{2898, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{85, fxTone, 0, -1.3, [3]int{3895, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{125, fxTone, 0, 2.6, [3]int{3895, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{160, fxTone, 0, -3.4, [3]int{3895, 268, 0}, [3]float64{0, -60.0, 0.0}},
		{165, fxTone, 0, -4.9, [3]int{3895, 268, 0}, [3]float64{0, -31.7, 0.0}},
		{170, fxTone, 0, -6.7, [3]int{3895, 379, 0}, [3]float64{0, -17.6, 0.0}},
		{175, fxTone, 0, -26.2, [3]int{422, 414, 0}, [3]float64{0, -4.2, 0.0}},
		{180, fxTone, 0, -60.0, [3]int{391, 997, 0}, [3]float64{0, -0.4, 0.0}},
		{185, fxTone, 0, -60.0, [3]int{373, 997, 0}, [3]float64{0, -4.3, 0.0}},
		{190, fxTone, 0, -60.0, [3]int{274, 1003, 0}, [3]float64{0, -10.1, 0.0}},
		{195, fxTone, 0, -60.0, [3]int{239, 997, 0}, [3]float64{0, -0.1, 0.0}},
		{200, fxTone, 0, -60.0, [3]int{239, 997, 0}, [3]float64{0, -60.0, 0.0}},
	},
	228: { // B8
		{5, fxTone, 0, -60.0, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{20, fxTone, 0, -17.3, [3]int{245, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{25, fxTone, 0, -9.5, [3]int{2053, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{35, fxTone, 0, -5.9, [3]int{2053, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{150, fxTone, 0, -4.2, [3]int{1481, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{220, fxTone, 0, -3.7, [3]int{1324, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{290, fxTone, 0, -5.5, [3]int{1300, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{320, fxTone, 0, -13.1, [3]int{1300, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{340, fxTone, 0, -60.0, [3]int{1271, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{345, fxTone, 0, -60.0, [3]int{239, 0, 0}, [3]float64{0, 0.0, 0.0}},
		{355, fxTone, 0, -60.0, [3]int{227, 0, 0}, [3]float64{0, 0.0, 0.0}},
	},
	229: { // B9
		{5, fxTone, 0, -60.0, [3]int{233, 898, 2700}, [3]float64{0, -60.0, -60.0}},
		{20, fxTone, 0, -28.7, [3]int{245, 898, 2700}, [3]float64{0, -60.0, -60.0}},
		{25, fxTone, 0, -15.5, [3]int{894, 898, 2700}, [3]float64{0, -60.0, -60.0}},
		{30, fxTone, 0, -12.1, [3]int{896, 898, 2700}, [3]float64{0, -0.0, -60.0}},
		{35, fxTone, 0, -8.2, [3]int{1802, 898, 2700}, [3]float64{0, -4.4, 1.4}},
		{80, fxTone, 0, -7.6, [3]int{1802, 904, 2700}, [3]float64{0, -3.9, 1.2}},
		{85, fxTone, 0, -10.8, [3]int{900, 268, 2700}, [3]float64{0, 0.1, -60.0}},
		{95, fxTone, 0, -15.2, [3]int{787, 268, 2700}, [3]float64{0, -12.6, -60.0}},
		{100, fxTone, 0, -12.6, [3]int{776, 408, 2700}, [3]float64{0, -11.9, -60.0}},
		{115, fxTone, 0, -8.8, [3]int{676, 362, 2700}, [3]float64{0, -11.6, -60.0}},
		{120, fxTone, 0, -9.3, [3]int{589, 344, 2700}, [3]float64{0, -13.8, -60.0}},
		{125, fxTone, 0, -6.3, [3]int{563, 344, 2700}, [3]float64{0, -60.0, -60.0}},
		{140, fxTone, 0, -5.2, [3]int{524, 344, 2700}, [3]float64{0, -60.0, -60.0}},
		{145, fxBuzz, 87, -5.3, [3]int{524, 344, 268}, [3]float64{0, -60.0, -60.0}},
		{150, fxBuzz, 87, -4.8, [3]int{525, 344, 268}, [3]float64{0, -60.0, -60.0}},
		{155, fxTone, 0, -6.1, [3]int{525, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{160, fxBuzz, 87, -4.8, [3]int{525, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{165, fxTone, 0, -6.8, [3]int{525, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{175, fxTone, 0, -6.2, [3]int{543, 286, 268}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, -6.0, [3]int{590, 286, 268}, [3]float64{0, 0.1, -60.0}},
		{185, fxTone, 0, -5.8, [3]int{636, 414, 268}, [3]float64{0, -10.9, -60.0}},
		{195, fxTone, 0, -6.3, [3]int{717, 437, 268}, [3]float64{0, -11.1, -60.0}},
		{200, fxTone, 0, -8.5, [3]int{764, 507, 268}, [3]float64{0, 0.1, 0.1}},
		{210, fxTone, 0, -6.9, [3]int{892, 531, 268}, [3]float64{0, -10.9, -14.9}},
		{215, fxTone, 0, -7.1, [3]int{892, 898, 601}, [3]float64{0, -0.0, 0.1}},
		{220, fxTone, 0, -4.9, [3]int{1802, 898, 3493}, [3]float64{0, -3.0, 3.0}},
		{270, fxTone, 0, -7.6, [3]int{1802, 904, 3493}, [3]float64{0, -6.7, 2.8}},
		{275, fxTone, 0, -8.3, [3]int{1802, 3493, 898}, [3]float64{0, 2.8, -12.0}},
		{285, fxTone, 0, -10.6, [3]int{1802, 3493, 892}, [3]float64{0, 2.8, -1.9}},
		{290, fxTone, 0, -12.0, [3]int{1802, 3493, 881}, [3]float64{0, 2.8, -16.3}},
		{295, fxTone, 0, -14.1, [3]int{1802, 3493, 881}, [3]float64{0, 2.7, -60.0}},
		{300, fxTone, 0, -16.2, [3]int{1802, 3493, 881}, [3]float64{0, 2.8, -60.0}},
		{305, fxTone, 0, -21.4, [3]int{932, 1802, 881}, [3]float64{0, 1.9, -60.0}},
		{310, fxTone, 0, -24.9, [3]int{884, 986, 881}, [3]float64{0, 0.0, -60.0}},
		{315, fxTone, 0, -28.6, [3]int{257, 986, 881}, [3]float64{0, -60.0, -60.0}},
		{320, fxTone, 0, -60.0, [3]int{245, 467, 881}, [3]float64{0, -13.3, -60.0}},
		{325, fxTone, 0, -60.0, [3]int{239, 467, 881}, [3]float64{0, -60.0, -60.0}},
		{330, fxTone, 0, -60.0, [3]int{239, 467, 881}, [3]float64{0, -60.0, -60.0}},
	},
	230: { // C0
		{5, fxTone, 0, -60.0, [3]int{239, 991, 1545}, [3]float64{0, -14.9, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 991, 1545}, [3]float64{0, -14.9, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 991, 1545}, [3]float64{0, -0.1, -60.0}},
		{25, fxTone, 0, -4.2, [3]int{2315, 991, 1545}, [3]float64{0, -5.4, -60.0}},
		{30, fxTone, 0, -3.9, [3]int{2315, 2058, 1545}, [3]float64{0, -14.2, -60.0}},
		{35, fxTone, 0, -0.8, [3]int{2315, 2058, 1545}, [3]float64{0, -16.1, -60.0}},
		{40, fxTone, 0, -2.1, [3]int{2058, 2058, 1545}, [3]float64{0, -2.2, -60.0}},
		{45, fxTone, 0, -2.7, [3]int{2053, 2058, 1545}, [3]float64{0, 0.0, -2.6}},
		{55, fxTone, 0, -4.0, [3]int{1790, 1767, 2327}, [3]float64{0, -13.5, -10.6}},
		{70, fxTone, 0, -1.1, [3]int{1528, 1767, 2327}, [3]float64{0, -11.8, -9.0}},
		{75, fxTone, 0, -1.0, [3]int{1265, 1767, 2327}, [3]float64{0, -3.5, -4.9}},
		{80, fxTone, 0, -1.4, [3]int{1265, 2298, 1277}, [3]float64{0, -6.9, -14.6}},
		{85, fxTone, 0, 0.5, [3]int{1260, 1516, 2327}, [3]float64{0, -0.6, -2.9}},
		{90, fxTone, 0, -1.4, [3]int{1009, 1015, 2239}, [3]float64{0, -11.7, -9.3}},
		{95, fxTone, 0, 1.4, [3]int{991, 1003, 2239}, [3]float64{0, -8.1, -5.9}},
		{100, fxTone, 0, 0.7, [3]int{781, 764, 2239}, [3]float64{0, -10.2, -8.0}},
		{105, fxTone, 0, 1.7, [3]int{764, 577, 2239}, [3]float64{0, -13.7, 4.8}},
		{110, fxTone, 0, 2.3, [3]int{723, 402, 245}, [3]float64{0, -12.3, -13.3}},
		{115, fxTone, 0, 0.2, [3]int{647, 309, 245}, [3]float64{0, 0.1, -60.0}},
		{120, fxTone, 0, 2.8, [3]int{630, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{125, fxTone, 0, 0.9, [3]int{519, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{130, fxTone, 0, 2.5, [3]int{501, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{135, fxTone, 0, 3.1, [3]int{362, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{140, fxTone, 0, 0.6, [3]int{344, 309, 245}, [3]float64{0, -60.0, -60.0}},
		{145, fxTone, 0, 3.6, [3]int{338, 455, 245}, [3]float64{0, -60.0, -60.0}},
		{150, fxTone, 0, 2.0, [3]int{198, 455, 583}, [3]float64{0, -60.0, -60.0}},
		{165, fxTone, 0, -5.1, [3]int{198, 455, 583}, [3]float64{0, -60.0, -60.0}},
		{170, fxTone, 0, -6.2, [3]int{297, 455, 583}, [3]float64{0, -6.3, -60.0}},
		{175, fxTone, 0, -5.8, [3]int{327, 595, 583}, [3]float64{0, -0.1, -60.0}},
		{180, fxTone, 0, -7.0, [3]int{297, 455, 583}, [3]float64{0, -6.0, -14.7}},
		{185, fxTone, 0, -9.9, [3]int{327, 595, 787}, [3]float64{0, -2.7, -6.1}},
		{190, fxTone, 0, -9.0, [3]int{327, 595, 898}, [3]float64{0, -6.4, -9.9}},
		{200, fxTone, 0, -13.4, [3]int{251, 484, 787}, [3]float64{0, -8.2, -13.8}},
		{205, fxTone, 0, -12.7, [3]int{251, 566, 881}, [3]float64{0, -9.3, -12.0}},
		{210, fxTone, 0, -14.5, [3]int{251, 484, 881}, [3]float64{0, -0.0, -0.1}},
		{215, fxTone, 0, -17.3, [3]int{216, 490, 881}, [3]float64{0, -8.8, -60.0}},
		{220, fxTone, 0, -17.1, [3]int{216, 490, 881}, [3]float64{0, -0.0, -60.0}},
		{225, fxTone, 0, -60.0, [3]int{268, 490, 881}, [3]float64{0, -10.7, -60.0}},
		{240, fxTone, 0, -60.0, [3]int{239, 461, 881}, [3]float64{0, -0.0, -60.0}},
		{245, fxTone, 0, -60.0, [3]int{239, 461, 881}, [3]float64{0, -60.0, -60.0}},
	},
	231: { // C1
		{5, fxTone, 0, -60.0, [3]int{239, 449, 426}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 449, 426}, [3]float64{0, -13.5, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{245, 449, 426}, [3]float64{0, -14.1, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{653, 478, 426}, [3]float64{0, -1.8, -14.5}},
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
		{305, fxTone, 0, -60.0, [3]int{519, 222, 490}, [3]float64{0, 0.0, 0.0}},
		{310, fxTone, 0, -60.0, [3]int{490, 280, 799}, [3]float64{0, -3.2, -4.5}},
		{315, fxTone, 0, -60.0, [3]int{245, 472, 886}, [3]float64{0, -2.0, -4.9}},
		{325, fxTone, 0, -60.0, [3]int{239, 472, 799}, [3]float64{0, -11.2, -12.6}},
		{330, fxTone, 0, -60.0, [3]int{227, 472, 799}, [3]float64{0, -14.7, -60.0}},
	},
	232: { // C2
		{5, fxTone, 0, -60.0, [3]int{233, 216, 974}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 216, 974}, [3]float64{0, -60.0, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 216, 974}, [3]float64{0, 0.0, -60.0}},
		{25, fxTone, 0, -60.0, [3]int{455, 216, 974}, [3]float64{0, -2.1, -60.0}},
		{30, fxTone, 0, -60.0, [3]int{478, 910, 974}, [3]float64{0, -5.1, -0.1}},
		{35, fxTone, 0, -6.8, [3]int{499, 904, 974}, [3]float64{0, -0.1, -0.1}},
		{40, fxTone, 0, -6.4, [3]int{501, 904, 974}, [3]float64{0, -2.9, -60.0}},
		{45, fxTone, 0, -6.4, [3]int{507, 904, 169}, [3]float64{0, -2.8, -11.0}},
		{50, fxTone, 0, -0.8, [3]int{507, 910, 2210}, [3]float64{0, -0.1, 4.5}},
		{55, fxTone, 0, -1.4, [3]int{507, 916, 2198}, [3]float64{0, -2.7, -2.8}},
		{60, fxTone, 0, -1.5, [3]int{507, 921, 175}, [3]float64{0, -2.6, -10.3}},
		{65, fxTone, 0, 0.9, [3]int{519, 933, 2012}, [3]float64{0, -2.9, -7.5}},
		{75, fxTone, 0, -1.6, [3]int{525, 986, 2006}, [3]float64{0, -3.2, -6.4}},
		{80, fxTone, 0, 0.7, [3]int{531, 1009, 239}, [3]float64{0, -4.3, -9.4}},
		{85, fxTone, 0, -1.0, [3]int{543, 1020, 1814}, [3]float64{0, -0.0, 1.9}},
		{90, fxTone, 0, -1.9, [3]int{548, 986, 239}, [3]float64{0, -8.3, -10.9}},
		{95, fxTone, 0, 0.1, [3]int{549, 986, 239}, [3]float64{0, -60.0, -60.0}},
		{100, fxTone, 0, -1.6, [3]int{566, 1114, 239}, [3]float64{0, -0.0, 0.1}},
		{105, fxTone, 0, -2.9, [3]int{566, 1114, 239}, [3]float64{0, -60.0, -60.0}},
		{110, fxTone, 0, -1.6, [3]int{566, 1155, 245}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, -3.9, [3]int{566, 1155, 245}, [3]float64{0, 0.0, 0.1}},
		{135, fxTone, 0, -9.7, [3]int{560, 1172, 245}, [3]float64{0, -3.5, -9.9}},
		{140, fxTone, 0, -9.5, [3]int{554, 1172, 344}, [3]float64{0, -7.0, -12.4}},
		{165, fxTone, 0, -25.7, [3]int{449, 1155, 251}, [3]float64{0, -14.3, 0.0}},
		{170, fxTone, 0, -25.2, [3]int{449, 1155, 251}, [3]float64{0, -9.1, -60.0}},
		{175, fxTone, 0, -22.7, [3]int{449, 175, 251}, [3]float64{0, 0.0, -60.0}},
		{180, fxTone, 0, -9.3, [3]int{3312, 175, 251}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, -8.0, [3]int{3336, 175, 251}, [3]float64{0, -60.0, -60.0}},
		{190, fxTone, 0, -3.9, [3]int{3336, 175, 251}, [3]float64{0, -13.1, -60.0}},
		{195, fxTone, 0, -4.4, [3]int{3336, 175, 618}, [3]float64{0, -26.6, -60.0}},
		{200, fxTone, 0, -4.2, [3]int{3336, 175, 618}, [3]float64{0, -60.0, -60.0}},
		{205, fxTone, 0, 0.3, [3]int{3341, 192, 618}, [3]float64{0, -13.2, -60.0}},
		{215, fxTone, 0, -1.0, [3]int{3341, 216, 618}, [3]float64{0, -26.5, -60.0}},
		{220, fxTone, 0, -0.8, [3]int{3341, 297, 618}, [3]float64{0, -17.5, -22.8}},
		{225, fxTone, 0, -20.7, [3]int{1674, 315, 618}, [3]float64{0, -6.6, -10.5}},
		{230, fxTone, 0, -21.3, [3]int{1668, 315, 618}, [3]float64{0, -7.2, -10.7}},
		{235, fxTone, 0, -22.4, [3]int{262, 472, 618}, [3]float64{0, -2.9, -9.2}},
		{240, fxTone, 0, -60.0, [3]int{245, 472, 484}, [3]float64{0, -14.6, -0.0}},
		{245, fxTone, 0, -60.0, [3]int{239, 472, 484}, [3]float64{0, -12.0, -60.0}},
		{250, fxTone, 0, -60.0, [3]int{233, 472, 484}, [3]float64{0, -60.0, -60.0}},
	},
	233: { // C3
		{5, fxTone, 0, -60.0, [3]int{222, 490, 414}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 490, 414}, [3]float64{0, -12.0, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 490, 414}, [3]float64{0, -13.2, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{268, 490, 414}, [3]float64{0, -7.4, -60.0}},
		{25, fxTone, 0, -9.4, [3]int{3143, 216, 414}, [3]float64{0, -11.7, -60.0}},
		{30, fxTone, 0, -8.3, [3]int{3161, 175, 414}, [3]float64{0, -19.1, -60.0}},
		{40, fxTone, 0, -3.0, [3]int{3166, 187, 414}, [3]float64{0, -11.8, -60.0}},
		{45, fxTone, 0, -3.2, [3]int{3172, 175, 414}, [3]float64{0, -25.7, -60.0}},
		{50, fxTone, 0, -0.9, [3]int{3166, 175, 414}, [3]float64{0, -60.0, -60.0}},
		{115, fxTone, 0, 3.8, [3]int{3376, 175, 414}, [3]float64{0, -60.0, -60.0}},
		{120, fxTone, 0, 3.9, [3]int{3470, 204, 414}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, 4.3, [3]int{3785, 204, 414}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, 4.4, [3]int{3785, 204, 414}, [3]float64{0, -16.8, -60.0}},
		{190, fxTone, 0, 4.4, [3]int{3785, 204, 414}, [3]float64{0, -16.8, -60.0}},
		{195, fxTone, 0, 4.4, [3]int{3785, 204, 414}, [3]float64{0, -26.4, -60.0}},
		{200, fxTone, 0, 0.5, [3]int{3785, 338, 414}, [3]float64{0, -18.9, -60.0}},
		{205, fxTone, 0, 0.7, [3]int{3773, 338, 414}, [3]float64{0, -30.2, -60.0}},
		{210, fxTone, 0, 0.6, [3]int{3773, 356, 414}, [3]float64{0, -31.1, -16.8}},
		{215, fxTone, 0, 0.6, [3]int{3773, 356, 414}, [3]float64{0, -27.2, -31.4}},
		{220, fxTone, 0, -18.9, [3]int{233, 3779, 571}, [3]float64{0, 10.3, -8.6}},
		{225, fxTone, 0, -20.3, [3]int{233, 3779, 741}, [3]float64{0, 16.5, -10.4}},
		{230, fxTone, 0, -20.5, [3]int{233, 3779, 571}, [3]float64{0, 16.7, -8.9}},
		{235, fxTone, 0, -20.3, [3]int{239, 3755, 571}, [3]float64{0, 10.1, -8.0}},
		{240, fxTone, 0, -60.0, [3]int{239, 997, 571}, [3]float64{0, -13.4, -60.0}},
		{250, fxTone, 0, -60.0, [3]int{239, 991, 571}, [3]float64{0, -13.5, -60.0}},
	},
	234: { // C4
		{5, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -14.7, 0.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 991, 0}, [3]float64{0, -14.9, 0.0}},
		{25, fxTone, 0, -19.4, [3]int{927, 402, 0}, [3]float64{0, -8.5, 0.0}},
		{35, fxTone, 0, -12.2, [3]int{962, 350, 0}, [3]float64{0, -13.7, 0.0}},
		{45, fxTone, 0, -9.2, [3]int{962, 338, 0}, [3]float64{0, 0.1, 0.0}},
		{50, fxTone, 0, -3.9, [3]int{962, 735, 0}, [3]float64{0, -7.4, 0.0}},
		{60, fxTone, 0, -2.2, [3]int{962, 735, 0}, [3]float64{0, -14.8, 0.0}},
		{70, fxTone, 0, -1.9, [3]int{962, 735, 0}, [3]float64{0, 0.0, 0.0}},
		{75, fxTone, 0, 0.9, [3]int{956, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{85, fxTone, 0, 0.8, [3]int{956, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{90, fxTone, 0, 1.0, [3]int{839, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{95, fxTone, 0, 0.3, [3]int{839, 735, 0}, [3]float64{0, -60.0, 0.0}},
		{100, fxBuzz, 87, 1.1, [3]int{839, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{105, fxTone, 0, -0.5, [3]int{839, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{110, fxTone, 0, 1.2, [3]int{956, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{115, fxTone, 0, -0.6, [3]int{956, 257, 0}, [3]float64{0, -60.0, 0.0}},
		{120, fxTone, 0, -0.7, [3]int{956, 257, 0}, [3]float64{0, 0.1, 0.0}},
		{125, fxTone, 0, -1.5, [3]int{956, 962, 0}, [3]float64{0, -4.1, 0.0}},
		{130, fxTone, 0, -2.8, [3]int{1913, 986, 0}, [3]float64{0, -2.5, 0.0}},
		{135, fxTone, 0, -2.5, [3]int{1913, 1330, 0}, [3]float64{0, -10.2, 0.0}},
		{140, fxTone, 0, -4.4, [3]int{1913, 986, 0}, [3]float64{0, -17.1, 0.0}},
		{145, fxTone, 0, -3.5, [3]int{1913, 986, 0}, [3]float64{0, -10.8, 0.0}},
		{150, fxNoise, 0, -5.1, [3]int{1913, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{155, fxNoise, 0, -5.7, [3]int{974, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{160, fxNoise, 0, -7.6, [3]int{974, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{165, fxNoise, 0, -5.5, [3]int{1913, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{175, fxNoise, 0, -9.3, [3]int{1914, 986, 0}, [3]float64{0, -60.0, 0.0}},
		{180, fxNoise, 0, -1.0, [3]int{2957, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{185, fxNoise, 0, -3.8, [3]int{2957, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{190, fxTone, 0, -2.5, [3]int{2957, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{205, fxTone, 0, -2.8, [3]int{2962, 222, 0}, [3]float64{0, -60.0, 0.0}},
		{210, fxTone, 0, -3.4, [3]int{2980, 222, 0}, [3]float64{0, -25.3, 0.0}},
		{215, fxTone, 0, -3.2, [3]int{2986, 222, 0}, [3]float64{0, -10.5, 0.0}},
		{220, fxTone, 0, -7.0, [3]int{2986, 222, 0}, [3]float64{0, -24.2, 0.0}},
		{230, fxTone, 0, -8.5, [3]int{2986, 227, 0}, [3]float64{0, -10.5, 0.0}},
		{235, fxTone, 0, -9.9, [3]int{2986, 239, 0}, [3]float64{0, -21.4, 0.0}},
		{240, fxTone, 0, -20.6, [3]int{239, 688, 0}, [3]float64{0, -4.7, 0.0}},
		{245, fxTone, 0, -60.0, [3]int{239, 997, 0}, [3]float64{0, -12.9, 0.0}},
		{255, fxTone, 0, -60.0, [3]int{239, 997, 0}, [3]float64{0, -15.0, 0.0}},
		{265, fxTone, 0, -60.0, [3]int{239, 478, 0}, [3]float64{0, -12.0, 0.0}},
	},
	235: { // C5
		{5, fxTone, 0, -60.0, [3]int{239, 467, 297}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 467, 297}, [3]float64{0, -0.0, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 665, 297}, [3]float64{0, -14.5, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{239, 665, 297}, [3]float64{0, -2.1, -60.0}},
		{25, fxTone, 0, -60.0, [3]int{309, 764, 297}, [3]float64{0, -10.9, -60.0}},
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
		{150, fxTone, 0, 0.0, [3]int{297, 297, 892}, [3]float64{0, 0.0, -60.0}},
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
		{360, fxTone, 0, -60.0, [3]int{187, 612, 892}, [3]float64{0, -11.8, -60.0}},
		{365, fxTone, 0, -60.0, [3]int{198, 612, 892}, [3]float64{0, -9.6, -60.0}},
		{370, fxTone, 0, -60.0, [3]int{239, 991, 892}, [3]float64{0, -5.0, -60.0}},
		{375, fxTone, 0, -60.0, [3]int{239, 991, 892}, [3]float64{0, -14.5, -60.0}},
		{380, fxTone, 0, -60.0, [3]int{239, 991, 892}, [3]float64{0, -13.8, -60.0}},
	},
	236: { // C6
		{5, fxTone, 0, -60.0, [3]int{233, 478, 583}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{233, 478, 583}, [3]float64{0, -0.0, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 980, 583}, [3]float64{0, -13.4, -0.1}},
		{20, fxTone, 0, -60.0, [3]int{239, 980, 583}, [3]float64{0, -11.9, -12.9}},
		{25, fxTone, 0, -60.0, [3]int{2012, 991, 595}, [3]float64{0, -14.7, -15.1}},
		{30, fxTone, 0, -22.1, [3]int{2029, 980, 997}, [3]float64{0, -10.5, -15.7}},
		{35, fxTone, 0, -13.4, [3]int{2041, 227, 595}, [3]float64{0, -7.5, -9.8}},
		{40, fxTone, 0, -13.0, [3]int{2029, 227, 1569}, [3]float64{0, -14.8, -17.3}},
		{55, fxTone, 0, -8.2, [3]int{2023, 227, 1569}, [3]float64{0, -3.1, -60.0}},
		{65, fxTone, 0, -4.2, [3]int{2023, 227, 1569}, [3]float64{0, -60.0, -60.0}},
		{70, fxTone, 0, -4.9, [3]int{2023, 1475, 1125}, [3]float64{0, -60.0, -60.0}},
		{75, fxTone, 0, -5.2, [3]int{2023, 1475, 1125}, [3]float64{0, -2.6, -60.0}},
		{80, fxTone, 0, -1.4, [3]int{2023, 1475, 1125}, [3]float64{0, -60.0, -60.0}},
		{90, fxTone, 0, -1.9, [3]int{1738, 1475, 1125}, [3]float64{0, -0.9, -1.5}},
		{100, fxTone, 0, -0.6, [3]int{1376, 1143, 1125}, [3]float64{0, -14.2, -14.8}},
		{110, fxTone, 0, 1.1, [3]int{1067, 1143, 2799}, [3]float64{0, -9.8, -2.4}},
		{130, fxTone, 0, -1.4, [3]int{904, 647, 641}, [3]float64{0, 0.0, 0.0}},
		{135, fxTone, 0, -2.3, [3]int{904, 1942, 641}, [3]float64{0, 2.7, 0.0}},
		{140, fxTone, 0, -2.8, [3]int{904, 1942, 910}, [3]float64{0, 1.8, -7.0}},
		{145, fxTone, 0, -1.9, [3]int{1930, 1942, 910}, [3]float64{0, -1.8, -6.2}},
		{150, fxTone, 0, -1.8, [3]int{1936, 1942, 1679}, [3]float64{0, -9.1, -13.1}},
		{160, fxTone, 0, -2.0, [3]int{1930, 898, 910}, [3]float64{0, -5.1, -8.5}},
		{165, fxTone, 0, -3.0, [3]int{906, 898, 647}, [3]float64{0, -1.4, -10.3}},
		{175, fxTone, 0, -1.8, [3]int{898, 898, 647}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, 0.7, [3]int{904, 641, 2782}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, 1.2, [3]int{904, 641, 2782}, [3]float64{0, -13.7, -5.0}},
		{200, fxTone, 0, -0.0, [3]int{1225, 1096, 2898}, [3]float64{0, -0.5, 7.4}},
		{205, fxTone, 0, -0.4, [3]int{1330, 1073, 1382}, [3]float64{0, -10.7, -12.7}},
		{240, fxTone, 0, -6.0, [3]int{2029, 1096, 1195}, [3]float64{0, -60.0, -60.0}},
		{245, fxTone, 0, -5.1, [3]int{2035, 338, 338}, [3]float64{0, -60.0, -60.0}},
		{265, fxTone, 0, -9.7, [3]int{2035, 338, 338}, [3]float64{0, -60.0, -60.0}},
		{280, fxTone, 0, -16.1, [3]int{2035, 571, 531}, [3]float64{0, -7.5, -11.0}},
		{285, fxTone, 0, -60.0, [3]int{239, 571, 531}, [3]float64{0, -8.0, -10.3}},
		{295, fxTone, 0, -60.0, [3]int{239, 764, 566}, [3]float64{0, -14.6, -0.1}},
		{300, fxTone, 0, -60.0, [3]int{239, 764, 566}, [3]float64{0, -10.4, -60.0}},
		{305, fxTone, 0, -60.0, [3]int{239, 478, 566}, [3]float64{0, -13.0, -60.0}},
	},
	237: { // C7
		{5, fxTone, 0, -60.0, [3]int{239, 472, 332}, [3]float64{0, -60.0, -60.0}},
		{10, fxTone, 0, -60.0, [3]int{239, 472, 332}, [3]float64{0, -12.2, -60.0}},
		{20, fxTone, 0, -60.0, [3]int{245, 472, 332}, [3]float64{0, -12.2, -60.0}},
		{25, fxTone, 0, -60.0, [3]int{490, 262, 332}, [3]float64{0, -4.9, -60.0}},
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
		{405, fxTone, 0, -60.0, [3]int{484, 227, 157}, [3]float64{0, -4.2, -60.0}},
		{410, fxTone, 0, -60.0, [3]int{239, 227, 157}, [3]float64{0, -13.8, -60.0}},
		{420, fxTone, 0, -60.0, [3]int{239, 227, 157}, [3]float64{0, -60.0, -60.0}},
	},
	252: { // M0
		{5, fxTone, 0, -60.0, [3]int{239, 986, 1627}, [3]float64{0, -13.9, -60.0}},
		{15, fxTone, 0, -60.0, [3]int{239, 991, 1627}, [3]float64{0, -14.8, -60.0}},
		{25, fxTone, 0, -2.4, [3]int{1755, 1609, 1627}, [3]float64{0, -11.4, -60.0}},
		{35, fxTone, 0, 1.1, [3]int{1790, 1609, 1627}, [3]float64{0, -12.9, -60.0}},
		{60, fxTone, 0, 0.8, [3]int{1814, 1574, 1627}, [3]float64{0, -60.0, -60.0}},
		{180, fxTone, 0, -2.1, [3]int{1819, 1598, 1627}, [3]float64{0, -60.0, -60.0}},
		{185, fxTone, 0, -2.1, [3]int{1819, 1598, 1627}, [3]float64{0, -1.0, -60.0}},
		{210, fxTone, 0, -2.9, [3]int{1784, 1574, 1627}, [3]float64{0, -0.9, -60.0}},
		{215, fxTone, 0, -3.1, [3]int{1784, 1574, 1627}, [3]float64{0, -60.0, -60.0}},
		{930, fxTone, 0, -24.3, [3]int{1796, 1580, 1627}, [3]float64{0, -14.7, -60.0}},
		{935, fxTone, 0, -60.0, [3]int{1796, 251, 1627}, [3]float64{0, -16.4, -60.0}},
		{950, fxTone, 0, -60.0, [3]int{1819, 222, 1627}, [3]float64{0, -15.6, -60.0}},
		{980, fxTone, 0, -60.0, [3]int{1773, 251, 1627}, [3]float64{0, -60.0, -60.0}},
		{985, fxTone, 0, -60.0, [3]int{1796, 251, 1627}, [3]float64{0, -11.5, -60.0}},
		{990, fxTone, 0, -60.0, [3]int{1825, 251, 1627}, [3]float64{0, -12.3, -0.9}},
		{995, fxTone, 0, -60.0, [3]int{1825, 1475, 1627}, [3]float64{0, -13.8, -13.4}},
		{1000, fxTone, 0, -60.0, [3]int{1825, 1475, 1627}, [3]float64{0, -11.2, -15.2}},
		{1005, fxTone, 0, -60.0, [3]int{1814, 251, 1627}, [3]float64{0, -14.1, -13.7}},
		{1010, fxTone, 0, -60.0, [3]int{1805, 245, 478}, [3]float64{0, -12.5, -14.4}},
		{1030, fxTone, 0, -60.0, [3]int{1800, 245, 478}, [3]float64{0, -60.0, -60.0}},
		{1035, fxTone, 0, -60.0, [3]int{1795, 245, 478}, [3]float64{0, -1.7, -60.0}},
		{1060, fxTone, 0, -60.0, [3]int{1808, 233, 583}, [3]float64{0, -10.6, -14.1}},
		{1075, fxTone, 0, -60.0, [3]int{1831, 227, 1679}, [3]float64{0, -11.5, -12.2}},
		{1085, fxTone, 0, -60.0, [3]int{1819, 513, 1679}, [3]float64{0, -13.5, -15.3}},
		{1110, fxTone, 0, -60.0, [3]int{1796, 292, 443}, [3]float64{0, -14.4, -16.5}},
		{1130, fxTone, 0, -60.0, [3]int{1779, 612, 840}, [3]float64{0, -8.9, -12.9}},
		{1135, fxBuzz, 128, -60.0, [3]int{957, 1781, 2639}, [3]float64{0, 26.7, 11.7}},
		{1160, fxBuzz, 128, -60.0, [3]int{1014, 1784, 2636}, [3]float64{0, 24.6, 10.0}},
		{1165, fxTone, 0, -60.0, [3]int{1790, 239, 1586}, [3]float64{0, -9.2, -13.0}},
		{1210, fxTone, 0, -60.0, [3]int{1802, 251, 461}, [3]float64{0, -4.6, -9.0}},
		{1215, fxTone, 0, -60.0, [3]int{239, 496, 443}, [3]float64{0, -5.8, -7.1}},
		{1220, fxTone, 0, -60.0, [3]int{239, 991, 443}, [3]float64{0, -12.5, -14.5}},
		{1230, fxTone, 0, -60.0, [3]int{233, 496, 397}, [3]float64{0, -2.2, -60.0}},
	},
	254: { // M2
		{5, fxBuzz, 128, -60.0, [3]int{540, 1687, 2856}, [3]float64{0, 37.6, 43.5}},
		{10, fxBuzz, 177, -60.0, [3]int{540, 1707, 2858}, [3]float64{0, 38.1, 44.6}},
		{15, fxBuzz, 498, -60.0, [3]int{540, 1707, 2866}, [3]float64{0, 4.2, 8.3}},
		{20, fxBuzz, 498, -11.0, [3]int{539, 1703, 2866}, [3]float64{0, 2.2, 6.6}},
		{25, fxBuzz, 85, -9.1, [3]int{530, 1703, 2866}, [3]float64{0, 2.2, 6.6}},
		{45, fxBuzz, 87, -4.3, [3]int{502, 1724, 2919}, [3]float64{0, 2.9, 7.1}},
		{155, fxBuzz, 87, -7.1, [3]int{735, 1954, 3166}, [3]float64{0, 3.2, 7.2}},
		{285, fxBuzz, 87, -6.6, [3]int{987, 2198, 3417}, [3]float64{0, 5.9, 9.2}},
		{410, fxBuzz, 87, -5.4, [3]int{714, 1934, 3145}, [3]float64{0, 2.8, 6.9}},
		{525, fxBuzz, 87, -6.4, [3]int{488, 1710, 2903}, [3]float64{0, 2.8, 7.5}},
		{530, fxBuzz, 103, -6.7, [3]int{488, 1710, 2903}, [3]float64{0, 2.5, 7.2}},
		{535, fxBuzz, 492, -13.6, [3]int{489, 1710, 2902}, [3]float64{0, 2.4, 7.2}},
		{540, fxBuzz, 83, -13.8, [3]int{489, 1710, 2902}, [3]float64{0, 2.7, 7.2}},
		{545, fxBuzz, 492, -22.2, [3]int{491, 1712, 2903}, [3]float64{0, 4.7, 9.4}},
		{550, fxBuzz, 492, -60.0, [3]int{491, 1710, 2901}, [3]float64{0, 5.4, 9.9}},
		{555, fxBuzz, 492, -60.0, [3]int{491, 1707, 2899}, [3]float64{0, 38.2, 43.8}},
	},
}

// the effects the chip gates: their tones pulse on and off at a steady rate,
// which the chip's envelope oscillator does. on the VoxRecord recording's third
// take, measured from each effect's envelope a millisecond at a time: C0 and C4
// pulse at the default speech pitch (85 and 87Hz), C2 at 66Hz, C3 at 59Hz, C6
// at 125Hz, on for a quarter to two fifths of each cycle (B9 pulses at 87Hz
// too, but plays closer to the recording ungated). C8 and C9 are gated noise
// (see cNoise). the
// envelope shape for each is the one that brought it closest to the recording.
// effects not listed aren't gated
type fxGate struct {
	hz   int
	wave envelopeWave
}

var fxGates = map[int]fxGate{
	230: {85, envSquare},
	232: {66, envSine},
	233: {59, envSaw},
	234: {87, envSine},
	236: {125, envSaw},
}

// the noise under some of the traced effects. on both units of the VoxRecord
// recording C0, C1, C2, C4, C6 and C7 have a broad noise under their tones
// that the traces, which take the strongest peaks as tones, leave out (C1 is as
// noisy as it is tonal). here oscillators 4 and 5 add it, at a level against
// the tones' and gated as deep as fitted, along with their frequencies and
// width, to both units at once. C0 70 to 78% and 2.7 to 2.3dB, C1 42 to 67% and
// 5.7 to 1.7dB, C2 74 to 79% and 5.1 to 3.3dB, C4 68 to 75% and 4.8 to 4.3dB,
// C6 66 to 72% and 3.8 to 2.7dB, C7 58 to 65% and 6.3 to 1.9dB. C3's made it
// match one unit worse, so it has none.
//
// the tone effects (see fx2Points) have it too: on both units a pure tone sits
// on a skirt of noise and scattered lines 15 to 30dB under it, across the band,
// 15 to 30dB over the recording's background, where these had clean sines with
// nothing. fitted the same way (depth 0: they aren't gated) and kept where
// neither unit's spectrogram match fell more than half a point: M0 67 to 97% and
// 5.6 to 0.5dB, A0 66 to 86% and 7.9 to 2.0dB, B6 6.1 to 0.7dB, R2 3.3 to
// 0.6dB, R9 5.5 to 1.8dB, A6 3.7 to 1.4dB. B7's made its spectrogram worse, as
// did R3's, R7's, R8's, A5's, B3's and B4's
type fxNoiseFloor struct {
	n1, n2, dist, depth, db float64
}

var fxNoiseFloors = map[int]fxNoiseFloor{
	230: {2361.6, 2282.7, 27.5, 0.371, -12.86},
	231: {1810.5, 2003.4, 147.2, 0.324, -1.82},
	232: {1935.8, 1748.1, 54.8, 0.287, -12.15},
	234: {1704.5, 3553.6, 228.3, 0.909, -9.64},
	236: {2357.3, 2947.3, 10.8, 0.670, -6.44},
	237: {442.9, 1765.9, 199.9, 0.832, -7.05},
	200: {600.9, 527.1, 135.1, 0, -17.96},
	201: {1143.9, 3430.0, 31.9, 0, -10.65},
	202: {1162.4, 1103.3, 73.7, 0, -18.28},
	204: {1437.6, 1456.3, 28.0, 0, -18.56},
	205: {2417.5, 2751.8, 44.1, 0, -15.10},
	206: {1735.5, 2187.7, 10.6, 0, -3.58},
	209: {341.9, 838.6, 254.3, 0, -17.03},
	210: {419.0, 533.5, 158.7, 0, -8.41},
	214: {2232.5, 3476.6, 176.3, 0, -13.45},
	216: {2026.9, 2048.9, 0.2, 0, -13.53},
	217: {2308.5, 1467.2, 205.5, 0, -23.03},
	218: {478.3, 2534.5, 114.5, 0, -26.93},
	219: {2499.3, 2824.7, 50.3, 0, -18.69},
	220: {686.8, 3002.8, 237.0, 0, -35.74},
	221: {1303.1, 2945.7, 163.1, 0, -27.86},
	222: {625.0, 537.8, 121.2, 0, -38.33},
	225: {388.1, 2892.3, 75.7, 0, -32.91},
	226: {2473.9, 2831.0, 65.7, 0, -6.71},
	228: {1613.6, 3166.5, 19.9, 0, -38.21},
	235: {774.1, 3426.3, 192.9, 0, -22.45},
	252: {1312.8, 2050.9, 46.6, 0, -9.72},
}

// toneNoise adds the noise under a tone effect's tones (see fxNoiseFloors), on
// oscillators 4 and 5 where the effect has no tone on them
func toneNoise(s sound, code int) sound {
	n, ok := fxNoiseFloors[code]
	if !ok {
		return s
	}
	for i := range s.parts {
		f := &s.parts[i].target
		if f.vol[3] != 0 || f.vol[4] != 0 {
			continue
		}
		f.freq[3] = n.n1
		f.freq[4] = n.n2
		f.distortion = n.dist
		f.vol[3] = 0.5 * (f.vol[0] + f.vol[1] + f.vol[2]) * math.Pow(10, n.db/20)
		f.vol[4] = f.vol[3]
	}
	return s
}

func fxFrame(p fxPoint, code int) frame {
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
			f.wave = envVoice
			f.voiced = true
		} else if gt, ok := fxGates[code]; ok {
			f.pitch = float64(gt.hz)
			f.wave = gt.wave
			f.voiced = true
		}
	}
	if n, ok := fxNoiseFloors[code]; ok {
		v := f.vol[0] + f.vol[1] + f.vol[2]
		if p.kind == fxNoise {
			v = f.vol[3] + f.vol[4]
		}
		f.freq[3] = n.n1
		f.freq[4] = n.n2
		f.distortion = n.dist
		f.vol[3] = 0.5 * v * math.Pow(10, n.db/20)
		f.vol[4] = f.vol[3]
		f.noiseDepth = 0
		if f.pitch > 0 {
			f.noiseDepth = n.depth
		}
	}
	if p.db <= -60 {
		return quiet(f)
	}
	return f
}

// C8 and C9 are noise from about 400Hz to 2.8kHz, flat within a few dB, the
// same on both units of the VoxRecord recording and nearly the same as each
// other, gated at 52Hz 13 to 16dB deep. their traces had taken the noise's
// strongest peaks for three tones, falling away above 1.5kHz. here they are
// oscillators 4 and 5 spread wide, gated by a saw (see noiseDepth), every 5ms
// at the level both units have (the mean of the two, in dB against the
// loudest); the frequencies, width and depth are fitted to both units at once,
// and the depth comes out at 0.83 (-15dB), as measured. C8 and C9 34% to 58%
// and 8.4 to 2.9dB
var c8DB = []float64{
	-16.2, -11.8, -9.5, -5.5, -1.1, -0.5, -0.4, -0.3, -0.1, -0.3,
	-0.2, -0.1, -0.3, -0.5, -0.5, -0.2, -0.9, -1.3, -1.8, -5.2,
}

var c9DB = []float64{
	-18.9, -13.6, -11.4, -7.8, -3.1, -1.8, -1.6, -1.1, -0.8,
	-0.7, -0.5, -0.0, -0.5, -1.2, -1.8, -4.7, -16.4,
}

func cNoise(name string, db []float64) sound {
	f := noiseFrame(lGlottal, 1316.3, 1855.6, 0, 0, 203.2)
	f.pitch = 52
	f.wave = envSaw
	f.noiseDepth = 0.826
	s := effect(name)
	for i, d := range db {
		f.vol[3] = 20 * math.Pow(10, d/20)
		f.vol[4] = f.vol[3]
		glide := 5.0
		if i == 0 {
			glide = 2
		}
		s.parts = append(s.parts, hold(5, glide, f)...)
	}
	f.vol[3], f.vol[4] = 0, 0
	s.parts = append(s.parts, hold(3, 3, f)...)
	return s
}

// contour plays a traced effect: a quick start on the first point, then a
// glide to each of the others in turn, then 3ms to silence
func contour(name string, p []fxPoint, code int) sound {
	s := effect(name, hold(float64(p[0].ms), 2, fxFrame(p[0], code)))
	for i := 1; i < len(p); i++ {
		ms := float64(p[i].ms - p[i-1].ms)
		s.parts = append(s.parts, hold(ms, ms, fxFrame(p[i], code))...)
	}
	s.parts = append(s.parts, hold(3, 3, quiet(fxFrame(p[len(p)-1], code)))...)
	return s
}

// FXV-BEGIN
// the sound effects that are a buzz, voiced at the default speech pitch with
// formants that move, played as the chip plays them, pulse by pulse: measured
// the same way as the allophones played from the chip's frames (see vfPoints),
// on both units. each point is a pulse: its time after the one before (glided
// to in a straight line), the three formants and their volumes. A2 is played
// this way; A1, A3 and M2 measured this way came out further from the
// recordings (A1 is more than three voiced formants after its first 300ms), and
// keep their traces
type fxvPoint struct {
	ms   float64
	freq formants
	vol  [3]float64
}

// the buzz effects played from the chip's frames, by code
var fxvPoints = map[int][]fxvPoint{
	212: { // A2
		{1.00, formants{262, 1651, 1422}, [3]float64{5.46, 9.84, 3.08}},
		{125.61, formants{260, 1549, 2556}, [3]float64{5.65, 9.24, 9.35}},
		{22.95, formants{235, 1533, 2488}, [3]float64{6.63, 9.81, 8.93}},
		{24.34, formants{226, 1523, 2417}, [3]float64{7.24, 12.13, 10.91}},
		{23.09, formants{679, 1509, 2347}, [3]float64{4.85, 15.62, 14.52}},
		{11.52, formants{219, 1500, 2311}, [3]float64{7.76, 16.22, 14.07}},
		{11.48, formants{227, 1494, 2265}, [3]float64{6.96, 15.86, 15.20}},
		{11.50, formants{224, 1486, 2262}, [3]float64{7.05, 15.82, 15.94}},
		{11.52, formants{215, 1468, 2226}, [3]float64{7.11, 15.58, 15.38}},
		{11.53, formants{209, 1458, 2209}, [3]float64{7.23, 15.80, 13.78}},
		{11.45, formants{209, 1444, 2193}, [3]float64{6.94, 15.75, 12.62}},
		{11.45, formants{189, 1426, 2102}, [3]float64{7.67, 15.86, 13.59}},
		{11.66, formants{197, 1434, 2015}, [3]float64{8.29, 16.90, 13.79}},
		{11.50, formants{255, 1420, 2216}, [3]float64{6.54, 17.48, 8.01}},
		{11.46, formants{211, 1405, 1940}, [3]float64{9.34, 17.45, 15.49}},
		{11.43, formants{208, 1402, 1918}, [3]float64{10.11, 17.49, 15.89}},
		{11.42, formants{198, 1395, 1883}, [3]float64{10.80, 17.58, 15.01}},
		{11.78, formants{204, 1395, 1803}, [3]float64{9.51, 18.15, 16.25}},
		{11.46, formants{209, 1375, 1783}, [3]float64{8.66, 17.79, 16.84}},
		{11.44, formants{236, 1352, 1772}, [3]float64{6.21, 17.09, 16.15}},
		{11.44, formants{230, 1340, 1734}, [3]float64{6.65, 16.83, 16.72}},
		{11.45, formants{224, 1328, 1674}, [3]float64{6.81, 17.20, 16.27}},
		{11.46, formants{252, 1332, 1615}, [3]float64{5.75, 17.88, 16.68}},
		{11.45, formants{239, 1305, 1598}, [3]float64{5.87, 17.50, 16.52}},
		{11.44, formants{250, 1285, 1562}, [3]float64{5.76, 17.22, 16.06}},
		{11.47, formants{679, 1265, 1514}, [3]float64{5.10, 16.54, 15.33}},
		{11.45, formants{679, 1279, 1469}, [3]float64{4.86, 17.50, 17.17}},
		{11.41, formants{258, 1257, 1422}, [3]float64{5.25, 18.15, 18.08}},
		{11.50, formants{262, 1315, 1999}, [3]float64{6.11, 17.10, 9.94}},
		{11.45, formants{282, 1227, 1977}, [3]float64{5.22, 16.80, 9.15}},
		{11.44, formants{282, 1264, 2558}, [3]float64{5.28, 23.19, 5.94}},
		{11.42, formants{272, 1245, 2488}, [3]float64{5.50, 26.77, 6.38}},
		{11.43, formants{270, 1220, 2578}, [3]float64{5.71, 29.62, 7.58}},
		{11.44, formants{275, 1193, 2581}, [3]float64{5.23, 31.00, 5.37}},
		{11.46, formants{269, 1174, 2578}, [3]float64{5.50, 31.00, 6.95}},
		{11.38, formants{679, 1143, 1422}, [3]float64{4.63, 31.00, 3.15}},
		{11.20, formants{250, 1118, 2492}, [3]float64{6.85, 31.00, 5.91}},
		{11.50, formants{242, 1092, 1422}, [3]float64{6.88, 31.00, 4.31}},
		{11.39, formants{192, 1064, 1422}, [3]float64{11.19, 31.00, 1.65}},
		{11.46, formants{183, 1033, 1422}, [3]float64{11.78, 31.00, 3.34}},
		{11.29, formants{679, 1018, 1422}, [3]float64{3.22, 28.65, 2.44}},
		{11.59, formants{911, 1078, 1422}, [3]float64{17.97, 16.82, 2.81}},
		{11.97, formants{881, 1054, 1422}, [3]float64{18.24, 17.03, 2.42}},
		{10.87, formants{850, 1040, 1422}, [3]float64{19.04, 17.42, 2.80}},
		{12.09, formants{810, 1031, 1422}, [3]float64{18.82, 16.88, 1.58}},
		{11.45, formants{781, 1013, 1422}, [3]float64{19.69, 17.52, 2.05}},
		{10.74, formants{740, 1001, 1422}, [3]float64{19.92, 17.81, 3.16}},
		{11.46, formants{715, 980, 1422}, [3]float64{20.36, 18.38, 2.57}},
		{11.45, formants{679, 968, 1422}, [3]float64{20.58, 18.73, 2.70}},
		{14.17, formants{672, 915, 2536}, [3]float64{16.48, 15.21, 4.63}},
		{10.75, formants{630, 912, 2600}, [3]float64{12.57, 11.88, 4.62}},
		{11.58, formants{601, 918, 2495}, [3]float64{14.14, 13.33, 4.72}},
		{11.54, formants{549, 911, 2445}, [3]float64{14.24, 13.27, 6.28}},
		{11.52, formants{535, 924, 2445}, [3]float64{14.22, 13.99, 8.79}},
		{11.52, formants{517, 921, 2432}, [3]float64{13.84, 13.86, 7.64}},
		{11.42, formants{396, 877, 1422}, [3]float64{15.13, 13.10, 2.09}},
		{10.82, formants{409, 845, 1422}, [3]float64{15.81, 14.73, 2.60}},
		{11.52, formants{386, 841, 1422}, [3]float64{14.56, 15.01, 3.18}},
		{11.57, formants{366, 831, 2576}, [3]float64{12.89, 14.69, 4.58}},
		{11.54, formants{313, 831, 1422}, [3]float64{12.18, 14.11, 3.99}},
		{11.50, formants{215, 797, 1422}, [3]float64{19.92, 14.28, 3.13}},
		{10.75, formants{200, 803, 1422}, [3]float64{19.90, 13.82, 2.59}},
		{11.59, formants{186, 786, 1422}, [3]float64{17.65, 12.75, 1.26}},
		{11.50, formants{189, 769, 1422}, [3]float64{12.85, 12.85, 2.06}},
		{45.73, formants{206, 711, 1422}, [3]float64{9.25, 16.00, 2.64}},
		{11.41, formants{220, 714, 1422}, [3]float64{6.82, 14.01, 2.82}},
		{12.17, formants{679, 1143, 1422}, [3]float64{3.07, 7.34, 0.90}},
		{11.50, formants{679, 1143, 1422}, [3]float64{3.07, 7.34, 0.90}},
	},
}

// contourVoiced plays a buzz effect from the chip's frames, voiced at the
// default pitch (which the Pitch control scales)
func contourVoiced(name string, p []fxvPoint) sound {
	s := effect(name)
	var f frame
	for _, q := range p {
		f = voicedFrame(q.freq, q.vol[0], q.vol[1], q.vol[2])
		f.pitch = voiceHz(defaultPitch)
		s.parts = append(s.parts, hold(q.ms, q.ms, f)...)
	}
	s.parts = append(s.parts, hold(3, 3, quiet(f))...)
	return s
}

// FXV-END
// FX2-BEGIN
// the sound effects that are pure tones, traced again from both units'
// VoxRecord recordings: the tones were tracked every millisecond (12ms
// windows), each tone followed while it lasts and split where its level jumps,
// the two units' tracks paired and averaged (and a strong one heard on only one
// unit kept), and each reduced to the points where it bends (10Hz, 2dB). each
// tone gets an oscillator of its own, which takes its frequency while silent
// before the tone starts, so tones start and stop in place rather than sweeping
// in from another frequency as the first traces did. the levels are the
// recording's with the output filter's loss added back, scaled so that the DTMF
// tones play at their measured level. effects whose recordings are mostly not
// tones (buzzes and noise) keep fxPoints. each point is its time after the one
// before (glided to in a straight line), the five oscillators' frequencies and
// their volumes. A4's trace stopped at 1315Hz, where its volume reached the
// oscillators' limit; on both units it goes on, faster, to 3.9kHz by 190ms,
// holds there and dies away by 245ms. from 1083Hz on it is tracked again every
// 4ms (the two units within 50Hz and 0.5dB of each other) with this library's
// output filter loss added back, against its level at 1083Hz. the volumes had
// been cut off at the oscillators' limit (31), which flattened the loudest part
// of 30 effects (A0's swell, B6's and B7's fall, the DTMF tones' whole body);
// each effect is now scaled so that its loudest point is at the limit instead,
// which keeps its shape below it: effects 82.9 to 83.5% and 2.91 to 2.45dB, B2
// 4.9 to 0.6dB, R0 3.1 to 1.0dB. A4, M0 and R6 keep their cut-off volumes,
// which matched both units better
type fx2Point struct {
	ms   float64
	freq [5]float64
	vol  [5]float64
}

// the tone effects traced again, by code
var fx2Points = map[int][]fx2Point{
	200: { // R0
		{1.0, [5]float64{794, 409, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{794, 409, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{794, 409, 0, 0, 0}, [5]float64{2.52, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{798, 409, 0, 0, 0}, [5]float64{16.51, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{803, 409, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{803, 409, 0, 0, 0}, [5]float64{30.98, 4.08, 0.00, 0.00, 0.00}},
		{11.0, [5]float64{803, 450, 0, 0, 0}, [5]float64{30.78, 7.00, 0.00, 0.00, 0.00}},
		{24.0, [5]float64{802, 539, 0, 0, 0}, [5]float64{30.33, 8.44, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{802, 570, 0, 0, 0}, [5]float64{30.24, 10.56, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{802, 570, 0, 0, 0}, [5]float64{30.22, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{802, 570, 0, 0, 0}, [5]float64{30.11, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{804, 570, 0, 0, 0}, [5]float64{19.17, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{809, 570, 0, 0, 0}, [5]float64{6.41, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{809, 570, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{809, 570, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	201: { // R1
		{1.0, [5]float64{635, 1105, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{635, 1105, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{635, 1105, 0, 0, 0}, [5]float64{0.37, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{681, 1105, 0, 0, 0}, [5]float64{0.79, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{681, 1105, 0, 0, 0}, [5]float64{1.01, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{681, 1105, 0, 0, 0}, [5]float64{1.31, 2.09, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{681, 1105, 0, 0, 0}, [5]float64{1.69, 3.31, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{685, 1105, 0, 0, 0}, [5]float64{1.72, 13.19, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{686, 1105, 0, 0, 0}, [5]float64{1.74, 15.42, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{660, 1105, 0, 0, 0}, [5]float64{2.47, 28.86, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{699, 1105, 0, 0, 0}, [5]float64{3.24, 29.03, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{698, 1105, 0, 0, 0}, [5]float64{3.47, 29.37, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{733, 1105, 0, 0, 0}, [5]float64{3.41, 29.67, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{735, 1105, 0, 0, 0}, [5]float64{3.35, 30.33, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{776, 1105, 0, 0, 0}, [5]float64{3.42, 30.69, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{766, 1105, 0, 0, 0}, [5]float64{3.42, 30.73, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{777, 1105, 0, 0, 0}, [5]float64{3.40, 30.82, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{765, 1105, 0, 0, 0}, [5]float64{3.35, 30.86, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{766, 1105, 0, 0, 0}, [5]float64{3.31, 31.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{767, 1105, 0, 0, 0}, [5]float64{3.27, 27.70, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{715, 1105, 0, 0, 0}, [5]float64{2.66, 24.75, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{710, 1104, 0, 0, 0}, [5]float64{2.55, 23.40, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{701, 1105, 0, 0, 0}, [5]float64{2.33, 16.96, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{733, 1107, 0, 0, 0}, [5]float64{2.51, 10.46, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{744, 1107, 0, 0, 0}, [5]float64{2.57, 7.34, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{768, 1107, 0, 0, 0}, [5]float64{1.45, 3.62, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{768, 1107, 0, 0, 0}, [5]float64{0.00, 2.54, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{768, 1107, 0, 0, 0}, [5]float64{0.00, 0.37, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{768, 1107, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{768, 1107, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	202: { // R2
		{1.0, [5]float64{1300, 733, 1697, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1300, 733, 1697, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1300, 733, 1697, 0, 0}, [5]float64{0.18, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1300, 733, 1697, 0, 0}, [5]float64{1.01, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1300, 733, 1697, 0, 0}, [5]float64{2.44, 1.03, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1300, 653, 1697, 0, 0}, [5]float64{4.18, 0.75, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1300, 655, 1697, 0, 0}, [5]float64{7.15, 1.01, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1301, 659, 1697, 0, 0}, [5]float64{18.65, 2.41, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1303, 678, 1697, 0, 0}, [5]float64{27.16, 2.89, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1304, 685, 1697, 0, 0}, [5]float64{30.78, 3.08, 3.67, 0.00, 0.00}},
		{4.0, [5]float64{1304, 662, 1667, 0, 0}, [5]float64{30.80, 3.11, 5.18, 0.00, 0.00}},
		{4.0, [5]float64{1304, 639, 1606, 0, 0}, [5]float64{30.81, 3.14, 6.90, 0.00, 0.00}},
		{1.0, [5]float64{1304, 633, 1606, 0, 0}, [5]float64{30.82, 3.15, 0.00, 0.00, 0.00}},
		{42.0, [5]float64{1304, 392, 1606, 0, 0}, [5]float64{31.00, 3.52, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1303, 375, 1606, 0, 0}, [5]float64{24.71, 3.55, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1302, 376, 1606, 0, 0}, [5]float64{21.24, 3.34, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1303, 377, 1606, 0, 0}, [5]float64{17.46, 3.24, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1304, 415, 1606, 0, 0}, [5]float64{11.80, 2.59, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1305, 418, 1606, 0, 0}, [5]float64{9.70, 2.39, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1306, 386, 1606, 0, 0}, [5]float64{5.39, 2.52, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1305, 376, 1606, 0, 0}, [5]float64{7.11, 2.57, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1305, 388, 1606, 0, 0}, [5]float64{0.00, 2.45, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1305, 412, 1606, 0, 0}, [5]float64{0.00, 2.23, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1305, 421, 1606, 0, 0}, [5]float64{0.00, 0.24, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1305, 420, 1606, 0, 0}, [5]float64{0.00, 0.60, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1305, 420, 1606, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1305, 420, 1606, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	203: { // R3
		{1.0, [5]float64{1605, 636, 345, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1605, 636, 345, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1605, 636, 345, 0, 0}, [5]float64{0.95, 0.87, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1605, 636, 345, 0, 0}, [5]float64{4.51, 2.96, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1604, 645, 345, 0, 0}, [5]float64{13.51, 3.97, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1605, 653, 345, 0, 0}, [5]float64{24.38, 5.32, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1606, 662, 345, 0, 0}, [5]float64{29.19, 7.12, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1607, 670, 345, 0, 0}, [5]float64{31.00, 7.18, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1607, 678, 345, 0, 0}, [5]float64{30.94, 7.24, 1.21, 0.00, 0.00}},
		{1.0, [5]float64{1607, 686, 360, 0, 0}, [5]float64{30.88, 7.30, 1.70, 0.00, 0.00}},
		{1.0, [5]float64{1607, 694, 357, 0, 0}, [5]float64{30.83, 7.36, 1.70, 0.00, 0.00}},
		{4.0, [5]float64{1607, 697, 345, 0, 0}, [5]float64{30.60, 7.44, 1.69, 0.00, 0.00}},
		{4.0, [5]float64{1607, 699, 351, 0, 0}, [5]float64{30.37, 7.52, 2.28, 0.00, 0.00}},
		{6.0, [5]float64{1607, 728, 360, 0, 0}, [5]float64{30.03, 6.62, 3.54, 0.00, 0.00}},
		{1.0, [5]float64{1606, 729, 361, 0, 0}, [5]float64{29.97, 6.43, 3.81, 0.00, 0.00}},
		{15.0, [5]float64{1606, 737, 364, 0, 0}, [5]float64{29.14, 4.21, 4.08, 0.00, 0.00}},
		{7.0, [5]float64{1606, 771, 365, 0, 0}, [5]float64{28.76, 3.20, 4.21, 0.00, 0.00}},
		{8.0, [5]float64{1606, 769, 367, 0, 0}, [5]float64{28.34, 3.57, 4.36, 0.00, 0.00}},
		{1.0, [5]float64{1606, 758, 357, 0, 0}, [5]float64{28.28, 3.49, 4.14, 0.00, 0.00}},
		{4.0, [5]float64{1606, 717, 318, 0, 0}, [5]float64{17.71, 3.18, 3.36, 0.00, 0.00}},
		{3.0, [5]float64{1607, 726, 296, 0, 0}, [5]float64{12.47, 3.04, 1.77, 0.00, 0.00}},
		{1.0, [5]float64{1607, 728, 296, 0, 0}, [5]float64{11.09, 3.00, 1.43, 0.00, 0.00}},
		{1.0, [5]float64{1607, 731, 296, 0, 0}, [5]float64{8.30, 2.95, 1.16, 0.00, 0.00}},
		{1.0, [5]float64{1607, 731, 296, 0, 0}, [5]float64{6.22, 2.53, 0.94, 0.00, 0.00}},
		{1.0, [5]float64{1607, 731, 296, 0, 0}, [5]float64{4.65, 2.17, 0.88, 0.00, 0.00}},
		{3.0, [5]float64{1607, 731, 298, 0, 0}, [5]float64{1.09, 1.36, 0.71, 0.00, 0.00}},
		{1.0, [5]float64{1606, 731, 298, 0, 0}, [5]float64{1.35, 1.17, 0.66, 0.00, 0.00}},
		{1.0, [5]float64{1606, 731, 299, 0, 0}, [5]float64{0.00, 1.15, 0.62, 0.00, 0.00}},
		{1.0, [5]float64{1606, 731, 299, 0, 0}, [5]float64{0.00, 1.14, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1606, 731, 299, 0, 0}, [5]float64{0.00, 1.08, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1606, 731, 299, 0, 0}, [5]float64{0.00, 0.35, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1606, 731, 299, 0, 0}, [5]float64{0.00, 0.19, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1606, 706, 299, 0, 0}, [5]float64{0.00, 0.07, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1606, 706, 299, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1606, 706, 299, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	204: { // R4
		{1.0, [5]float64{1901, 629, 462, 282, 1476}, [5]float64{0.13, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1901, 629, 462, 282, 1476}, [5]float64{0.75, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1901, 629, 462, 282, 1476}, [5]float64{1.83, 0.59, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1901, 608, 462, 282, 1476}, [5]float64{3.34, 0.55, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1901, 603, 462, 282, 1476}, [5]float64{6.08, 0.85, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1904, 588, 462, 282, 1476}, [5]float64{15.30, 3.12, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1905, 572, 462, 282, 1476}, [5]float64{20.81, 3.49, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1905, 555, 462, 282, 1476}, [5]float64{22.99, 3.90, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1906, 549, 462, 282, 1476}, [5]float64{31.00, 4.25, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1906, 629, 462, 282, 1476}, [5]float64{30.89, 4.75, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1907, 635, 462, 282, 1476}, [5]float64{30.82, 4.69, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1907, 671, 462, 282, 1476}, [5]float64{30.75, 4.21, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1907, 682, 462, 282, 1476}, [5]float64{30.71, 4.10, 5.21, 0.00, 0.00}},
		{1.0, [5]float64{1907, 693, 463, 282, 1476}, [5]float64{30.67, 4.00, 4.02, 0.00, 0.00}},
		{7.0, [5]float64{1907, 770, 505, 282, 1476}, [5]float64{30.42, 3.33, 4.32, 0.00, 0.00}},
		{3.0, [5]float64{1907, 790, 522, 282, 1476}, [5]float64{30.32, 3.18, 4.46, 0.00, 0.00}},
		{1.0, [5]float64{1907, 797, 531, 282, 1476}, [5]float64{30.28, 3.13, 4.36, 1.97, 0.00}},
		{1.0, [5]float64{1907, 804, 540, 281, 1476}, [5]float64{30.25, 3.08, 4.25, 2.05, 0.00}},
		{2.0, [5]float64{1907, 833, 558, 279, 1476}, [5]float64{30.17, 2.58, 4.04, 2.22, 0.00}},
		{1.0, [5]float64{1907, 848, 567, 282, 1476}, [5]float64{30.14, 2.37, 3.94, 2.41, 1.77}},
		{1.0, [5]float64{1907, 863, 576, 286, 1472}, [5]float64{30.10, 2.17, 3.85, 2.62, 1.75}},
		{1.0, [5]float64{1907, 863, 585, 289, 1468}, [5]float64{30.07, 2.07, 3.75, 2.84, 1.74}},
		{4.0, [5]float64{1907, 863, 621, 304, 1514}, [5]float64{29.93, 1.71, 3.40, 3.94, 2.59}},
		{1.0, [5]float64{1907, 863, 630, 307, 1535}, [5]float64{29.89, 1.63, 3.32, 4.27, 2.70}},
		{2.0, [5]float64{1907, 863, 641, 314, 1578}, [5]float64{29.82, 1.48, 3.30, 5.03, 2.94}},
		{1.0, [5]float64{1907, 863, 646, 318, 1581}, [5]float64{29.79, 1.41, 3.30, 5.00, 2.80}},
		{1.0, [5]float64{1907, 863, 651, 321, 1618}, [5]float64{29.75, 1.35, 3.29, 4.96, 2.86}},
		{1.0, [5]float64{1907, 863, 657, 324, 1634}, [5]float64{29.72, 1.28, 3.28, 4.93, 2.98}},
		{1.0, [5]float64{1907, 863, 662, 327, 1634}, [5]float64{29.68, 1.22, 3.27, 4.90, 0.00}},
		{3.0, [5]float64{1907, 863, 678, 337, 1634}, [5]float64{29.58, 1.06, 3.25, 4.80, 0.00}},
		{1.0, [5]float64{1907, 861, 683, 340, 1634}, [5]float64{29.54, 0.87, 3.24, 4.77, 0.00}},
		{3.0, [5]float64{1907, 857, 710, 350, 1634}, [5]float64{29.44, 0.48, 3.50, 4.68, 0.00}},
		{1.0, [5]float64{1907, 857, 719, 354, 1634}, [5]float64{29.41, 0.00, 3.58, 4.64, 0.00}},
		{4.0, [5]float64{1907, 857, 734, 367, 1634}, [5]float64{29.27, 0.00, 3.87, 4.52, 0.00}},
		{1.0, [5]float64{1907, 857, 737, 366, 1634}, [5]float64{29.23, 0.00, 3.94, 4.43, 0.00}},
		{1.0, [5]float64{1907, 857, 729, 366, 1634}, [5]float64{29.20, 0.00, 3.70, 4.34, 0.00}},
		{3.0, [5]float64{1907, 857, 702, 365, 1634}, [5]float64{20.17, 0.00, 3.07, 4.08, 0.00}},
		{1.0, [5]float64{1907, 857, 709, 373, 1634}, [5]float64{17.83, 0.00, 3.03, 3.55, 0.00}},
		{1.0, [5]float64{1907, 857, 715, 441, 1634}, [5]float64{15.76, 0.00, 3.00, 3.73, 0.00}},
		{1.0, [5]float64{1907, 857, 722, 441, 1634}, [5]float64{13.94, 0.00, 2.96, 0.00, 0.00}},
		{1.0, [5]float64{1907, 857, 728, 441, 1634}, [5]float64{12.32, 0.00, 2.92, 0.00, 0.00}},
		{1.0, [5]float64{1906, 857, 728, 441, 1634}, [5]float64{10.89, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1905, 857, 728, 441, 1634}, [5]float64{1.30, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1905, 857, 728, 441, 1634}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1905, 857, 728, 441, 1634}, [5]float64{0, 0, 0, 0, 0}},
	},
	205: { // R5
		{1.0, [5]float64{347, 2206, 810, 0, 0}, [5]float64{0.12, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{347, 2206, 810, 0, 0}, [5]float64{0.19, 0.12, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{347, 2206, 810, 0, 0}, [5]float64{0.30, 0.43, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{347, 2206, 810, 0, 0}, [5]float64{0.75, 2.36, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{347, 2206, 810, 0, 0}, [5]float64{1.85, 6.25, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{334, 2206, 810, 0, 0}, [5]float64{2.07, 10.17, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{321, 2206, 810, 0, 0}, [5]float64{2.31, 12.71, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{307, 2207, 810, 0, 0}, [5]float64{2.58, 15.89, 3.46, 0.00, 0.00}},
		{2.0, [5]float64{302, 2207, 813, 0, 0}, [5]float64{2.91, 24.83, 3.25, 0.00, 0.00}},
		{1.0, [5]float64{310, 2208, 804, 0, 0}, [5]float64{3.06, 25.95, 3.41, 0.00, 0.00}},
		{2.0, [5]float64{359, 2208, 786, 0, 0}, [5]float64{3.72, 28.36, 3.76, 0.00, 0.00}},
		{2.0, [5]float64{379, 2208, 769, 0, 0}, [5]float64{4.26, 31.00, 4.14, 0.00, 0.00}},
		{45.0, [5]float64{381, 2208, 768, 0, 0}, [5]float64{4.19, 25.60, 4.24, 0.00, 0.00}},
		{2.0, [5]float64{382, 2208, 757, 0, 0}, [5]float64{4.19, 25.38, 4.16, 0.00, 0.00}},
		{3.0, [5]float64{362, 2208, 740, 0, 0}, [5]float64{4.07, 25.06, 4.04, 0.00, 0.00}},
		{1.0, [5]float64{356, 2208, 812, 0, 0}, [5]float64{4.03, 21.18, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{349, 2208, 812, 0, 0}, [5]float64{3.99, 17.90, 1.71, 0.00, 0.00}},
		{1.0, [5]float64{361, 2208, 833, 0, 0}, [5]float64{4.02, 15.13, 2.11, 0.00, 0.00}},
		{2.0, [5]float64{384, 2207, 831, 0, 0}, [5]float64{4.07, 10.81, 2.12, 0.00, 0.00}},
		{3.0, [5]float64{419, 2207, 798, 0, 0}, [5]float64{4.16, 6.52, 2.17, 0.00, 0.00}},
		{1.0, [5]float64{431, 2207, 787, 0, 0}, [5]float64{4.19, 4.20, 2.18, 0.00, 0.00}},
		{1.0, [5]float64{446, 2207, 776, 0, 0}, [5]float64{3.51, 2.71, 2.20, 0.00, 0.00}},
		{1.0, [5]float64{462, 2207, 756, 0, 0}, [5]float64{2.93, 1.74, 1.96, 0.00, 0.00}},
		{1.0, [5]float64{477, 2208, 736, 0, 0}, [5]float64{2.46, 0.97, 1.75, 0.00, 0.00}},
		{1.0, [5]float64{492, 2208, 736, 0, 0}, [5]float64{2.06, 0.54, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{492, 2209, 736, 0, 0}, [5]float64{1.34, 0.17, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{492, 2209, 736, 0, 0}, [5]float64{1.08, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{492, 2209, 736, 0, 0}, [5]float64{0.88, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{496, 2209, 736, 0, 0}, [5]float64{1.03, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{496, 2209, 736, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{496, 2209, 736, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	206: { // R6
		{1.0, [5]float64{1668, 477, 2511, 760, 878}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1668, 477, 2511, 760, 878}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1668, 477, 2511, 760, 878}, [5]float64{0.26, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1668, 477, 2511, 760, 878}, [5]float64{0.58, 1.39, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1668, 475, 2511, 760, 878}, [5]float64{1.27, 2.00, 0.75, 0.00, 0.00}},
		{2.0, [5]float64{1654, 471, 2511, 760, 878}, [5]float64{2.21, 4.14, 3.26, 0.00, 0.00}},
		{1.0, [5]float64{1646, 469, 2511, 760, 878}, [5]float64{2.91, 4.40, 5.29, 0.00, 0.00}},
		{2.0, [5]float64{1597, 464, 2511, 760, 878}, [5]float64{2.79, 4.96, 13.94, 0.00, 0.00}},
		{1.0, [5]float64{1573, 462, 2510, 760, 878}, [5]float64{2.74, 5.26, 18.16, 0.00, 0.00}},
		{1.0, [5]float64{1548, 434, 2510, 760, 878}, [5]float64{2.68, 5.42, 23.67, 0.00, 0.00}},
		{1.0, [5]float64{1489, 405, 2510, 760, 878}, [5]float64{2.17, 5.60, 30.84, 0.00, 0.00}},
		{1.0, [5]float64{1707, 377, 2509, 760, 878}, [5]float64{0.00, 5.78, 31.00, 3.20, 0.00}},
		{1.0, [5]float64{1707, 367, 2509, 713, 878}, [5]float64{0.00, 6.33, 31.00, 4.66, 0.00}},
		{1.0, [5]float64{1707, 358, 2509, 719, 878}, [5]float64{0.00, 6.95, 31.00, 4.95, 0.00}},
		{3.0, [5]float64{1707, 366, 2509, 739, 878}, [5]float64{0.00, 7.07, 31.00, 5.92, 0.00}},
		{3.0, [5]float64{1707, 374, 2509, 735, 878}, [5]float64{0.00, 7.20, 31.00, 6.05, 0.00}},
		{3.0, [5]float64{1707, 363, 2509, 712, 878}, [5]float64{0.00, 7.11, 31.00, 6.06, 0.00}},
		{1.0, [5]float64{1707, 359, 2509, 704, 878}, [5]float64{2.00, 7.08, 31.00, 6.07, 1.06}},
		{3.0, [5]float64{1691, 348, 2509, 682, 877}, [5]float64{3.38, 6.99, 31.00, 6.08, 1.29}},
		{5.0, [5]float64{1643, 329, 2509, 644, 874}, [5]float64{4.26, 6.85, 31.00, 6.10, 1.77}},
		{6.0, [5]float64{1535, 307, 2509, 598, 871}, [5]float64{3.33, 6.69, 31.00, 6.14, 2.58}},
		{3.0, [5]float64{1482, 302, 2509, 575, 869}, [5]float64{2.95, 4.94, 31.00, 6.15, 3.12}},
		{1.0, [5]float64{1482, 301, 2509, 568, 869}, [5]float64{3.01, 4.47, 31.00, 6.16, 3.32}},
		{2.0, [5]float64{1482, 297, 2509, 553, 840}, [5]float64{3.12, 3.65, 31.00, 6.17, 3.86}},
		{1.0, [5]float64{542, 296, 2509, 549, 826}, [5]float64{0.00, 3.30, 31.00, 6.34, 4.17}},
		{2.0, [5]float64{542, 292, 2509, 541, 798}, [5]float64{0.00, 2.70, 31.00, 6.69, 4.84}},
		{3.0, [5]float64{542, 287, 2509, 529, 772}, [5]float64{0.00, 1.99, 31.00, 7.25, 5.20}},
		{1.0, [5]float64{542, 1622, 2509, 522, 763}, [5]float64{0.00, 0.00, 31.00, 7.32, 5.33}},
		{8.0, [5]float64{542, 1622, 2510, 466, 692}, [5]float64{0.00, 0.00, 31.00, 7.94, 6.45}},
		{1.0, [5]float64{542, 1622, 2510, 466, 683}, [5]float64{0.00, 0.00, 31.00, 0.00, 6.60}},
		{2.0, [5]float64{542, 1622, 2510, 466, 643}, [5]float64{0.00, 0.00, 31.00, 0.00, 6.82}},
		{4.0, [5]float64{542, 1622, 2509, 466, 625}, [5]float64{0.00, 0.00, 31.00, 0.00, 7.46}},
		{1.0, [5]float64{542, 1622, 2509, 466, 625}, [5]float64{5.79, 0.00, 31.00, 0.00, 0.00}},
		{1.0, [5]float64{538, 1622, 2509, 466, 625}, [5]float64{5.62, 3.46, 31.00, 0.00, 0.00}},
		{2.0, [5]float64{530, 1600, 2509, 466, 625}, [5]float64{5.29, 2.72, 26.30, 0.00, 0.00}},
		{1.0, [5]float64{526, 1607, 2509, 466, 625}, [5]float64{5.13, 2.69, 22.60, 0.00, 0.00}},
		{2.0, [5]float64{518, 1668, 2508, 466, 625}, [5]float64{4.84, 2.84, 16.68, 0.00, 0.00}},
		{1.0, [5]float64{502, 1670, 2508, 466, 625}, [5]float64{4.03, 2.63, 14.33, 0.00, 0.00}},
		{2.0, [5]float64{469, 1675, 2508, 466, 625}, [5]float64{2.79, 2.25, 6.77, 0.00, 0.00}},
		{2.0, [5]float64{451, 1680, 2508, 466, 625}, [5]float64{3.64, 2.48, 3.20, 0.00, 0.00}},
		{1.0, [5]float64{442, 1683, 2508, 466, 625}, [5]float64{4.16, 2.61, 1.72, 0.00, 0.00}},
		{2.0, [5]float64{473, 1688, 2506, 466, 625}, [5]float64{3.73, 2.87, 0.50, 0.00, 0.00}},
		{1.0, [5]float64{488, 1685, 2506, 466, 625}, [5]float64{3.53, 2.34, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{504, 1682, 2506, 466, 625}, [5]float64{3.35, 1.91, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{504, 1681, 2506, 466, 625}, [5]float64{0.00, 1.19, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{504, 1678, 2506, 466, 625}, [5]float64{0.00, 0.29, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{504, 1678, 2506, 466, 625}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{504, 1678, 2506, 466, 625}, [5]float64{0, 0, 0, 0, 0}},
	},
	207: { // R7
		{1.0, [5]float64{2805, 593, 639, 424, 901}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2805, 593, 639, 424, 901}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2805, 593, 639, 424, 901}, [5]float64{0.73, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2805, 593, 639, 424, 901}, [5]float64{1.60, 1.12, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2805, 600, 639, 424, 901}, [5]float64{3.51, 2.17, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2805, 565, 639, 424, 901}, [5]float64{8.33, 3.45, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2805, 529, 639, 424, 901}, [5]float64{12.82, 3.40, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2806, 492, 639, 424, 901}, [5]float64{15.84, 3.34, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2808, 480, 639, 424, 901}, [5]float64{19.56, 3.37, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2809, 477, 639, 424, 901}, [5]float64{24.16, 3.51, 2.99, 0.00, 0.00}},
		{1.0, [5]float64{2809, 475, 625, 424, 901}, [5]float64{25.72, 3.67, 3.43, 0.00, 0.00}},
		{1.0, [5]float64{2810, 493, 611, 424, 901}, [5]float64{27.37, 4.86, 3.94, 0.00, 0.00}},
		{1.0, [5]float64{2810, 1002, 597, 424, 901}, [5]float64{29.13, 0.00, 4.52, 0.00, 0.00}},
		{1.0, [5]float64{2811, 1002, 597, 424, 901}, [5]float64{31.00, 0.00, 4.78, 0.00, 0.00}},
		{1.0, [5]float64{2811, 1002, 596, 424, 901}, [5]float64{30.97, 3.33, 5.06, 0.00, 0.00}},
		{2.0, [5]float64{2811, 1021, 596, 424, 901}, [5]float64{30.91, 3.13, 5.66, 0.00, 0.00}},
		{2.0, [5]float64{2811, 1041, 614, 424, 901}, [5]float64{30.85, 2.94, 5.33, 0.00, 0.00}},
		{1.0, [5]float64{2811, 1051, 623, 424, 901}, [5]float64{30.82, 2.85, 5.17, 4.19, 0.00}},
		{2.0, [5]float64{2810, 1070, 641, 430, 901}, [5]float64{30.77, 2.67, 4.86, 4.44, 0.00}},
		{1.0, [5]float64{2810, 1083, 650, 433, 901}, [5]float64{30.74, 2.36, 4.72, 4.57, 0.00}},
		{3.0, [5]float64{2810, 1120, 664, 442, 901}, [5]float64{30.65, 1.62, 4.26, 4.98, 0.00}},
		{1.0, [5]float64{2810, 1133, 669, 445, 901}, [5]float64{30.62, 1.43, 4.11, 5.12, 2.83}},
		{1.0, [5]float64{2810, 260, 674, 448, 907}, [5]float64{30.59, 0.00, 3.97, 5.27, 2.93}},
		{2.0, [5]float64{2810, 260, 683, 455, 919}, [5]float64{30.53, 0.00, 3.71, 5.59, 3.13}},
		{2.0, [5]float64{2810, 260, 693, 466, 931}, [5]float64{30.47, 0.00, 3.46, 5.47, 3.35}},
		{1.0, [5]float64{2810, 260, 697, 471, 943}, [5]float64{30.45, 0.00, 3.34, 5.41, 3.44}},
		{1.0, [5]float64{2810, 260, 761, 477, 956}, [5]float64{30.42, 0.00, 0.00, 5.36, 3.54}},
		{3.0, [5]float64{2810, 260, 761, 494, 993}, [5]float64{30.33, 0.00, 0.00, 5.19, 3.84}},
		{1.0, [5]float64{2810, 260, 761, 499, 1005}, [5]float64{30.30, 1.88, 0.00, 5.14, 3.95}},
		{1.0, [5]float64{2810, 261, 761, 505, 1018}, [5]float64{30.27, 1.98, 0.00, 5.09, 4.06}},
		{1.0, [5]float64{2810, 262, 761, 507, 1023}, [5]float64{30.24, 2.09, 1.83, 4.99, 4.02}},
		{1.0, [5]float64{2810, 263, 764, 510, 1028}, [5]float64{30.22, 2.20, 2.23, 4.90, 3.97}},
		{7.0, [5]float64{2810, 269, 778, 526, 1062}, [5]float64{30.02, 3.18, 1.37, 4.29, 3.69}},
		{1.0, [5]float64{2810, 270, 780, 529, 1073}, [5]float64{29.99, 3.35, 1.28, 4.21, 3.70}},
		{1.0, [5]float64{2810, 271, 1077, 534, 1083}, [5]float64{29.96, 3.53, 0.00, 4.16, 3.70}},
		{4.0, [5]float64{2810, 275, 1077, 557, 1125}, [5]float64{29.84, 4.36, 0.00, 3.95, 3.73}},
		{6.0, [5]float64{2809, 281, 1077, 591, 1188}, [5]float64{29.68, 4.40, 0.00, 3.66, 3.77}},
		{3.0, [5]float64{2809, 284, 1077, 595, 1196}, [5]float64{29.59, 4.42, 0.00, 3.87, 3.83}},
		{2.0, [5]float64{2809, 286, 1077, 579, 1204}, [5]float64{23.89, 4.43, 0.00, 3.73, 3.14}},
		{1.0, [5]float64{2809, 287, 1077, 571, 1231}, [5]float64{21.47, 4.44, 0.00, 3.67, 2.81}},
		{1.0, [5]float64{2809, 296, 1077, 563, 1258}, [5]float64{19.29, 4.45, 2.55, 3.60, 2.51}},
		{1.0, [5]float64{2809, 304, 1072, 566, 1266}, [5]float64{17.34, 4.46, 2.63, 3.55, 1.87}},
		{1.0, [5]float64{2809, 321, 1066, 569, 1273}, [5]float64{15.58, 4.50, 2.71, 3.51, 1.39}},
		{1.0, [5]float64{2809, 337, 1082, 572, 1273}, [5]float64{14.00, 4.54, 2.81, 3.46, 0.00}},
		{1.0, [5]float64{2809, 354, 1098, 572, 1273}, [5]float64{12.58, 4.59, 2.90, 0.00, 0.00}},
		{1.0, [5]float64{2809, 370, 1114, 572, 1273}, [5]float64{9.15, 4.63, 3.00, 0.00, 0.00}},
		{2.0, [5]float64{2810, 403, 1126, 572, 1273}, [5]float64{4.85, 4.72, 2.91, 0.00, 0.00}},
		{2.0, [5]float64{2811, 403, 1138, 572, 1273}, [5]float64{2.57, 3.62, 2.82, 0.00, 0.00}},
		{1.0, [5]float64{2811, 404, 1139, 572, 1273}, [5]float64{1.87, 3.17, 2.14, 0.00, 0.00}},
		{1.0, [5]float64{2811, 405, 1139, 572, 1273}, [5]float64{0.00, 2.36, 1.62, 0.00, 0.00}},
		{3.0, [5]float64{2811, 409, 1141, 572, 1273}, [5]float64{0.00, 0.97, 0.71, 0.00, 0.00}},
		{6.0, [5]float64{2811, 409, 1141, 572, 1273}, [5]float64{0.00, 0.85, 0.78, 0.00, 0.00}},
		{2.0, [5]float64{2811, 410, 1144, 572, 1273}, [5]float64{0.00, 0.48, 0.35, 0.00, 0.00}},
		{1.0, [5]float64{2811, 410, 1144, 572, 1273}, [5]float64{0.00, 0.36, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2811, 410, 1144, 572, 1273}, [5]float64{0.00, 0.20, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2811, 410, 1144, 572, 1273}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2811, 410, 1144, 572, 1273}, [5]float64{0, 0, 0, 0, 0}},
	},
	208: { // R8
		{1.0, [5]float64{728, 3111, 575, 2311, 334}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{728, 3111, 575, 2311, 334}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{728, 3111, 575, 2311, 334}, [5]float64{0.59, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{709, 3111, 575, 2311, 334}, [5]float64{1.42, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{707, 3111, 575, 2311, 334}, [5]float64{2.26, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{706, 3111, 575, 2311, 334}, [5]float64{3.60, 1.75, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{748, 3111, 575, 2311, 334}, [5]float64{7.00, 14.17, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{756, 3111, 575, 2311, 334}, [5]float64{8.00, 16.31, 6.02, 0.00, 0.00}},
		{1.0, [5]float64{2783, 3111, 607, 2311, 334}, [5]float64{0.00, 18.79, 6.34, 0.00, 0.00}},
		{1.0, [5]float64{2783, 3111, 621, 2311, 334}, [5]float64{2.74, 21.63, 7.62, 0.00, 0.00}},
		{1.0, [5]float64{2779, 3111, 634, 2311, 334}, [5]float64{3.08, 24.91, 9.15, 0.00, 0.00}},
		{1.0, [5]float64{2774, 3110, 643, 2311, 334}, [5]float64{3.46, 28.68, 9.52, 0.00, 0.00}},
		{1.0, [5]float64{2770, 3111, 652, 2311, 334}, [5]float64{3.89, 28.73, 9.90, 0.00, 0.00}},
		{2.0, [5]float64{2725, 3111, 670, 2311, 334}, [5]float64{3.90, 28.83, 10.70, 0.00, 0.00}},
		{1.0, [5]float64{2702, 3111, 679, 2311, 334}, [5]float64{3.90, 28.87, 11.13, 1.07, 0.89}},
		{1.0, [5]float64{2679, 3111, 677, 2317, 334}, [5]float64{3.90, 28.92, 11.13, 0.91, 0.95}},
		{2.0, [5]float64{2655, 3111, 674, 2317, 332}, [5]float64{4.18, 29.02, 11.13, 1.14, 1.06}},
		{4.0, [5]float64{2658, 3111, 669, 2317, 330}, [5]float64{2.75, 29.21, 11.13, 1.80, 1.33}},
		{4.0, [5]float64{2660, 3111, 663, 2316, 350}, [5]float64{1.80, 29.40, 11.13, 2.83, 1.51}},
		{3.0, [5]float64{2663, 3111, 659, 2316, 345}, [5]float64{1.32, 29.55, 11.13, 3.99, 1.45}},
		{1.0, [5]float64{2663, 3111, 658, 2304, 330}, [5]float64{1.18, 29.60, 11.13, 3.97, 1.37}},
		{1.0, [5]float64{2670, 3111, 657, 2291, 314}, [5]float64{1.48, 29.65, 11.13, 3.95, 1.29}},
		{1.0, [5]float64{1910, 3111, 655, 2278, 314}, [5]float64{0.00, 29.70, 11.13, 3.93, 1.26}},
		{1.0, [5]float64{1910, 3111, 654, 2266, 313}, [5]float64{0.00, 29.75, 11.13, 3.92, 1.22}},
		{3.0, [5]float64{1910, 3111, 650, 2228, 322}, [5]float64{0.00, 29.89, 11.13, 3.86, 1.21}},
		{1.0, [5]float64{1910, 3111, 648, 2228, 325}, [5]float64{0.00, 29.94, 11.13, 3.71, 1.20}},
		{1.0, [5]float64{1910, 3111, 647, 2228, 328}, [5]float64{0.57, 29.99, 11.13, 3.56, 1.20}},
		{5.0, [5]float64{1905, 3111, 640, 2228, 320}, [5]float64{1.50, 30.24, 11.13, 2.90, 1.18}},
		{2.0, [5]float64{1903, 3111, 637, 2228, 338}, [5]float64{2.21, 30.34, 11.13, 2.67, 1.38}},
		{1.0, [5]float64{1903, 3111, 636, 2228, 336}, [5]float64{2.68, 30.39, 11.13, 2.57, 1.42}},
		{1.0, [5]float64{1903, 3111, 634, 2228, 335}, [5]float64{2.81, 30.44, 11.13, 2.46, 1.47}},
		{1.0, [5]float64{1902, 3111, 633, 2224, 334}, [5]float64{2.94, 30.49, 11.13, 2.18, 1.52}},
		{1.0, [5]float64{1902, 3111, 632, 2220, 326}, [5]float64{3.08, 30.54, 11.13, 1.93, 1.50}},
		{1.0, [5]float64{1890, 3112, 630, 2211, 319}, [5]float64{3.16, 30.59, 11.13, 2.42, 1.49}},
		{1.0, [5]float64{1877, 3112, 629, 2211, 311}, [5]float64{3.25, 30.64, 11.13, 0.00, 1.48}},
		{2.0, [5]float64{1851, 3112, 626, 2211, 295}, [5]float64{3.43, 30.75, 11.13, 0.00, 1.46}},
		{2.0, [5]float64{1825, 3112, 623, 2211, 296}, [5]float64{3.63, 30.85, 11.13, 0.00, 1.44}},
		{3.0, [5]float64{1827, 3112, 619, 2211, 297}, [5]float64{3.80, 31.00, 11.13, 0.00, 1.40}},
		{1.0, [5]float64{1827, 3112, 618, 2211, 297}, [5]float64{3.86, 30.05, 11.13, 0.00, 1.39}},
		{2.0, [5]float64{1828, 3112, 615, 2211, 317}, [5]float64{3.99, 28.23, 11.13, 0.00, 1.70}},
		{1.0, [5]float64{1823, 3112, 614, 2211, 317}, [5]float64{3.84, 27.37, 11.13, 0.00, 0.00}},
		{2.0, [5]float64{1814, 3112, 584, 2211, 317}, [5]float64{3.57, 25.71, 10.23, 0.00, 0.00}},
		{1.0, [5]float64{1809, 3111, 569, 2211, 317}, [5]float64{3.43, 22.89, 9.81, 0.00, 0.00}},
		{2.0, [5]float64{1792, 3111, 540, 2211, 317}, [5]float64{3.04, 18.14, 9.02, 0.00, 0.00}},
		{1.0, [5]float64{1730, 3111, 540, 2211, 317}, [5]float64{2.57, 16.15, 8.59, 0.00, 0.00}},
		{1.0, [5]float64{689, 3111, 540, 2211, 317}, [5]float64{0.00, 14.38, 8.18, 0.00, 0.00}},
		{0.7, [5]float64{689, 3111, 540, 2211, 317}, [5]float64{0.00, 11.91, 7.89, 0.00, 0.00}},
		{1.0, [5]float64{689, 3111, 541, 2211, 317}, [5]float64{5.15, 9.25, 7.51, 0.00, 0.00}},
		{0.5, [5]float64{688, 3111, 541, 2211, 317}, [5]float64{5.15, 8.67, 7.41, 0.00, 0.00}},
		{1.0, [5]float64{688, 3111, 574, 2211, 317}, [5]float64{5.14, 6.73, 11.66, 0.00, 0.00}},
		{0.7, [5]float64{687, 3111, 574, 2211, 317}, [5]float64{5.13, 5.58, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{682, 3111, 574, 2211, 317}, [5]float64{5.12, 5.23, 0.00, 0.00, 0.00}},
		{0.7, [5]float64{667, 3111, 574, 2211, 317}, [5]float64{5.08, 3.59, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{624, 3111, 574, 2211, 317}, [5]float64{5.32, 2.15, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{614, 3111, 574, 2211, 317}, [5]float64{5.20, 1.29, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{614, 3111, 574, 2211, 317}, [5]float64{4.75, 1.12, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{615, 3112, 574, 2211, 317}, [5]float64{3.40, 1.26, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{616, 3112, 574, 2211, 317}, [5]float64{2.43, 0.00, 0.00, 0.00, 0.00}},
		{3.7, [5]float64{619, 3112, 574, 2211, 317}, [5]float64{0.70, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{619, 3112, 574, 2211, 317}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{619, 3112, 574, 2211, 317}, [5]float64{0, 0, 0, 0, 0}},
	},
	209: { // R9
		{1.0, [5]float64{680, 1693, 3412, 334, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{680, 1693, 3412, 334, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{680, 1693, 3412, 334, 0}, [5]float64{1.06, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{678, 1693, 3412, 334, 0}, [5]float64{1.84, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{676, 1693, 3412, 334, 0}, [5]float64{3.19, 1.18, 1.00, 0.00, 0.00}},
		{2.0, [5]float64{682, 1710, 3412, 334, 0}, [5]float64{5.65, 2.51, 3.49, 0.00, 0.00}},
		{2.0, [5]float64{686, 1756, 3412, 334, 0}, [5]float64{7.27, 2.94, 7.91, 0.00, 0.00}},
		{1.0, [5]float64{689, 1763, 3412, 334, 0}, [5]float64{8.24, 3.05, 11.89, 0.00, 0.00}},
		{1.0, [5]float64{681, 1771, 3413, 334, 0}, [5]float64{8.97, 3.16, 14.63, 0.00, 0.00}},
		{1.0, [5]float64{673, 1800, 3414, 334, 0}, [5]float64{9.76, 3.64, 18.00, 0.00, 0.00}},
		{1.0, [5]float64{666, 1677, 3414, 334, 0}, [5]float64{10.63, 0.00, 22.15, 0.00, 0.00}},
		{2.0, [5]float64{651, 1677, 3414, 334, 0}, [5]float64{12.59, 0.00, 25.10, 0.00, 0.00}},
		{1.0, [5]float64{657, 1677, 3413, 334, 0}, [5]float64{12.84, 0.00, 26.72, 0.00, 0.00}},
		{1.0, [5]float64{664, 1677, 3413, 334, 0}, [5]float64{13.10, 2.96, 28.44, 0.00, 0.00}},
		{1.0, [5]float64{671, 1702, 3413, 334, 0}, [5]float64{13.36, 3.71, 28.41, 0.00, 0.00}},
		{1.0, [5]float64{678, 1695, 3413, 334, 0}, [5]float64{13.62, 3.90, 28.39, 2.60, 0.00}},
		{2.0, [5]float64{677, 1681, 3413, 334, 0}, [5]float64{13.63, 4.32, 28.33, 1.49, 0.00}},
		{1.0, [5]float64{676, 1674, 3413, 350, 0}, [5]float64{13.63, 4.54, 28.31, 1.43, 0.00}},
		{1.0, [5]float64{675, 1667, 3413, 349, 0}, [5]float64{13.63, 4.78, 28.28, 1.38, 0.00}},
		{2.0, [5]float64{674, 1667, 3413, 348, 0}, [5]float64{13.64, 4.77, 28.23, 1.29, 0.00}},
		{1.0, [5]float64{673, 1667, 3413, 328, 0}, [5]float64{13.64, 4.76, 28.20, 1.19, 0.00}},
		{2.0, [5]float64{671, 1667, 3413, 347, 0}, [5]float64{13.64, 4.75, 28.15, 1.07, 0.00}},
		{1.0, [5]float64{671, 1667, 3413, 324, 0}, [5]float64{13.65, 4.74, 28.12, 1.01, 0.00}},
		{1.0, [5]float64{670, 1667, 3413, 335, 0}, [5]float64{13.65, 4.74, 28.09, 0.97, 0.00}},
		{1.0, [5]float64{669, 1667, 3413, 344, 0}, [5]float64{13.65, 4.73, 28.07, 0.97, 0.00}},
		{1.0, [5]float64{668, 1668, 3413, 327, 0}, [5]float64{13.65, 4.72, 28.04, 0.98, 0.00}},
		{2.0, [5]float64{667, 1668, 3413, 360, 0}, [5]float64{13.66, 4.71, 27.99, 1.08, 0.00}},
		{1.0, [5]float64{663, 1657, 3413, 331, 0}, [5]float64{13.61, 4.71, 27.96, 1.00, 0.00}},
		{3.0, [5]float64{651, 1626, 3413, 300, 0}, [5]float64{13.45, 4.69, 27.88, 1.10, 0.00}},
		{2.0, [5]float64{643, 1605, 3413, 308, 0}, [5]float64{13.35, 4.69, 27.83, 1.34, 0.00}},
		{1.0, [5]float64{643, 1605, 3413, 312, 0}, [5]float64{13.38, 4.67, 27.80, 1.49, 0.00}},
		{2.0, [5]float64{643, 1605, 3413, 340, 0}, [5]float64{13.42, 4.65, 27.75, 1.84, 0.00}},
		{1.0, [5]float64{642, 1604, 3413, 323, 0}, [5]float64{13.44, 4.63, 27.73, 1.95, 0.00}},
		{1.0, [5]float64{642, 1604, 3413, 688, 0}, [5]float64{13.47, 4.62, 27.70, 0.00, 0.00}},
		{7.0, [5]float64{640, 1602, 3413, 688, 0}, [5]float64{13.63, 4.52, 27.52, 0.00, 0.00}},
		{1.0, [5]float64{640, 1594, 3413, 688, 0}, [5]float64{13.65, 4.58, 27.49, 0.00, 0.00}},
		{7.0, [5]float64{626, 1538, 3413, 688, 0}, [5]float64{13.34, 4.97, 27.31, 0.00, 0.00}},
		{6.0, [5]float64{614, 1542, 3413, 688, 0}, [5]float64{13.09, 4.37, 27.16, 0.00, 0.00}},
		{1.0, [5]float64{612, 1543, 3413, 688, 0}, [5]float64{13.05, 4.27, 24.61, 0.00, 0.00}},
		{4.0, [5]float64{534, 1546, 3412, 688, 0}, [5]float64{10.63, 3.92, 16.60, 0.00, 0.00}},
		{2.0, [5]float64{531, 1548, 3412, 688, 0}, [5]float64{9.77, 3.75, 13.64, 0.00, 0.00}},
		{1.7, [5]float64{528, 1517, 3411, 688, 0}, [5]float64{9.07, 3.38, 8.78, 0.00, 0.00}},
		{0.5, [5]float64{528, 1512, 3411, 688, 0}, [5]float64{8.98, 3.33, 8.24, 0.00, 0.00}},
		{0.7, [5]float64{526, 1519, 3411, 688, 0}, [5]float64{8.70, 3.28, 6.83, 6.21, 0.00}},
		{0.5, [5]float64{526, 1522, 3411, 688, 0}, [5]float64{8.60, 3.27, 6.40, 6.23, 0.00}},
		{1.0, [5]float64{545, 1531, 3411, 688, 0}, [5]float64{12.47, 3.20, 4.98, 6.31, 0.00}},
		{0.7, [5]float64{545, 1538, 3410, 689, 0}, [5]float64{0.00, 3.16, 4.13, 6.37, 0.00}},
		{0.5, [5]float64{545, 1540, 3410, 684, 0}, [5]float64{0.00, 3.14, 3.87, 6.36, 0.00}},
		{0.7, [5]float64{545, 1574, 3410, 672, 0}, [5]float64{0.00, 2.99, 3.21, 6.32, 0.00}},
		{0.5, [5]float64{545, 1587, 3410, 661, 0}, [5]float64{0.00, 2.94, 3.01, 6.39, 0.00}},
		{0.7, [5]float64{545, 1587, 3410, 630, 0}, [5]float64{0.00, 0.00, 0.00, 6.60, 0.00}},
		{0.5, [5]float64{545, 1587, 3410, 627, 0}, [5]float64{0.00, 0.00, 0.00, 6.59, 0.00}},
		{0.7, [5]float64{545, 1587, 3410, 618, 0}, [5]float64{0.00, 0.00, 0.00, 6.58, 0.00}},
		{2.0, [5]float64{545, 1587, 3410, 623, 0}, [5]float64{0.00, 0.00, 0.00, 4.44, 0.00}},
		{4.0, [5]float64{545, 1587, 3410, 628, 0}, [5]float64{0.00, 0.00, 0.00, 0.94, 0.00}},
		{1.0, [5]float64{545, 1587, 3410, 628, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{545, 1587, 3410, 628, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	210: { // A0
		{1.0, [5]float64{1615, 305, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{11.0, [5]float64{1615, 305, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1615, 305, 0, 0, 0}, [5]float64{5.71, 0.00, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{1695, 305, 0, 0, 0}, [5]float64{7.41, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1718, 305, 0, 0, 0}, [5]float64{6.47, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1724, 305, 0, 0, 0}, [5]float64{6.25, 1.92, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1729, 301, 0, 0, 0}, [5]float64{6.04, 1.32, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1735, 300, 0, 0, 0}, [5]float64{5.84, 1.67, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1763, 294, 0, 0, 0}, [5]float64{5.89, 5.35, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1803, 215, 0, 0, 0}, [5]float64{5.97, 2.89, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1814, 228, 0, 0, 0}, [5]float64{6.00, 3.45, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1831, 281, 0, 0, 0}, [5]float64{6.04, 6.05, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1870, 242, 0, 0, 0}, [5]float64{6.13, 5.04, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1904, 283, 0, 0, 0}, [5]float64{6.21, 8.02, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1932, 262, 0, 0, 0}, [5]float64{6.29, 6.60, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1961, 292, 0, 0, 0}, [5]float64{6.37, 8.83, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1989, 277, 0, 0, 0}, [5]float64{6.45, 7.50, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2017, 303, 0, 0, 0}, [5]float64{6.53, 9.33, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2045, 293, 0, 0, 0}, [5]float64{6.62, 7.76, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2073, 316, 0, 0, 0}, [5]float64{6.71, 9.39, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2096, 303, 0, 0, 0}, [5]float64{6.79, 8.56, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2124, 327, 0, 0, 0}, [5]float64{6.89, 9.42, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2147, 312, 0, 0, 0}, [5]float64{6.97, 8.93, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2169, 340, 0, 0, 0}, [5]float64{7.05, 8.40, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2197, 314, 0, 0, 0}, [5]float64{7.16, 9.27, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2220, 360, 0, 0, 0}, [5]float64{7.25, 8.16, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2237, 322, 0, 0, 0}, [5]float64{7.31, 10.79, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2254, 269, 0, 0, 0}, [5]float64{7.38, 8.45, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2259, 274, 0, 0, 0}, [5]float64{7.40, 7.49, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2265, 361, 0, 0, 0}, [5]float64{7.43, 9.21, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2271, 224, 0, 0, 0}, [5]float64{7.45, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{2305, 224, 0, 0, 0}, [5]float64{7.59, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2310, 224, 0, 0, 0}, [5]float64{7.62, 7.84, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2316, 266, 0, 0, 0}, [5]float64{7.64, 8.66, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2333, 291, 0, 0, 0}, [5]float64{7.71, 12.42, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2362, 236, 0, 0, 0}, [5]float64{7.44, 11.42, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2385, 280, 0, 0, 0}, [5]float64{7.22, 15.38, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2408, 247, 0, 0, 0}, [5]float64{7.01, 14.21, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2431, 279, 0, 0, 0}, [5]float64{6.81, 17.69, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2449, 258, 0, 0, 0}, [5]float64{6.66, 16.99, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2472, 280, 0, 0, 0}, [5]float64{6.47, 19.24, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2495, 263, 0, 0, 0}, [5]float64{6.28, 18.30, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2518, 285, 0, 0, 0}, [5]float64{6.10, 20.93, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2536, 273, 0, 0, 0}, [5]float64{5.96, 20.68, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2559, 289, 0, 0, 0}, [5]float64{5.79, 22.30, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2576, 288, 0, 0, 0}, [5]float64{5.66, 22.38, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{2599, 285, 0, 0, 0}, [5]float64{5.45, 22.62, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2608, 299, 0, 0, 0}, [5]float64{5.37, 24.10, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2613, 301, 0, 0, 0}, [5]float64{5.32, 24.22, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2653, 304, 0, 0, 0}, [5]float64{5.67, 24.41, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2653, 308, 0, 0, 0}, [5]float64{4.43, 24.72, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2574, 309, 0, 0, 0}, [5]float64{5.12, 24.78, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{415, 310, 0, 0, 0}, [5]float64{0.00, 24.85, 0.00, 0.00, 0.00}},
		{87.0, [5]float64{415, 384, 0, 0, 0}, [5]float64{0.00, 31.00, 0.00, 0.00, 0.00}},
		{18.0, [5]float64{415, 342, 0, 0, 0}, [5]float64{0.00, 17.90, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{415, 308, 0, 0, 0}, [5]float64{0.00, 13.56, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{415, 305, 0, 0, 0}, [5]float64{0.00, 16.94, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{415, 305, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{415, 305, 0, 0, 0}, [5]float64{10.70, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{437, 305, 0, 0, 0}, [5]float64{12.05, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{388, 305, 0, 0, 0}, [5]float64{10.02, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{390, 305, 0, 0, 0}, [5]float64{8.95, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{385, 305, 0, 0, 0}, [5]float64{6.16, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{371, 305, 0, 0, 0}, [5]float64{4.87, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{327, 305, 0, 0, 0}, [5]float64{6.31, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{327, 305, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{327, 305, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	214: { // A4
		{1.0, [5]float64{247, 0, 0, 0, 0}, [5]float64{0.37, 0.00, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{247, 0, 0, 0, 0}, [5]float64{10.50, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{247, 0, 0, 0, 0}, [5]float64{20.36, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{247, 0, 0, 0, 0}, [5]float64{19.65, 0.00, 0.00, 0.00, 0.00}},
		{11.0, [5]float64{261, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{12.0, [5]float64{303, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{49.0, [5]float64{496, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{534, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{45.0, [5]float64{1083, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{3.1, [5]float64{1122, 0, 0, 0, 0}, [5]float64{27.99, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1257, 0, 0, 0, 0}, [5]float64{23.91, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1408, 0, 0, 0, 0}, [5]float64{22.51, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1559, 0, 0, 0, 0}, [5]float64{21.35, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1712, 0, 0, 0, 0}, [5]float64{20.41, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1863, 0, 0, 0, 0}, [5]float64{20.10, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2015, 0, 0, 0, 0}, [5]float64{19.50, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2169, 0, 0, 0, 0}, [5]float64{19.68, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2320, 0, 0, 0, 0}, [5]float64{19.73, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2464, 0, 0, 0, 0}, [5]float64{18.74, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2689, 0, 0, 0, 0}, [5]float64{14.39, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2998, 0, 0, 0, 0}, [5]float64{13.33, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3298, 0, 0, 0, 0}, [5]float64{11.45, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3578, 0, 0, 0, 0}, [5]float64{10.41, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3878, 0, 0, 0, 0}, [5]float64{18.59, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3911, 0, 0, 0, 0}, [5]float64{22.95, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{21.53, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{19.52, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{17.75, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{16.17, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{14.74, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3912, 0, 0, 0, 0}, [5]float64{13.01, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{11.02, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3914, 0, 0, 0, 0}, [5]float64{9.23, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3912, 0, 0, 0, 0}, [5]float64{7.60, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3914, 0, 0, 0, 0}, [5]float64{6.08, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{4.17, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{3913, 0, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	215: { // A5
		{1.0, [5]float64{644, 333, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{644, 333, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{644, 333, 0, 0, 0}, [5]float64{1.67, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{641, 333, 0, 0, 0}, [5]float64{1.67, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{641, 333, 0, 0, 0}, [5]float64{6.38, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{641, 333, 0, 0, 0}, [5]float64{12.67, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{636, 333, 0, 0, 0}, [5]float64{18.94, 0.00, 0.00, 0.00, 0.00}},
		{173.0, [5]float64{420, 333, 0, 0, 0}, [5]float64{19.05, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{405, 333, 0, 0, 0}, [5]float64{19.13, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{417, 333, 0, 0, 0}, [5]float64{18.90, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{389, 333, 0, 0, 0}, [5]float64{19.80, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{413, 333, 0, 0, 0}, [5]float64{18.27, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{358, 333, 0, 0, 0}, [5]float64{21.81, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{409, 333, 0, 0, 0}, [5]float64{17.66, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{409, 333, 0, 0, 0}, [5]float64{17.47, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{345, 333, 0, 0, 0}, [5]float64{24.17, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{338, 333, 0, 0, 0}, [5]float64{25.84, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{341, 333, 0, 0, 0}, [5]float64{24.61, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{404, 333, 0, 0, 0}, [5]float64{16.39, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{409, 333, 0, 0, 0}, [5]float64{15.78, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{341, 333, 0, 0, 0}, [5]float64{23.88, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{336, 333, 0, 0, 0}, [5]float64{24.87, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{368, 333, 0, 0, 0}, [5]float64{15.99, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{377, 333, 0, 0, 0}, [5]float64{15.81, 0.00, 0.00, 0.00, 0.00}},
		{0.8, [5]float64{410, 333, 0, 0, 0}, [5]float64{15.24, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{410, 333, 0, 0, 0}, [5]float64{0.00, 10.62, 0.00, 0.00, 0.00}},
		{0.8, [5]float64{410, 333, 0, 0, 0}, [5]float64{0.00, 10.02, 0.00, 0.00, 0.00}},
		{1.2, [5]float64{410, 333, 0, 0, 0}, [5]float64{0.00, 9.13, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{410, 331, 0, 0, 0}, [5]float64{0.00, 25.97, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{410, 331, 0, 0, 0}, [5]float64{0.00, 31.00, 0.00, 0.00, 0.00}},
		{12.0, [5]float64{410, 331, 0, 0, 0}, [5]float64{0.00, 15.08, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{410, 333, 0, 0, 0}, [5]float64{0.00, 6.31, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 393, 0, 0, 0}, [5]float64{0.00, 4.61, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 409, 0, 0, 0}, [5]float64{0.00, 4.21, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 390, 0, 0, 0}, [5]float64{0.00, 5.41, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 333, 0, 0, 0}, [5]float64{0.00, 7.96, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{410, 331, 0, 0, 0}, [5]float64{0.00, 26.45, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{410, 332, 0, 0, 0}, [5]float64{0.00, 21.68, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 334, 0, 0, 0}, [5]float64{0.00, 16.97, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 384, 0, 0, 0}, [5]float64{0.00, 12.30, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 411, 0, 0, 0}, [5]float64{0.00, 10.39, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 401, 0, 0, 0}, [5]float64{0.00, 11.37, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 333, 0, 0, 0}, [5]float64{0.00, 19.58, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{410, 333, 0, 0, 0}, [5]float64{0.00, 24.72, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 335, 0, 0, 0}, [5]float64{0.00, 20.98, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 405, 0, 0, 0}, [5]float64{0.00, 13.64, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 413, 0, 0, 0}, [5]float64{0.00, 13.07, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 384, 0, 0, 0}, [5]float64{0.00, 15.35, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 338, 0, 0, 0}, [5]float64{0.00, 19.51, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 335, 0, 0, 0}, [5]float64{0.00, 22.71, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 342, 0, 0, 0}, [5]float64{0.00, 20.17, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 386, 0, 0, 0}, [5]float64{0.00, 16.49, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 409, 0, 0, 0}, [5]float64{0.00, 14.96, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 414, 0, 0, 0}, [5]float64{0.00, 14.60, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 388, 0, 0, 0}, [5]float64{0.00, 16.25, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 350, 0, 0, 0}, [5]float64{0.00, 18.84, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 348, 0, 0, 0}, [5]float64{0.00, 19.63, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 403, 0, 0, 0}, [5]float64{0.00, 16.18, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 414, 0, 0, 0}, [5]float64{0.00, 15.63, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 408, 0, 0, 0}, [5]float64{0.00, 15.85, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 376, 0, 0, 0}, [5]float64{0.00, 17.51, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{410, 418, 0, 0, 0}, [5]float64{0.00, 16.15, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{410, 403, 0, 0, 0}, [5]float64{0.00, 16.28, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{410, 423, 0, 0, 0}, [5]float64{0.00, 16.21, 0.00, 0.00, 0.00}},
		{180.4, [5]float64{410, 628, 0, 0, 0}, [5]float64{0.00, 14.44, 0.00, 0.00, 0.00}},
		{27.9, [5]float64{410, 628, 0, 0, 0}, [5]float64{0.00, 7.07, 0.00, 0.00, 0.00}},
		{16.9, [5]float64{410, 628, 0, 0, 0}, [5]float64{0.00, 2.45, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{410, 628, 0, 0, 0}, [5]float64{0.00, 0.64, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 628, 0, 0, 0}, [5]float64{0.00, 0.23, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{410, 628, 0, 0, 0}, [5]float64{0.00, 0.05, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{410, 628, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{410, 628, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	216: { // A6
		{1.0, [5]float64{308, 262, 0, 0, 0}, [5]float64{0.22, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{308, 262, 0, 0, 0}, [5]float64{1.71, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{308, 262, 0, 0, 0}, [5]float64{12.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{368, 262, 0, 0, 0}, [5]float64{22.68, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{457, 262, 0, 0, 0}, [5]float64{28.15, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{475, 262, 0, 0, 0}, [5]float64{30.38, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{503, 262, 0, 0, 0}, [5]float64{29.91, 21.41, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{561, 263, 0, 0, 0}, [5]float64{29.03, 20.34, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{704, 304, 0, 0, 0}, [5]float64{27.04, 30.57, 0.00, 0.00, 0.00}},
		{33.0, [5]float64{1649, 662, 0, 0, 0}, [5]float64{18.59, 30.66, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{1649, 759, 0, 0, 0}, [5]float64{14.88, 30.87, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1649, 799, 0, 0, 0}, [5]float64{13.15, 31.00, 0.00, 0.00, 0.00}},
		{37.0, [5]float64{1649, 802, 0, 0, 0}, [5]float64{5.27, 30.87, 0.00, 0.00, 0.00}},
		{11.0, [5]float64{1649, 803, 0, 0, 0}, [5]float64{2.52, 30.84, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.72, 30.81, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 30.81, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 30.79, 0.00, 0.00, 0.00}},
		{31.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 15.09, 0.00, 0.00, 0.00}},
		{12.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 8.37, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 3.78, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 1.35, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 0.32, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1649, 804, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	217: { // A7
		{1.0, [5]float64{1145, 418, 876, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1145, 418, 876, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1145, 418, 876, 0, 0}, [5]float64{0.67, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1142, 418, 876, 0, 0}, [5]float64{4.17, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1142, 418, 876, 0, 0}, [5]float64{11.24, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1153, 418, 876, 0, 0}, [5]float64{19.99, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1156, 418, 876, 0, 0}, [5]float64{23.09, 0.48, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1161, 418, 876, 0, 0}, [5]float64{22.83, 0.41, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1163, 418, 876, 0, 0}, [5]float64{22.70, 0.50, 0.48, 0.00, 0.00}},
		{3.0, [5]float64{1169, 418, 875, 0, 0}, [5]float64{22.31, 0.97, 0.74, 0.00, 0.00}},
		{6.0, [5]float64{1182, 418, 873, 0, 0}, [5]float64{21.55, 1.86, 1.78, 0.00, 0.00}},
		{1.0, [5]float64{1184, 418, 873, 0, 0}, [5]float64{21.43, 2.07, 1.87, 0.00, 0.00}},
		{8.0, [5]float64{1202, 418, 873, 0, 0}, [5]float64{20.46, 2.93, 2.81, 0.00, 0.00}},
		{8.0, [5]float64{1219, 418, 873, 0, 0}, [5]float64{19.55, 4.15, 3.36, 0.00, 0.00}},
		{44.0, [5]float64{1314, 418, 874, 0, 0}, [5]float64{15.21, 9.86, 9.01, 0.00, 0.00}},
		{9.0, [5]float64{1333, 418, 894, 0, 0}, [5]float64{14.45, 11.77, 9.54, 0.00, 0.00}},
		{60.0, [5]float64{1462, 458, 1027, 0, 0}, [5]float64{10.32, 17.55, 13.88, 0.00, 0.00}},
		{58.0, [5]float64{1463, 497, 1156, 0, 0}, [5]float64{5.28, 25.83, 20.00, 0.00, 0.00}},
		{27.0, [5]float64{1463, 515, 1215, 0, 0}, [5]float64{2.33, 30.93, 23.72, 0.00, 0.00}},
		{1.0, [5]float64{1463, 516, 1218, 0, 0}, [5]float64{2.26, 30.93, 23.88, 0.00, 0.00}},
		{5.0, [5]float64{1463, 524, 1229, 0, 0}, [5]float64{2.32, 30.93, 24.65, 0.00, 0.00}},
		{29.0, [5]float64{1463, 567, 1293, 0, 0}, [5]float64{2.67, 30.94, 24.17, 0.00, 0.00}},
		{16.0, [5]float64{1463, 590, 1328, 0, 0}, [5]float64{2.06, 30.95, 23.92, 0.00, 0.00}},
		{10.0, [5]float64{1463, 605, 1350, 0, 0}, [5]float64{1.11, 30.96, 23.77, 0.00, 0.00}},
		{22.0, [5]float64{1463, 638, 1398, 0, 0}, [5]float64{0.90, 30.99, 23.47, 0.00, 0.00}},
		{4.0, [5]float64{1463, 644, 1407, 0, 0}, [5]float64{1.08, 31.00, 23.41, 0.00, 0.00}},
		{3.0, [5]float64{1463, 648, 1413, 0, 0}, [5]float64{1.24, 30.95, 23.37, 0.00, 0.00}},
		{11.0, [5]float64{1463, 664, 1438, 0, 0}, [5]float64{0.88, 30.75, 23.23, 0.00, 0.00}},
		{6.0, [5]float64{1463, 673, 1456, 0, 0}, [5]float64{0.73, 30.64, 23.15, 0.00, 0.00}},
		{13.0, [5]float64{1463, 692, 1494, 0, 0}, [5]float64{1.08, 30.42, 22.99, 0.00, 0.00}},
		{1.0, [5]float64{1463, 694, 1496, 0, 0}, [5]float64{1.11, 30.40, 22.98, 0.00, 0.00}},
		{14.0, [5]float64{1463, 715, 1524, 0, 0}, [5]float64{0.70, 30.16, 22.84, 0.00, 0.00}},
		{26.0, [5]float64{1463, 753, 1574, 0, 0}, [5]float64{1.01, 29.72, 22.60, 0.00, 0.00}},
		{11.0, [5]float64{1463, 770, 1596, 0, 0}, [5]float64{0.65, 29.54, 22.51, 0.00, 0.00}},
		{1.0, [5]float64{1456, 771, 1598, 0, 0}, [5]float64{0.85, 29.53, 22.50, 0.00, 0.00}},
		{1.0, [5]float64{1456, 772, 1600, 0, 0}, [5]float64{0.00, 29.51, 22.49, 0.00, 0.00}},
		{4.0, [5]float64{1456, 778, 1608, 0, 0}, [5]float64{0.00, 29.44, 22.46, 0.00, 0.00}},
		{15.0, [5]float64{1456, 801, 1658, 0, 0}, [5]float64{0.00, 29.20, 22.24, 0.00, 0.00}},
		{9.0, [5]float64{1456, 814, 1659, 0, 0}, [5]float64{0.00, 29.06, 22.10, 0.00, 0.00}},
		{5.0, [5]float64{1456, 821, 1690, 0, 0}, [5]float64{0.00, 28.98, 22.01, 0.00, 0.00}},
		{9.0, [5]float64{1456, 835, 1694, 0, 0}, [5]float64{0.00, 28.84, 22.08, 0.00, 0.00}},
		{16.0, [5]float64{1456, 858, 1748, 0, 0}, [5]float64{0.00, 28.59, 21.93, 0.00, 0.00}},
		{6.0, [5]float64{1456, 867, 1748, 0, 0}, [5]float64{0.00, 28.50, 21.90, 0.00, 0.00}},
		{42.0, [5]float64{1456, 919, 1816, 0, 0}, [5]float64{0.00, 16.03, 13.46, 0.00, 0.00}},
		{3.0, [5]float64{1456, 920, 1821, 0, 0}, [5]float64{0.00, 14.48, 13.00, 0.00, 0.00}},
		{35.0, [5]float64{1456, 928, 1821, 0, 0}, [5]float64{0.00, 4.39, 5.13, 0.00, 0.00}},
		{8.0, [5]float64{1456, 928, 1821, 0, 0}, [5]float64{0.00, 2.03, 4.15, 0.00, 0.00}},
		{1.0, [5]float64{1456, 928, 1821, 0, 0}, [5]float64{0.00, 1.84, 3.91, 0.00, 0.00}},
		{8.0, [5]float64{1456, 928, 1821, 0, 0}, [5]float64{0.00, 0.33, 2.46, 0.00, 0.00}},
		{1.0, [5]float64{1456, 928, 1821, 0, 0}, [5]float64{0.00, 0.00, 2.32, 0.00, 0.00}},
		{4.0, [5]float64{1456, 928, 1821, 0, 0}, [5]float64{0.00, 0.00, 1.84, 0.00, 0.00}},
		{3.0, [5]float64{1456, 928, 1821, 0, 0}, [5]float64{0.00, 0.00, 1.01, 0.00, 0.00}},
		{4.0, [5]float64{1456, 928, 1815, 0, 0}, [5]float64{0.00, 0.00, 0.22, 0.00, 0.00}},
		{1.0, [5]float64{1456, 928, 1815, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1456, 928, 1815, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	218: { // A8
		{1.0, [5]float64{348, 289, 0, 0, 0}, [5]float64{0.16, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{348, 289, 0, 0, 0}, [5]float64{0.98, 0.00, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{348, 289, 0, 0, 0}, [5]float64{6.76, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{348, 289, 0, 0, 0}, [5]float64{10.80, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{389, 289, 0, 0, 0}, [5]float64{11.75, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{423, 289, 0, 0, 0}, [5]float64{14.24, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{484, 289, 0, 0, 0}, [5]float64{15.53, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{500, 289, 0, 0, 0}, [5]float64{15.87, 11.06, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{577, 290, 0, 0, 0}, [5]float64{17.71, 13.80, 0.00, 0.00, 0.00}},
		{23.0, [5]float64{931, 449, 0, 0, 0}, [5]float64{29.84, 24.49, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{999, 491, 0, 0, 0}, [5]float64{31.00, 28.49, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{999, 498, 0, 0, 0}, [5]float64{30.38, 29.22, 0.00, 0.00, 0.00}},
		{22.0, [5]float64{1003, 501, 0, 0, 0}, [5]float64{19.48, 17.99, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1003, 502, 0, 0, 0}, [5]float64{17.19, 16.84, 0.00, 0.00, 0.00}},
		{21.0, [5]float64{1003, 501, 0, 0, 0}, [5]float64{7.18, 6.86, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1003, 501, 0, 0, 0}, [5]float64{5.59, 3.93, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1003, 501, 0, 0, 0}, [5]float64{2.54, 1.19, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1003, 544, 0, 0, 0}, [5]float64{2.23, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1003, 544, 0, 0, 0}, [5]float64{1.50, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1003, 544, 0, 0, 0}, [5]float64{0.33, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{284, 544, 0, 0, 0}, [5]float64{0.00, 0.15, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{284, 544, 0, 0, 0}, [5]float64{0.00, 0.06, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{284, 544, 0, 0, 0}, [5]float64{0.00, 0.07, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{284, 544, 0, 0, 0}, [5]float64{0.00, 0.11, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{284, 544, 0, 0, 0}, [5]float64{0.00, 1.77, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{284, 544, 0, 0, 0}, [5]float64{0.00, 5.53, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{284, 575, 0, 0, 0}, [5]float64{0.00, 5.78, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{284, 606, 0, 0, 0}, [5]float64{0.44, 6.06, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{280, 916, 0, 0, 0}, [5]float64{4.00, 9.66, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{280, 1134, 0, 0, 0}, [5]float64{11.72, 13.49, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{294, 1196, 0, 0, 0}, [5]float64{13.52, 14.85, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{336, 1383, 0, 0, 0}, [5]float64{20.75, 16.11, 0.00, 0.00, 0.00}},
		{17.0, [5]float64{461, 1913, 0, 0, 0}, [5]float64{27.33, 22.10, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{476, 1966, 0, 0, 0}, [5]float64{28.24, 23.70, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{491, 1997, 0, 0, 0}, [5]float64{29.18, 25.07, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{498, 2001, 0, 0, 0}, [5]float64{29.66, 24.91, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{499, 2009, 0, 0, 0}, [5]float64{27.84, 24.58, 0.00, 0.00, 0.00}},
		{26.0, [5]float64{500, 2009, 0, 0, 0}, [5]float64{12.22, 12.76, 0.00, 0.00, 0.00}},
		{19.0, [5]float64{501, 2008, 0, 0, 0}, [5]float64{6.70, 5.25, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{501, 2008, 0, 0, 0}, [5]float64{5.85, 5.01, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{501, 2008, 0, 0, 0}, [5]float64{2.26, 1.48, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{501, 2008, 0, 0, 0}, [5]float64{1.59, 1.24, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{501, 2008, 0, 0, 0}, [5]float64{0.80, 0.52, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{502, 2008, 0, 0, 0}, [5]float64{0.39, 0.33, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{502, 2008, 0, 0, 0}, [5]float64{0.19, 0.18, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{502, 2008, 0, 0, 0}, [5]float64{0.00, 0.09, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{502, 2008, 0, 0, 0}, [5]float64{0.00, 0.05, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{502, 2008, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{502, 2008, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	219: { // A9
		{1.0, [5]float64{1303, 2431, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1303, 2431, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1303, 2431, 0, 0, 0}, [5]float64{0.88, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1302, 2431, 0, 0, 0}, [5]float64{1.79, 1.29, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1302, 2449, 0, 0, 0}, [5]float64{3.62, 3.67, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1301, 2449, 0, 0, 0}, [5]float64{7.35, 5.76, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1301, 2449, 0, 0, 0}, [5]float64{9.31, 9.02, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1301, 2449, 0, 0, 0}, [5]float64{18.90, 14.36, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1301, 2449, 0, 0, 0}, [5]float64{18.93, 16.76, 0.00, 0.00, 0.00}},
		{45.0, [5]float64{1295, 2449, 0, 0, 0}, [5]float64{20.68, 18.19, 0.00, 0.00, 0.00}},
		{20.0, [5]float64{1169, 2449, 0, 0, 0}, [5]float64{21.57, 18.87, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1137, 2431, 0, 0, 0}, [5]float64{21.82, 19.13, 0.00, 0.00, 0.00}},
		{132.0, [5]float64{304, 1531, 0, 0, 0}, [5]float64{31.00, 21.28, 0.00, 0.00, 0.00}},
		{12.0, [5]float64{252, 1449, 0, 0, 0}, [5]float64{12.16, 21.91, 0.00, 0.00, 0.00}},
		{14.0, [5]float64{251, 1354, 0, 0, 0}, [5]float64{0.67, 22.76, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{251, 1347, 0, 0, 0}, [5]float64{0.00, 22.83, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{251, 1334, 0, 0, 0}, [5]float64{0.00, 22.96, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{251, 1308, 0, 0, 0}, [5]float64{0.00, 10.28, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{251, 1308, 0, 0, 0}, [5]float64{0.00, 3.93, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{251, 1308, 0, 0, 0}, [5]float64{0.00, 0.46, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{251, 1308, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{251, 1308, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	220: { // B0
		{1.0, [5]float64{2603, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2603, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2603, 0, 0, 0, 0}, [5]float64{1.02, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2597, 0, 0, 0, 0}, [5]float64{3.97, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2597, 0, 0, 0, 0}, [5]float64{12.47, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2602, 0, 0, 0, 0}, [5]float64{22.56, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2606, 0, 0, 0, 0}, [5]float64{30.76, 0.00, 0.00, 0.00, 0.00}},
		{104.0, [5]float64{2607, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{29.0, [5]float64{2605, 0, 0, 0, 0}, [5]float64{15.71, 0.00, 0.00, 0.00, 0.00}},
		{17.0, [5]float64{2605, 0, 0, 0, 0}, [5]float64{6.63, 0.00, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{2605, 0, 0, 0, 0}, [5]float64{2.08, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{2605, 0, 0, 0, 0}, [5]float64{0.31, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2605, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2605, 0, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	221: { // B1
		{1.0, [5]float64{1421, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1421, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1421, 0, 0, 0, 0}, [5]float64{1.69, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1416, 0, 0, 0, 0}, [5]float64{15.65, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1473, 0, 0, 0, 0}, [5]float64{29.09, 0.00, 0.00, 0.00, 0.00}},
		{87.0, [5]float64{2986, 0, 0, 0, 0}, [5]float64{16.04, 0.00, 0.00, 0.00, 0.00}},
		{49.0, [5]float64{3837, 0, 0, 0, 0}, [5]float64{8.91, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3868, 0, 0, 0, 0}, [5]float64{9.53, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3879, 0, 0, 0, 0}, [5]float64{9.96, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3822, 0, 0, 0, 0}, [5]float64{8.75, 0.00, 0.00, 0.00, 0.00}},
		{35.0, [5]float64{3070, 0, 0, 0, 0}, [5]float64{14.74, 0.00, 0.00, 0.00, 0.00}},
		{71.0, [5]float64{1546, 0, 0, 0, 0}, [5]float64{27.52, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1443, 0, 0, 0, 0}, [5]float64{29.51, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1418, 0, 0, 0, 0}, [5]float64{30.74, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{61.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{15.11, 0.00, 0.00, 0.00, 0.00}},
		{32.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{6.24, 0.00, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{3.50, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{1.53, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{0.51, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{0.11, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1407, 0, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	222: { // B2
		{1.0, [5]float64{1406, 260, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1406, 260, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1406, 260, 0, 0, 0}, [5]float64{1.17, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1406, 260, 0, 0, 0}, [5]float64{3.34, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1406, 260, 0, 0, 0}, [5]float64{11.39, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1409, 260, 0, 0, 0}, [5]float64{21.47, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1412, 260, 0, 0, 0}, [5]float64{30.32, 0.00, 0.00, 0.00, 0.00}},
		{108.0, [5]float64{1406, 260, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1414, 260, 0, 0, 0}, [5]float64{30.18, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1443, 260, 0, 0, 0}, [5]float64{26.56, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1471, 260, 0, 0, 0}, [5]float64{23.98, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1562, 260, 0, 0, 0}, [5]float64{16.52, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1869, 260, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1869, 260, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1869, 260, 0, 0, 0}, [5]float64{24.59, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{2266, 260, 0, 0, 0}, [5]float64{25.25, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2395, 260, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2395, 260, 0, 0, 0}, [5]float64{26.78, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2468, 260, 0, 0, 0}, [5]float64{26.16, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2507, 260, 0, 0, 0}, [5]float64{30.23, 0.00, 0.00, 0.00, 0.00}},
		{13.0, [5]float64{2508, 260, 0, 0, 0}, [5]float64{30.07, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2487, 260, 0, 0, 0}, [5]float64{22.17, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2461, 260, 0, 0, 0}, [5]float64{17.31, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1767, 260, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1767, 260, 0, 0, 0}, [5]float64{16.98, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1719, 260, 0, 0, 0}, [5]float64{21.73, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1707, 260, 0, 0, 0}, [5]float64{29.06, 0.00, 0.00, 0.00, 0.00}},
		{33.0, [5]float64{1707, 260, 0, 0, 0}, [5]float64{29.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1707, 260, 0, 0, 0}, [5]float64{29.00, 0.30, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1707, 257, 0, 0, 0}, [5]float64{29.00, 0.32, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{1707, 259, 0, 0, 0}, [5]float64{28.98, 2.51, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1707, 298, 0, 0, 0}, [5]float64{28.97, 5.09, 0.00, 0.00, 0.00}},
		{33.0, [5]float64{1707, 573, 0, 0, 0}, [5]float64{28.91, 5.19, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1707, 601, 0, 0, 0}, [5]float64{28.90, 5.10, 0.00, 0.00, 0.00}},
		{21.0, [5]float64{1707, 601, 0, 0, 0}, [5]float64{28.87, 4.12, 0.00, 0.00, 0.00}},
		{68.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{12.02, 2.07, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{11.03, 2.01, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{7.19, 1.22, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{6.23, 0.79, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{5.52, 0.66, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{3.59, 0.63, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{2.99, 0.44, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{2.65, 0.25, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1708, 603, 0, 0, 0}, [5]float64{1.91, 0.15, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1708, 600, 0, 0, 0}, [5]float64{1.38, 0.17, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1708, 600, 0, 0, 0}, [5]float64{1.18, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1708, 600, 0, 0, 0}, [5]float64{0.85, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1709, 600, 0, 0, 0}, [5]float64{0.10, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1709, 600, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1709, 600, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	223: { // B3
		{1.0, [5]float64{3103, 615, 1416, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3103, 615, 1416, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3103, 615, 1416, 0, 0}, [5]float64{1.30, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3102, 615, 1416, 0, 0}, [5]float64{2.81, 1.30, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3100, 612, 1416, 0, 0}, [5]float64{6.04, 1.89, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3103, 605, 1416, 0, 0}, [5]float64{12.93, 3.95, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3105, 602, 1416, 0, 0}, [5]float64{14.29, 5.73, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3114, 596, 1416, 0, 0}, [5]float64{21.35, 8.15, 0.00, 0.00, 0.00}},
		{98.0, [5]float64{3112, 599, 1416, 0, 0}, [5]float64{20.72, 8.15, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3104, 599, 1416, 0, 0}, [5]float64{17.08, 8.15, 0.67, 0.00, 0.00}},
		{3.0, [5]float64{3083, 599, 1422, 0, 0}, [5]float64{9.57, 8.15, 6.19, 0.00, 0.00}},
		{1.0, [5]float64{3076, 599, 1418, 0, 0}, [5]float64{7.88, 8.15, 9.20, 0.00, 0.00}},
		{1.0, [5]float64{3076, 599, 1415, 0, 0}, [5]float64{5.01, 8.15, 13.68, 0.00, 0.00}},
		{1.0, [5]float64{3075, 599, 1412, 0, 0}, [5]float64{1.54, 8.15, 20.35, 0.00, 0.00}},
		{1.0, [5]float64{3075, 599, 1409, 0, 0}, [5]float64{0.00, 8.15, 23.14, 0.00, 0.00}},
		{2.0, [5]float64{3075, 599, 1404, 0, 0}, [5]float64{0.00, 8.15, 29.94, 0.00, 0.00}},
		{9.0, [5]float64{3075, 599, 1404, 0, 0}, [5]float64{0.00, 8.15, 31.00, 0.00, 0.00}},
		{99.0, [5]float64{3075, 602, 1404, 0, 0}, [5]float64{0.00, 8.15, 30.83, 0.00, 0.00}},
		{3.0, [5]float64{3075, 602, 1404, 0, 0}, [5]float64{0.00, 7.73, 30.82, 0.00, 0.00}},
		{31.0, [5]float64{3075, 601, 1403, 0, 0}, [5]float64{0.00, 4.46, 18.58, 0.00, 0.00}},
		{5.0, [5]float64{3075, 600, 1403, 0, 0}, [5]float64{0.00, 4.08, 15.24, 0.00, 0.00}},
		{19.0, [5]float64{3075, 604, 1403, 0, 0}, [5]float64{0.00, 1.89, 7.17, 0.00, 0.00}},
		{8.0, [5]float64{3075, 604, 1403, 0, 0}, [5]float64{0.00, 0.94, 5.22, 0.00, 0.00}},
		{2.0, [5]float64{3075, 604, 1403, 0, 0}, [5]float64{0.00, 0.79, 3.99, 0.00, 0.00}},
		{5.0, [5]float64{3075, 602, 1403, 0, 0}, [5]float64{0.00, 0.25, 2.03, 0.00, 0.00}},
		{1.0, [5]float64{3075, 602, 1403, 0, 0}, [5]float64{0.00, 0.00, 1.51, 0.00, 0.00}},
		{3.0, [5]float64{3075, 602, 1403, 0, 0}, [5]float64{0.00, 0.00, 0.63, 0.00, 0.00}},
		{1.0, [5]float64{3075, 602, 1404, 0, 0}, [5]float64{0.00, 0.00, 0.70, 0.00, 0.00}},
		{1.0, [5]float64{3075, 602, 1404, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{3075, 602, 1404, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	224: { // B4
		{1.0, [5]float64{1828, 1620, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1828, 1620, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1828, 1620, 0, 0, 0}, [5]float64{2.16, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1828, 1620, 0, 0, 0}, [5]float64{8.65, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1828, 1620, 0, 0, 0}, [5]float64{16.32, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1823, 1620, 0, 0, 0}, [5]float64{24.15, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1855, 1620, 0, 0, 0}, [5]float64{30.20, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1825, 1620, 0, 0, 0}, [5]float64{19.76, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1835, 1620, 0, 0, 0}, [5]float64{16.78, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1844, 1620, 0, 0, 0}, [5]float64{18.82, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1805, 1620, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1841, 1620, 0, 0, 0}, [5]float64{24.95, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1829, 1620, 0, 0, 0}, [5]float64{25.50, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1829, 1620, 0, 0, 0}, [5]float64{25.50, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1829, 1620, 0, 0, 0}, [5]float64{25.50, 15.24, 0.00, 0.00, 0.00}},
		{106.0, [5]float64{1834, 357, 0, 0, 0}, [5]float64{25.31, 21.67, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1834, 325, 0, 0, 0}, [5]float64{25.30, 21.15, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1834, 353, 0, 0, 0}, [5]float64{25.29, 21.63, 0.00, 0.00, 0.00}},
		{104.0, [5]float64{1839, 1574, 0, 0, 0}, [5]float64{25.10, 14.87, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1839, 1617, 0, 0, 0}, [5]float64{25.09, 19.19, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1839, 1617, 0, 0, 0}, [5]float64{25.09, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1839, 1617, 0, 0, 0}, [5]float64{25.09, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1805, 1617, 0, 0, 0}, [5]float64{29.51, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1853, 1617, 0, 0, 0}, [5]float64{16.82, 0.00, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{1807, 1617, 0, 0, 0}, [5]float64{14.98, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1858, 1617, 0, 0, 0}, [5]float64{30.74, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1825, 1617, 0, 0, 0}, [5]float64{25.39, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1838, 1617, 0, 0, 0}, [5]float64{25.34, 0.00, 0.00, 0.00, 0.00}},
		{31.0, [5]float64{1832, 1617, 0, 0, 0}, [5]float64{15.28, 0.00, 0.00, 0.00, 0.00}},
		{28.0, [5]float64{1834, 1617, 0, 0, 0}, [5]float64{5.56, 0.00, 0.00, 0.00, 0.00}},
		{11.0, [5]float64{1834, 1617, 0, 0, 0}, [5]float64{1.66, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1834, 1617, 0, 0, 0}, [5]float64{0.76, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1834, 1617, 0, 0, 0}, [5]float64{0.29, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1839, 1617, 0, 0, 0}, [5]float64{0.27, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1839, 1617, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1839, 1617, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	225: { // B5
		{1.0, [5]float64{982, 3508, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{982, 3508, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{982, 3508, 0, 0, 0}, [5]float64{1.45, 0.56, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{981, 3502, 0, 0, 0}, [5]float64{2.77, 2.12, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{979, 3502, 0, 0, 0}, [5]float64{10.13, 6.67, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{977, 3506, 0, 0, 0}, [5]float64{18.58, 11.02, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{973, 3508, 0, 0, 0}, [5]float64{20.69, 14.16, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{965, 3510, 0, 0, 0}, [5]float64{25.65, 15.72, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{956, 3511, 0, 0, 0}, [5]float64{25.71, 16.56, 0.00, 0.00, 0.00}},
		{65.0, [5]float64{320, 3513, 0, 0, 0}, [5]float64{31.00, 16.18, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{260, 3514, 0, 0, 0}, [5]float64{13.75, 16.12, 0.00, 0.00, 0.00}},
		{16.0, [5]float64{258, 3514, 0, 0, 0}, [5]float64{0.33, 16.03, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{258, 3514, 0, 0, 0}, [5]float64{0.00, 16.03, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{258, 3514, 0, 0, 0}, [5]float64{0.00, 15.99, 0.00, 0.00, 0.00}},
		{34.0, [5]float64{258, 3516, 0, 0, 0}, [5]float64{0.00, 6.12, 0.00, 0.00, 0.00}},
		{18.0, [5]float64{258, 3517, 0, 0, 0}, [5]float64{0.00, 1.32, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{258, 3517, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{258, 3517, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	226: { // B6
		{1.0, [5]float64{3503, 1821, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3503, 1821, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3503, 1821, 0, 0, 0}, [5]float64{1.41, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3495, 1821, 0, 0, 0}, [5]float64{7.11, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{3507, 1821, 0, 0, 0}, [5]float64{22.97, 0.00, 0.00, 0.00, 0.00}},
		{14.0, [5]float64{3506, 1821, 0, 0, 0}, [5]float64{24.65, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3500, 1821, 0, 0, 0}, [5]float64{20.49, 1.01, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3486, 1821, 0, 0, 0}, [5]float64{14.16, 4.47, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3452, 1821, 0, 0, 0}, [5]float64{13.05, 9.41, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3493, 1817, 0, 0, 0}, [5]float64{0.00, 12.47, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{3493, 1805, 0, 0, 0}, [5]float64{0.00, 29.04, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{3493, 1804, 0, 0, 0}, [5]float64{0.00, 30.86, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3493, 1804, 0, 0, 0}, [5]float64{0.00, 19.87, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3493, 1803, 0, 0, 0}, [5]float64{11.54, 15.94, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3501, 1803, 0, 0, 0}, [5]float64{17.05, 10.26, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3506, 1803, 0, 0, 0}, [5]float64{20.73, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3510, 1803, 0, 0, 0}, [5]float64{25.20, 0.00, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{3513, 1803, 0, 0, 0}, [5]float64{26.47, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3512, 1803, 0, 0, 0}, [5]float64{25.20, 0.47, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3511, 1802, 0, 0, 0}, [5]float64{23.99, 1.62, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3509, 1802, 0, 0, 0}, [5]float64{22.84, 3.61, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3505, 1802, 0, 0, 0}, [5]float64{18.34, 8.04, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3495, 1802, 0, 0, 0}, [5]float64{11.82, 18.57, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3491, 1802, 0, 0, 0}, [5]float64{9.49, 22.02, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3469, 1803, 0, 0, 0}, [5]float64{0.00, 26.11, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3469, 1803, 0, 0, 0}, [5]float64{0.00, 30.96, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{3469, 1809, 0, 0, 0}, [5]float64{0.00, 25.78, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3469, 1817, 0, 0, 0}, [5]float64{0.00, 13.98, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3469, 1821, 0, 0, 0}, [5]float64{12.74, 10.30, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3497, 1823, 0, 0, 0}, [5]float64{18.34, 2.98, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3500, 1824, 0, 0, 0}, [5]float64{19.98, 1.61, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3503, 1822, 0, 0, 0}, [5]float64{21.77, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3509, 1822, 0, 0, 0}, [5]float64{25.86, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3503, 1822, 0, 0, 0}, [5]float64{22.12, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3498, 1822, 0, 0, 0}, [5]float64{18.93, 1.17, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3488, 1822, 0, 0, 0}, [5]float64{13.86, 4.96, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3463, 1822, 0, 0, 0}, [5]float64{13.13, 10.23, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3463, 1817, 0, 0, 0}, [5]float64{0.00, 13.90, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{3463, 1808, 0, 0, 0}, [5]float64{0.00, 25.65, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{3463, 1804, 0, 0, 0}, [5]float64{0.00, 31.00, 0.00, 0.00, 0.00}},
		{26.0, [5]float64{3463, 1805, 0, 0, 0}, [5]float64{0.00, 15.96, 0.00, 0.00, 0.00}},
		{16.0, [5]float64{3463, 1805, 0, 0, 0}, [5]float64{0.00, 6.83, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{3463, 1805, 0, 0, 0}, [5]float64{0.00, 2.03, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{3463, 1805, 0, 0, 0}, [5]float64{0.00, 0.33, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3463, 1805, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{3463, 1805, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	227: { // B7
		{1.0, [5]float64{2908, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2908, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2908, 0, 0, 0, 0}, [5]float64{1.74, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2910, 0, 0, 0, 0}, [5]float64{11.64, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{2914, 0, 0, 0, 0}, [5]float64{29.00, 0.00, 0.00, 0.00, 0.00}},
		{31.0, [5]float64{2912, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2910, 0, 0, 0, 0}, [5]float64{25.56, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2907, 0, 0, 0, 0}, [5]float64{12.20, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2907, 0, 0, 0, 0}, [5]float64{4.34, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2907, 0, 0, 0, 0}, [5]float64{0.70, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3884, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{3884, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3884, 0, 0, 0, 0}, [5]float64{1.15, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3903, 0, 0, 0, 0}, [5]float64{4.60, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{3897, 0, 0, 0, 0}, [5]float64{13.53, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{3918, 0, 0, 0, 0}, [5]float64{28.13, 0.00, 0.00, 0.00, 0.00}},
		{33.0, [5]float64{3916, 0, 0, 0, 0}, [5]float64{27.42, 0.00, 0.00, 0.00, 0.00}},
		{29.0, [5]float64{3918, 0, 0, 0, 0}, [5]float64{13.85, 0.00, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{3916, 0, 0, 0, 0}, [5]float64{6.25, 0.00, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{3917, 0, 0, 0, 0}, [5]float64{2.02, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{3917, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{3917, 0, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	228: { // B8
		{1.0, [5]float64{2085, 3618, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2085, 3618, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2085, 3618, 0, 0, 0}, [5]float64{0.84, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{2086, 3618, 0, 0, 0}, [5]float64{2.40, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2086, 3618, 0, 0, 0}, [5]float64{9.29, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{2086, 3618, 0, 0, 0}, [5]float64{18.76, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{2082, 3618, 0, 0, 0}, [5]float64{28.04, 0.00, 0.00, 0.00, 0.00}},
		{24.0, [5]float64{1961, 3618, 0, 0, 0}, [5]float64{28.72, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1956, 3618, 0, 0, 0}, [5]float64{28.75, 4.55, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{1946, 3621, 0, 0, 0}, [5]float64{28.82, 9.15, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1925, 3621, 0, 0, 0}, [5]float64{28.76, 14.01, 0.00, 0.00, 0.00}},
		{17.0, [5]float64{1808, 3619, 0, 0, 0}, [5]float64{28.55, 22.37, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{1752, 3501, 0, 0, 0}, [5]float64{28.53, 23.14, 0.00, 0.00, 0.00}},
		{23.0, [5]float64{1654, 3280, 0, 0, 0}, [5]float64{29.03, 24.86, 0.00, 0.00, 0.00}},
		{35.0, [5]float64{1505, 3005, 0, 0, 0}, [5]float64{30.12, 28.94, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1501, 3001, 0, 0, 0}, [5]float64{30.15, 28.94, 0.00, 0.00, 0.00}},
		{76.0, [5]float64{1330, 2709, 0, 0, 0}, [5]float64{31.00, 28.72, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1325, 2686, 0, 0, 0}, [5]float64{29.86, 28.62, 0.00, 0.00, 0.00}},
		{23.0, [5]float64{1305, 2611, 0, 0, 0}, [5]float64{25.88, 24.51, 0.00, 0.00, 0.00}},
		{39.0, [5]float64{1305, 2609, 0, 0, 0}, [5]float64{24.36, 24.54, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1305, 2610, 0, 0, 0}, [5]float64{24.17, 22.01, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{1305, 2611, 0, 0, 0}, [5]float64{15.99, 17.71, 0.00, 0.00, 0.00}},
		{21.0, [5]float64{1305, 2612, 0, 0, 0}, [5]float64{6.71, 6.24, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1305, 2612, 0, 0, 0}, [5]float64{2.79, 2.47, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1305, 2612, 0, 0, 0}, [5]float64{1.24, 0.97, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1305, 2612, 0, 0, 0}, [5]float64{0.77, 0.71, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1305, 2613, 0, 0, 0}, [5]float64{0.49, 0.82, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1306, 2613, 0, 0, 0}, [5]float64{0.49, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1306, 2613, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1306, 2613, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	235: { // C5
		{1.0, [5]float64{716, 2705, 302, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{716, 2705, 302, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{716, 2705, 302, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{716, 2705, 302, 0, 0}, [5]float64{1.28, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{716, 2705, 302, 0, 0}, [5]float64{1.28, 8.05, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{716, 2708, 302, 0, 0}, [5]float64{1.49, 3.89, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{716, 2710, 302, 0, 0}, [5]float64{2.33, 9.25, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{716, 2710, 302, 0, 0}, [5]float64{2.68, 9.99, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{716, 2710, 302, 0, 0}, [5]float64{2.68, 12.78, 0.00, 0.00, 0.00}},
		{31.0, [5]float64{716, 2709, 302, 0, 0}, [5]float64{2.67, 12.87, 0.00, 0.00, 0.00}},
		{1.8, [5]float64{716, 2701, 302, 0, 0}, [5]float64{2.66, 12.40, 0.00, 0.00, 0.00}},
		{1.2, [5]float64{716, 2696, 302, 0, 0}, [5]float64{2.94, 12.11, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{716, 2668, 302, 0, 0}, [5]float64{3.49, 10.62, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{716, 2708, 302, 0, 0}, [5]float64{3.80, 15.95, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{716, 901, 302, 0, 0}, [5]float64{4.13, 0.00, 0.00, 0.00, 0.00}},
		{9.8, [5]float64{716, 901, 302, 0, 0}, [5]float64{9.48, 0.00, 0.00, 0.00, 0.00}},
		{14.2, [5]float64{640, 901, 302, 0, 0}, [5]float64{16.96, 0.00, 0.00, 0.00, 0.00}},
		{0.7, [5]float64{636, 901, 302, 0, 0}, [5]float64{17.48, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{636, 901, 302, 0, 0}, [5]float64{17.56, 13.06, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{620, 902, 302, 0, 0}, [5]float64{19.12, 16.49, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{617, 902, 302, 0, 0}, [5]float64{19.45, 17.28, 2.54, 0.00, 0.00}},
		{4.0, [5]float64{605, 903, 301, 0, 0}, [5]float64{20.82, 20.83, 3.77, 0.00, 0.00}},
		{2.7, [5]float64{597, 903, 301, 0, 0}, [5]float64{21.79, 20.25, 4.93, 0.00, 0.00}},
		{7.3, [5]float64{596, 902, 299, 0, 0}, [5]float64{21.75, 18.76, 10.20, 0.00, 0.00}},
		{4.0, [5]float64{595, 902, 299, 0, 0}, [5]float64{21.72, 17.99, 12.86, 0.00, 0.00}},
		{6.0, [5]float64{594, 900, 299, 0, 0}, [5]float64{21.69, 13.66, 18.22, 0.00, 0.00}},
		{3.0, [5]float64{593, 900, 299, 0, 0}, [5]float64{21.67, 11.90, 18.52, 0.00, 0.00}},
		{14.6, [5]float64{590, 898, 300, 0, 0}, [5]float64{21.59, 3.07, 20.04, 0.00, 0.00}},
		{1.4, [5]float64{586, 898, 300, 0, 0}, [5]float64{21.65, 2.68, 20.20, 0.00, 0.00}},
		{1.0, [5]float64{583, 899, 300, 0, 0}, [5]float64{21.70, 3.61, 20.31, 0.00, 0.00}},
		{1.0, [5]float64{580, 281, 300, 0, 0}, [5]float64{21.75, 0.00, 20.42, 0.00, 0.00}},
		{5.5, [5]float64{563, 281, 300, 0, 0}, [5]float64{22.02, 0.00, 21.05, 0.00, 0.00}},
		{10.5, [5]float64{512, 281, 300, 0, 0}, [5]float64{21.38, 0.00, 22.28, 0.00, 0.00}},
		{1.0, [5]float64{507, 281, 300, 0, 0}, [5]float64{21.32, 0.00, 0.00, 0.00, 0.00}},
		{7.5, [5]float64{471, 281, 300, 0, 0}, [5]float64{20.89, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{472, 281, 300, 0, 0}, [5]float64{21.36, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{421, 281, 300, 0, 0}, [5]float64{21.88, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{453, 281, 300, 0, 0}, [5]float64{19.85, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{453, 281, 300, 0, 0}, [5]float64{19.85, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{447, 281, 300, 0, 0}, [5]float64{19.47, 17.55, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{441, 307, 300, 0, 0}, [5]float64{19.09, 20.06, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{339, 334, 300, 0, 0}, [5]float64{0.00, 22.94, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{339, 361, 300, 0, 0}, [5]float64{0.00, 26.22, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{339, 360, 300, 0, 0}, [5]float64{0.00, 25.45, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{339, 417, 300, 0, 0}, [5]float64{0.00, 16.80, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{339, 424, 300, 0, 0}, [5]float64{0.00, 15.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{339, 406, 300, 0, 0}, [5]float64{0.00, 16.27, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{339, 406, 300, 0, 0}, [5]float64{0.00, 16.28, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{339, 406, 300, 0, 0}, [5]float64{21.31, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{339, 406, 300, 0, 0}, [5]float64{21.32, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{332, 406, 300, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{324, 406, 300, 0, 0}, [5]float64{11.53, 0.00, 0.00, 0.00, 0.00}},
		{0.6, [5]float64{406, 406, 300, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{0.5, [5]float64{406, 406, 300, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{0.6, [5]float64{406, 406, 300, 0, 0}, [5]float64{17.57, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{357, 406, 300, 0, 0}, [5]float64{10.11, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{318, 406, 300, 0, 0}, [5]float64{13.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{313, 406, 300, 0, 0}, [5]float64{22.88, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{302, 406, 300, 0, 0}, [5]float64{25.32, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{323, 406, 300, 0, 0}, [5]float64{23.83, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{289, 406, 300, 0, 0}, [5]float64{23.22, 0.00, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{299, 406, 300, 0, 0}, [5]float64{21.74, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{8.52, 0.00, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.57, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{6.87, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.71, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{6.80, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.67, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{6.79, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.66, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{6.79, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.73, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{6.88, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.73, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{6.77, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.63, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{6.70, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{3.99, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{4.05, 0.00, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{305, 406, 300, 0, 0}, [5]float64{0.68, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{316, 406, 300, 0, 0}, [5]float64{1.23, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{316, 406, 300, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{316, 406, 300, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	240: { // D0
		{1.0, [5]float64{965, 1334, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{965, 1334, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{965, 1334, 0, 0, 0}, [5]float64{3.18, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{968, 1334, 0, 0, 0}, [5]float64{7.64, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{963, 1334, 0, 0, 0}, [5]float64{17.75, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{960, 1334, 0, 0, 0}, [5]float64{20.41, 17.54, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{950, 1340, 0, 0, 0}, [5]float64{31.00, 21.88, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{950, 1346, 0, 0, 0}, [5]float64{30.99, 27.30, 0.00, 0.00, 0.00}},
		{84.0, [5]float64{950, 1341, 0, 0, 0}, [5]float64{30.66, 27.11, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{950, 1341, 0, 0, 0}, [5]float64{30.65, 25.67, 0.00, 0.00, 0.00}},
		{18.0, [5]float64{950, 1342, 0, 0, 0}, [5]float64{17.88, 18.53, 0.00, 0.00, 0.00}},
		{23.0, [5]float64{949, 1342, 0, 0, 0}, [5]float64{8.99, 6.41, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{949, 1342, 0, 0, 0}, [5]float64{5.35, 4.86, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{949, 1342, 0, 0, 0}, [5]float64{4.13, 3.02, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{949, 1342, 0, 0, 0}, [5]float64{2.36, 1.88, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{949, 1342, 0, 0, 0}, [5]float64{1.96, 1.36, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{949, 1342, 0, 0, 0}, [5]float64{0.98, 0.72, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{949, 1342, 0, 0, 0}, [5]float64{0.69, 0.43, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{950, 1342, 0, 0, 0}, [5]float64{0.79, 0.26, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{950, 1344, 0, 0, 0}, [5]float64{0.00, 0.30, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{950, 1344, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{950, 1344, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	241: { // D1
		{1.0, [5]float64{690, 1231, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{690, 1231, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{690, 1231, 0, 0, 0}, [5]float64{3.77, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{697, 1231, 0, 0, 0}, [5]float64{8.78, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1231, 0, 0, 0}, [5]float64{13.38, 12.47, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{705, 1213, 0, 0, 0}, [5]float64{20.40, 17.27, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{707, 1214, 0, 0, 0}, [5]float64{31.00, 25.89, 0.00, 0.00, 0.00}},
		{90.0, [5]float64{701, 1215, 0, 0, 0}, [5]float64{30.46, 20.47, 0.00, 0.00, 0.00}},
		{13.0, [5]float64{701, 1215, 0, 0, 0}, [5]float64{20.16, 19.78, 0.00, 0.00, 0.00}},
		{20.0, [5]float64{701, 1215, 0, 0, 0}, [5]float64{10.69, 10.64, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{701, 1215, 0, 0, 0}, [5]float64{8.29, 6.40, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{701, 1215, 0, 0, 0}, [5]float64{5.59, 4.97, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{701, 1215, 0, 0, 0}, [5]float64{3.42, 2.50, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{701, 1215, 0, 0, 0}, [5]float64{2.09, 1.90, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{700, 1215, 0, 0, 0}, [5]float64{0.78, 0.50, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{700, 1215, 0, 0, 0}, [5]float64{0.61, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{700, 1215, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{700, 1215, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	242: { // D2
		{1.0, [5]float64{714, 1350, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{714, 1350, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{714, 1350, 0, 0, 0}, [5]float64{3.28, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{693, 1350, 0, 0, 0}, [5]float64{4.11, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{692, 1350, 0, 0, 0}, [5]float64{6.72, 4.53, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{690, 1343, 0, 0, 0}, [5]float64{17.89, 14.05, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{704, 1346, 0, 0, 0}, [5]float64{30.98, 22.51, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{704, 1347, 0, 0, 0}, [5]float64{30.98, 25.33, 0.00, 0.00, 0.00}},
		{90.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{31.00, 21.88, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{26.18, 21.63, 0.00, 0.00, 0.00}},
		{19.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{16.55, 12.46, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{12.34, 10.47, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{7.57, 5.50, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{4.82, 4.26, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{3.07, 2.18, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{1.79, 1.56, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{1.37, 1.05, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{0.57, 0.48, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{0.36, 0.25, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1341, 0, 0, 0}, [5]float64{0.38, 0.13, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1342, 0, 0, 0}, [5]float64{0.00, 0.15, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1342, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{701, 1342, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	243: { // D3
		{1.0, [5]float64{699, 1489, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{699, 1489, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{699, 1489, 0, 0, 0}, [5]float64{2.30, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{699, 1489, 0, 0, 0}, [5]float64{4.24, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{699, 1489, 0, 0, 0}, [5]float64{7.83, 10.19, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{699, 1487, 0, 0, 0}, [5]float64{14.47, 11.73, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{706, 1479, 0, 0, 0}, [5]float64{31.00, 23.66, 0.00, 0.00, 0.00}},
		{89.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{30.55, 20.34, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{25.08, 20.04, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{18.04, 13.25, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{13.71, 11.23, 0.00, 0.00, 0.00}},
		{13.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{7.56, 5.50, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{6.04, 4.93, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{2.76, 1.85, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{2.03, 1.61, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{0.59, 0.35, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{0.43, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{701, 1480, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	244: { // D4
		{1.0, [5]float64{794, 1202, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{794, 1202, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{794, 1202, 0, 0, 0}, [5]float64{5.90, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{780, 1202, 0, 0, 0}, [5]float64{15.68, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{775, 1202, 0, 0, 0}, [5]float64{21.73, 18.39, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{773, 1216, 0, 0, 0}, [5]float64{30.98, 26.27, 0.00, 0.00, 0.00}},
		{87.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{31.00, 21.14, 0.00, 0.00, 0.00}},
		{15.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{23.03, 20.36, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{772, 1215, 0, 0, 0}, [5]float64{20.05, 15.47, 0.00, 0.00, 0.00}},
		{20.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{9.23, 7.05, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{6.49, 6.03, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{3.83, 2.93, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{2.96, 2.60, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{0.82, 0.65, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{0.48, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{0.28, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{0.09, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{773, 1215, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	245: { // D5
		{1.0, [5]float64{765, 1354, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{765, 1354, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{765, 1354, 0, 0, 0}, [5]float64{2.21, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{765, 1354, 0, 0, 0}, [5]float64{7.72, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{765, 1354, 0, 0, 0}, [5]float64{14.41, 14.60, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{772, 1348, 0, 0, 0}, [5]float64{31.00, 23.16, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{772, 1346, 0, 0, 0}, [5]float64{30.99, 25.40, 0.00, 0.00, 0.00}},
		{89.0, [5]float64{773, 1342, 0, 0, 0}, [5]float64{30.52, 20.40, 0.00, 0.00, 0.00}},
		{12.0, [5]float64{773, 1341, 0, 0, 0}, [5]float64{23.04, 19.80, 0.00, 0.00, 0.00}},
		{13.0, [5]float64{773, 1341, 0, 0, 0}, [5]float64{16.99, 11.77, 0.00, 0.00, 0.00}},
		{19.0, [5]float64{773, 1341, 0, 0, 0}, [5]float64{6.84, 5.50, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{773, 1341, 0, 0, 0}, [5]float64{2.94, 2.33, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{772, 1342, 0, 0, 0}, [5]float64{0.54, 0.48, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{772, 1342, 0, 0, 0}, [5]float64{0.39, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{772, 1342, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{772, 1342, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	246: { // D6
		{1.0, [5]float64{780, 1476, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{780, 1476, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{780, 1476, 0, 0, 0}, [5]float64{4.49, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{758, 1476, 0, 0, 0}, [5]float64{5.03, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{758, 1476, 0, 0, 0}, [5]float64{7.82, 4.24, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{758, 1470, 0, 0, 0}, [5]float64{12.17, 10.89, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{757, 1473, 0, 0, 0}, [5]float64{18.93, 14.95, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{771, 1479, 0, 0, 0}, [5]float64{31.00, 24.40, 0.00, 0.00, 0.00}},
		{91.0, [5]float64{773, 1480, 0, 0, 0}, [5]float64{30.28, 23.33, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1480, 0, 0, 0}, [5]float64{29.31, 23.32, 0.00, 0.00, 0.00}},
		{27.0, [5]float64{773, 1480, 0, 0, 0}, [5]float64{12.10, 11.78, 0.00, 0.00, 0.00}},
		{13.0, [5]float64{773, 1481, 0, 0, 0}, [5]float64{7.91, 5.91, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{773, 1481, 0, 0, 0}, [5]float64{6.39, 5.31, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{773, 1481, 0, 0, 0}, [5]float64{3.03, 2.14, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1481, 0, 0, 0}, [5]float64{2.27, 1.88, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{773, 1480, 0, 0, 0}, [5]float64{0.54, 0.28, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{773, 1480, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{773, 1480, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	247: { // D7
		{1.0, [5]float64{895, 1220, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{895, 1220, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{895, 1220, 0, 0, 0}, [5]float64{5.77, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{862, 1220, 0, 0, 0}, [5]float64{4.45, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{856, 1220, 0, 0, 0}, [5]float64{11.19, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{852, 1220, 0, 0, 0}, [5]float64{17.76, 8.60, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{854, 1212, 0, 0, 0}, [5]float64{19.85, 18.94, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{861, 1214, 0, 0, 0}, [5]float64{31.00, 27.18, 0.00, 0.00, 0.00}},
		{92.0, [5]float64{856, 1214, 0, 0, 0}, [5]float64{27.69, 25.70, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{856, 1214, 0, 0, 0}, [5]float64{27.59, 23.24, 0.00, 0.00, 0.00}},
		{25.0, [5]float64{855, 1215, 0, 0, 0}, [5]float64{13.95, 10.07, 0.00, 0.00, 0.00}},
		{11.0, [5]float64{855, 1215, 0, 0, 0}, [5]float64{7.51, 6.97, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{855, 1215, 0, 0, 0}, [5]float64{6.35, 5.12, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{855, 1215, 0, 0, 0}, [5]float64{3.17, 3.06, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{855, 1215, 0, 0, 0}, [5]float64{2.40, 1.78, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{855, 1215, 0, 0, 0}, [5]float64{0.73, 0.78, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{856, 1215, 0, 0, 0}, [5]float64{0.49, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{856, 1215, 0, 0, 0}, [5]float64{0.33, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{856, 1215, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{856, 1215, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	248: { // D8
		{1.0, [5]float64{878, 1360, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{878, 1360, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{878, 1360, 0, 0, 0}, [5]float64{2.56, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{853, 1360, 0, 0, 0}, [5]float64{3.07, 0.00, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{850, 1360, 0, 0, 0}, [5]float64{9.08, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{849, 1360, 0, 0, 0}, [5]float64{15.60, 8.53, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{852, 1344, 0, 0, 0}, [5]float64{17.90, 17.35, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{860, 1346, 0, 0, 0}, [5]float64{31.00, 26.07, 0.00, 0.00, 0.00}},
		{90.0, [5]float64{856, 1342, 0, 0, 0}, [5]float64{30.44, 23.89, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{856, 1342, 0, 0, 0}, [5]float64{27.55, 23.80, 0.00, 0.00, 0.00}},
		{25.0, [5]float64{856, 1342, 0, 0, 0}, [5]float64{14.76, 9.98, 0.00, 0.00, 0.00}},
		{13.0, [5]float64{855, 1342, 0, 0, 0}, [5]float64{7.33, 6.35, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{855, 1342, 0, 0, 0}, [5]float64{6.94, 5.66, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{855, 1342, 0, 0, 0}, [5]float64{2.40, 2.24, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{855, 1342, 0, 0, 0}, [5]float64{2.10, 1.59, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{854, 1342, 0, 0, 0}, [5]float64{0.47, 0.41, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{854, 1342, 0, 0, 0}, [5]float64{0.32, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{854, 1342, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{854, 1342, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	249: { // D9
		{1.0, [5]float64{865, 1486, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{865, 1486, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{865, 1486, 0, 0, 0}, [5]float64{3.72, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{861, 1486, 0, 0, 0}, [5]float64{5.67, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{858, 1486, 0, 0, 0}, [5]float64{8.64, 7.37, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{850, 1478, 0, 0, 0}, [5]float64{20.07, 16.23, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{860, 1480, 0, 0, 0}, [5]float64{31.00, 25.01, 0.00, 0.00, 0.00}},
		{92.0, [5]float64{856, 1480, 0, 0, 0}, [5]float64{29.91, 23.41, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{856, 1480, 0, 0, 0}, [5]float64{27.99, 23.37, 0.00, 0.00, 0.00}},
		{37.0, [5]float64{856, 1481, 0, 0, 0}, [5]float64{8.21, 7.04, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{856, 1481, 0, 0, 0}, [5]float64{7.68, 5.73, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{856, 1481, 0, 0, 0}, [5]float64{3.01, 2.51, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{856, 1482, 0, 0, 0}, [5]float64{2.68, 1.97, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{856, 1483, 0, 0, 0}, [5]float64{0.49, 0.59, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{856, 1483, 0, 0, 0}, [5]float64{0.35, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{856, 1483, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{856, 1483, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	250: { // D10
		{1.0, [5]float64{936, 1211, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{936, 1211, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{936, 1211, 0, 0, 0}, [5]float64{5.07, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{937, 1211, 0, 0, 0}, [5]float64{20.72, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{940, 1211, 0, 0, 0}, [5]float64{22.92, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{944, 1211, 0, 0, 0}, [5]float64{25.35, 21.74, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{950, 1214, 0, 0, 0}, [5]float64{31.00, 25.90, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{950, 1215, 0, 0, 0}, [5]float64{31.00, 28.27, 0.00, 0.00, 0.00}},
		{87.0, [5]float64{949, 1215, 0, 0, 0}, [5]float64{30.65, 27.86, 0.00, 0.00, 0.00}},
		{34.0, [5]float64{948, 1216, 0, 0, 0}, [5]float64{11.60, 11.60, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{948, 1216, 0, 0, 0}, [5]float64{10.05, 7.80, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{948, 1216, 0, 0, 0}, [5]float64{4.57, 3.53, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{948, 1216, 0, 0, 0}, [5]float64{3.30, 3.01, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{948, 1217, 0, 0, 0}, [5]float64{2.80, 3.75, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{948, 1217, 0, 0, 0}, [5]float64{2.38, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{948, 1217, 0, 0, 0}, [5]float64{0.93, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{949, 1217, 0, 0, 0}, [5]float64{1.11, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{949, 1217, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{949, 1217, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	251: { // D11
		{1.0, [5]float64{953, 1470, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{953, 1470, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{953, 1470, 0, 0, 0}, [5]float64{8.48, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{945, 1470, 0, 0, 0}, [5]float64{8.22, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{946, 1470, 0, 0, 0}, [5]float64{11.58, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{947, 1470, 0, 0, 0}, [5]float64{16.30, 15.30, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{947, 1473, 0, 0, 0}, [5]float64{22.95, 17.90, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{948, 1477, 0, 0, 0}, [5]float64{26.67, 24.51, 0.00, 0.00, 0.00}},
		{2.0, [5]float64{949, 1478, 0, 0, 0}, [5]float64{31.00, 24.94, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{950, 1479, 0, 0, 0}, [5]float64{30.89, 26.03, 0.00, 0.00, 0.00}},
		{81.0, [5]float64{950, 1479, 0, 0, 0}, [5]float64{29.12, 25.79, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{950, 1479, 0, 0, 0}, [5]float64{28.99, 22.86, 0.00, 0.00, 0.00}},
		{19.0, [5]float64{949, 1480, 0, 0, 0}, [5]float64{18.08, 15.61, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{949, 1479, 0, 0, 0}, [5]float64{15.19, 11.98, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{949, 1479, 0, 0, 0}, [5]float64{9.92, 8.86, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{949, 1479, 0, 0, 0}, [5]float64{6.83, 5.17, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{949, 1479, 0, 0, 0}, [5]float64{4.19, 3.81, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{949, 1479, 0, 0, 0}, [5]float64{2.90, 2.10, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{949, 1479, 0, 0, 0}, [5]float64{1.24, 1.16, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{949, 1478, 0, 0, 0}, [5]float64{0.53, 0.29, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{949, 1478, 0, 0, 0}, [5]float64{0.00, 0.19, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{949, 1478, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{949, 1478, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
	252: { // M0
		{1.0, [5]float64{1732, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1732, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1732, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1784, 0, 0, 0, 0}, [5]float64{19.71, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1810, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1826, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1786, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1810, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1789, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1836, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1818, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1838, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1822, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1841, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1799, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1815, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1778, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1822, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1784, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1795, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1851, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1816, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1803, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{11.0, [5]float64{1837, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{14.0, [5]float64{1798, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{1834, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1790, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1832, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1818, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1838, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1826, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1785, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1803, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1795, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1828, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1802, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{1795, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{9.0, [5]float64{1826, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1777, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1821, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1804, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{24.0, [5]float64{1812, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1787, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1797, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1782, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1796, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1844, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1810, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1787, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{5.0, [5]float64{1808, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1797, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1818, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{8.0, [5]float64{1765, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{7.0, [5]float64{1806, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{6.0, [5]float64{1791, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1802, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1799, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{10.0, [5]float64{1782, 0, 0, 0, 0}, [5]float64{31.00, 0.00, 0.00, 0.00, 0.00}},
		{503.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{6.77, 0.00, 0.00, 0.00, 0.00}},
		{20.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{4.28, 0.00, 0.00, 0.00, 0.00}},
		{113.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{3.57, 0.00, 0.00, 0.00, 0.00}},
		{13.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{2.44, 0.00, 0.00, 0.00, 0.00}},
		{107.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{2.53, 0.00, 0.00, 0.00, 0.00}},
		{12.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{1.33, 0.00, 0.00, 0.00, 0.00}},
		{84.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{1.30, 0.00, 0.00, 0.00, 0.00}},
		{4.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{0.98, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1783, 0, 0, 0, 0}, [5]float64{0.41, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1794, 0, 0, 0, 0}, [5]float64{0.70, 0.00, 0.00, 0.00, 0.00}},
		{1.0, [5]float64{1794, 0, 0, 0, 0}, [5]float64{0.00, 0.00, 0.00, 0.00, 0.00}},
		{3.0, [5]float64{1794, 0, 0, 0, 0}, [5]float64{0, 0, 0, 0, 0}},
	},
}

// contour2 plays a tone effect traced again: each point glided to in a
// straight line
func contour2(name string, p []fx2Point) sound {
	s := effect(name)
	for _, q := range p {
		var f frame
		f.freq = q.freq
		f.vol = q.vol
		s.parts = append(s.parts, hold(q.ms, q.ms, f)...)
	}
	return s
}

// FX2-END
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
	// the DTMF tones' two frequencies in freqStepHz steps: the chip's, measured
	// on the second VoxProbe recording, which are a few hertz from the
	// standard ones (1477Hz is 1468.75)
	dtmfs := [12][2]int{{215, 304}, {159, 275}, {159, 304}, {159, 335}, {175, 275}, {175, 304},
		{175, 335}, {194, 275}, {194, 304}, {194, 335}, {215, 275}, {215, 335}}
	for i, name := range effectNames {
		code := 200 + i
		switch {
		case fxvPoints[code] != nil:
			e[i] = contourVoiced(name, fxvPoints[code])
		case fx2Points[code] != nil:
			e[i] = toneNoise(contour2(name, fx2Points[code]), code)
		case fxPoints[code] != nil:
			e[i] = contour(name, fxPoints[code], code)
		case code >= 240 && code <= 251:
			e[i] = dtmf(name, float64(dtmfs[code-240][0])*freqStepHz, float64(dtmfs[code-240][1])*freqStepHz)
		case code == 253:
			e[i] = m1()
		case code == 238:
			e[i] = cNoise(name, c8DB)
		case code == 239:
			e[i] = cNoise(name, c9DB)
		}
	}
	return
}()

// the noise refitted at the chip's own rate. the noise sounds were fitted while
// this library ran at 8192 samples a second; at chipRate the same values make
// different noise (it is a frequency jittered every sample, so its spread, and
// where it folds back around half the rate, move with the rate), and those
// sounds matched both units 1 to 15 points worse. so each sound with noise was
// fitted again at chipRate, with changes to the noise oscillators (4 and 5) in
// all its parts: a scale on their frequencies, a change to their width
// (distortion) and the balance of their levels, against both units of the
// VoxRecord recording with the background taken out and clipped sounds left
// out. the sound's overall level is held where it was (each sound is scored
// against its own loudest, so a fit left free to change it made SE and others
// louder against the vowels, and the connected groups, which nothing was fitted
// to, worse). each is kept only where it matches both units better, by at least
// 0.05dB or half a point, and doesn't match the second take (which nothing was
// fitted to) worse: TS 28 to 54%, CH 67 to 83%, TT 29 to 45%, KE 62 to 72%, C9
// 51 to 58% on unit B. HO, WH, FF, SO, SH, TH, R5, R8, A7, A9, B0, B1, B2, B5,
// B8, C5, M0 and M1 (clipped on both units) keep theirs
type noiseFit struct {
	freqScale, distortion, db4, db5 float64
}

var noiseRateFit = map[int]noiseFit{
	165: {0.9836, -0.3, -0.28, 1.63},
	166: {0.9824, 6.7, 3.06, -0.80},
	170: {0.9310, -65.4, 3.72, -3.84},
	172: {1.0617, -43.2, 8.47, -9.02},
	173: {0.9253, 78.7, -1.43, -1.04},
	174: {0.9881, -26.6, -1.23, 1.34},
	182: {0.9778, 4.5, 0.80, -1.31},
	183: {1.0414, -17.3, 0.07, 2.65},
	187: {1.0507, -14.3, 3.45, -0.95},
	191: {1.0258, 9.5, -2.09, 1.07},
	192: {0.9785, -13.6, 1.15, -0.49},
	193: {1.0706, -13.3, 10.28, -7.08},
	194: {1.0126, 29.7, 0.95, -1.50},
	195: {1.0266, 38.7, 0.68, -0.16},
	196: {1.0444, -8.5, 5.29, 0.25},
	197: {0.9356, 13.6, -0.62, 1.86},
	198: {0.9956, 2.6, 0.55, -0.66},
	199: {0.9547, 72.6, -0.02, -0.38},
	200: {0.9870, 54.5, -7.08, 6.43},
	201: {0.9336, -10.3, 1.43, -1.06},
	202: {1.0186, 11.8, -4.17, 4.26},
	204: {0.9880, -20.2, -0.10, 0.58},
	206: {0.9968, -16.9, -1.95, 1.83},
	207: {0.9756, 76.1, -0.32, -0.13},
	209: {0.9111, -9.6, -1.21, 1.75},
	210: {1.0497, 15.8, 2.86, -2.60},
	214: {0.9786, 65.5, -1.43, 1.26},
	216: {1.0994, -3.7, -0.16, -0.05},
	218: {0.9956, 14.7, 8.70, -9.23},
	226: {1.0477, -5.8, 0.02, 0.45},
	230: {1.0215, 17.9, -0.72, 0.24},
	231: {1.0252, -41.2, -1.28, 1.67},
	232: {1.0196, 0.3, 3.26, -4.28},
	234: {1.0933, 49.7, 2.56, -2.96},
	236: {0.9841, 31.2, 1.12, -1.36},
	237: {0.9611, 36.2, -0.42, -0.06},
	238: {1.0423, 12.9, -0.26, 0.61},
	239: {1.0203, 39.1, 2.42, -6.94},
}

func (n noiseFit) apply(s sound) sound {
	parts := make([]part, len(s.parts))
	copy(parts, s.parts)
	g4 := math.Pow(10, n.db4/20)
	g5 := math.Pow(10, n.db5/20)
	for i := range parts {
		f := &parts[i].target
		f.freq[3] *= n.freqScale
		f.freq[4] *= n.freqScale
		f.distortion = math.Min(math.Max(f.distortion+n.distortion, 0), 255)
		f.vol[3] *= g4
		f.vol[4] *= g5
	}
	s.parts = parts
	return s
}

func init() {
	for code, n := range noiseRateFit {
		if code < 200 {
			allophones[code-128] = n.apply(allophones[code-128])
		} else {
			effects[code-200] = n.apply(effects[code-200])
		}
	}
}
