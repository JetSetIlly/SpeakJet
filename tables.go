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

	// the sound doesn't follow the Bend control (see expand())
	noBend bool

	// Slow doesn't lengthen the sound (see expand())
	noSlow bool

	// the copies' length when repeated, against the manual's: the second play
	// and the copies after it (see copy1Length)
	copy1Length float64
	copyLength  float64

	// a multiple of the voicing pitch (see voicingPitch), 0 for none
	pitchScale float64

	// a voiced effect whose pitch is its own, not moved by the Pitch control
	// (see c1())
	ownPitch bool

	// the sound is played from the chip's own frames (see vfPoints)
	framed bool

	// the sound's gate keeps time with the Speed control (see cNoise)
	speedGate bool

	// the sound is a stop, its first part the closure (see plosive() and
	// msa.closure())
	stop bool
}

// the lengths the stops and affricates are built with, against the manual's:
// steady sounds about 1.5 times, diphthongs 1.2 times, stops as given (from a
// recording of a real SpeakJet). the other allophones have their own (see
// allophoneLength)
const (
	steadyLength    = 1.5
	diphthongLength = 1.2
	stopLength      = 1.0
)

// how much longer the chip plays each allophone than the manual's length (what
// its parts add up to), at the default speed, measured and fitted on both
// units, alone and in connected groups. the framed sounds (see vfPoints) are
// the measured length already, so theirs is near 1
var allophoneLength = [...]float64{
	1.4, 0.95, 1.96, 1.47, 2.0825, 1.96, 1.4, 1.7325, 1.4,
	1.68, 1.68, 1.4, 1.4, 1.33, 1.0375, 1.2425, 1.3475, 1.3825,
	1.0625, 1.54, 1.5925, 0.9375, 1.177734, 1, 1.025, 1.0492, 1.169697,
	1.254, 0.875, 0.921081, 1.05, 1.0625, 1, 1.05, 1, 1.0125,
	1.025, 0.890315, 0.8295, 0.97966, 1.175, 0.8625, 1.63, 0.975, 2.87025,
	2.738625, 1.3475, 1, 0.952656, 0.8875, 1.0875, 1, 1.075, 0.8328,
	2.800463, 1.17, 1.12125, 1.11, 1.49, 2.45, 1.862719, 1.038375, 1.28325,
	2.07, 2.67575, 0.922059, 2.658075, 1.085, 2.49, 2.706, 1.4175, 1.37984,
}

// how much louder (in dB) the chip plays each allophone than its oscillator
// volumes give: against AW on its own, and refitted where sounds sit next to
// each other
var allophoneGainDB = [...]float64{
	3.54, 0, 2, 1.2, 2, 1.7, 1.3, 3.99, 0,
	-1.8, -1.9, -1.3, -5.96, -10.97, 0, -4.5, -4.7, -4.2,
	0, -4.2, 0.4, 0, 0.8, 0, 0, 0, 1.2,
	0.5, 0, 0.9, 0, 0, 0, 0, 0, 0,
	0, -11.3, -2.1, 4.31, 0, 0, 5.9, 0, -2.6,
	2.7, -0.1, 0, 0, 0, 0, 0, 0, 0,
	-15.5, -26.49, -31.1, -29.8, -26.3, -15.9, -17.6, -16.6, -26.8,
	-2.1, -3.3, -26.4, 1.5, -3, -2.38, -3.8, 6.4, 5.9,
}

// finalLength multiplies an allophone's length when it is the last before a
// pause, a command or the end and isn't repeated: on the chip a vowel played
// last is longer than inside a phrase (IY and AW 1.25 times) and TT and FF
// shorter (0.64, 0.79). a command counts as on both units of the fifth
// VoxProbe: AW and IY before a Pitch or Delay are as long as before a pause,
// and the same length whatever the pause, and whether or not anything is
// waiting in the buffer
var finalLength = [...]float64{
	1.25, 1.175, 1, 1.175, 1, 1, 1, 1, 1.25,
	1, 1, 1.15, 1, 1, 0.9, 1.025, 0.9375, 1,
	1, 1.05, 0.9875, 1.1375, 1, 1, 1, 0.9125, 1,
	1, 1.1931, 1.515937, 0.925, 0.975, 1.0375, 0.8875, 1.025, 1.0625,
	1, 1, 1.2656, 1, 1, 1.159, 1, 1, 1,
	1, 0.975, 0.8875, 1.357187, 1.2, 1, 1.075, 1.2125, 1.0875,
	1, 1, 1, 1, 0.7875, 0.9625, 1, 1, 1,
	0.6375, 1.05, 1, 1, 1.3, 1, 0.8, 1, 0.925,
}

// the copies after the first, when a sound is repeated, as a multiple of the
// manual's length (see expand()): the second play of a sound, copy1Length, and
// the copies after it, copyLength. on both units of the fifth VoxProbe the
// second play runs longer than the rest (AW 72ms then 70, SE 82 then 70, BE 60
// then 70) and most copies are the manual's 70ms; a stop's copies keep their
// burst and release and put the rest of their time into the closure (EB: a
// burst every 45ms)
var copy1Length = [...]float64{
	1.1, 0.7, 1, 1, 1, 1, 1.5, 1, 1.25,
	1, 1, 1, 1, 1, 0.75, 0.9875, 0.9065, 1,
	0.75, 1, 1, 1.0625, 0.65, 0.9625, 1, 0.9125, 1.0931,
	0.8625, 1.075, 0.7088, 0.931875, 0.9625, 0.5655, 0.9125, 1.05, 0.925,
	0.9375, 0.9375, 1, 0.6359, 0.975, 0.7625, 1.346, 0.825, 4.5,
	0.8, 1.25, 0.78, 0.95, 0.97625, 0.867188, 0.9875, 0.7, 0.8125,
	1, 1, 1, 1, 1, 1.165, 1, 1, 1,
	0.75, 1, 1, 1, 1, 1, 1, 1, 1,
}

var copyLength = [...]float64{
	1, 0.7, 1, 1, 1, 1, 1.125, 1, 1.075,
	1, 1, 1, 1, 1, 0.75, 0.9875, 0.93425, 1,
	0.75, 1, 1, 1.0625, 0.65, 0.9625, 1, 0.9125, 1.1925,
	0.8625, 1.075, 1.215, 0.931875, 0.9625, 1.098375, 0.9125, 1.05, 0.925,
	0.9375, 0.9375, 1, 0.6875, 0.975, 0.7625, 1.346, 0.825, 4,
	0.8, 0.875, 0.675, 0.95, 0.97625, 0.867188, 0.9875, 0.7, 0.8125,
	1, 1, 1, 1, 1, 1.835, 1, 1, 1,
	1.125, 1, 1, 1, 1, 1, 1, 1, 1,
}

// REFIT-END

// voiced sounds played as the chip plays them, frame by frame: each point is a
// voicing pulse fitted as three formant sines with the chip's fade and output
// filter (see voiceFade), the two units averaged: its time after the one
// before, the glide to it, the three frequencies and their volumes (AW's first
// formant is 24). for the diphthongs (which move in steps), the voiced
// fricatives (mostly voicing) and voiced stops such as BO and GE (a voice bar,
// then a short vowel-like release)
type vfPoint struct {
	ms, glide float64
	freq      formants
	vol       [3]float64
}

// the allophones played from the chip's frames, by code
// each frame as its time and glide in hundredths of a millisecond, the three
// formants in freqStepHz steps and their volumes in eighths (see vfPoint)
var vfPointsRows = map[int][][8]float64{
	129: { // IH
		{100, 100, 80, 399, 576, 85, 56, 36},
		{7995, 7995, 79, 404, 564, 182, 140, 81},
		{2425, 2425, 79, 399, 570, 47, 45, 23},
		{1150, 1150, 79, 399, 570, 47, 45, 23},
	},
	142: { // NO
		{100, 100, 57, 262, 592, 6, 2, 2},
		{1239, 1239, 58, 255, 594, 60, 14, 26},
		{5719, 5719, 58, 263, 593, 74, 16, 32},
		{1152, 1152, 59, 264, 583, 56, 10, 23},
		{1150, 1150, 59, 264, 583, 56, 10, 23},
	},
	146: { // LO
		{100, 100, 101, 200, 546, 38, 40, 3},
		{5831, 5831, 102, 200, 546, 74, 83, 14},
		{2285, 2285, 101, 198, 559, 44, 48, 4},
		{1150, 1150, 101, 198, 559, 44, 48, 4},
	},
	149: { // IYRR
		{100, 100, 82, 305, 567, 32, 19, 23},
		{1215, 1215, 53, 504, 582, 96, 64, 50},
		{14859, 14859, 90, 344, 557, 129, 103, 8},
		{6789, 6789, 98, 302, 557, 82, 76, 10},
	},
	151: { // AXRR
		{100, 100, 121, 343, 540, 96, 64, 47},
		{5804, 5804, 116, 335, 553, 200, 133, 49},
		{6886, 6886, 109, 321, 441, 171, 121, 48},
		{1146, 1146, 107, 316, 443, 171, 120, 44},
		{1150, 1150, 106, 314, 535, 158, 121, 17},
		{4523, 4523, 99, 306, 535, 156, 129, 7},
		{3504, 3504, 99, 302, 535, 59, 54, 6},
	},
	152: { // AWRR
		{100, 100, 166, 212, 577, 59, 54, 20},
		{2376, 2376, 158, 224, 561, 182, 170, 50},
		{11565, 11565, 125, 273, 450, 142, 127, 34},
		{1141, 1141, 121, 281, 508, 145, 126, 33},
		{8002, 8002, 102, 301, 547, 76, 77, 8},
	},
	153: { // OWRR
		{100, 100, 118, 208, 547, 8, 7, 7},
		{1289, 1289, 128, 197, 547, 55, 62, 7},
		{968, 968, 121, 201, 554, 132, 161, 37},
		{3645, 3645, 120, 217, 547, 94, 108, 60},
		{1074, 1074, 119, 226, 528, 114, 129, 23},
		{1134, 1134, 118, 234, 547, 118, 133, 9},
		{2287, 2287, 113, 247, 583, 119, 127, 26},
		{3423, 3423, 109, 267, 589, 139, 129, 20},
		{2281, 2281, 106, 282, 543, 139, 128, 16},
		{2274, 2274, 103, 298, 589, 139, 127, 15},
		{1262, 1262, 103, 302, 566, 99, 90, 32},
		{959, 959, 102, 300, 547, 88, 80, 14},
		{1329, 1329, 103, 208, 547, 46, 41, 3},
		{1150, 1150, 103, 208, 547, 46, 41, 3},
	},
	156: { // OWIY
		{100, 100, 84, 229, 561, 3, 4, 4},
		{1315, 1315, 126, 199, 561, 46, 49, 9},
		{1223, 1223, 126, 206, 580, 81, 98, 34},
		{1138, 1138, 122, 209, 571, 94, 104, 56},
		{3337, 3337, 109, 264, 546, 121, 126, 45},
		{1138, 1138, 104, 281, 553, 131, 125, 30},
		{1081, 1081, 99, 298, 554, 120, 108, 59},
		{4662, 4662, 87, 364, 592, 134, 101, 39},
		{3480, 3480, 77, 413, 575, 129, 105, 57},
		{4582, 4582, 55, 477, 584, 129, 69, 64},
		{2291, 2291, 53, 509, 584, 113, 68, 56},
		{1268, 1268, 57, 509, 580, 57, 41, 28},
		{1150, 1150, 57, 509, 580, 57, 41, 28},
	},
	158: { // IYEH
		{100, 100, 136, 395, 591, 9, 3, 3},
		{1369, 1369, 136, 487, 582, 43, 31, 30},
		{1056, 1056, 60, 499, 569, 110, 85, 63},
		{1267, 1267, 58, 495, 576, 134, 80, 69},
		{1147, 1147, 60, 489, 591, 140, 84, 70},
		{2250, 2250, 79, 469, 574, 154, 86, 83},
		{6829, 6829, 120, 409, 573, 182, 103, 60},
		{4530, 4530, 140, 385, 537, 207, 123, 80},
		{2519, 2519, 139, 386, 534, 72, 42, 27},
		{1150, 1150, 139, 386, 534, 72, 42, 27},
	},
	159: { // EHLL
		{100, 100, 141, 383, 551, 5, 3, 2},
		{1047, 1047, 141, 383, 551, 39, 26, 19},
		{2420, 2420, 141, 383, 537, 202, 117, 86},
		{9172, 9172, 114, 264, 600, 119, 90, 29},
		{3482, 3482, 104, 221, 551, 77, 83, 10},
		{1147, 1147, 141, 220, 551, 42, 49, 4},
		{1150, 1150, 141, 220, 551, 42, 49, 4},
	},
	160: { // IYUW
		{100, 100, 49, 255, 571, 34, 23, 20},
		{1244, 1244, 47, 496, 581, 112, 62, 46},
		{1108, 1108, 49, 504, 572, 148, 79, 64},
		{2298, 2298, 59, 466, 582, 117, 66, 66},
		{2251, 2251, 60, 419, 571, 122, 89, 48},
		{1023, 1023, 70, 403, 540, 126, 116, 60},
		{2455, 2455, 66, 364, 552, 106, 73, 27},
		{5579, 5579, 76, 275, 501, 109, 101, 31},
		{2300, 2300, 77, 243, 534, 120, 116, 25},
		{1212, 1212, 74, 242, 582, 71, 68, 5},
		{1150, 1150, 74, 242, 582, 71, 68, 5},
	},
	161: { // AXUW
		{100, 100, 154, 275, 542, 18, 17, 12},
		{1099, 1099, 121, 343, 538, 108, 70, 52},
		{1164, 1164, 121, 342, 542, 197, 126, 93},
		{11541, 11541, 156, 259, 555, 185, 137, 70},
		{2246, 2246, 117, 249, 534, 149, 122, 30},
		{1156, 1156, 93, 246, 573, 130, 109, 25},
		{1154, 1154, 78, 242, 542, 114, 110, 10},
		{1158, 1158, 72, 242, 542, 64, 59, 4},
		{1150, 1150, 72, 242, 542, 64, 59, 4},
	},
	162: { // IHWW
		{100, 100, 80, 380, 555, 35, 17, 14},
		{3495, 3495, 80, 391, 555, 180, 136, 77},
		{6975, 6975, 87, 309, 556, 121, 103, 23},
		{5700, 5700, 91, 243, 508, 105, 107, 30},
		{1147, 1147, 93, 231, 555, 98, 106, 13},
		{4693, 4693, 98, 185, 555, 54, 73, 18},
		{1216, 1216, 80, 183, 555, 26, 39, 9},
		{1150, 1150, 80, 183, 555, 26, 39, 9},
	},
	163: { // AYWW
		{100, 100, 155, 357, 496, 8, 6, 6},
		{2429, 2429, 162, 381, 595, 155, 78, 17},
		{6824, 6824, 144, 319, 481, 195, 113, 38},
		{5752, 5752, 128, 255, 508, 137, 107, 35},
		{1147, 1147, 123, 243, 617, 128, 107, 19},
		{5851, 5851, 105, 186, 561, 58, 79, 20},
		{1177, 1177, 155, 180, 496, 37, 48, 10},
		{1150, 1150, 155, 180, 496, 37, 48, 10},
	},
	164: { // OWWW
		{100, 100, 129, 197, 570, 52, 56, 7},
		{1198, 1198, 125, 206, 577, 81, 96, 33},
		{1148, 1148, 128, 203, 566, 82, 105, 20},
		{1146, 1146, 124, 203, 570, 80, 102, 33},
		{1147, 1147, 125, 198, 544, 80, 103, 18},
		{1143, 1143, 118, 198, 593, 71, 96, 22},
		{2292, 2292, 119, 190, 572, 73, 94, 27},
		{1146, 1146, 111, 189, 580, 69, 91, 16},
		{1148, 1148, 114, 191, 577, 66, 89, 9},
		{1146, 1146, 111, 188, 592, 65, 88, 27},
		{2294, 2294, 107, 187, 578, 57, 81, 20},
		{1147, 1147, 106, 188, 589, 56, 77, 30},
		{2287, 2287, 103, 181, 569, 51, 70, 18},
		{1241, 1241, 128, 180, 570, 20, 32, 8},
		{1150, 1150, 128, 180, 570, 20, 32, 8},
	},
	167: { // ZZ
		{100, 100, 62, 342, 544, 3, 2, 2},
		{1145, 1145, 68, 340, 546, 38, 28, 30},
		{1139, 1139, 62, 342, 546, 80, 55, 58},
		{7893, 7893, 63, 344, 546, 66, 42, 43},
	},
	168: { // ZH
		{100, 100, 99, 382, 585, 54, 37, 15},
		{6710, 6710, 100, 382, 570, 53, 36, 19},
	},
	169: { // DH
		{100, 100, 64, 362, 536, 31, 31, 10},
		{9033, 9033, 56, 364, 532, 20, 16, 6},
	},
	171: { // BO
		{100, 100, 142, 321, 546, 10, 3, 1},
		{1112, 1112, 62, 321, 546, 55, 3, 5},
		{1137, 1137, 62, 321, 546, 59, 2, 6},
		{1130, 1130, 94, 321, 546, 68, 12, 13},
		{1187, 1187, 142, 321, 546, 161, 120, 62},
		{3503, 3503, 142, 319, 546, 65, 51, 25},
	},
	175: { // DO
		{100, 100, 100, 339, 567, 3, 2, 3},
		{1133, 1133, 60, 339, 567, 37, 2, 5},
		{2163, 2163, 62, 339, 567, 60, 3, 6},
		{1234, 1234, 100, 339, 567, 84, 57, 33},
		{1241, 1241, 100, 339, 573, 63, 44, 22},
		{1177, 1177, 100, 339, 567, 13, 8, 2},
		{1150, 1150, 100, 339, 567, 13, 8, 2},
	},
	176: { // ED
		{100, 100, 58, 450, 548, 20, 14, 8},
		{1153, 1153, 55, 445, 601, 32, 15, 12},
		{1284, 1284, 47, 445, 601, 16, 3, 1},
		{1150, 1150, 47, 445, 601, 16, 3, 1},
	},
	177: { // OD
		{100, 100, 60, 343, 563, 27, 26, 12},
		{1081, 1081, 59, 347, 558, 25, 16, 8},
		{1292, 1292, 62, 343, 563, 13, 4, 2},
		{1150, 1150, 62, 343, 563, 13, 4, 2},
	},
	178: { // GE
		{100, 100, 58, 298, 575, 24, 3, 4},
		{1053, 1053, 60, 242, 582, 51, 2, 6},
		{1146, 1146, 60, 298, 572, 51, 3, 5},
		{1141, 1141, 61, 447, 589, 46, 3, 7},
		{1150, 1150, 61, 447, 589, 46, 3, 7},
	},
	179: { // GO
		{100, 100, 57, 459, 544, 45, 2, 4},
		{1145, 1145, 59, 459, 566, 55, 8, 4},
		{1143, 1143, 60, 328, 586, 54, 4, 7},
		{2297, 2297, 64, 392, 582, 24, 5, 3},
	},
	180: { // EG
		{100, 100, 60, 483, 566, 64, 41, 9},
		{1145, 1145, 59, 484, 563, 44, 27, 5},
		{1150, 1150, 59, 484, 563, 44, 27, 5},
	},
	181: { // OG
		{100, 100, 64, 300, 571, 8, 4, 3},
		{1062, 1062, 64, 300, 553, 60, 50, 8},
		{2294, 2294, 60, 301, 566, 44, 32, 7},
	},
}

var vfPoints = func() map[int][]vfPoint {
	m := map[int][]vfPoint{}
	for code, rows := range vfPointsRows {
		for _, r := range rows {
			m[code] = append(m[code], vfPoint{ms: r[0] / 100, glide: r[1] / 100, freq: formants{r[2] * freqStepHz, r[3] * freqStepHz, r[4] * freqStepHz}, vol: [3]float64{r[5] / 8, r[6] / 8, r[7] / 8}})
		}
	}
	return m
}()

// framed plays an allophone from the chip's frames, at their measured length
func framed(name string, p []vfPoint) sound {
	s := sound{name: name, speech: true, length: 1, framed: true}
	for _, q := range p {
		s.parts = append(s.parts, part{ms: q.ms, glideMs: q.glide, target: voicedFrame(q.freq, q.vol[0], q.vol[1], q.vol[2])})
	}
	return s
}

// VF-END

// the hiss of ZZ and ZH: oscillators 4 and 5 at 3.5 to 4.5kHz, about 20dB under
// the voicing's loudest band, half enveloped by the voicing and following each
// frame's level
var framedHiss = map[int]struct{ n1, n2, db, dist float64 }{
	167: {3600, 4100, -22, 30}, // ZZ
	168: {3300, 4100, -24, 60}, // ZH
}

func framedNoise(s *sound, code int) {
	h, ok := framedHiss[code]
	if !ok {
		return
	}
	loudest := 0.0
	for _, p := range s.parts {
		loudest = math.Max(loudest, p.target.vol[0])
	}
	for i := range s.parts {
		f := &s.parts[i].target
		v := 24 * math.Pow(10, h.db/20) * f.vol[0] / loudest
		f.freq[3] = h.n1
		f.freq[4] = h.n2
		f.vol[3] = v
		f.vol[4] = v * 0.7
		f.distortion = h.dist
		f.halfEnv = true
	}
}

func init() {
	for i, l := range allophoneLength {
		g := math.Pow(10, allophoneGainDB[i]/20)
		if p := vfPoints[128+i]; p != nil {
			allophones[i] = framed(allophones[i].name, p)
			allophones[i].length = l
			for j := range allophones[i].parts {
				allophones[i].parts[j].target.onsetMs = onsetFor(128 + i)
			}
			framedNoise(&allophones[i], 128+i)
			for j := range allophones[i].parts {
				for k := range 5 {
					allophones[i].parts[j].target.vol[k] *= g
				}
			}
			continue
		}
		allophones[i].length = l
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

// formant targets in Hz. the vowels, nasals and liquids are the chip's,
// measured from single voicing pulses on two units with the chip's fade and
// output filter (see voiceFade): their F1 and F2 are whole multiples of about
// 88.4Hz less 3.5Hz. the rest are published values for an adult male speaker,
// or the mean of two independent measurements of the chip where they agree
type formants [3]float64

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

// the chip doesn't restart all three formant sines at the same phase: each
// sound's second and third formants start at their own phase against the first
// (as fractions of a cycle). only the sounds that match the chip better with
// them at speaking pitch have them
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

// how much quieter the chip's noise is than levels first chosen here: hiss sits
// 16 to 23dB under the vowels beside it. the fricatives have their levels in
// their frames (see frSE()); these are for the bursts and aspiration
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

	return sound{name: name, speech: true, length: stopLength, parts: parts, stop: true}
}

// TS is hiss that starts at its loudest, peaked at 3.5kHz, and dies away (6dB
// down at 40ms, 14dB by 120ms), spreading out to a flat spectrum
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

// the noise frames of the voiceless fricatives, fitted as whole sounds to both
// units: SO's hiss is a band at 3 to 4kHz, HE and HO's around 1 to 1.5kHz, WH's
// at 0.6 to 0.9kHz. SE and SH are broad: SE from 2 to 5kHz strongest at the
// oscillators' top, SH from 1.5 to 4kHz (and its image to 6.5kHz), a wide
// spread every sample where a narrow one made a line. their levels are set by
// allophoneGainDB
var (
	frSE = noiseFrame(lAlveolar, maxFrequency, 2491.4, 35.96, 3.36, 41.0)
	frSO = noiseFrame(lAlveolarBak, 2953.5, 3580, 17.5, 30.41, 27.3)
	frSH = noiseFrame(lPalatal, 3662.7, 2581.1, 14.17, 22.8, 77.5)
	frZH = hush(voicedNoiseFrame(formants{260, 1800, 2500}, 14, 2400, 3200, 14, 10, 200), fricativeDB)
)

// a stop's noise widths, aspiration volumes and voicing default to the first
// ones chosen here (see stop.withDefaults()). the stops and affricates are
// fitted as whole sounds to both units, each keeping the length it has (see
// allophoneLength): a closure that is silent can't be measured from a sound
// alone
// allophones, codes 128 to 199. durations are from the manual
var allophones = [...]sound{
	// the vowels' and liquids' three volumes are the chip's, from single voicing
	// pulses (see voiceFade). some sounds are played from the chip's own frames
	// instead (see vfPoints)
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
	// NGE and NGO are fitted to their harmonics on both units: on the chip NGE's
	// second and third formants are one peak near 2.3kHz
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

// a bang of noise, then hiss around 1 to 2kHz that dies away by 27dB and stops
// at 300ms
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

// the effects traced from recordings that are not pure tones (the tone effects
// are in fxTones). each point is where it is reached (ms from the start of the
// sound), its kind, a buzz's pitch, its level in dB (-60 is silence), up to
// three frequencies (0 for none) and the second and third's levels against the
// first. a tone is a pure sine; noise is noise around its frequency; a buzz is
// voiced at its pitch with oscillators 1 to 3 on its formants. the levels have
// the output filter's loss added back (see chipRate). A1 is three formants
// (steady at 175Hz, rising 720 to 1645Hz, and rising faster from 120 to 2870Hz)
const (
	fxTone = iota
	fxNoise
	fxBuzz
)

type fxPoint struct {
	ms, kind, pitch int
	db              float64
	freq            [3]float64
	rel             [3]float64
}

// each traced point as ms, kind, pitch, its level in tenths of a dB, the three
// frequencies in freqStepHz steps and the second and third's levels against the
// first in tenths of a dB (see fxPoint)
var fxRows = map[int][][9]float64{
	211: { // A1
		{15, 2, 87, -600, 40, 164, 27, 150, -30},
		{35, 2, 87, -161, 40, 164, 27, 150, -30},
		{60, 2, 87, 47, 40, 171, 39, -53, -30},
		{85, 2, 87, 182, 40, 178, 51, -167, -30},
		{160, 2, 87, 79, 40, 198, 98, -43, 36},
		{185, 2, 87, 57, 40, 205, 113, -29, -10},
		{210, 2, 87, 76, 40, 212, 129, -108, -64},
		{235, 2, 87, 67, 40, 219, 146, -12, -69},
		{260, 2, 87, 84, 40, 226, 162, -119, -39},
		{285, 2, 87, 85, 40, 234, 177, -30, -30},
		{485, 2, 87, 29, 40, 277, 330, -30, -30},
		{510, 2, 87, 23, 40, 283, 348, -34, -77},
		{535, 2, 87, 15, 40, 288, 366, -26, -14},
		{560, 2, 87, 27, 40, 295, 385, -75, -61},
		{585, 2, 87, 39, 40, 303, 404, -35, -76},
		{610, 2, 87, 20, 40, 309, 424, -36, -3},
		{635, 2, 87, 48, 40, 317, 443, -94, -113},
		{660, 2, 87, 26, 40, 323, 462, -11, -12},
		{710, 2, 87, 33, 40, 336, 500, -55, -81},
		{735, 2, 87, 26, 40, 342, 518, -55, -11},
		{760, 2, 87, 54, 40, 346, 535, -72, -126},
		{785, 2, 87, 28, 40, 350, 552, -39, 28},
		{810, 2, 87, 35, 40, 354, 570, -65, -90},
		{860, 2, 87, 21, 40, 363, 604, -58, -27},
		{885, 2, 87, 52, 40, 367, 621, -68, -124},
		{910, 2, 87, 79, 40, 371, 638, -104, -78},
		{975, 2, 87, 60, 40, 380, 681, -30, -30},
	},
	213: { // A3
		{5, 2, 88, -600, 131, 223, 455, 272, 365},
		{20, 2, 88, -600, 131, 226, 458, 263, 380},
		{25, 2, 88, -65, 132, 226, 544, -350, -121},
		{30, 2, 88, -55, 132, 223, 544, -16, -131},
		{35, 2, 88, -39, 132, 223, 544, -10, -139},
		{80, 2, 88, -46, 132, 243, 565, 10, -140},
		{120, 2, 88, -35, 143, 265, 568, 2, -152},
		{125, 2, 88, -55, 144, 268, 422, 1, 36},
		{135, 2, 88, -49, 147, 273, 422, -1, 27},
		{520, 2, 88, -87, 331, 538, 779, 41, 60},
		{545, 2, 88, -142, 334, 543, 786, 52, 66},
		{550, 2, 88, -260, 333, 543, 783, 40, 65},
		{555, 2, 88, -600, 297, 543, 781, -217, 65},
		{570, 2, 88, -600, 231, 453, 679, 90, 166},
	},
	229: { // B9
		{2, 0, 0, -600, 0, 0, 0, -600, -600},
		{18, 0, 0, -600, 0, 0, 0, -600, -600},
		{20, 0, 0, -247, 443, 61, 0, -258, -600},
		{22, 0, 0, -101, 614, 416, 201, -6, -123},
		{26, 0, 0, -8, 617, 412, 204, 2, -121},
		{78, 0, 0, -51, 609, 409, 192, 5, -123},
		{80, 0, 0, -600, 609, 174, 236, -600, -600},
		{98, 0, 0, -600, 609, 172, 55, -600, -600},
		{100, 0, 0, -395, 143, 227, 185, 58, 49},
		{104, 0, 0, -119, 152, 227, 185, -600, -600},
		{110, 0, 0, -278, 142, 227, 31, -600, -264},
		{112, 0, 0, -329, 126, 227, 31, -600, -600},
		{116, 0, 0, -108, 130, 227, 31, -600, -600},
		{120, 0, 0, -194, 125, 227, 31, -600, -600},
		{122, 0, 0, -139, 857, 570, 104, 24, -175},
		{124, 0, 0, -54, 862, 571, 278, 27, -180},
		{126, 0, 0, -600, 862, 572, 116, -600, -600},
		{130, 0, 0, -600, 862, 573, 120, -600, -600},
		{132, 0, 0, -28, 870, 573, 118, 26, -190},
		{134, 0, 0, -28, 870, 573, 93, 26, -356},
		{136, 0, 0, -28, 870, 573, 78, 26, -244},
		{138, 0, 0, -600, 870, 573, 115, -600, -600},
		{140, 0, 0, -600, 870, 572, 119, -600, -600},
		{142, 0, 0, -26, 869, 573, 120, 23, -128},
		{144, 0, 0, -26, 869, 573, 119, 24, -228},
		{146, 0, 0, -27, 869, 573, 163, 25, -296},
		{148, 0, 0, -27, 869, 573, 109, 25, -167},
		{150, 0, 0, -600, 869, 573, 117, -600, -600},
		{152, 0, 0, -600, 869, 573, 119, -600, -600},
		{154, 0, 0, -28, 870, 573, 119, 26, -149},
		{156, 0, 0, -27, 869, 573, 115, 25, -266},
		{158, 0, 0, -28, 869, 573, 40, 27, -365},
		{160, 0, 0, -34, 870, 571, 115, 21, -128},
		{162, 0, 0, -600, 870, 571, 118, -600, -600},
		{190, 0, 0, -600, 870, 62, 164, -600, -600},
		{192, 0, 0, -364, 117, 159, 199, 49, 67},
		{194, 0, 0, -600, 117, 171, 199, -600, -600},
		{206, 0, 0, -600, 117, 188, 200, -600, -600},
		{208, 0, 0, -114, 203, 797, 409, 76, 101},
		{264, 0, 0, -130, 205, 799, 412, 107, 129},
		{268, 0, 0, -212, 204, 800, 412, 189, 210},
		{270, 0, 0, -600, 204, 799, 412, -600, -600},
		{274, 0, 0, -600, 204, 800, 412, -600, -600},
		{276, 0, 0, -200, 202, 799, 412, 162, 182},
		{280, 0, 0, -268, 198, 799, 412, 217, 238},
		{282, 0, 0, -600, 198, 800, 412, -600, -600},
		{286, 0, 0, -600, 198, 801, 412, -600, -600},
		{288, 0, 0, -267, 205, 800, 412, 180, 207},
		{292, 0, 0, -323, 213, 799, 412, 214, 242},
		{294, 0, 0, -600, 213, 799, 412, -600, -600},
		{298, 0, 0, -600, 213, 801, 412, -600, -600},
		{300, 0, 0, -422, 172, 799, 413, 242, 265},
		{302, 0, 0, -462, 149, 798, 412, 250, 271},
		{304, 0, 0, -420, 186, 104, 414, -83, 176},
		{306, 0, 0, -600, 186, 82, 414, -600, -600},
		{319, 0, 0, -600, 186, 47, 414, -600, -600},
	},
	230: { // C0
		{5, 0, 0, -600, 55, 226, 352, -149, -600},
		{20, 0, 0, -600, 55, 226, 352, -1, -600},
		{25, 0, 0, -42, 528, 226, 352, -54, -600},
		{30, 0, 0, -39, 528, 469, 352, -142, -600},
		{35, 0, 0, -8, 528, 469, 352, -161, -600},
		{40, 0, 0, -21, 469, 469, 352, -22, -600},
		{45, 0, 0, -27, 468, 469, 352, 0, -26},
		{55, 0, 0, -40, 408, 403, 531, -135, -106},
		{70, 0, 0, -11, 349, 403, 531, -118, -90},
		{75, 0, 0, -10, 289, 403, 531, -35, -49},
		{80, 0, 0, -14, 289, 524, 291, -69, -146},
		{85, 0, 0, 5, 287, 346, 531, -6, -29},
		{90, 0, 0, -14, 230, 231, 511, -117, -93},
		{95, 0, 0, 14, 226, 229, 511, -81, -59},
		{100, 0, 0, 7, 178, 174, 511, -102, -80},
		{105, 0, 0, 17, 174, 132, 511, -137, 48},
		{110, 0, 0, 23, 165, 92, 56, -123, -133},
		{115, 0, 0, 2, 148, 70, 56, 1, -600},
		{130, 0, 0, 25, 114, 70, 56, -600, -600},
		{135, 0, 0, 31, 83, 70, 56, -600, -600},
		{145, 0, 0, 36, 77, 104, 56, -600, -600},
		{150, 0, 0, 20, 45, 104, 133, -600, -600},
		{165, 0, 0, -51, 45, 104, 133, -600, -600},
		{170, 0, 0, -62, 68, 104, 133, -63, -600},
		{175, 0, 0, -58, 75, 136, 133, -1, -600},
		{180, 0, 0, -70, 68, 104, 133, -60, -147},
		{185, 0, 0, -99, 75, 136, 179, -27, -61},
		{190, 0, 0, -90, 75, 136, 205, -64, -99},
		{200, 0, 0, -134, 57, 110, 179, -82, -138},
		{205, 0, 0, -127, 57, 129, 201, -93, -120},
		{210, 0, 0, -145, 57, 110, 201, 0, -1},
		{215, 0, 0, -173, 49, 112, 201, -88, -600},
		{220, 0, 0, -171, 49, 112, 201, 0, -600},
		{225, 0, 0, -600, 61, 112, 201, -107, -600},
		{245, 0, 0, -600, 55, 105, 201, -600, -600},
	},
	232: { // C2
		{5, 0, 0, -600, 53, 49, 222, -600, -600},
		{30, 0, 0, -600, 109, 208, 222, -51, -1},
		{35, 0, 0, -68, 114, 206, 222, -1, -1},
		{45, 0, 0, -64, 116, 206, 39, -28, -110},
		{50, 0, 0, -8, 116, 208, 504, -1, 45},
		{55, 0, 0, -14, 116, 209, 501, -27, -28},
		{60, 0, 0, -15, 116, 210, 40, -26, -103},
		{65, 0, 0, 9, 118, 213, 459, -29, -75},
		{75, 0, 0, -16, 120, 225, 458, -32, -64},
		{80, 0, 0, 7, 121, 230, 55, -43, -94},
		{85, 0, 0, -10, 124, 233, 414, 0, 19},
		{90, 0, 0, -19, 125, 225, 55, -83, -109},
		{100, 0, 0, -16, 129, 254, 55, 0, 1},
		{135, 0, 0, -97, 128, 267, 56, -35, -99},
		{140, 0, 0, -95, 126, 267, 78, -70, -124},
		{165, 0, 0, -257, 102, 263, 57, -143, 0},
		{170, 0, 0, -252, 102, 263, 57, -91, -600},
		{175, 0, 0, -227, 102, 40, 57, 0, -600},
		{180, 0, 0, -93, 755, 40, 57, -600, -600},
		{190, 0, 0, -39, 761, 40, 57, -131, -600},
		{195, 0, 0, -44, 761, 40, 141, -266, -600},
		{205, 0, 0, 3, 762, 44, 141, -132, -600},
		{215, 0, 0, -10, 762, 49, 141, -265, -600},
		{220, 0, 0, -8, 762, 68, 141, -175, -228},
		{225, 0, 0, -207, 382, 72, 141, -66, -105},
		{230, 0, 0, -213, 380, 72, 141, -72, -107},
		{235, 0, 0, -224, 60, 108, 141, -29, -92},
		{240, 0, 0, -600, 56, 108, 110, -146, 0},
		{250, 0, 0, -600, 53, 108, 110, -600, -600},
	},
	233: { // C3
		{5, 0, 0, -600, 51, 112, 94, -600, -600},
		{20, 0, 0, -600, 61, 112, 94, -74, -600},
		{25, 0, 0, -94, 717, 49, 94, -117, -600},
		{30, 0, 0, -83, 721, 40, 94, -191, -600},
		{40, 0, 0, -30, 722, 43, 94, -118, -600},
		{45, 0, 0, -32, 723, 40, 94, -257, -600},
		{50, 0, 0, -9, 722, 40, 94, -600, -600},
		{115, 0, 0, 38, 770, 40, 94, -600, -600},
		{120, 0, 0, 39, 791, 47, 94, -600, -600},
		{180, 0, 0, 43, 863, 47, 94, -600, -600},
		{185, 0, 0, 44, 863, 47, 94, -168, -600},
		{190, 0, 0, 44, 863, 47, 94, -168, -600},
		{195, 0, 0, 44, 863, 47, 94, -264, -600},
		{200, 0, 0, 5, 863, 77, 94, -189, -600},
		{205, 0, 0, 7, 861, 77, 94, -302, -600},
		{210, 0, 0, 6, 861, 81, 94, -311, -168},
		{215, 0, 0, 6, 861, 81, 94, -272, -314},
		{220, 0, 0, -189, 53, 862, 130, 103, -86},
		{225, 0, 0, -203, 53, 862, 169, 165, -104},
		{230, 0, 0, -205, 53, 862, 130, 167, -89},
		{235, 0, 0, -203, 55, 856, 130, 101, -80},
		{240, 0, 0, -600, 55, 227, 130, -134, -600},
		{250, 0, 0, -600, 55, 226, 130, -135, -600},
	},
	234: { // C4
		{2, 0, 0, -600, 0, 0, 0, -600, -600},
		{16, 0, 0, -574, 48, 0, 0, -600, -600},
		{18, 0, 0, -361, 219, 0, 0, -600, -600},
		{24, 0, 0, -600, 215, 0, 0, -600, -600},
		{26, 0, 0, -246, 208, 0, 0, -600, -600},
		{28, 0, 0, -147, 217, 0, 0, -600, -600},
		{34, 0, 0, -277, 218, 0, 0, -600, -600},
		{36, 0, 0, -545, 44, 0, 0, -600, -600},
		{38, 0, 0, -494, 42, 0, 0, -600, -600},
		{40, 0, 0, -140, 213, 0, 0, -600, -600},
		{42, 0, 0, -73, 218, 0, 0, -600, -600},
		{48, 0, 0, -255, 221, 0, 0, -600, -600},
		{50, 0, 0, -361, 295, 0, 0, -600, -600},
		{52, 0, 0, -138, 209, 0, 0, -600, -600},
		{54, 0, 0, -47, 218, 0, 0, -600, -600},
		{58, 0, 0, -92, 220, 0, 0, -600, -600},
		{62, 0, 0, -279, 249, 0, 0, -600, -600},
		{66, 0, 0, -27, 218, 0, 0, -600, -600},
		{72, 0, 0, -208, 218, 0, 0, -600, -600},
		{74, 0, 0, -253, 177, 0, 0, -600, -600},
		{76, 0, 0, -73, 213, 0, 0, -600, -600},
		{78, 0, 0, -22, 218, 0, 0, -600, -600},
		{84, 0, 0, -251, 216, 0, 0, -600, -600},
		{86, 0, 0, -198, 180, 0, 0, -600, -600},
		{90, 0, 0, -20, 219, 0, 0, -600, -600},
		{96, 0, 0, -278, 221, 0, 0, -600, -600},
		{98, 0, 0, -128, 448, 0, 0, -600, -600},
		{100, 0, 0, -36, 215, 0, 0, -600, -600},
		{102, 0, 0, -25, 442, 0, 0, -600, -600},
		{108, 0, 0, -240, 474, 0, 0, -600, -600},
		{112, 0, 0, -7, 442, 0, 0, -600, -600},
		{118, 0, 0, -199, 442, 0, 0, -600, -600},
		{124, 0, 0, -3, 440, 0, 0, -600, -600},
		{130, 0, 0, -241, 441, 0, 0, -600, -600},
		{136, 0, 0, 0, 437, 0, 0, -600, -600},
		{142, 0, 0, -271, 440, 0, 0, -600, -600},
		{146, 0, 0, -11, 435, 0, 0, -600, -600},
		{152, 0, 0, -172, 440, 0, 0, -600, -600},
		{154, 0, 0, -239, 708, 0, 0, -600, -600},
		{158, 0, 0, -15, 684, 0, 0, -600, -600},
		{164, 0, 0, -216, 678, 0, 0, -600, -600},
		{170, 0, 0, -13, 683, 0, 0, -600, -600},
		{176, 0, 0, -268, 702, 0, 0, -600, -600},
		{180, 0, 0, -33, 681, 0, 0, -600, -600},
		{182, 0, 0, -15, 681, 0, 0, -600, -600},
		{186, 0, 0, -153, 681, 0, 0, -600, -600},
		{188, 0, 0, -600, 681, 0, 0, -600, -600},
		{190, 0, 0, -102, 678, 0, 0, -600, -600},
		{192, 0, 0, -22, 681, 0, 0, -600, -600},
		{200, 0, 0, -237, 708, 0, 0, -600, -600},
		{204, 0, 0, -20, 682, 0, 0, -600, -600},
		{210, 0, 0, -247, 676, 0, 0, -600, -600},
		{212, 0, 0, -541, 58, 0, 0, -600, -600},
		{214, 0, 0, -110, 680, 0, 0, -600, -600},
		{216, 0, 0, -52, 681, 0, 0, -600, -600},
		{222, 0, 0, -255, 683, 0, 0, -600, -600},
		{224, 0, 0, -555, 54, 0, 0, -600, -600},
		{226, 0, 0, -561, 52, 0, 0, -600, -600},
		{228, 0, 0, -166, 680, 0, 0, -600, -600},
		{230, 0, 0, -133, 682, 0, 0, -600, -600},
		{234, 0, 0, -261, 684, 0, 0, -600, -600},
		{236, 0, 0, -600, 684, 0, 0, -600, -600},
		{238, 0, 0, -590, 52, 0, 0, -600, -600},
		{249, 0, 0, -600, 52, 0, 0, -600, -600},
	},
	236: { // C6
		{5, 0, 0, -600, 53, 109, 133, -600, -600},
		{25, 0, 0, -600, 459, 226, 136, -147, -151},
		{30, 0, 0, -221, 463, 224, 227, -105, -157},
		{35, 0, 0, -134, 466, 52, 136, -75, -98},
		{40, 0, 0, -130, 463, 52, 358, -148, -173},
		{55, 0, 0, -82, 461, 52, 358, -31, -600},
		{75, 0, 0, -52, 461, 336, 257, -26, -600},
		{90, 0, 0, -19, 396, 336, 257, -9, -15},
		{100, 0, 0, -6, 314, 261, 257, -142, -148},
		{110, 0, 0, 11, 243, 261, 638, -98, -24},
		{130, 0, 0, -14, 206, 148, 146, 0, 0},
		{135, 0, 0, -23, 206, 443, 146, 27, 0},
		{140, 0, 0, -28, 206, 443, 208, 18, -70},
		{145, 0, 0, -19, 440, 443, 208, -18, -62},
		{150, 0, 0, -18, 442, 443, 383, -91, -131},
		{160, 0, 0, -20, 440, 205, 208, -51, -85},
		{165, 0, 0, -30, 207, 205, 148, -14, -103},
		{185, 0, 0, 12, 206, 146, 635, -137, -50},
		{200, 0, 0, 0, 279, 250, 661, -5, 74},
		{205, 0, 0, -4, 303, 245, 315, -107, -127},
		{240, 0, 0, -60, 463, 250, 273, -600, -600},
		{280, 0, 0, -161, 464, 130, 121, -75, -110},
		{285, 0, 0, -600, 55, 130, 121, -80, -103},
		{305, 0, 0, -600, 55, 109, 129, -130, -600},
	},
	237: { // C7
		{2, 0, 0, -600, 0, 0, 0, -600, -600},
		{20, 0, 0, -554, 60, 0, 0, -600, -600},
		{22, 0, 0, -322, 114, 0, 0, -600, -600},
		{26, 0, 0, -260, 120, 0, 0, -600, -600},
		{30, 0, 0, -428, 118, 0, 0, -600, -600},
		{32, 0, 0, -570, 42, 0, 0, -600, -600},
		{34, 0, 0, -462, 46, 0, 0, -600, -600},
		{36, 0, 0, -239, 111, 0, 0, -600, -600},
		{38, 0, 0, -178, 117, 0, 0, -600, -600},
		{44, 0, 0, -378, 119, 0, 0, -600, -600},
		{46, 0, 0, -497, 36, 0, 0, -600, -600},
		{48, 0, 0, -270, 105, 0, 0, -600, -600},
		{50, 0, 0, -158, 116, 0, 0, -600, -600},
		{56, 0, 0, -275, 120, 0, 0, -600, -600},
		{58, 0, 0, -494, 28, 0, 0, -600, -600},
		{60, 0, 0, -262, 105, 0, 0, -600, -600},
		{64, 0, 0, -112, 120, 0, 0, -600, -600},
		{68, 0, 0, -228, 139, 0, 0, -600, -600},
		{70, 0, 0, -337, 194, 0, 0, -600, -600},
		{74, 0, 0, -45, 235, 0, 0, -600, -600},
		{82, 0, 0, -246, 312, 0, 0, -600, -600},
		{84, 0, 0, -63, 360, 0, 0, -600, -600},
		{86, 0, 0, -6, 374, 0, 0, -600, -600},
		{92, 0, 0, -214, 420, 0, 0, -600, -600},
		{98, 0, 0, 0, 511, 0, 0, -600, -600},
		{104, 0, 0, -255, 555, 0, 0, -600, -600},
		{108, 0, 0, -28, 623, 0, 0, -600, -600},
		{110, 0, 0, -39, 603, 0, 0, -600, -600},
		{116, 0, 0, -327, 349, 0, 0, -600, -600},
		{118, 0, 0, -198, 124, 0, 0, -600, -600},
		{122, 0, 0, -88, 155, 0, 0, -600, -600},
		{126, 0, 0, -229, 188, 0, 0, -600, -600},
		{132, 0, 0, -25, 272, 0, 0, -600, -600},
		{136, 0, 0, -141, 307, 0, 0, -600, -600},
		{140, 0, 0, -13, 372, 0, 0, -600, -600},
		{154, 0, 0, -77, 503, 0, 0, -600, -600},
		{158, 0, 0, -34, 450, 0, 0, -600, -600},
		{162, 0, 0, -151, 358, 0, 0, -600, -600},
		{164, 0, 0, -112, 134, 0, 0, -600, -600},
		{166, 0, 0, -101, 131, 0, 0, -600, -600},
		{202, 0, 0, -34, 475, 0, 0, -600, -600},
		{206, 0, 0, -47, 364, 0, 0, -600, -600},
		{208, 0, 0, -129, 174, 0, 0, -600, -600},
		{212, 0, 0, -106, 143, 0, 0, -600, -600},
		{220, 0, 0, -63, 188, 0, 0, -600, -600},
		{224, 0, 0, -87, 176, 0, 0, -600, -600},
		{234, 0, 0, -64, 235, 0, 0, -600, -600},
		{236, 0, 0, -26, 296, 0, 0, -600, -600},
		{244, 0, 0, -45, 293, 0, 0, -600, -600},
		{246, 0, 0, -14, 355, 0, 0, -600, -600},
		{252, 0, 0, -36, 276, 0, 0, -600, -600},
		{254, 0, 0, -95, 174, 0, 0, -600, -600},
		{258, 0, 0, -106, 124, 0, 0, -600, -600},
		{264, 0, 0, -127, 106, 0, 0, -600, -600},
		{268, 0, 0, -74, 167, 0, 0, -600, -600},
		{276, 0, 0, -80, 161, 0, 0, -600, -600},
		{280, 0, 0, -43, 214, 0, 0, -600, -600},
		{288, 0, 0, -42, 223, 0, 0, -600, -600},
		{290, 0, 0, -47, 284, 0, 0, -600, -600},
		{294, 0, 0, -12, 280, 0, 0, -600, -600},
		{302, 0, 0, -73, 195, 0, 0, -600, -600},
		{304, 0, 0, -116, 111, 0, 0, -600, -600},
		{306, 0, 0, -129, 98, 0, 0, -600, -600},
		{322, 0, 0, -139, 102, 0, 0, -600, -600},
		{330, 0, 0, -188, 84, 0, 0, -600, -600},
		{332, 0, 0, -158, 103, 0, 0, -600, -600},
		{334, 0, 0, -221, 82, 0, 0, -600, -600},
		{336, 0, 0, -204, 86, 0, 0, -600, -600},
		{340, 0, 0, -164, 128, 0, 0, -600, -600},
		{354, 0, 0, -211, 100, 0, 0, -600, -600},
		{358, 0, 0, -250, 107, 0, 0, -600, -600},
		{360, 0, 0, -241, 140, 0, 0, -600, -600},
		{362, 0, 0, -226, 105, 0, 0, -600, -600},
		{372, 0, 0, -256, 109, 0, 0, -600, -600},
		{376, 0, 0, -396, 127, 0, 0, -600, -600},
		{382, 0, 0, -327, 110, 0, 0, -600, -600},
		{388, 0, 0, -600, 127, 0, 0, -600, -600},
		{405, 0, 0, -600, 127, 0, 0, -600, -600},
	},
	254: { // M2
		{5, 2, 128, -600, 123, 385, 651, 376, 435},
		{10, 2, 177, -600, 123, 389, 652, 381, 446},
		{15, 2, 498, -600, 123, 389, 654, 42, 83},
		{20, 2, 498, -110, 123, 388, 654, 22, 66},
		{25, 2, 85, -91, 121, 388, 654, 22, 66},
		{45, 2, 87, -43, 114, 393, 666, 29, 71},
		{155, 2, 87, -71, 168, 446, 722, 32, 72},
		{285, 2, 87, -66, 225, 501, 779, 59, 92},
		{410, 2, 87, -54, 163, 441, 717, 28, 69},
		{525, 2, 87, -64, 111, 390, 662, 28, 75},
		{530, 2, 103, -67, 111, 390, 662, 25, 72},
		{535, 2, 492, -136, 112, 390, 662, 24, 72},
		{540, 2, 83, -138, 112, 390, 662, 27, 72},
		{545, 2, 492, -222, 112, 390, 662, 47, 94},
		{550, 2, 492, -600, 112, 390, 662, 54, 99},
		{555, 2, 492, -600, 112, 389, 661, 382, 438},
	},
}

// the traced effects, by code (from fxRows)
var fxPoints = func() map[int][]fxPoint {
	m := map[int][]fxPoint{}
	for code, rows := range fxRows {
		for _, r := range rows {
			m[code] = append(m[code], fxPoint{ms: int(r[0]), kind: int(r[1]), pitch: int(r[2]), db: r[3] / 10,
				freq: [3]float64{r[4] * freqStepHz, r[5] * freqStepHz, r[6] * freqStepHz}, rel: [3]float64{0, r[7] / 10, r[8] / 10}})
		}
	}
	return m
}()

// the effects the chip gates: their tones pulse on and off at a steady rate
// (the envelope oscillator). 0 is not gated. for codes 200 to 254. C8 and C9
// are gated noise (see cNoise()), and C0, C2, C3, C4 and C6 pulse with the
// voicing instead (see fxPulses)
type fxGate struct {
	hz   int
	wave envelopeWave
}

var fxGates = map[int]fxGate{
	232: {66, envSine},
	234: {87, envSine},
	236: {125, envSaw},
}

// the effects that pulse with the voicing: each pulse restarts the tones at
// full level and fades along voiceFade stretched by fade, the pulses a steady
// distance (offset_hz) from the voicing frequency, so they follow the Pitch
// control: C0 4Hz under it, C2 20.5Hz under, C4 on it, C6 42.7Hz over. C3's
// distance grows along the sound (drift, in Hz a millisecond), so its pulses
// slow as it plays. R8 and R9 restart at half the frequency of their loudest
// tone
type fxPulse struct {
	offsetHz, fade, drift float64
}

var fxPulses = map[int]fxPulse{
	232: {-20.5, 1.0, 0},
	234: {0.2, 1.0, 0},
	236: {42.7, 1.6, 0},
	208: {0, 0.7, 0},
	209: {0, 0.7, 0},
	233: {-13.0, 1.075, 0.14},
	230: {-4.0, 0.6625, 0},
}

// the noise under some of the traced effects, on oscillators 4 and 5: their
// frequencies, width (distortion), how deep it is gated and its level against
// the tones. the C series have a broad noise under their tones, and some tone
// effects a skirt of noise 15 to 30dB under them
type fxNoiseFloor struct {
	n1, n2, dist, depth, db float64
}

var fxNoiseFloors = map[int]fxNoiseFloor{
	230: {2361.6, 2282.7, 27.5, 0.371, -18.86},
	231: {1810.5, 2003.4, 147.2, 0.324, -1.82},
	232: {1935.8, 1748.1, 54.8, 0.4, -15.15},
	234: {1704.5, 3553.6, 228.3, 1.0, -12.64},
	236: {2357.3, 2947.3, 10.8, 0.0, -9.44},
	237: {442.9, 1765.9, 199.9, 0.832, -7.05},
	200: {600.9, 527.1, 135.1, 0, -17.96},
	201: {1143.9, 3430.0, 31.9, 0, -10.65},
	202: {1162.4, 1103.3, 73.7, 0, -18.28},
	204: {1437.6, 1456.3, 28.0, 0, -18.56},
	205: {2417.5, 2751.8, 44.1, 0, -15.10},
	206: {1735.5, 2187.7, 10.6, 0, -3.58},
	210: {419.0, 533.5, 158.7, 0, -8.41},
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
	if _, pulsed := fxPulses[code]; !ok || pulsed {
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
		f = noiseFrame(lGlottal, p.freq[0]*0.85, p.freq[0]*1.15, 24*g, 18*g, 200)
	} else {
		base := 28.0
		if p.freq[1] > 0 {
			base = 24
		}
		for i := range 3 {
			f.freq[i] = p.freq[i]
			if p.freq[i] > 0 {
				f.vol[i] = base * g * math.Pow(10, p.rel[i]/20.0)
			}
		}
		if p.kind == fxBuzz {
			f.pitch = float64(p.pitch)
			f.wave = envVoice
			f.voiced = true
		} else if pl, ok := fxPulses[code]; ok {
			f.pulseOffset = pl.offsetHz - pl.drift*float64(p.ms)
			f.pitch = voiceHz(defaultPitch) + f.pulseOffset
			f.wave = envVoice
			f.voiced = true
			f.pulsed = true
			f.fadeScale = pl.fade
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

// C8 and C9 are a narrow band of noise swept by the envelope oscillator: on
// both units its middle climbs from about 650Hz to 3kHz over the first 0.42 of
// each period (19.5ms at the default speed) and falls back over the rest,
// sweepDB quieter at the top, about a tenth of its frequency wide (measured
// every millisecond, the period folded). the levels are fitted to the M score
const (
	sweepLoHz   = 650.0
	sweepHiHz   = 3000.0
	sweepPeak   = 0.42
	sweepDB     = 8.0
	sweepSpread = 35.0 / 255.0 * 1.5
)

// C8 and C9 are a band of noise swept up and down every 19.5ms (see
// sweepLoHz), each random value held 0.75ms (a band with edges, as on the
// chip), at these levels in dB every 5ms (C8 is 80ms on both units). the sweep
// keeps time with Speed, not Pitch
var c8DB = []float64{
	-16.2, -11.8, -9.5, -5.5, -1.1, -0.5, -0.2, -0.1, -0.3, -0.5,
	-0.5, -0.2, -0.9, -1.3, -1.8, -5.2,
}

var c9DB = []float64{
	-18.9, -13.6, -11.4, -7.8, -3.1, -1.8, -1.6, -1.1, -0.8,
	-0.7, -0.5, -0.0, -0.5, -1.2, -1.8, -4.7, -16.4,
}

func cNoise(name string, db []float64) sound {
	f := noiseFrame(lGlottal, 1416.3, 1905.6, 0, 0, 100)
	f.noiseHoldMs = 0.75
	f.pitch = 52
	f.wave = envSaw
	f.sweep = true
	s := effect(name)
	s.speedGate = true
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

// contour plays a traced effect: a quick start on the first point, a glide to
// each of the others in turn, then 3ms to silence. a tone doesn't move while it
// is silent: it fades out where it was and fades in where it is going, rather
// than sweeping in from wherever the trace's silent points were
func contour(name string, p []fxPoint, code int) sound {
	n := len(p)
	fr := make([]frame, n)
	for i := range p {
		fr[i] = fxFrame(p[i], code)
	}
	for j := range 3 {
		on := make([]bool, n)
		for i := range p {
			on[i] = p[i].db > -60 && p[i].freq[j] > 0 && (j == 0 || p[i].rel[j] > -59)
		}
		for i := range n {
			if on[i] {
				continue
			}
			if i > 0 && on[i-1] {
				fr[i].freq[j] = fr[i-1].freq[j]
				continue
			}
			k := i + 1
			for k < n && !on[k] {
				k++
			}
			if k < n {
				fr[i].freq[j] = fr[k].freq[j]
			} else if i > 0 {
				fr[i].freq[j] = fr[i-1].freq[j]
			}
		}
	}
	s := effect(name, hold(float64(p[0].ms), 2, fr[0]))
	for i := 1; i < n; i++ {
		ms := float64(p[i].ms - p[i-1].ms)
		s.parts = append(s.parts, hold(ms, ms, fr[i])...)
	}
	s.parts = append(s.parts, hold(3, 3, quiet(fr[n-1]))...)
	return s
}

// the buzz effects played as the chip plays them, pulse by pulse (measured as
// vfPoints are): each point is a pulse's time after the one before, the three
// formants and their volumes
type fxvPoint struct {
	ms   float64
	freq formants
	vol  [3]float64
}

// the buzz effects played from the chip's frames, by code
// each pulse as its time in hundredths of a millisecond, the three formants in
// freqStepHz steps and their volumes in eighths (see fxvPoint)
var fxvPointsRows = map[int][][7]float64{
	212: { // A2
		{100, 60, 377, 324, 44, 79, 25},
		{12561, 59, 353, 583, 45, 74, 75},
		{2295, 54, 350, 567, 53, 78, 71},
		{2434, 52, 347, 551, 58, 97, 87},
		{2309, 155, 344, 535, 39, 125, 116},
		{1152, 50, 342, 527, 62, 130, 113},
		{1148, 52, 341, 517, 56, 127, 122},
		{1150, 51, 339, 516, 56, 127, 128},
		{1152, 49, 335, 508, 57, 125, 123},
		{1153, 48, 333, 504, 58, 126, 110},
		{1145, 48, 329, 500, 56, 126, 101},
		{1145, 43, 325, 479, 61, 127, 109},
		{1166, 45, 327, 460, 66, 135, 110},
		{1150, 58, 324, 505, 52, 140, 64},
		{1146, 48, 320, 442, 75, 140, 124},
		{1143, 47, 320, 437, 81, 140, 127},
		{1142, 45, 318, 429, 86, 141, 120},
		{1178, 47, 318, 411, 76, 145, 130},
		{1146, 48, 314, 407, 69, 142, 135},
		{1144, 54, 308, 404, 50, 137, 129},
		{1144, 52, 306, 395, 53, 135, 134},
		{1145, 51, 303, 382, 54, 138, 130},
		{1146, 57, 304, 368, 46, 143, 133},
		{1145, 55, 298, 364, 47, 140, 132},
		{1144, 57, 293, 356, 46, 138, 128},
		{1147, 155, 289, 345, 41, 132, 123},
		{1145, 155, 292, 335, 39, 140, 137},
		{1141, 59, 287, 324, 42, 145, 145},
		{1150, 60, 300, 456, 49, 137, 80},
		{1145, 64, 280, 451, 42, 134, 73},
		{1144, 64, 288, 583, 42, 186, 48},
		{1142, 62, 284, 567, 44, 214, 51},
		{1143, 62, 278, 588, 46, 237, 61},
		{1144, 63, 272, 589, 42, 248, 43},
		{1146, 61, 268, 588, 44, 248, 56},
		{1138, 155, 261, 324, 37, 248, 25},
		{1120, 57, 255, 568, 55, 248, 47},
		{1150, 55, 249, 324, 55, 248, 34},
		{1139, 44, 243, 324, 90, 248, 13},
		{1146, 42, 236, 324, 94, 248, 27},
		{1129, 155, 232, 324, 26, 229, 20},
		{1159, 208, 246, 324, 144, 135, 22},
		{1197, 201, 240, 324, 146, 136, 19},
		{1087, 194, 237, 324, 152, 139, 22},
		{1209, 185, 235, 324, 151, 135, 13},
		{1145, 178, 231, 324, 158, 140, 16},
		{1074, 169, 228, 324, 159, 142, 25},
		{1146, 163, 224, 324, 163, 147, 21},
		{1145, 155, 221, 324, 165, 150, 22},
		{1417, 153, 209, 578, 132, 122, 37},
		{1075, 144, 208, 593, 101, 95, 37},
		{1158, 137, 209, 569, 113, 107, 38},
		{1154, 125, 208, 558, 114, 106, 50},
		{1152, 122, 211, 558, 114, 112, 70},
		{1152, 118, 210, 555, 111, 111, 61},
		{1142, 90, 200, 324, 121, 105, 17},
		{1082, 93, 193, 324, 126, 118, 21},
		{1152, 88, 192, 324, 116, 120, 25},
		{1157, 83, 190, 588, 103, 118, 37},
		{1154, 71, 190, 324, 97, 113, 32},
		{1150, 49, 182, 324, 159, 114, 25},
		{1075, 46, 183, 324, 159, 111, 21},
		{1159, 42, 179, 324, 141, 102, 10},
		{1150, 43, 175, 324, 103, 103, 16},
		{4573, 47, 162, 324, 74, 128, 21},
		{1141, 50, 163, 324, 55, 112, 23},
		{1217, 155, 261, 324, 25, 59, 7},
		{1150, 155, 261, 324, 25, 59, 7},
	},
}

var fxvPoints = func() map[int][]fxvPoint {
	m := map[int][]fxvPoint{}
	for code, rows := range fxvPointsRows {
		for _, r := range rows {
			m[code] = append(m[code], fxvPoint{ms: r[0] / 100, freq: formants{r[1] * freqStepHz, r[2] * freqStepHz, r[3] * freqStepHz}, vol: [3]float64{r[4] / 8, r[5] / 8, r[6] / 8}})
		}
	}
	return m
}()

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

// C1 is voiced: a harmonic series whose own pitch sweeps, in whole voicing
// ticks (see voiceHz()), from 333Hz up to 570Hz by 80ms and down to 200Hz, with
// three formants moving through it; the Pitch control doesn't move it (see
// sound.ownPitch). over its last 200ms one line rises from 2.0 to 2.9kHz
// through the harmonics: oscillator 3 there runs free of the voicing (see
// frame.freeOsc). each point: its time after the one before, the pitch, the
// formants and their volumes
type c1Point struct {
	ms, pitch float64
	freq      formants
	vol       [3]float64
}

// each point as its time in hundredths of a millisecond, its voicing period in
// voiceTickHz ticks, the three formants in freqStepHz steps and their volumes
// in eighths (see c1Point)
var c1PointsRows = [][8]float64{
	{100, 24, 200, 329, 455, 81, 10, 0},
	{190, 24, 174, 328, 476, 53, 5, 0},
	{190, 24, 193, 309, 478, 93, 31, 0},
	{380, 24, 254, 267, 471, 208, 99, 0},
	{380, 23, 204, 366, 502, 138, 90, 0},
	{190, 23, 188, 371, 513, 131, 117, 0},
	{380, 23, 170, 292, 524, 175, 231, 0},
	{380, 23, 130, 359, 497, 181, 131, 0},
	{190, 23, 127, 379, 487, 188, 99, 2},
	{380, 23, 231, 309, 499, 239, 191, 14},
	{190, 23, 245, 328, 506, 206, 188, 12},
	{380, 23, 228, 431, 518, 70, 124, 0},
	{380, 20, 381, 473, 488, 206, 206, 43},
	{190, 20, 408, 471, 480, 230, 225, 61},
	{380, 19, 301, 390, 483, 136, 189, 85},
	{380, 19, 262, 348, 441, 161, 168, 61},
	{190, 18, 245, 336, 418, 181, 158, 47},
	{380, 18, 195, 405, 390, 191, 124, 35},
	{190, 17, 170, 380, 395, 188, 121, 38},
	{380, 17, 122, 211, 440, 164, 143, 62},
	{190, 16, 99, 246, 463, 141, 153, 96},
	{190, 16, 76, 297, 487, 116, 164, 133},
	{190, 16, 60, 323, 506, 89, 165, 160},
	{190, 15, 64, 273, 509, 53, 136, 160},
	{190, 15, 67, 223, 512, 17, 107, 160},
	{190, 15, 84, 205, 515, 12, 88, 164},
	{190, 15, 109, 207, 518, 25, 76, 170},
	{190, 14, 135, 209, 521, 38, 64, 176},
	{380, 14, 30, 286, 527, 52, 119, 194},
	{190, 14, 13, 322, 523, 48, 147, 192},
	{190, 14, 55, 354, 508, 27, 175, 171},
	{190, 15, 97, 385, 493, 5, 203, 149},
	{190, 15, 86, 372, 494, 29, 181, 146},
	{190, 15, 57, 345, 499, 67, 142, 149},
	{190, 15, 29, 330, 498, 95, 113, 148},
	{380, 15, -16, 481, 399, 19, 187, 84},
	{190, 15, -29, 498, 380, 3, 200, 70},
	{570, 15, -35, 368, 417, 19, 162, 84},
	{190, 16, -37, 324, 429, 24, 149, 89},
	{190, 16, -36, 294, 440, 28, 139, 93},
	{1330, 16, 24, 344, 492, 19, 114, 125},
	{190, 16, 30, 352, 498, 17, 115, 123},
	{190, 17, 30, 362, 501, 16, 122, 111},
	{1520, 17, 32, 437, 524, 2, 178, 14},
	{190, 18, 32, 446, 527, 0, 185, 2},
	{190, 18, 46, 434, 522, 2, 176, 11},
	{1710, 19, 193, 303, 462, 23, 71, 120},
	{190, 19, 211, 302, 460, 26, 64, 125},
	{380, 19, 252, 343, 470, 35, 64, 106},
	{190, 20, 273, 364, 475, 39, 64, 97},
	{1330, 20, 417, 508, 511, 71, 64, 33},
	{760, 22, 341, 478, 524, 63, 63, 38},
	{1140, 23, 220, 427, 543, 49, 60, 47},
	{190, 23, 209, 420, 545, 48, 59, 48},
	{1900, 26, 239, 372, 551, 36, 24, 56},
	{2280, 29, 195, 368, 601, 15, 16, 31},
	{1710, 33, 171, 360, 617, 6, 0, 24},
	{2090, 38, 121, 392, 639, 0, 0, 17},
	{190, 40, 121, 392, 644, 0, 1, 15},
	{1140, 40, 121, 392, 676, 0, 6, 1},
	{760, 40, 121, 392, 667, 0, 2, 0},
}

var c1Points = func() []c1Point {
	var p []c1Point
	for _, r := range c1PointsRows {
		p = append(p, c1Point{ms: r[0] / 100, pitch: voiceTickHz / r[1], freq: formants{r[2] * freqStepHz, r[3] * freqStepHz, r[4] * freqStepHz}, vol: [3]float64{r[5] / 8, r[6] / 8, r[7] / 8}})
	}
	return p
}()

func c1(name string) sound {
	s := effect(name)
	var f frame
	for i, p := range c1Points {
		f = voicedFrame(p.freq, p.vol[0], p.vol[1], p.vol[2])
		f.pitch = p.pitch
		f.freeOsc = 4
		glide := p.ms
		if i == 0 {
			glide = 1
		}
		s.parts = append(s.parts, hold(p.ms, glide, f)...)
	}
	s.parts = append(s.parts, hold(3, 3, quiet(f))...)
	s.ownPitch = true
	return s
}

// the effects that are pure tones, traced from both units every millisecond,
// the levels with the output filter's loss added back, each effect scaled so
// its loudest point is at the oscillators' limit. each tone has an oscillator
// of its own and is kept as its knots: its time from the start of the sound in
// half milliseconds, its frequency in freqStepHz steps and its volume in eighths,
// glided between in straight lines (only the knots a straight line can't pass
// close enough to are kept). before its first knot a tone sits silent at its
// first frequency
type fx2Point struct {
	ms   float64
	freq [5]float64
	vol  [5]float64
}

// toneKnots is a tone of a traced effect: its oscillator and its knots, in threes
type toneKnots struct {
	osc   int
	knots []int
}

// the tone effects, by code
var fxTones = map[int][]toneKnots{
	200: { // R0
		{0, []int{
			2, 181, 0, 10, 181, 0, 12, 181, 20, 34, 183, 248, 130, 183, 241, 156, 185, 51, 158, 185, 0, 164, 185, 0,
		}},
		{1, []int{
			2, 93, 0, 34, 93, 0, 36, 93, 33, 58, 103, 56, 106, 123, 68, 116, 130, 84, 118, 130, 0, 164, 130, 0,
		}},
	},
	201: { // R1
		{0, []int{
			2, 145, 0, 6, 145, 0, 8, 145, 3, 10, 155, 6, 16, 155, 14, 24, 156, 14, 32, 151, 20, 40, 159, 26,
			56, 159, 28, 70, 167, 27, 100, 168, 27, 116, 177, 27, 134, 175, 26, 138, 163, 21, 144, 160, 19, 152, 170, 21,
			156, 175, 12, 158, 175, 0, 172, 175, 0,
		}},
		{1, []int{
			2, 252, 0, 12, 252, 0, 14, 252, 17, 16, 252, 26, 22, 252, 106, 32, 252, 231, 130, 252, 248, 156, 252, 29,
			158, 252, 20, 164, 252, 3, 172, 252, 0,
		}},
	},
	202: { // R2
		{0, []int{
			2, 296, 0, 8, 296, 1, 12, 296, 8, 14, 296, 20, 16, 296, 33, 18, 296, 57, 24, 297, 149, 32, 297, 246,
			134, 297, 248, 144, 297, 170, 150, 297, 94, 152, 298, 78, 158, 298, 43, 160, 298, 57, 162, 298, 0, 190, 298, 0,
		}},
		{1, []int{
			2, 167, 0, 12, 167, 0, 14, 167, 8, 16, 149, 6, 24, 150, 19, 32, 156, 25, 140, 86, 28, 146, 86, 26,
			150, 95, 21, 152, 95, 19, 160, 86, 21, 166, 94, 18, 180, 96, 2, 182, 96, 5, 184, 96, 0, 190, 96, 0,
		}},
		{2, []int{
			2, 387, 0, 30, 387, 0, 32, 387, 29, 40, 380, 41, 48, 366, 55, 50, 366, 0, 190, 366, 0,
		}},
	},
	203: { // R3
		{0, []int{
			2, 366, 0, 6, 366, 0, 8, 366, 8, 14, 366, 36, 26, 366, 195, 132, 366, 226, 160, 367, 9, 162, 366, 11,
			164, 366, 0, 192, 366, 0,
		}},
		{1, []int{
			2, 145, 0, 6, 145, 0, 40, 158, 59, 114, 176, 26, 192, 161, 0,
		}},
		{2, []int{
			2, 79, 0, 34, 79, 0, 36, 79, 10, 130, 84, 35, 146, 68, 14, 164, 68, 5, 166, 68, 0, 192, 68, 0,
		}},
	},
	204: { // R4
		{0, []int{
			2, 434, 1, 6, 434, 6, 8, 434, 15, 10, 434, 27, 28, 435, 248, 130, 435, 234, 158, 434, 10, 160, 434, 0,
			166, 434, 0,
		}},
		{1, []int{
			2, 143, 0, 6, 143, 0, 8, 143, 5, 12, 138, 7, 18, 134, 25, 22, 127, 31, 28, 125, 34, 34, 143, 38,
			78, 197, 17, 108, 197, 8, 118, 195, 0, 166, 195, 0,
		}},
		{2, []int{
			2, 105, 0, 42, 105, 0, 44, 105, 42, 46, 106, 32, 118, 164, 29, 128, 168, 32, 136, 160, 25, 144, 166, 23,
			146, 166, 0, 166, 166, 0,
		}},
		{3, []int{
			2, 64, 0, 66, 64, 0, 68, 64, 16, 74, 64, 18, 94, 72, 40, 126, 84, 36, 136, 83, 33, 138, 85, 28,
			140, 101, 30, 142, 101, 0, 166, 101, 0,
		}},
		{4, []int{
			2, 337, 0, 74, 337, 0, 76, 337, 14, 100, 373, 24, 102, 373, 0, 166, 373, 0,
		}},
	},
	205: { // R5
		{0, []int{
			2, 79, 1, 10, 79, 6, 14, 79, 15, 20, 70, 21, 24, 69, 23, 26, 71, 24, 30, 82, 30, 34, 86, 34,
			128, 87, 34, 138, 80, 32, 152, 98, 34, 160, 112, 16, 166, 112, 9, 170, 113, 8, 172, 113, 0, 178, 113, 0,
		}},
		{1, []int{
			2, 503, 0, 6, 503, 3, 10, 503, 19, 14, 503, 50, 24, 503, 199, 134, 504, 200, 138, 504, 143, 154, 503, 22,
			158, 504, 8, 166, 504, 0, 178, 504, 0,
		}},
		{2, []int{
			2, 185, 0, 18, 185, 0, 20, 185, 28, 34, 175, 33, 128, 173, 33, 134, 169, 32, 136, 185, 0, 138, 185, 14,
			144, 190, 17, 152, 179, 17, 158, 168, 14, 160, 168, 0, 178, 168, 0,
		}},
	},
	206: { // R6
		{0, []int{
			2, 380, 0, 8, 380, 5, 16, 375, 23, 26, 340, 17, 28, 389, 0, 50, 389, 0, 52, 389, 16, 68, 375, 34,
			92, 338, 25, 94, 124, 0, 136, 124, 0, 138, 124, 46, 150, 118, 39, 156, 107, 22, 162, 101, 33, 170, 115, 27,
			172, 115, 0, 186, 115, 0,
		}},
		{1, []int{
			2, 109, 0, 6, 109, 0, 8, 109, 11, 14, 107, 33, 22, 105, 42, 28, 86, 46, 32, 82, 56, 44, 85, 58,
			80, 70, 54, 104, 65, 16, 106, 370, 0, 138, 370, 0, 140, 370, 28, 152, 381, 21, 166, 385, 23, 172, 383, 10,
			180, 383, 0, 186, 383, 0,
		}},
		{2, []int{
			2, 573, 0, 8, 573, 0, 10, 573, 6, 14, 573, 26, 16, 573, 42, 26, 572, 247, 140, 572, 248, 156, 572, 54,
			162, 572, 14, 166, 572, 4, 168, 572, 0, 186, 572, 0,
		}},
		{3, []int{
			2, 173, 0, 26, 173, 0, 28, 173, 26, 30, 163, 37, 44, 168, 48, 122, 106, 64, 124, 106, 0, 186, 106, 0,
		}},
		{4, []int{
			2, 200, 0, 50, 200, 0, 52, 200, 8, 88, 198, 27, 128, 147, 55, 136, 143, 60, 138, 143, 0, 186, 143, 0,
		}},
	},
	207: { // R7
		{0, []int{
			2, 640, 0, 4, 640, 0, 6, 640, 6, 8, 640, 13, 22, 641, 193, 128, 641, 237, 150, 641, 39, 158, 641, 0,
			194, 641, 0,
		}},
		{1, []int{
			2, 135, 0, 6, 135, 0, 8, 135, 9, 14, 129, 28, 18, 112, 27, 24, 108, 29, 26, 112, 39, 28, 229, 0,
			30, 229, 0, 32, 229, 27, 56, 258, 11, 58, 59, 0, 76, 59, 0, 78, 59, 15, 134, 65, 36, 150, 92, 38,
			164, 93, 8, 194, 94, 0,
		}},
		{2, []int{
			2, 146, 0, 20, 146, 0, 22, 146, 24, 36, 136, 45, 68, 159, 27, 70, 174, 0, 80, 174, 0, 82, 174, 15,
			100, 178, 10, 102, 246, 0, 134, 246, 0, 136, 246, 20, 154, 260, 23, 164, 260, 6, 176, 260, 6, 182, 261, 0,
			194, 261, 0,
		}},
		{3, []int{
			2, 97, 0, 40, 97, 0, 42, 97, 34, 122, 135, 29, 142, 130, 28, 144, 130, 0, 194, 130, 0,
		}},
		{4, []int{
			2, 205, 0, 54, 205, 0, 56, 205, 23, 128, 273, 31, 140, 290, 11, 142, 290, 0, 194, 290, 0,
		}},
	},
	208: { // R8
		{0, []int{
			2, 166, 0, 4, 166, 0, 6, 166, 7, 12, 161, 43, 24, 172, 96, 26, 635, 0, 28, 635, 33, 34, 632, 47,
			46, 606, 50, 62, 607, 22, 70, 607, 14, 72, 609, 18, 74, 436, 0, 84, 436, 0, 86, 436, 7, 96, 434, 18,
			102, 434, 32, 132, 417, 48, 144, 409, 36, 146, 395, 31, 148, 157, 0, 149, 157, 0, 151, 157, 62, 157, 156, 61,
			160, 142, 64, 162, 140, 62, 167, 140, 29, 177, 141, 0, 183, 141, 0,
		}},
		{2, []int{
			2, 131, 0, 22, 131, 0, 24, 131, 72, 28, 142, 91, 40, 155, 134, 134, 140, 134, 144, 123, 108, 152, 123, 89,
			154, 131, 140, 156, 131, 0, 183, 131, 0,
		}},
		{3, []int{
			2, 710, 0, 10, 710, 0, 12, 710, 14, 22, 710, 113, 32, 709, 229, 138, 710, 206, 151, 710, 74, 160, 710, 17,
			162, 710, 10, 165, 710, 10, 167, 710, 0, 183, 710, 0,
		}},
		{4, []int{
			2, 527, 0, 38, 527, 0, 40, 527, 9, 54, 528, 14, 68, 528, 32, 82, 508, 31, 108, 506, 15, 110, 504, 19,
			112, 504, 0, 183, 504, 0,
		}},
	},
	209: { // R9
		{0, []int{
			2, 155, 0, 4, 155, 0, 8, 155, 16, 14, 156, 50, 20, 157, 72, 30, 148, 111, 38, 155, 120, 134, 140, 115,
			142, 122, 94, 146, 121, 86, 153, 120, 76, 155, 124, 110, 156, 124, 0, 183, 124, 0,
		}},
		{1, []int{
			2, 386, 0, 8, 386, 0, 10, 386, 10, 14, 390, 22, 24, 411, 32, 26, 382, 0, 32, 382, 0, 34, 382, 26,
			46, 380, 42, 68, 380, 41, 80, 366, 41, 104, 365, 40, 120, 351, 44, 146, 353, 33, 150, 345, 29, 160, 362, 26,
			161, 362, 0, 183, 362, 0,
		}},
		{3, []int{
			2, 778, 0, 8, 778, 0, 10, 778, 12, 14, 778, 42, 18, 778, 95, 34, 778, 341, 132, 778, 326, 152, 778, 82,
			156, 778, 50, 160, 778, 36, 161, 778, 0, 183, 778, 0,
		}},
		{4, []int{
			2, 76, 0, 36, 76, 0, 38, 76, 21, 42, 76, 12, 44, 80, 11, 50, 79, 10, 52, 75, 10, 56, 79, 9,
			58, 74, 8, 62, 78, 8, 64, 75, 8, 68, 82, 9, 70, 75, 8, 76, 68, 9, 82, 71, 12, 86, 78, 15,
			88, 74, 16, 90, 157, 0, 150, 157, 0, 152, 157, 50, 157, 156, 51, 161, 144, 53, 163, 141, 53, 177, 143, 0,
			183, 143, 0,
		}},
	},
	210: { // A0
		{0, []int{
			2, 368, 0, 24, 368, 0, 26, 368, 46, 408, 605, 35, 410, 587, 41, 412, 95, 0, 634, 95, 0, 636, 95, 86,
			644, 100, 96, 670, 88, 49, 674, 75, 50, 676, 75, 0, 682, 75, 0,
		}},
		{1, []int{
			2, 70, 0, 64, 70, 0, 66, 70, 15, 68, 69, 11, 80, 67, 43, 94, 49, 23, 104, 64, 48, 118, 55, 40,
			130, 65, 64, 140, 60, 53, 224, 78, 67, 234, 72, 74, 242, 82, 65, 254, 61, 68, 256, 62, 60, 258, 82, 74,
			260, 51, 0, 272, 51, 0, 274, 51, 63, 276, 61, 69, 282, 66, 99, 292, 54, 91, 300, 64, 123, 308, 56, 114,
			316, 64, 142, 338, 60, 146, 586, 88, 248, 632, 70, 136, 634, 70, 0, 682, 70, 0,
		}},
	},
	214: { // A4
		{0, []int{
			2, 56, 3, 22, 56, 84, 28, 56, 163, 42, 56, 157, 64, 60, 248, 186, 113, 248, 194, 122, 248, 290, 256, 224,
			322, 390, 163, 362, 562, 150, 370, 613, 115, 394, 816, 83, 402, 884, 149, 410, 892, 184, 418, 892, 172, 498, 892, 33,
			500, 892, 0, 506, 892, 0,
		}},
	},
	215: { // A5
		{0, []int{
			2, 147, 0, 6, 147, 0, 8, 147, 13, 10, 146, 13, 18, 146, 101, 26, 145, 152, 378, 92, 153, 384, 95, 151,
			390, 89, 158, 398, 94, 146, 406, 82, 174, 412, 93, 141, 414, 93, 140, 420, 79, 193, 422, 77, 207, 424, 78, 197,
			432, 93, 126, 438, 78, 191, 448, 77, 199, 453, 86, 126, 455, 94, 122, 456, 94, 0, 1118, 94, 0,
		}},
		{1, []int{
			2, 76, 0, 455, 76, 0, 456, 76, 85, 460, 76, 73, 480, 75, 248, 514, 76, 50, 516, 90, 37, 518, 93, 34,
			522, 89, 43, 524, 76, 64, 536, 75, 212, 552, 76, 136, 554, 88, 98, 558, 94, 83, 560, 91, 91, 564, 76, 157,
			572, 76, 198, 574, 76, 168, 578, 92, 109, 580, 94, 105, 584, 88, 123, 586, 77, 156, 590, 76, 182, 592, 78, 161,
			594, 88, 132, 598, 94, 117, 602, 88, 130, 604, 80, 151, 606, 79, 157, 612, 94, 125, 616, 93, 127, 620, 86, 140,
			626, 95, 129, 632, 92, 130, 640, 96, 130, 1000, 143, 116, 1102, 143, 5, 1118, 143, 0,
		}},
	},
	216: { // A6
		{0, []int{
			2, 70, 2, 14, 70, 14, 22, 70, 96, 30, 104, 225, 114, 376, 149, 254, 376, 6, 256, 376, 0, 394, 376, 0,
		}},
		{1, []int{
			2, 60, 0, 32, 60, 0, 34, 60, 171, 48, 69, 245, 142, 182, 248, 330, 183, 121, 380, 183, 11, 388, 183, 0,
			394, 183, 0,
		}},
	},
	217: { // A7
		{0, []int{
			2, 261, 0, 4, 261, 0, 6, 261, 5, 24, 264, 185, 308, 333, 83, 480, 334, 18, 600, 334, 9, 692, 334, 6,
			824, 332, 7, 826, 332, 0, 1180, 332, 0,
		}},
		{1, []int{
			2, 95, 0, 22, 95, 0, 24, 95, 4, 30, 95, 4, 188, 95, 94, 478, 117, 247, 954, 198, 228, 1130, 212, 16,
			1150, 212, 0, 1180, 212, 0,
		}},
		{2, []int{
			2, 200, 0, 28, 200, 0, 30, 200, 4, 170, 199, 72, 490, 280, 197, 954, 399, 175, 1174, 414, 0, 1180, 414, 0,
		}},
	},
	218: { // A8
		{0, []int{
			2, 79, 1, 14, 79, 8, 40, 79, 86, 128, 228, 248, 260, 229, 3, 262, 65, 0, 324, 65, 0, 346, 64, 32,
			360, 64, 94, 420, 114, 237, 538, 114, 3, 552, 114, 0,
		}},
		{1, []int{
			2, 66, 0, 58, 66, 0, 60, 66, 88, 70, 66, 110, 130, 114, 234, 246, 114, 10, 248, 124, 0, 292, 124, 1,
			322, 124, 44, 414, 448, 190, 536, 458, 4, 552, 458, 0,
		}},
	},
	219: { // A9
		{0, []int{
			2, 297, 0, 4, 297, 0, 8, 297, 14, 10, 297, 29, 20, 297, 151, 112, 295, 165, 426, 69, 248, 450, 57, 97,
			480, 57, 0, 512, 57, 0,
		}},
		{1, []int{
			2, 554, 0, 6, 554, 0, 8, 554, 10, 22, 559, 134, 162, 554, 153, 484, 304, 184, 498, 298, 31, 504, 298, 4,
			506, 298, 0, 512, 298, 0,
		}},
	},
	220: { // B0
		{0, []int{
			2, 594, 0, 4, 594, 0, 6, 594, 8, 22, 594, 246, 230, 595, 248, 348, 594, 2, 356, 594, 0,
		}},
	},
	221: { // B1
		{0, []int{
			2, 324, 0, 6, 324, 0, 8, 324, 14, 16, 323, 125, 26, 336, 233, 298, 875, 71, 306, 885, 80, 314, 872, 70,
			526, 353, 220, 540, 323, 246, 774, 321, 4, 788, 321, 0,
		}},
	},
	222: { // B2
		{0, []int{
			2, 321, 0, 6, 321, 0, 8, 321, 9, 10, 321, 27, 24, 322, 243, 244, 322, 241, 254, 356, 132, 256, 426, 0,
			262, 426, 0, 264, 426, 197, 278, 517, 202, 280, 546, 0, 282, 546, 214, 292, 572, 242, 318, 572, 241, 326, 561, 138,
			328, 403, 0, 330, 403, 136, 342, 389, 232, 564, 389, 231, 784, 390, 9, 798, 390, 1, 806, 390, 0,
		}},
		{1, []int{
			2, 59, 0, 408, 59, 0, 414, 59, 3, 432, 59, 20, 446, 68, 41, 522, 137, 41, 784, 137, 0, 806, 137, 0,
		}},
	},
	223: { // B3
		{0, []int{
			2, 708, 0, 4, 708, 0, 6, 708, 10, 24, 710, 171, 220, 710, 166, 230, 702, 63, 234, 701, 12, 236, 701, 0,
			620, 701, 0,
		}},
		{1, []int{
			2, 140, 0, 6, 140, 0, 8, 140, 10, 24, 136, 65, 456, 137, 65, 604, 137, 0, 620, 137, 0,
		}},
		{2, []int{
			2, 323, 0, 220, 323, 0, 222, 323, 5, 228, 324, 50, 240, 320, 240, 462, 320, 247, 614, 320, 0, 620, 320, 0,
		}},
	},
	224: { // B4
		{0, []int{
			2, 417, 0, 4, 417, 0, 6, 417, 17, 14, 417, 131, 28, 423, 242, 46, 419, 134, 62, 412, 248, 72, 420, 200,
			546, 412, 236, 556, 423, 135, 572, 412, 120, 584, 424, 246, 738, 418, 13, 748, 418, 2, 758, 419, 0,
		}},
		{1, []int{
			2, 369, 0, 80, 369, 0, 82, 369, 122, 294, 81, 173, 304, 74, 169, 312, 81, 173, 520, 359, 119, 528, 369, 154,
			530, 369, 0, 758, 369, 0,
		}},
	},
	225: { // B5
		{0, []int{
			2, 224, 0, 6, 224, 0, 10, 224, 22, 24, 220, 205, 156, 73, 248, 174, 59, 110, 206, 59, 3, 208, 59, 0,
			332, 59, 0,
		}},
		{1, []int{
			2, 800, 0, 6, 800, 0, 8, 800, 4, 10, 799, 17, 26, 801, 132, 220, 801, 128, 324, 802, 11, 326, 802, 0,
			332, 802, 0,
		}},
	},
	226: { // B6
		{0, []int{
			2, 799, 0, 4, 799, 0, 6, 799, 11, 18, 800, 184, 54, 787, 104, 56, 797, 0, 96, 797, 0, 98, 797, 92,
			122, 801, 212, 136, 796, 76, 138, 791, 0, 164, 791, 0, 166, 791, 102, 178, 800, 207, 188, 790, 105, 190, 790, 0,
			342, 790, 0,
		}},
		{1, []int{
			2, 415, 0, 46, 415, 0, 48, 415, 8, 52, 415, 36, 62, 412, 232, 92, 411, 247, 102, 411, 82, 104, 411, 0,
			122, 411, 0, 124, 411, 4, 126, 411, 13, 128, 411, 29, 140, 411, 248, 160, 413, 206, 170, 416, 24, 174, 416, 0,
			180, 416, 0, 182, 416, 9, 186, 416, 40, 194, 412, 205, 224, 411, 248, 324, 412, 16, 334, 412, 3, 342, 412, 0,
		}},
	},
	227: { // B7
		{0, []int{
			2, 663, 0, 4, 663, 0, 6, 663, 14, 12, 664, 93, 20, 665, 232, 82, 664, 248, 88, 664, 204, 98, 663, 35,
			102, 663, 6, 104, 886, 0, 122, 886, 0, 124, 886, 9, 126, 890, 37, 132, 889, 108, 146, 894, 225, 212, 893, 219,
			270, 894, 111, 318, 893, 16, 320, 893, 0, 326, 893, 0,
		}},
	},
	228: { // B8
		{0, []int{
			2, 476, 0, 4, 476, 0, 6, 476, 7, 8, 476, 19, 22, 475, 224, 132, 400, 228, 250, 342, 241, 402, 303, 248,
			548, 298, 193, 630, 298, 10, 634, 298, 4, 644, 298, 0,
		}},
		{1, []int{
			2, 825, 0, 70, 825, 0, 72, 825, 36, 82, 826, 112, 116, 825, 179, 248, 685, 232, 460, 596, 196, 538, 595, 196,
			568, 596, 142, 630, 596, 8, 634, 596, 7, 636, 596, 0, 644, 596, 0,
		}},
	},
	235: { // C5
		{0, []int{
			2, 163, 0, 17, 163, 0, 19, 163, 10, 30, 163, 21, 102, 163, 21, 132, 163, 76, 188, 136, 174, 260, 134, 173,
			319, 108, 171, 325, 96, 175, 329, 103, 159, 334, 101, 153, 336, 77, 0, 353, 77, 0, 355, 77, 170, 366, 76, 248,
			376, 74, 92, 378, 93, 0, 379, 93, 0, 380, 93, 141, 382, 81, 81, 386, 73, 104, 402, 69, 203, 410, 74, 191,
			422, 66, 186, 452, 68, 174, 466, 70, 68, 486, 70, 37, 498, 70, 55, 510, 70, 38, 522, 70, 54, 532, 70, 37,
			544, 70, 54, 556, 70, 37, 566, 70, 54, 578, 70, 38, 590, 70, 55, 602, 70, 38, 614, 70, 54, 624, 70, 37,
			636, 70, 54, 648, 70, 32, 660, 70, 32, 676, 70, 5, 678, 72, 10, 680, 72, 0, 686, 72, 0,
		}},
		{1, []int{
			2, 617, 0, 19, 617, 0, 20, 617, 64, 22, 618, 31, 36, 618, 102, 108, 609, 85, 110, 618, 128, 112, 205, 0,
			161, 205, 0, 162, 205, 104, 182, 206, 167, 210, 206, 144, 258, 205, 25, 260, 205, 21, 262, 205, 29, 264, 64, 0,
			330, 64, 0, 332, 64, 140, 338, 82, 210, 342, 82, 204, 346, 95, 134, 350, 97, 120, 353, 93, 130, 355, 93, 0,
			686, 93, 0,
		}},
		{2, []int{
			2, 69, 0, 172, 69, 0, 174, 69, 20, 188, 69, 39, 222, 68, 146, 258, 68, 160, 296, 68, 178, 298, 68, 0,
			686, 68, 0,
		}},
	},
	240: { // D0
		{0, []int{
			2, 220, 0, 6, 220, 0, 8, 220, 25, 10, 221, 61, 14, 220, 142, 22, 217, 248, 202, 217, 245, 238, 217, 143,
			284, 216, 72, 296, 216, 43, 308, 216, 19, 314, 216, 8, 318, 217, 6, 320, 217, 0, 328, 217, 0,
		}},
		{1, []int{
			2, 304, 0, 14, 304, 0, 16, 304, 140, 28, 307, 218, 196, 306, 217, 238, 306, 148, 284, 306, 51, 296, 306, 39,
			302, 306, 24, 310, 306, 11, 318, 306, 2, 328, 307, 0,
		}},
	},
	241: { // D1
		{0, []int{
			2, 157, 0, 8, 157, 0, 10, 157, 30, 14, 159, 70, 18, 161, 163, 26, 161, 248, 206, 160, 244, 232, 160, 161,
			272, 160, 86, 288, 160, 66, 310, 160, 17, 322, 160, 0, 328, 160, 0,
		}},
		{1, []int{
			2, 281, 0, 14, 281, 0, 16, 281, 100, 18, 277, 138, 26, 277, 207, 206, 277, 164, 232, 277, 158, 306, 277, 20,
			320, 277, 0, 328, 277, 0,
		}},
	},
	242: { // D2
		{0, []int{
			2, 163, 0, 8, 163, 0, 10, 163, 26, 12, 158, 33, 26, 161, 248, 208, 160, 248, 318, 160, 5, 324, 160, 0,
			332, 160, 0,
		}},
		{1, []int{
			2, 308, 0, 12, 308, 0, 18, 306, 112, 28, 307, 203, 222, 306, 173, 300, 306, 34, 318, 306, 4, 332, 306, 0,
		}},
	},
	243: { // D3
		{0, []int{
			2, 159, 0, 8, 159, 0, 12, 159, 34, 14, 159, 63, 16, 159, 116, 26, 161, 248, 204, 160, 244, 310, 160, 16,
			322, 160, 0, 328, 160, 0,
		}},
		{1, []int{
			2, 340, 0, 12, 340, 0, 14, 340, 82, 26, 337, 189, 222, 338, 160, 308, 338, 15, 320, 338, 0, 328, 338, 0,
		}},
	},
	244: { // D4
		{0, []int{
			2, 181, 0, 8, 181, 0, 16, 178, 125, 26, 176, 248, 200, 176, 248, 318, 176, 4, 330, 176, 0,
		}},
		{1, []int{
			2, 274, 0, 16, 274, 0, 18, 274, 147, 230, 277, 163, 292, 277, 48, 318, 277, 0, 330, 277, 0,
		}},
	},
	245: { // D5
		{0, []int{
			2, 174, 0, 8, 174, 0, 10, 174, 18, 14, 174, 62, 16, 174, 115, 26, 176, 248, 206, 176, 244, 318, 176, 4,
			322, 176, 0, 328, 176, 0,
		}},
		{1, []int{
			2, 309, 0, 14, 309, 0, 16, 309, 117, 28, 307, 203, 230, 306, 158, 256, 306, 94, 308, 306, 19, 320, 306, 0,
			328, 306, 0,
		}},
	},
	246: { // D6
		{0, []int{
			2, 178, 0, 8, 178, 0, 10, 178, 36, 12, 173, 40, 14, 173, 63, 18, 173, 151, 26, 176, 248, 208, 176, 242,
			264, 176, 97, 290, 176, 63, 310, 176, 18, 322, 176, 0, 328, 176, 0,
		}},
		{1, []int{
			2, 337, 0, 12, 337, 0, 14, 337, 34, 16, 335, 87, 26, 337, 195, 210, 338, 187, 308, 338, 17, 322, 338, 0,
			328, 338, 0,
		}},
	},
	247: { // D7
		{0, []int{
			2, 204, 0, 6, 204, 0, 8, 204, 46, 10, 197, 36, 26, 196, 248, 216, 195, 221, 314, 195, 6, 320, 195, 0,
			326, 195, 0,
		}},
		{1, []int{
			2, 278, 0, 14, 278, 0, 18, 276, 152, 26, 277, 217, 210, 277, 206, 316, 277, 0, 326, 277, 0,
		}},
	},
	248: { // D8
		{0, []int{
			2, 200, 0, 6, 200, 0, 10, 195, 25, 26, 196, 248, 206, 195, 244, 318, 195, 4, 328, 195, 0,
		}},
		{1, []int{
			2, 310, 0, 14, 310, 0, 16, 310, 68, 26, 307, 209, 214, 306, 190, 318, 306, 3, 320, 306, 0, 328, 306, 0,
		}},
	},
	249: { // D9
		{0, []int{
			2, 197, 0, 8, 197, 0, 18, 194, 161, 210, 195, 239, 320, 195, 4, 330, 195, 0,
		}},
		{1, []int{
			2, 339, 0, 12, 339, 0, 14, 339, 59, 26, 338, 200, 214, 338, 187, 322, 338, 0, 330, 338, 0,
		}},
	},
	250: { // D10
		{0, []int{
			2, 213, 0, 8, 213, 0, 10, 213, 41, 26, 217, 248, 202, 216, 245, 314, 216, 7, 316, 216, 9, 318, 216, 0,
			324, 216, 0,
		}},
		{1, []int{
			2, 276, 0, 20, 276, 0, 22, 276, 174, 28, 277, 226, 202, 277, 223, 300, 277, 28, 304, 277, 24, 306, 278, 30,
			308, 278, 0, 324, 278, 0,
		}},
	},
	251: { // D11
		{0, []int{
			2, 217, 0, 8, 217, 0, 10, 217, 68, 26, 216, 248, 210, 217, 232, 312, 216, 10, 320, 216, 0, 328, 216, 0,
		}},
		{1, []int{
			2, 335, 0, 14, 335, 0, 16, 335, 122, 198, 337, 206, 318, 337, 2, 328, 337, 0,
		}},
	},
	252: { // M0
		{0, []int{
			2, 395, 0, 10, 395, 0, 12, 395, 248, 14, 407, 158, 20, 413, 248, 514, 406, 248, 536, 421, 248, 672, 406, 248,
			1718, 407, 34, 1944, 407, 29, 1970, 407, 20, 2184, 407, 20, 2208, 407, 11, 2376, 407, 10, 2390, 407, 3, 2392, 409, 6,
			2394, 409, 0, 2400, 409, 0,
		}},
	},
}

// tonePoints rebuilds an effect's parts from its tones' knots (see fxTones): a
// part at every knot of any of them, each tone at its straight line's value
// there
func tonePoints(tones []toneKnots) []fx2Point {
	var times []int
	for _, t := range tones {
		for i := 0; i < len(t.knots); i += 3 {
			k := t.knots[i]
			j := 0
			for j < len(times) && times[j] < k {
				j++
			}
			if j < len(times) && times[j] == k {
				continue
			}
			times = append(times, 0)
			copy(times[j+1:], times[j:])
			times[j] = k
		}
	}
	p := make([]fx2Point, len(times))
	for i, tm := range times {
		prev := 0
		if i > 0 {
			prev = times[i-1]
		}
		p[i].ms = float64(float32(float64(tm-prev) / 2))
		for _, t := range tones {
			q := t.knots
			c := len(q) / 3
			var f, v float64
			switch {
			case tm <= q[0]:
				f, v = float64(q[1]), float64(q[2])
			case tm >= q[3*(c-1)]:
				f, v = float64(q[3*(c-1)+1]), float64(q[3*(c-1)+2])
			default:
				j := 0
				for q[3*(j+1)] <= tm {
					j++
				}
				w := float64(tm-q[3*j]) / float64(q[3*(j+1)]-q[3*j])
				f = float64(q[3*j+1]) + w*float64(q[3*(j+1)+1]-q[3*j+1])
				v = float64(q[3*j+2]) + w*float64(q[3*(j+1)+2]-q[3*j+2])
			}
			p[i].freq[t.osc] = float64(float32(f * freqStepHz))
			p[i].vol[t.osc] = float64(float32(v / 8))
		}
	}
	return p
}

// contour2 plays a tone effect traced again: each point glided to in a
// straight line
// A4 is not a sine: its tone carries its third harmonic, 10 to 14dB under it
// and gone as it nears 4kHz, and no noise under it (see speakjet.c's
// third_harmonics)
var thirdHarmonics = map[int]float64{
	214: -14.0, // A4
}

func contour2(name string, p []fx2Point, code int) sound {
	s := effect(name)
	pl, pulsed := fxPulses[code]
	third := 0.0
	if db, ok := thirdHarmonics[code]; ok {
		third = math.Pow(10, db/20)
	}
	for _, q := range p {
		var f frame
		f.freq = q.freq
		f.vol = q.vol
		// the third harmonic, held still and silent once it would pass 3990Hz
		if third > 0 {
			if 3*f.freq[0] < 3990 {
				f.freq[1] = 3 * f.freq[0]
				f.vol[1] = f.vol[0] * third
			} else {
				f.freq[1] = 3990
				f.vol[1] = 0
			}
		}
		// a pulsed tone effect restarts its tones at half the frequency of the
		// loudest (see fxPulses)
		if pulsed {
			c := 0
			for j := 1; j < 3; j++ {
				if q.vol[j] > q.vol[c] {
					c = j
				}
			}
			f.pulseOffset = q.freq[c]/2 - voiceHz(defaultPitch) + pl.offsetHz
			f.pitch = voiceHz(defaultPitch) + f.pulseOffset
			f.wave = envVoice
			f.voiced = true
			f.pulsed = true
			f.fadeScale = pl.fade
		}
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
	for i, name := range effectNames {
		code := 200 + i
		switch {
		case code == 231:
			e[i] = c1(name)
		case fxvPoints[code] != nil:
			e[i] = contourVoiced(name, fxvPoints[code])
		case fxTones[code] != nil:
			e[i] = toneNoise(contour2(name, tonePoints(fxTones[code]), code), code)
		case fxPoints[code] != nil:
			e[i] = contour(name, fxPoints[code], code)
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

// the noise sounds refitted at the chip's own rate (the noise is a frequency
// jittered every sample, so it moves with the rate): a scale on oscillators 4
// and 5's frequencies, a change to their width (distortion) and their levels
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
