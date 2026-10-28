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

	// the copies' length when repeated, against the manual's (see copyLength)
	copyLength float64

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
	1.4, 1, 1.96, 1.47, 2.0825, 1.96, 1.4, 1.7325, 1.4,
	1.68, 1.68, 1.4, 1.4, 1.33, 1.0375, 1.2425, 1.3475, 1.3825,
	1.0625, 1.54, 1.5925, 0.9375, 1.177734, 1, 1.025, 1.0492, 1.169697,
	1.254, 0.875, 0.921081, 1.05, 1.0625, 1, 1.05, 1, 1.0125,
	1.025, 0.890315, 0.8295, 0.8906, 1.175, 1, 1.63, 0.975, 2.87025,
	3.27, 1.3475, 1, 0.8375, 0.8875, 1.0875, 1, 1, 0.8328,
	2.800463, 1.17, 1.12125, 1.11, 1.49, 2.45, 1.862719, 1.17, 1.28325,
	2.07, 2.67575, 0.922059, 2.5315, 1.085, 2.49, 2.706, 1.4175, 1.37984,
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
// pause or the end and isn't repeated: on the chip a vowel played last is
// longer than inside a phrase (IY and AW 1.25 times) and TT and FF shorter
// (0.64, 0.79)
var finalLength = [...]float64{
	1.25, 1.175, 1, 1.175, 1, 1, 1, 1, 1.25,
	1, 1, 1.15, 1, 1, 0.9, 1.025, 0.9375, 1,
	1, 1.05, 0.9875, 1.1375, 1, 1, 1, 0.9125, 1,
	1, 1.1931, 1.2375, 0.925, 0.975, 1.0375, 1, 1.025, 1.0625,
	1, 1, 1.2656, 1, 1, 1, 1, 1, 1,
	1, 0.975, 0.8875, 1.2625, 1.2, 1, 1.075, 1.2125, 1,
	1, 1, 1, 1, 0.7875, 0.9625, 1, 1, 1,
	0.6375, 1.05, 1, 1, 1.3, 1, 0.8, 1, 0.925,
}

// the copies after the first, when a sound is repeated, as a multiple of the
// manual's length (see expand()): on the chip many are shorter than the manual
// says (DO 0.7 times, ZZ 0.69, EB 0.51)
var copyLength = [...]float64{
	1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 0.9875, 0.925, 1,
	0.75, 1, 1, 1.0625, 1, 0.9625, 1, 0.9125, 1,
	0.8625, 1.075, 1, 0.8875, 0.9625, 1.0875, 0.9125, 1.05, 0.925,
	0.9375, 0.9375, 1, 0.6875, 0.975, 0.7625, 0.7625, 0.825, 0.5125,
	0.8, 1, 0.7, 1, 1.1, 0.925, 0.9875, 1, 0.8125,
	1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1,
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
var vfPointsRows = map[int][][8]float64{
	129: { // IH
		{1.00, 1.00, 352, 1750, 2524, 10.64, 7.00, 4.52},
		{79.95, 79.95, 346, 1770, 2471, 22.77, 17.49, 10.08},
		{24.25, 24.25, 348, 1751, 2497, 5.84, 5.68, 2.93},
		{11.50, 11.50, 348, 1751, 2497, 5.84, 5.68, 2.93},
	},
	142: { // NO
		{1.00, 1.00, 251, 1149, 2596, 0.81, 0.23, 0.25},
		{12.39, 12.39, 254, 1116, 2604, 7.53, 1.69, 3.23},
		{57.19, 57.19, 253, 1154, 2599, 9.20, 2.04, 4.05},
		{11.52, 11.52, 260, 1157, 2555, 6.99, 1.31, 2.92},
		{11.50, 11.50, 260, 1157, 2555, 6.99, 1.31, 2.92},
	},
	146: { // LO
		{1.00, 1.00, 442, 875, 2394, 4.79, 5.02, 0.42},
		{58.31, 58.31, 447, 876, 2394, 9.30, 10.39, 1.70},
		{22.85, 22.85, 443, 869, 2450, 5.54, 6.05, 0.56},
		{11.50, 11.50, 443, 869, 2450, 5.54, 6.05, 0.56},
	},
	149: { // IYRR
		{1.00, 1.00, 358, 1336, 2487, 3.97, 2.40, 2.83},
		{12.15, 12.15, 234, 2208, 2552, 12.00, 8.04, 6.25},
		{148.59, 148.59, 396, 1510, 2441, 16.18, 12.84, 1.04},
		{67.89, 67.89, 429, 1322, 2441, 10.24, 9.55, 1.29},
	},
	151: { // AXRR
		{1.00, 1.00, 532, 1502, 2368, 12.05, 7.97, 5.86},
		{58.04, 58.04, 507, 1468, 2426, 25.00, 16.61, 6.10},
		{68.86, 68.86, 476, 1408, 1933, 21.41, 15.09, 6.00},
		{11.46, 11.46, 468, 1384, 1943, 21.36, 14.98, 5.50},
		{11.50, 11.50, 463, 1377, 2347, 19.81, 15.16, 2.10},
		{45.23, 45.23, 435, 1340, 2347, 19.49, 16.07, 0.90},
		{35.04, 35.04, 436, 1324, 2347, 7.40, 6.72, 0.73},
	},
	152: { // AWRR
		{1.00, 1.00, 726, 930, 2530, 7.32, 6.79, 2.55},
		{23.76, 23.76, 694, 981, 2460, 22.75, 21.26, 6.22},
		{115.65, 115.65, 547, 1195, 1971, 17.72, 15.93, 4.21},
		{11.41, 11.41, 529, 1233, 2229, 18.15, 15.80, 4.08},
		{80.02, 80.02, 446, 1318, 2397, 9.51, 9.62, 0.97},
	},
	153: { // OWRR
		{1.00, 1.00, 519, 910, 2400, 1.06, 0.89, 0.85},
		{12.89, 12.89, 562, 865, 2400, 6.87, 7.72, 0.86},
		{9.68, 9.68, 529, 883, 2427, 16.52, 20.16, 4.58},
		{36.45, 36.45, 524, 950, 2400, 11.75, 13.56, 7.44},
		{10.74, 10.74, 520, 993, 2313, 14.28, 16.08, 2.91},
		{11.34, 11.34, 517, 1024, 2400, 14.71, 16.57, 1.08},
		{22.87, 22.87, 497, 1083, 2555, 14.91, 15.93, 3.20},
		{34.23, 34.23, 479, 1170, 2583, 17.39, 16.17, 2.47},
		{22.81, 22.81, 464, 1236, 2379, 17.32, 15.97, 2.06},
		{22.74, 22.74, 451, 1305, 2583, 17.41, 15.90, 1.92},
		{12.62, 12.62, 450, 1322, 2482, 12.43, 11.20, 4.01},
		{9.59, 9.59, 449, 1316, 2400, 11.05, 9.98, 1.70},
		{13.29, 13.29, 450, 910, 2400, 5.79, 5.13, 0.38},
		{11.50, 11.50, 450, 910, 2400, 5.79, 5.13, 0.38},
	},
	156: { // OWIY
		{1.00, 1.00, 367, 1006, 2461, 0.43, 0.48, 0.48},
		{13.15, 13.15, 551, 871, 2461, 5.79, 6.16, 1.14},
		{12.23, 12.23, 552, 904, 2543, 10.17, 12.25, 4.25},
		{11.38, 11.38, 534, 917, 2503, 11.71, 13.00, 7.05},
		{33.37, 33.37, 478, 1156, 2393, 15.07, 15.69, 5.67},
		{11.38, 11.38, 457, 1231, 2424, 16.43, 15.58, 3.73},
		{10.81, 10.81, 436, 1308, 2429, 14.99, 13.52, 7.32},
		{46.62, 46.62, 380, 1597, 2595, 16.72, 12.66, 4.93},
		{34.80, 34.80, 336, 1809, 2522, 16.10, 13.08, 7.15},
		{45.82, 45.82, 243, 2092, 2561, 16.16, 8.68, 8.00},
		{22.91, 22.91, 232, 2231, 2561, 14.13, 8.52, 7.00},
		{12.68, 12.68, 249, 2232, 2545, 7.17, 5.18, 3.47},
		{11.50, 11.50, 249, 2232, 2545, 7.17, 5.18, 3.47},
	},
	158: { // IYEH
		{1.00, 1.00, 595, 1732, 2592, 1.13, 0.41, 0.39},
		{13.69, 13.69, 595, 2137, 2553, 5.36, 3.89, 3.71},
		{10.56, 10.56, 263, 2190, 2496, 13.73, 10.61, 7.87},
		{12.67, 12.67, 255, 2169, 2526, 16.77, 10.04, 8.61},
		{11.47, 11.47, 262, 2146, 2592, 17.51, 10.52, 8.76},
		{22.50, 22.50, 347, 2055, 2518, 19.26, 10.71, 10.32},
		{68.29, 68.29, 524, 1793, 2512, 22.79, 12.85, 7.46},
		{45.30, 45.30, 613, 1687, 2356, 25.88, 15.32, 10.01},
		{25.19, 25.19, 610, 1692, 2342, 9.04, 5.22, 3.41},
		{11.50, 11.50, 610, 1692, 2342, 9.04, 5.22, 3.41},
	},
	159: { // EHLL
		{1.00, 1.00, 617, 1681, 2416, 0.57, 0.38, 0.19},
		{10.47, 10.47, 617, 1681, 2416, 4.82, 3.19, 2.38},
		{24.20, 24.20, 617, 1681, 2353, 25.21, 14.61, 10.75},
		{91.72, 91.72, 498, 1158, 2630, 14.83, 11.29, 3.62},
		{34.82, 34.82, 457, 971, 2416, 9.68, 10.34, 1.19},
		{11.47, 11.47, 617, 965, 2416, 5.21, 6.13, 0.49},
		{11.50, 11.50, 617, 965, 2416, 5.21, 6.13, 0.49},
	},
	160: { // IYUW
		{1.00, 1.00, 214, 1118, 2504, 4.23, 2.83, 2.45},
		{12.44, 12.44, 208, 2176, 2548, 14.05, 7.71, 5.76},
		{11.08, 11.08, 214, 2208, 2507, 18.49, 9.82, 8.01},
		{22.98, 22.98, 260, 2045, 2552, 14.67, 8.20, 8.25},
		{22.51, 22.51, 265, 1837, 2504, 15.25, 11.07, 6.06},
		{10.23, 10.23, 305, 1767, 2368, 15.78, 14.49, 7.55},
		{24.55, 24.55, 289, 1598, 2419, 13.28, 9.14, 3.37},
		{55.79, 55.79, 333, 1205, 2198, 13.66, 12.58, 3.89},
		{23.00, 23.00, 336, 1064, 2342, 14.98, 14.53, 3.10},
		{12.12, 12.12, 326, 1062, 2552, 8.86, 8.49, 0.65},
		{11.50, 11.50, 326, 1062, 2552, 8.86, 8.49, 0.65},
	},
	161: { // AXUW
		{1.00, 1.00, 675, 1207, 2378, 2.26, 2.07, 1.53},
		{10.99, 10.99, 531, 1504, 2358, 13.51, 8.71, 6.50},
		{11.64, 11.64, 530, 1500, 2376, 24.66, 15.78, 11.67},
		{115.41, 115.41, 683, 1137, 2432, 23.12, 17.14, 8.71},
		{22.46, 22.46, 515, 1090, 2341, 18.61, 15.28, 3.74},
		{11.56, 11.56, 407, 1080, 2511, 16.19, 13.59, 3.08},
		{11.54, 11.54, 343, 1061, 2378, 14.28, 13.74, 1.20},
		{11.58, 11.58, 317, 1059, 2378, 8.03, 7.35, 0.48},
		{11.50, 11.50, 317, 1059, 2378, 8.03, 7.35, 0.48},
	},
	162: { // IHWW
		{1.00, 1.00, 351, 1664, 2435, 4.35, 2.15, 1.70},
		{34.95, 34.95, 351, 1713, 2435, 22.44, 17.04, 9.58},
		{69.75, 69.75, 381, 1354, 2439, 15.07, 12.86, 2.93},
		{57.00, 57.00, 398, 1064, 2228, 13.16, 13.37, 3.76},
		{11.47, 11.47, 408, 1013, 2435, 12.30, 13.21, 1.68},
		{46.93, 46.93, 430, 811, 2435, 6.70, 9.13, 2.27},
		{12.16, 12.16, 351, 802, 2435, 3.25, 4.83, 1.14},
		{11.50, 11.50, 351, 802, 2435, 3.25, 4.83, 1.14},
	},
	163: { // AYWW
		{1.00, 1.00, 679, 1567, 2175, 1.02, 0.71, 0.70},
		{24.29, 24.29, 712, 1671, 2607, 19.40, 9.73, 2.08},
		{68.24, 68.24, 633, 1397, 2110, 24.43, 14.07, 4.74},
		{57.52, 57.52, 560, 1116, 2228, 17.16, 13.35, 4.35},
		{11.47, 11.47, 539, 1067, 2705, 16.03, 13.33, 2.33},
		{58.51, 58.51, 459, 814, 2459, 7.25, 9.89, 2.54},
		{11.77, 11.77, 679, 791, 2175, 4.58, 6.03, 1.26},
		{11.50, 11.50, 679, 791, 2175, 4.58, 6.03, 1.26},
	},
	164: { // OWWW
		{1.00, 1.00, 564, 862, 2501, 6.51, 7.02, 0.82},
		{11.98, 11.98, 549, 905, 2529, 10.07, 11.94, 4.16},
		{11.48, 11.48, 561, 891, 2482, 10.26, 13.15, 2.52},
		{11.46, 11.46, 542, 892, 2501, 10.05, 12.72, 4.18},
		{11.47, 11.47, 548, 866, 2387, 9.95, 12.89, 2.26},
		{11.43, 11.43, 519, 869, 2602, 8.82, 12.02, 2.70},
		{22.92, 22.92, 520, 832, 2506, 9.18, 11.78, 3.43},
		{11.46, 11.46, 485, 827, 2543, 8.68, 11.40, 2.06},
		{11.48, 11.48, 501, 837, 2529, 8.22, 11.09, 1.07},
		{11.46, 11.46, 487, 824, 2595, 8.10, 11.03, 3.34},
		{22.94, 22.94, 468, 818, 2535, 7.15, 10.09, 2.54},
		{11.47, 11.47, 466, 826, 2584, 7.05, 9.64, 3.75},
		{22.87, 22.87, 452, 793, 2494, 6.41, 8.73, 2.24},
		{12.41, 12.41, 561, 789, 2501, 2.56, 3.98, 1.02},
		{11.50, 11.50, 561, 789, 2501, 2.56, 3.98, 1.02},
	},
	167: { // ZZ
		{1.00, 1.00, 271, 1499, 2383, 0.40, 0.24, 0.21},
		{11.45, 11.45, 298, 1491, 2394, 4.81, 3.49, 3.70},
		{11.39, 11.39, 274, 1499, 2392, 10.03, 6.86, 7.25},
		{78.93, 78.93, 276, 1508, 2395, 8.23, 5.19, 5.43},
	},
	168: { // ZH
		{1.00, 1.00, 434, 1676, 2563, 6.69, 4.59, 1.88},
		{67.10, 67.10, 439, 1677, 2500, 6.58, 4.44, 2.37},
	},
	169: { // DH
		{1.00, 1.00, 282, 1589, 2352, 3.90, 3.82, 1.28},
		{90.33, 90.33, 246, 1598, 2331, 2.44, 2.02, 0.76},
	},
	171: { // BO
		{1.00, 1.00, 624, 1406, 2392, 1.19, 0.41, 0.17},
		{11.12, 11.12, 274, 1406, 2392, 6.89, 0.33, 0.61},
		{11.37, 11.37, 271, 1406, 2392, 7.40, 0.29, 0.72},
		{11.30, 11.30, 410, 1406, 2392, 8.53, 1.49, 1.65},
		{11.87, 11.87, 624, 1406, 2392, 20.14, 14.98, 7.74},
		{35.03, 35.03, 622, 1400, 2395, 8.18, 6.33, 3.09},
	},
	175: { // DO
		{1.00, 1.00, 440, 1486, 2485, 0.35, 0.29, 0.43},
		{11.33, 11.33, 263, 1486, 2485, 4.58, 0.28, 0.59},
		{21.63, 21.63, 274, 1486, 2485, 7.52, 0.38, 0.72},
		{12.34, 12.34, 440, 1486, 2485, 10.49, 7.15, 4.09},
		{12.41, 12.41, 439, 1486, 2514, 7.92, 5.46, 2.77},
		{11.77, 11.77, 440, 1486, 2485, 1.67, 1.00, 0.26},
		{11.50, 11.50, 440, 1486, 2485, 1.67, 1.00, 0.26},
	},
	176: { // ED
		{1.00, 1.00, 254, 1972, 2404, 2.56, 1.69, 1.06},
		{11.53, 11.53, 239, 1951, 2635, 3.99, 1.83, 1.51},
		{12.84, 12.84, 205, 1951, 2635, 1.99, 0.41, 0.18},
		{11.50, 11.50, 205, 1951, 2635, 1.99, 0.41, 0.18},
	},
	177: { // OD
		{1.00, 1.00, 265, 1504, 2470, 3.32, 3.27, 1.48},
		{10.81, 10.81, 260, 1522, 2447, 3.12, 2.00, 1.06},
		{12.92, 12.92, 273, 1504, 2470, 1.57, 0.48, 0.19},
		{11.50, 11.50, 273, 1504, 2470, 1.57, 0.48, 0.19},
	},
	178: { // GE
		{1.00, 1.00, 254, 1306, 2523, 3.01, 0.37, 0.49},
		{10.53, 10.53, 265, 1063, 2551, 6.37, 0.29, 0.69},
		{11.46, 11.46, 265, 1307, 2508, 6.36, 0.35, 0.61},
		{11.41, 11.41, 267, 1960, 2582, 5.72, 0.35, 0.91},
		{11.50, 11.50, 267, 1960, 2582, 5.72, 0.35, 0.91},
	},
	179: { // GO
		{1.00, 1.00, 250, 2012, 2386, 5.66, 0.19, 0.54},
		{11.45, 11.45, 258, 2012, 2480, 6.87, 1.03, 0.53},
		{11.43, 11.43, 264, 1436, 2570, 6.73, 0.55, 0.82},
		{22.97, 22.97, 282, 1718, 2551, 2.94, 0.62, 0.37},
	},
	180: { // EG
		{1.00, 1.00, 264, 2117, 2480, 8.05, 5.10, 1.17},
		{11.45, 11.45, 258, 2124, 2467, 5.51, 3.38, 0.63},
		{11.50, 11.50, 258, 2124, 2467, 5.51, 3.38, 0.63},
	},
	181: { // OG
		{1.00, 1.00, 279, 1314, 2503, 1.06, 0.51, 0.38},
		{10.62, 10.62, 279, 1314, 2423, 7.49, 6.23, 0.95},
		{22.94, 22.94, 262, 1319, 2480, 5.45, 3.96, 0.82},
	},
}

var vfPoints = func() map[int][]vfPoint {
	m := map[int][]vfPoint{}
	for code, rows := range vfPointsRows {
		for _, r := range rows {
			m[code] = append(m[code], vfPoint{ms: r[0], glide: r[1], freq: formants{r[2], r[3], r[4]}, vol: [3]float64{r[5], r[6], r[7]}})
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
// units: the chip's hiss is narrow bands (SE and SO at 3 to 4kHz, SH at 2.5 to
// 3.2kHz, HE and HO around 1 to 1.5kHz, WH at 0.6 to 0.9kHz). their levels are
// set by allophoneGainDB
var (
	frSE = noiseFrame(lAlveolar, 3629.7, 2609.1, 25.87, 5.78, 33.4)
	frSO = noiseFrame(lAlveolarBak, 2953.5, 3580, 17.5, 30.41, 27.3)
	frSH = noiseFrame(lPalatal, 3605.7, 2502.8, 17.72, 22.59, 44.4)
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
	freq            [3]int
	rel             [3]float64
}

// each traced point as ms, kind, pitch, db, the three frequencies and the second
// and third's levels against the first (see fxPoint)
var fxRows = map[int][][9]float64{
	211: { // A1
		{15, fxBuzz, 87, -60, 175, 720, 120, 15, -3},
		{35, fxBuzz, 87, -16.1, 175, 720, 120, 15, -3},
		{60, fxBuzz, 87, 4.7, 175, 750, 172, -5.3, -3},
		{85, fxBuzz, 87, 18.2, 175, 780, 224, -16.7, -3},
		{160, fxBuzz, 87, 7.9, 175, 868, 428, -4.3, 3.6},
		{185, fxBuzz, 87, 5.7, 175, 898, 495, -2.9, -1},
		{210, fxBuzz, 87, 7.6, 175, 929, 567, -10.8, -6.4},
		{235, fxBuzz, 87, 6.7, 175, 960, 640, -1.2, -6.9},
		{260, fxBuzz, 87, 8.4, 175, 992, 709, -11.9, -3.9},
		{285, fxBuzz, 87, 8.5, 175, 1024, 778, -3, -3},
		{485, fxBuzz, 87, 2.9, 175, 1215, 1445, -3, -3},
		{510, fxBuzz, 87, 2.3, 175, 1239, 1525, -3.4, -7.7},
		{535, fxBuzz, 87, 1.5, 175, 1263, 1605, -2.6, -1.4},
		{560, fxBuzz, 87, 2.7, 175, 1295, 1689, -7.5, -6.1},
		{585, fxBuzz, 87, 3.9, 175, 1327, 1773, -3.5, -7.6},
		{610, fxBuzz, 87, 2, 175, 1357, 1858, -3.6, -0.3},
		{635, fxBuzz, 87, 4.8, 175, 1388, 1943, -9.4, -11.3},
		{660, fxBuzz, 87, 2.6, 175, 1418, 2027, -1.1, -1.2},
		{710, fxBuzz, 87, 3.3, 175, 1473, 2192, -5.5, -8.1},
		{735, fxBuzz, 87, 2.6, 175, 1498, 2272, -5.5, -1.1},
		{760, fxBuzz, 87, 5.4, 175, 1516, 2347, -7.2, -12.6},
		{785, fxBuzz, 87, 2.8, 175, 1535, 2422, -3.9, 2.8},
		{810, fxBuzz, 87, 3.5, 175, 1554, 2497, -6.5, -9},
		{860, fxBuzz, 87, 2.1, 175, 1591, 2648, -5.8, -2.7},
		{885, fxBuzz, 87, 5.2, 175, 1610, 2723, -6.8, -12.4},
		{910, fxBuzz, 87, 7.9, 175, 1627, 2797, -10.4, -7.8},
		{975, fxBuzz, 87, 6, 175, 1668, 2984, -3, -3},
	},
	213: { // A3
		{5, fxBuzz, 88, -60, 576, 977, 1996, 27.2, 36.5},
		{20, fxBuzz, 88, -60, 576, 990, 2010, 26.3, 38},
		{25, fxBuzz, 88, -6.5, 577, 990, 2386, -35, -12.1},
		{30, fxBuzz, 88, -5.5, 577, 978, 2386, -1.6, -13.1},
		{35, fxBuzz, 88, -3.9, 577, 978, 2386, -1, -13.9},
		{80, fxBuzz, 88, -4.6, 578, 1065, 2476, 1, -14},
		{120, fxBuzz, 88, -3.5, 626, 1164, 2491, 0.2, -15.2},
		{125, fxBuzz, 88, -5.5, 632, 1177, 1852, 0.1, 3.6},
		{135, fxBuzz, 88, -4.9, 645, 1199, 1852, -0.1, 2.7},
		{520, fxBuzz, 88, -8.7, 1452, 2357, 3417, 4.1, 6},
		{545, fxBuzz, 88, -14.2, 1465, 2382, 3447, 5.2, 6.6},
		{550, fxBuzz, 88, -26, 1459, 2382, 3431, 4, 6.5},
		{555, fxBuzz, 88, -60, 1304, 2379, 3423, -21.7, 6.5},
		{570, fxBuzz, 88, -60, 1011, 1986, 2975, 9, 16.6},
	},
	229: { // B9: traced again every 2ms (see speakjet.c's fx_points)
		{2, fxTone, 0, -60, 0, 0, 0, -60, -60},
		{18, fxTone, 0, -60, 0, 0, 0, -60, -60},
		{20, fxTone, 0, -24.7, 1943, 266, 0, -25.8, -60},
		{22, fxTone, 0, -10.1, 2690, 1822, 883, -0.6, -12.3},
		{26, fxTone, 0, -0.8, 2707, 1805, 893, 0.2, -12.1},
		{78, fxTone, 0, -5.1, 2669, 1795, 840, 0.5, -12.3},
		{80, fxTone, 0, -60, 2669, 761, 1033, -60, -60},
		{98, fxTone, 0, -60, 2669, 753, 242, -60, -60},
		{100, fxTone, 0, -39.5, 628, 995, 809, 5.8, 4.9},
		{104, fxTone, 0, -11.9, 666, 995, 809, -60, -60},
		{110, fxTone, 0, -27.8, 624, 995, 134, -60, -26.4},
		{112, fxTone, 0, -32.9, 551, 995, 134, -60, -60},
		{116, fxTone, 0, -10.8, 569, 995, 134, -60, -60},
		{120, fxTone, 0, -19.4, 548, 995, 134, -60, -60},
		{122, fxTone, 0, -13.9, 3756, 2499, 454, 2.4, -17.5},
		{124, fxTone, 0, -5.4, 3781, 2502, 1220, 2.7, -18},
		{126, fxTone, 0, -60, 3781, 2510, 507, -60, -60},
		{130, fxTone, 0, -60, 3781, 2514, 527, -60, -60},
		{132, fxTone, 0, -2.8, 3813, 2512, 517, 2.6, -19},
		{134, fxTone, 0, -2.8, 3815, 2512, 406, 2.6, -35.6},
		{136, fxTone, 0, -2.8, 3813, 2514, 344, 2.6, -24.4},
		{138, fxTone, 0, -60, 3813, 2511, 505, -60, -60},
		{140, fxTone, 0, -60, 3813, 2510, 521, -60, -60},
		{142, fxTone, 0, -2.6, 3810, 2511, 525, 2.3, -12.8},
		{144, fxTone, 0, -2.6, 3810, 2512, 521, 2.4, -22.8},
		{146, fxTone, 0, -2.7, 3810, 2512, 713, 2.5, -29.6},
		{148, fxTone, 0, -2.7, 3812, 2514, 478, 2.5, -16.7},
		{150, fxTone, 0, -60, 3812, 2513, 511, -60, -60},
		{152, fxTone, 0, -60, 3812, 2513, 523, -60, -60},
		{154, fxTone, 0, -2.8, 3814, 2513, 523, 2.6, -14.9},
		{156, fxTone, 0, -2.7, 3811, 2513, 506, 2.5, -26.6},
		{158, fxTone, 0, -2.8, 3812, 2513, 176, 2.7, -36.5},
		{160, fxTone, 0, -3.4, 3816, 2502, 506, 2.1, -12.8},
		{162, fxTone, 0, -60, 3816, 2502, 516, -60, -60},
		{190, fxTone, 0, -60, 3816, 272, 718, -60, -60},
		{192, fxTone, 0, -36.4, 511, 695, 872, 4.9, 6.7},
		{194, fxTone, 0, -60, 511, 750, 872, -60, -60},
		{206, fxTone, 0, -60, 511, 823, 877, -60, -60},
		{208, fxTone, 0, -11.4, 892, 3494, 1794, 7.6, 10.1},
		{264, fxTone, 0, -13, 900, 3503, 1806, 10.7, 12.9},
		{268, fxTone, 0, -21.2, 896, 3506, 1807, 18.9, 21},
		{270, fxTone, 0, -60, 896, 3504, 1807, -60, -60},
		{274, fxTone, 0, -60, 896, 3506, 1807, -60, -60},
		{276, fxTone, 0, -20, 886, 3504, 1808, 16.2, 18.2},
		{280, fxTone, 0, -26.8, 868, 3505, 1807, 21.7, 23.8},
		{282, fxTone, 0, -60, 868, 3507, 1808, -60, -60},
		{286, fxTone, 0, -60, 868, 3510, 1808, -60, -60},
		{288, fxTone, 0, -26.7, 897, 3506, 1807, 18, 20.7},
		{292, fxTone, 0, -32.3, 936, 3503, 1808, 21.4, 24.2},
		{294, fxTone, 0, -60, 936, 3504, 1807, -60, -60},
		{298, fxTone, 0, -60, 936, 3510, 1807, -60, -60},
		{300, fxTone, 0, -42.2, 754, 3505, 1809, 24.2, 26.5},
		{302, fxTone, 0, -46.2, 655, 3498, 1806, 25, 27.1},
		{304, fxTone, 0, -42, 815, 454, 1815, -8.3, 17.6},
		{306, fxTone, 0, -60, 815, 359, 1815, -60, -60},
		{319, fxTone, 0, -60, 815, 204, 1815, -60, -60},
	},
	230: { // C0
		{5, fxTone, 0, -60, 239, 991, 1545, -14.9, -60},
		{20, fxTone, 0, -60, 239, 991, 1545, -0.1, -60},
		{25, fxTone, 0, -4.2, 2315, 991, 1545, -5.4, -60},
		{30, fxTone, 0, -3.9, 2315, 2058, 1545, -14.2, -60},
		{35, fxTone, 0, -0.8, 2315, 2058, 1545, -16.1, -60},
		{40, fxTone, 0, -2.1, 2058, 2058, 1545, -2.2, -60},
		{45, fxTone, 0, -2.7, 2053, 2058, 1545, 0, -2.6},
		{55, fxTone, 0, -4, 1790, 1767, 2327, -13.5, -10.6},
		{70, fxTone, 0, -1.1, 1528, 1767, 2327, -11.8, -9},
		{75, fxTone, 0, -1, 1265, 1767, 2327, -3.5, -4.9},
		{80, fxTone, 0, -1.4, 1265, 2298, 1277, -6.9, -14.6},
		{85, fxTone, 0, 0.5, 1260, 1516, 2327, -0.6, -2.9},
		{90, fxTone, 0, -1.4, 1009, 1015, 2239, -11.7, -9.3},
		{95, fxTone, 0, 1.4, 991, 1003, 2239, -8.1, -5.9},
		{100, fxTone, 0, 0.7, 781, 764, 2239, -10.2, -8},
		{105, fxTone, 0, 1.7, 764, 577, 2239, -13.7, 4.8},
		{110, fxTone, 0, 2.3, 723, 402, 245, -12.3, -13.3},
		{115, fxTone, 0, 0.2, 647, 309, 245, 0.1, -60},
		{130, fxTone, 0, 2.5, 501, 309, 245, -60, -60},
		{135, fxTone, 0, 3.1, 362, 309, 245, -60, -60},
		{145, fxTone, 0, 3.6, 338, 455, 245, -60, -60},
		{150, fxTone, 0, 2, 198, 455, 583, -60, -60},
		{165, fxTone, 0, -5.1, 198, 455, 583, -60, -60},
		{170, fxTone, 0, -6.2, 297, 455, 583, -6.3, -60},
		{175, fxTone, 0, -5.8, 327, 595, 583, -0.1, -60},
		{180, fxTone, 0, -7, 297, 455, 583, -6, -14.7},
		{185, fxTone, 0, -9.9, 327, 595, 787, -2.7, -6.1},
		{190, fxTone, 0, -9, 327, 595, 898, -6.4, -9.9},
		{200, fxTone, 0, -13.4, 251, 484, 787, -8.2, -13.8},
		{205, fxTone, 0, -12.7, 251, 566, 881, -9.3, -12},
		{210, fxTone, 0, -14.5, 251, 484, 881, -0, -0.1},
		{215, fxTone, 0, -17.3, 216, 490, 881, -8.8, -60},
		{220, fxTone, 0, -17.1, 216, 490, 881, -0, -60},
		{225, fxTone, 0, -60, 268, 490, 881, -10.7, -60},
		{245, fxTone, 0, -60, 239, 461, 881, -60, -60},
	},
	232: { // C2
		{5, fxTone, 0, -60, 233, 216, 974, -60, -60},
		{30, fxTone, 0, -60, 478, 910, 974, -5.1, -0.1},
		{35, fxTone, 0, -6.8, 499, 904, 974, -0.1, -0.1},
		{45, fxTone, 0, -6.4, 507, 904, 169, -2.8, -11},
		{50, fxTone, 0, -0.8, 507, 910, 2210, -0.1, 4.5},
		{55, fxTone, 0, -1.4, 507, 916, 2198, -2.7, -2.8},
		{60, fxTone, 0, -1.5, 507, 921, 175, -2.6, -10.3},
		{65, fxTone, 0, 0.9, 519, 933, 2012, -2.9, -7.5},
		{75, fxTone, 0, -1.6, 525, 986, 2006, -3.2, -6.4},
		{80, fxTone, 0, 0.7, 531, 1009, 239, -4.3, -9.4},
		{85, fxTone, 0, -1, 543, 1020, 1814, -0, 1.9},
		{90, fxTone, 0, -1.9, 548, 986, 239, -8.3, -10.9},
		{100, fxTone, 0, -1.6, 566, 1114, 239, -0, 0.1},
		{135, fxTone, 0, -9.7, 560, 1172, 245, -3.5, -9.9},
		{140, fxTone, 0, -9.5, 554, 1172, 344, -7, -12.4},
		{165, fxTone, 0, -25.7, 449, 1155, 251, -14.3, 0},
		{170, fxTone, 0, -25.2, 449, 1155, 251, -9.1, -60},
		{175, fxTone, 0, -22.7, 449, 175, 251, 0, -60},
		{180, fxTone, 0, -9.3, 3312, 175, 251, -60, -60},
		{190, fxTone, 0, -3.9, 3336, 175, 251, -13.1, -60},
		{195, fxTone, 0, -4.4, 3336, 175, 618, -26.6, -60},
		{205, fxTone, 0, 0.3, 3341, 192, 618, -13.2, -60},
		{215, fxTone, 0, -1, 3341, 216, 618, -26.5, -60},
		{220, fxTone, 0, -0.8, 3341, 297, 618, -17.5, -22.8},
		{225, fxTone, 0, -20.7, 1674, 315, 618, -6.6, -10.5},
		{230, fxTone, 0, -21.3, 1668, 315, 618, -7.2, -10.7},
		{235, fxTone, 0, -22.4, 262, 472, 618, -2.9, -9.2},
		{240, fxTone, 0, -60, 245, 472, 484, -14.6, -0},
		{250, fxTone, 0, -60, 233, 472, 484, -60, -60},
	},
	233: { // C3
		{5, fxTone, 0, -60, 222, 490, 414, -60, -60},
		{20, fxTone, 0, -60, 268, 490, 414, -7.4, -60},
		{25, fxTone, 0, -9.4, 3143, 216, 414, -11.7, -60},
		{30, fxTone, 0, -8.3, 3161, 175, 414, -19.1, -60},
		{40, fxTone, 0, -3, 3166, 187, 414, -11.8, -60},
		{45, fxTone, 0, -3.2, 3172, 175, 414, -25.7, -60},
		{50, fxTone, 0, -0.9, 3166, 175, 414, -60, -60},
		{115, fxTone, 0, 3.8, 3376, 175, 414, -60, -60},
		{120, fxTone, 0, 3.9, 3470, 204, 414, -60, -60},
		{180, fxTone, 0, 4.3, 3785, 204, 414, -60, -60},
		{185, fxTone, 0, 4.4, 3785, 204, 414, -16.8, -60},
		{190, fxTone, 0, 4.4, 3785, 204, 414, -16.8, -60},
		{195, fxTone, 0, 4.4, 3785, 204, 414, -26.4, -60},
		{200, fxTone, 0, 0.5, 3785, 338, 414, -18.9, -60},
		{205, fxTone, 0, 0.7, 3773, 338, 414, -30.2, -60},
		{210, fxTone, 0, 0.6, 3773, 356, 414, -31.1, -16.8},
		{215, fxTone, 0, 0.6, 3773, 356, 414, -27.2, -31.4},
		{220, fxTone, 0, -18.9, 233, 3779, 571, 10.3, -8.6},
		{225, fxTone, 0, -20.3, 233, 3779, 741, 16.5, -10.4},
		{230, fxTone, 0, -20.5, 233, 3779, 571, 16.7, -8.9},
		{235, fxTone, 0, -20.3, 239, 3755, 571, 10.1, -8},
		{240, fxTone, 0, -60, 239, 997, 571, -13.4, -60},
		{250, fxTone, 0, -60, 239, 991, 571, -13.5, -60},
	},
	234: { // C4: traced again every 2ms (see speakjet.c's fx_points)
		{2, fxTone, 0, -60, 0, 0, 0, -60, -60},
		{16, fxTone, 0, -57.4, 210, 0, 0, -60, -60},
		{18, fxTone, 0, -36.1, 959, 0, 0, -60, -60},
		{24, fxTone, 0, -60, 941, 0, 0, -60, -60},
		{26, fxTone, 0, -24.6, 910, 0, 0, -60, -60},
		{28, fxTone, 0, -14.7, 953, 0, 0, -60, -60},
		{34, fxTone, 0, -27.7, 956, 0, 0, -60, -60},
		{36, fxTone, 0, -54.5, 195, 0, 0, -60, -60},
		{38, fxTone, 0, -49.4, 182, 0, 0, -60, -60},
		{40, fxTone, 0, -14, 932, 0, 0, -60, -60},
		{42, fxTone, 0, -7.3, 957, 0, 0, -60, -60},
		{48, fxTone, 0, -25.5, 969, 0, 0, -60, -60},
		{50, fxTone, 0, -36.1, 1295, 0, 0, -60, -60},
		{52, fxTone, 0, -13.8, 916, 0, 0, -60, -60},
		{54, fxTone, 0, -4.7, 954, 0, 0, -60, -60},
		{58, fxTone, 0, -9.2, 965, 0, 0, -60, -60},
		{62, fxTone, 0, -27.9, 1093, 0, 0, -60, -60},
		{66, fxTone, 0, -2.7, 956, 0, 0, -60, -60},
		{72, fxTone, 0, -20.8, 956, 0, 0, -60, -60},
		{74, fxTone, 0, -25.3, 776, 0, 0, -60, -60},
		{76, fxTone, 0, -7.3, 935, 0, 0, -60, -60},
		{78, fxTone, 0, -2.2, 958, 0, 0, -60, -60},
		{84, fxTone, 0, -25.1, 946, 0, 0, -60, -60},
		{86, fxTone, 0, -19.8, 791, 0, 0, -60, -60},
		{90, fxTone, 0, -2, 960, 0, 0, -60, -60},
		{96, fxTone, 0, -27.8, 971, 0, 0, -60, -60},
		{98, fxTone, 0, -12.8, 1965, 0, 0, -60, -60},
		{100, fxTone, 0, -3.6, 943, 0, 0, -60, -60},
		{102, fxTone, 0, -2.5, 1937, 0, 0, -60, -60},
		{108, fxTone, 0, -24, 2078, 0, 0, -60, -60},
		{112, fxTone, 0, -0.7, 1936, 0, 0, -60, -60},
		{118, fxTone, 0, -19.9, 1940, 0, 0, -60, -60},
		{124, fxTone, 0, -0.3, 1930, 0, 0, -60, -60},
		{130, fxTone, 0, -24.1, 1933, 0, 0, -60, -60},
		{136, fxTone, 0, 0, 1915, 0, 0, -60, -60},
		{142, fxTone, 0, -27.1, 1927, 0, 0, -60, -60},
		{146, fxTone, 0, -1.1, 1908, 0, 0, -60, -60},
		{152, fxTone, 0, -17.2, 1928, 0, 0, -60, -60},
		{154, fxTone, 0, -23.9, 3103, 0, 0, -60, -60},
		{158, fxTone, 0, -1.5, 3001, 0, 0, -60, -60},
		{164, fxTone, 0, -21.6, 2974, 0, 0, -60, -60},
		{170, fxTone, 0, -1.3, 2994, 0, 0, -60, -60},
		{176, fxTone, 0, -26.8, 3080, 0, 0, -60, -60},
		{180, fxTone, 0, -3.3, 2986, 0, 0, -60, -60},
		{182, fxTone, 0, -1.5, 2988, 0, 0, -60, -60},
		{186, fxTone, 0, -15.3, 2987, 0, 0, -60, -60},
		{188, fxTone, 0, -60, 2987, 0, 0, -60, -60},
		{190, fxTone, 0, -10.2, 2972, 0, 0, -60, -60},
		{192, fxTone, 0, -2.2, 2988, 0, 0, -60, -60},
		{200, fxTone, 0, -23.7, 3106, 0, 0, -60, -60},
		{204, fxTone, 0, -2, 2990, 0, 0, -60, -60},
		{210, fxTone, 0, -24.7, 2962, 0, 0, -60, -60},
		{212, fxTone, 0, -54.1, 255, 0, 0, -60, -60},
		{214, fxTone, 0, -11, 2982, 0, 0, -60, -60},
		{216, fxTone, 0, -5.2, 2988, 0, 0, -60, -60},
		{222, fxTone, 0, -25.5, 2995, 0, 0, -60, -60},
		{224, fxTone, 0, -55.5, 237, 0, 0, -60, -60},
		{226, fxTone, 0, -56.1, 229, 0, 0, -60, -60},
		{228, fxTone, 0, -16.6, 2982, 0, 0, -60, -60},
		{230, fxTone, 0, -13.3, 2989, 0, 0, -60, -60},
		{234, fxTone, 0, -26.1, 2998, 0, 0, -60, -60},
		{236, fxTone, 0, -60, 2998, 0, 0, -60, -60},
		{238, fxTone, 0, -59, 226, 0, 0, -60, -60},
		{249, fxTone, 0, -60, 226, 0, 0, -60, -60},
	},
	236: { // C6
		{5, fxTone, 0, -60, 233, 478, 583, -60, -60},
		{25, fxTone, 0, -60, 2012, 991, 595, -14.7, -15.1},
		{30, fxTone, 0, -22.1, 2029, 980, 997, -10.5, -15.7},
		{35, fxTone, 0, -13.4, 2041, 227, 595, -7.5, -9.8},
		{40, fxTone, 0, -13, 2029, 227, 1569, -14.8, -17.3},
		{55, fxTone, 0, -8.2, 2023, 227, 1569, -3.1, -60},
		{75, fxTone, 0, -5.2, 2023, 1475, 1125, -2.6, -60},
		{90, fxTone, 0, -1.9, 1738, 1475, 1125, -0.9, -1.5},
		{100, fxTone, 0, -0.6, 1376, 1143, 1125, -14.2, -14.8},
		{110, fxTone, 0, 1.1, 1067, 1143, 2799, -9.8, -2.4},
		{130, fxTone, 0, -1.4, 904, 647, 641, 0, 0},
		{135, fxTone, 0, -2.3, 904, 1942, 641, 2.7, 0},
		{140, fxTone, 0, -2.8, 904, 1942, 910, 1.8, -7},
		{145, fxTone, 0, -1.9, 1930, 1942, 910, -1.8, -6.2},
		{150, fxTone, 0, -1.8, 1936, 1942, 1679, -9.1, -13.1},
		{160, fxTone, 0, -2, 1930, 898, 910, -5.1, -8.5},
		{165, fxTone, 0, -3, 906, 898, 647, -1.4, -10.3},
		{185, fxTone, 0, 1.2, 904, 641, 2782, -13.7, -5},
		{200, fxTone, 0, -0, 1225, 1096, 2898, -0.5, 7.4},
		{205, fxTone, 0, -0.4, 1330, 1073, 1382, -10.7, -12.7},
		{240, fxTone, 0, -6, 2029, 1096, 1195, -60, -60},
		{280, fxTone, 0, -16.1, 2035, 571, 531, -7.5, -11},
		{285, fxTone, 0, -60, 239, 571, 531, -8, -10.3},
		{305, fxTone, 0, -60, 239, 478, 566, -13, -60},
	},
	237: { // C7: traced again every 2ms, one tone sweeping in five arches (see speakjet.c's fx_points)
		{2, fxTone, 0, -60, 0, 0, 0, -60, -60},
		{20, fxTone, 0, -55.4, 263, 0, 0, -60, -60},
		{22, fxTone, 0, -32.2, 501, 0, 0, -60, -60},
		{26, fxTone, 0, -26, 524, 0, 0, -60, -60},
		{30, fxTone, 0, -42.8, 519, 0, 0, -60, -60},
		{32, fxTone, 0, -57, 182, 0, 0, -60, -60},
		{34, fxTone, 0, -46.2, 202, 0, 0, -60, -60},
		{36, fxTone, 0, -23.9, 488, 0, 0, -60, -60},
		{38, fxTone, 0, -17.8, 514, 0, 0, -60, -60},
		{44, fxTone, 0, -37.8, 520, 0, 0, -60, -60},
		{46, fxTone, 0, -49.7, 160, 0, 0, -60, -60},
		{48, fxTone, 0, -27, 461, 0, 0, -60, -60},
		{50, fxTone, 0, -15.8, 507, 0, 0, -60, -60},
		{56, fxTone, 0, -27.5, 525, 0, 0, -60, -60},
		{58, fxTone, 0, -49.4, 124, 0, 0, -60, -60},
		{60, fxTone, 0, -26.2, 459, 0, 0, -60, -60},
		{64, fxTone, 0, -11.2, 526, 0, 0, -60, -60},
		{68, fxTone, 0, -22.8, 609, 0, 0, -60, -60},
		{70, fxTone, 0, -33.7, 850, 0, 0, -60, -60},
		{74, fxTone, 0, -4.5, 1029, 0, 0, -60, -60},
		{82, fxTone, 0, -24.6, 1368, 0, 0, -60, -60},
		{84, fxTone, 0, -6.3, 1578, 0, 0, -60, -60},
		{86, fxTone, 0, -0.6, 1639, 0, 0, -60, -60},
		{92, fxTone, 0, -21.4, 1841, 0, 0, -60, -60},
		{98, fxTone, 0, -0, 2242, 0, 0, -60, -60},
		{104, fxTone, 0, -25.5, 2433, 0, 0, -60, -60},
		{108, fxTone, 0, -2.8, 2733, 0, 0, -60, -60},
		{110, fxTone, 0, -3.9, 2643, 0, 0, -60, -60},
		{116, fxTone, 0, -32.7, 1529, 0, 0, -60, -60},
		{118, fxTone, 0, -19.8, 542, 0, 0, -60, -60},
		{122, fxTone, 0, -8.8, 678, 0, 0, -60, -60},
		{126, fxTone, 0, -22.9, 825, 0, 0, -60, -60},
		{132, fxTone, 0, -2.5, 1191, 0, 0, -60, -60},
		{136, fxTone, 0, -14.1, 1345, 0, 0, -60, -60},
		{140, fxTone, 0, -1.3, 1630, 0, 0, -60, -60},
		{154, fxTone, 0, -7.7, 2207, 0, 0, -60, -60},
		{158, fxTone, 0, -3.4, 1974, 0, 0, -60, -60},
		{162, fxTone, 0, -15.1, 1571, 0, 0, -60, -60},
		{164, fxTone, 0, -11.2, 588, 0, 0, -60, -60},
		{166, fxTone, 0, -10.1, 576, 0, 0, -60, -60},
		{202, fxTone, 0, -3.4, 2081, 0, 0, -60, -60},
		{206, fxTone, 0, -4.7, 1594, 0, 0, -60, -60},
		{208, fxTone, 0, -12.9, 763, 0, 0, -60, -60},
		{212, fxTone, 0, -10.6, 627, 0, 0, -60, -60},
		{220, fxTone, 0, -6.3, 823, 0, 0, -60, -60},
		{224, fxTone, 0, -8.7, 771, 0, 0, -60, -60},
		{234, fxTone, 0, -6.4, 1030, 0, 0, -60, -60},
		{236, fxTone, 0, -2.6, 1296, 0, 0, -60, -60},
		{244, fxTone, 0, -4.5, 1286, 0, 0, -60, -60},
		{246, fxTone, 0, -1.4, 1557, 0, 0, -60, -60},
		{252, fxTone, 0, -3.6, 1209, 0, 0, -60, -60},
		{254, fxTone, 0, -9.5, 764, 0, 0, -60, -60},
		{258, fxTone, 0, -10.6, 542, 0, 0, -60, -60},
		{264, fxTone, 0, -12.7, 466, 0, 0, -60, -60},
		{268, fxTone, 0, -7.4, 731, 0, 0, -60, -60},
		{276, fxTone, 0, -8, 706, 0, 0, -60, -60},
		{280, fxTone, 0, -4.3, 939, 0, 0, -60, -60},
		{288, fxTone, 0, -4.2, 979, 0, 0, -60, -60},
		{290, fxTone, 0, -4.7, 1246, 0, 0, -60, -60},
		{294, fxTone, 0, -1.2, 1226, 0, 0, -60, -60},
		{302, fxTone, 0, -7.3, 856, 0, 0, -60, -60},
		{304, fxTone, 0, -11.6, 485, 0, 0, -60, -60},
		{306, fxTone, 0, -12.9, 428, 0, 0, -60, -60},
		{322, fxTone, 0, -13.9, 446, 0, 0, -60, -60},
		{330, fxTone, 0, -18.8, 370, 0, 0, -60, -60},
		{332, fxTone, 0, -15.8, 453, 0, 0, -60, -60},
		{334, fxTone, 0, -22.1, 360, 0, 0, -60, -60},
		{336, fxTone, 0, -20.4, 379, 0, 0, -60, -60},
		{340, fxTone, 0, -16.4, 561, 0, 0, -60, -60},
		{354, fxTone, 0, -21.1, 437, 0, 0, -60, -60},
		{358, fxTone, 0, -25, 469, 0, 0, -60, -60},
		{360, fxTone, 0, -24.1, 614, 0, 0, -60, -60},
		{362, fxTone, 0, -22.6, 459, 0, 0, -60, -60},
		{372, fxTone, 0, -25.6, 476, 0, 0, -60, -60},
		{376, fxTone, 0, -39.6, 556, 0, 0, -60, -60},
		{382, fxTone, 0, -32.7, 483, 0, 0, -60, -60},
		{388, fxTone, 0, -60, 558, 0, 0, -60, -60},
		{405, fxTone, 0, -60, 558, 0, 0, -60, -60},
	},
	254: { // M2
		{5, fxBuzz, 128, -60, 540, 1687, 2856, 37.6, 43.5},
		{10, fxBuzz, 177, -60, 540, 1707, 2858, 38.1, 44.6},
		{15, fxBuzz, 498, -60, 540, 1707, 2866, 4.2, 8.3},
		{20, fxBuzz, 498, -11, 539, 1703, 2866, 2.2, 6.6},
		{25, fxBuzz, 85, -9.1, 530, 1703, 2866, 2.2, 6.6},
		{45, fxBuzz, 87, -4.3, 502, 1724, 2919, 2.9, 7.1},
		{155, fxBuzz, 87, -7.1, 735, 1954, 3166, 3.2, 7.2},
		{285, fxBuzz, 87, -6.6, 987, 2198, 3417, 5.9, 9.2},
		{410, fxBuzz, 87, -5.4, 714, 1934, 3145, 2.8, 6.9},
		{525, fxBuzz, 87, -6.4, 488, 1710, 2903, 2.8, 7.5},
		{530, fxBuzz, 103, -6.7, 488, 1710, 2903, 2.5, 7.2},
		{535, fxBuzz, 492, -13.6, 489, 1710, 2902, 2.4, 7.2},
		{540, fxBuzz, 83, -13.8, 489, 1710, 2902, 2.7, 7.2},
		{545, fxBuzz, 492, -22.2, 491, 1712, 2903, 4.7, 9.4},
		{550, fxBuzz, 492, -60, 491, 1710, 2901, 5.4, 9.9},
		{555, fxBuzz, 492, -60, 491, 1707, 2899, 38.2, 43.8},
	},
}

// the traced effects, by code (from fxRows)
var fxPoints = func() map[int][]fxPoint {
	m := map[int][]fxPoint{}
	for code, rows := range fxRows {
		for _, r := range rows {
			m[code] = append(m[code], fxPoint{ms: int(r[0]), kind: int(r[1]), pitch: int(r[2]), db: r[3],
				freq: [3]int{int(r[4]), int(r[5]), int(r[6])}, rel: [3]float64{0, r[7], r[8]}})
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

// C8 and C9 are noise from about 400Hz to 2.8kHz, gated by a saw (see
// noiseDepth) about 15dB deep every 19ms: oscillators 4 and 5 spread wide, each
// random value held 0.75ms (a band with edges, as on the chip), at these levels
// in dB every 5ms. the gate keeps time with Speed, not Pitch
var c8DB = []float64{
	-16.2, -11.8, -9.5, -5.5, -1.1, -0.5, -0.4, -0.3, -0.1, -0.3,
	-0.2, -0.1, -0.3, -0.5, -0.5, -0.2, -0.9, -1.3, -1.8, -5.2,
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
	f.noiseDepth = 0.826
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
var fxvPointsRows = map[int][][7]float64{
	212: { // A2
		{1.00, 262, 1651, 1422, 5.46, 9.84, 3.08},
		{125.61, 260, 1549, 2556, 5.65, 9.24, 9.35},
		{22.95, 235, 1533, 2488, 6.63, 9.81, 8.93},
		{24.34, 226, 1523, 2417, 7.24, 12.13, 10.91},
		{23.09, 679, 1509, 2347, 4.85, 15.62, 14.52},
		{11.52, 219, 1500, 2311, 7.76, 16.22, 14.07},
		{11.48, 227, 1494, 2265, 6.96, 15.86, 15.20},
		{11.50, 224, 1486, 2262, 7.05, 15.82, 15.94},
		{11.52, 215, 1468, 2226, 7.11, 15.58, 15.38},
		{11.53, 209, 1458, 2209, 7.23, 15.80, 13.78},
		{11.45, 209, 1444, 2193, 6.94, 15.75, 12.62},
		{11.45, 189, 1426, 2102, 7.67, 15.86, 13.59},
		{11.66, 197, 1434, 2015, 8.29, 16.90, 13.79},
		{11.50, 255, 1420, 2216, 6.54, 17.48, 8.01},
		{11.46, 211, 1405, 1940, 9.34, 17.45, 15.49},
		{11.43, 208, 1402, 1918, 10.11, 17.49, 15.89},
		{11.42, 198, 1395, 1883, 10.80, 17.58, 15.01},
		{11.78, 204, 1395, 1803, 9.51, 18.15, 16.25},
		{11.46, 209, 1375, 1783, 8.66, 17.79, 16.84},
		{11.44, 236, 1352, 1772, 6.21, 17.09, 16.15},
		{11.44, 230, 1340, 1734, 6.65, 16.83, 16.72},
		{11.45, 224, 1328, 1674, 6.81, 17.20, 16.27},
		{11.46, 252, 1332, 1615, 5.75, 17.88, 16.68},
		{11.45, 239, 1305, 1598, 5.87, 17.50, 16.52},
		{11.44, 250, 1285, 1562, 5.76, 17.22, 16.06},
		{11.47, 679, 1265, 1514, 5.10, 16.54, 15.33},
		{11.45, 679, 1279, 1469, 4.86, 17.50, 17.17},
		{11.41, 258, 1257, 1422, 5.25, 18.15, 18.08},
		{11.50, 262, 1315, 1999, 6.11, 17.10, 9.94},
		{11.45, 282, 1227, 1977, 5.22, 16.80, 9.15},
		{11.44, 282, 1264, 2558, 5.28, 23.19, 5.94},
		{11.42, 272, 1245, 2488, 5.50, 26.77, 6.38},
		{11.43, 270, 1220, 2578, 5.71, 29.62, 7.58},
		{11.44, 275, 1193, 2581, 5.23, 31.00, 5.37},
		{11.46, 269, 1174, 2578, 5.50, 31.00, 6.95},
		{11.38, 679, 1143, 1422, 4.63, 31.00, 3.15},
		{11.20, 250, 1118, 2492, 6.85, 31.00, 5.91},
		{11.50, 242, 1092, 1422, 6.88, 31.00, 4.31},
		{11.39, 192, 1064, 1422, 11.19, 31.00, 1.65},
		{11.46, 183, 1033, 1422, 11.78, 31.00, 3.34},
		{11.29, 679, 1018, 1422, 3.22, 28.65, 2.44},
		{11.59, 911, 1078, 1422, 17.97, 16.82, 2.81},
		{11.97, 881, 1054, 1422, 18.24, 17.03, 2.42},
		{10.87, 850, 1040, 1422, 19.04, 17.42, 2.80},
		{12.09, 810, 1031, 1422, 18.82, 16.88, 1.58},
		{11.45, 781, 1013, 1422, 19.69, 17.52, 2.05},
		{10.74, 740, 1001, 1422, 19.92, 17.81, 3.16},
		{11.46, 715, 980, 1422, 20.36, 18.38, 2.57},
		{11.45, 679, 968, 1422, 20.58, 18.73, 2.70},
		{14.17, 672, 915, 2536, 16.48, 15.21, 4.63},
		{10.75, 630, 912, 2600, 12.57, 11.88, 4.62},
		{11.58, 601, 918, 2495, 14.14, 13.33, 4.72},
		{11.54, 549, 911, 2445, 14.24, 13.27, 6.28},
		{11.52, 535, 924, 2445, 14.22, 13.99, 8.79},
		{11.52, 517, 921, 2432, 13.84, 13.86, 7.64},
		{11.42, 396, 877, 1422, 15.13, 13.10, 2.09},
		{10.82, 409, 845, 1422, 15.81, 14.73, 2.60},
		{11.52, 386, 841, 1422, 14.56, 15.01, 3.18},
		{11.57, 366, 831, 2576, 12.89, 14.69, 4.58},
		{11.54, 313, 831, 1422, 12.18, 14.11, 3.99},
		{11.50, 215, 797, 1422, 19.92, 14.28, 3.13},
		{10.75, 200, 803, 1422, 19.90, 13.82, 2.59},
		{11.59, 186, 786, 1422, 17.65, 12.75, 1.26},
		{11.50, 189, 769, 1422, 12.85, 12.85, 2.06},
		{45.73, 206, 711, 1422, 9.25, 16.00, 2.64},
		{11.41, 220, 714, 1422, 6.82, 14.01, 2.82},
		{12.17, 679, 1143, 1422, 3.07, 7.34, 0.90},
		{11.50, 679, 1143, 1422, 3.07, 7.34, 0.90},
	},
}

var fxvPoints = func() map[int][]fxvPoint {
	m := map[int][]fxvPoint{}
	for code, rows := range fxvPointsRows {
		for _, r := range rows {
			m[code] = append(m[code], fxvPoint{ms: r[0], freq: formants{r[1], r[2], r[3]}, vol: [3]float64{r[4], r[5], r[6]}})
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

var c1PointsRows = [][8]float64{
	{1.00, 332.8, 877, 1443, 1997, 10.15, 1.21, 0.00},
	{1.90, 332.8, 765, 1439, 2089, 6.63, 0.64, 0.00},
	{1.90, 332.8, 848, 1354, 2096, 11.68, 3.93, 0.00},
	{3.80, 332.8, 1112, 1172, 2064, 25.95, 12.43, 0.00},
	{3.80, 347.2, 896, 1604, 2203, 17.19, 11.29, 0.00},
	{1.90, 347.2, 823, 1626, 2249, 16.38, 14.57, 0.00},
	{3.80, 347.2, 745, 1282, 2297, 21.90, 28.86, 0.00},
	{3.80, 347.2, 569, 1574, 2179, 22.61, 16.36, 0.00},
	{1.90, 347.2, 557, 1662, 2137, 23.54, 12.43, 0.19},
	{3.80, 347.2, 1014, 1357, 2190, 29.82, 23.86, 1.71},
	{1.90, 347.2, 1076, 1440, 2217, 25.70, 23.50, 1.52},
	{3.80, 347.2, 1000, 1890, 2270, 8.75, 15.50, 0.00},
	{3.80, 399.3, 1670, 2073, 2140, 25.70, 25.79, 5.33},
	{1.90, 399.3, 1791, 2064, 2103, 28.79, 28.14, 7.57},
	{3.80, 420.3, 1319, 1712, 2118, 16.98, 23.57, 10.62},
	{3.80, 420.3, 1147, 1526, 1933, 20.17, 21.00, 7.67},
	{1.90, 443.7, 1076, 1474, 1832, 22.58, 19.74, 5.90},
	{3.80, 443.7, 855, 1777, 1710, 23.92, 15.55, 4.38},
	{1.90, 469.8, 747, 1667, 1730, 23.49, 15.17, 4.76},
	{3.80, 469.8, 533, 926, 1928, 20.44, 17.83, 7.81},
	{1.90, 499.1, 433, 1078, 2032, 17.58, 19.17, 12.00},
	{1.90, 499.1, 334, 1303, 2136, 14.54, 20.50, 16.57},
	{1.90, 499.1, 264, 1418, 2218, 11.13, 20.60, 20.00},
	{1.90, 532.4, 279, 1198, 2232, 6.65, 16.98, 20.00},
	{1.90, 532.4, 294, 977, 2245, 2.18, 13.36, 20.00},
	{1.90, 532.4, 369, 900, 2258, 1.51, 11.05, 20.48},
	{1.90, 532.4, 480, 909, 2272, 3.13, 9.52, 21.24},
	{1.90, 570.4, 590, 918, 2285, 4.75, 8.00, 22.00},
	{3.80, 570.4, 133, 1255, 2312, 6.46, 14.86, 24.29},
	{1.90, 570.4, 59, 1412, 2295, 6.00, 18.32, 24.00},
	{1.90, 570.4, 241, 1550, 2228, 3.33, 21.85, 21.33},
	{1.90, 532.4, 424, 1688, 2162, 0.67, 25.37, 18.67},
	{1.90, 532.4, 376, 1632, 2164, 3.57, 22.61, 18.29},
	{1.90, 532.4, 250, 1511, 2188, 8.33, 17.75, 18.67},
	{1.90, 532.4, 128, 1447, 2183, 11.90, 14.08, 18.50},
	{3.80, 532.4, -70, 2111, 1749, 2.38, 23.42, 10.50},
	{1.90, 532.4, -125, 2182, 1667, 0.33, 24.96, 8.79},
	{5.70, 532.4, -152, 1612, 1829, 2.33, 20.25, 10.50},
	{1.90, 499.1, -161, 1422, 1882, 3.00, 18.68, 11.07},
	{1.90, 499.1, -158, 1288, 1931, 3.46, 17.39, 11.64},
	{13.30, 499.1, 106, 1510, 2157, 2.35, 14.28, 15.64},
	{1.90, 499.1, 130, 1545, 2182, 2.17, 14.33, 15.43},
	{1.90, 469.8, 131, 1586, 2195, 1.96, 15.21, 13.90},
	{15.20, 469.8, 139, 1914, 2298, 0.24, 22.26, 1.71},
	{1.90, 443.7, 140, 1955, 2311, 0.03, 23.14, 0.19},
	{1.90, 443.7, 202, 1904, 2288, 0.25, 21.98, 1.33},
	{17.10, 420.3, 845, 1330, 2027, 2.82, 8.91, 15.05},
	{1.90, 420.3, 924, 1324, 2017, 3.21, 8.00, 15.57},
	{3.80, 420.3, 1105, 1505, 2062, 4.36, 8.00, 13.29},
	{1.90, 399.3, 1195, 1595, 2084, 4.93, 8.00, 12.14},
	{13.30, 399.3, 1829, 2229, 2241, 8.93, 8.00, 4.14},
	{7.60, 363.0, 1497, 2096, 2298, 7.89, 7.82, 4.74},
	{11.40, 347.2, 965, 1873, 2381, 6.18, 7.53, 5.88},
	{1.90, 347.2, 915, 1842, 2391, 5.95, 7.34, 6.04},
	{19.00, 307.2, 1048, 1633, 2415, 4.52, 3.05, 6.99},
	{22.80, 275.4, 854, 1615, 2633, 1.84, 1.96, 3.87},
	{17.10, 242.0, 751, 1580, 2706, 0.76, 0.03, 3.01},
	{20.90, 210.2, 530, 1720, 2802, 0.00, 0.04, 2.14},
	{1.90, 199.7, 530, 1720, 2825, 0.00, 0.13, 1.86},
	{11.40, 199.7, 530, 1720, 2965, 0.00, 0.70, 0.14},
	{7.60, 199.7, 530, 1720, 2926, 0.00, 0.25, 0.00},
}

var c1Points = func() []c1Point {
	var p []c1Point
	for _, r := range c1PointsRows {
		p = append(p, c1Point{ms: r[0], pitch: r[1], freq: formants{r[2], r[3], r[4]}, vol: [3]float64{r[5], r[6], r[7]}})
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
// tenths of a millisecond, its frequency in Hz and its volume in hundredths,
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
			10, 794, 0, 50, 794, 0, 60, 794, 252, 170, 803, 3100, 650, 802, 3011, 780, 809, 641,
			790, 809, 0, 820, 809, 0,
		}},
		{1, []int{
			10, 409, 0, 170, 409, 0, 180, 409, 408, 290, 450, 700, 530, 539, 844, 580, 570, 1056,
			590, 570, 0, 820, 570, 0,
		}},
	},
	201: { // R1
		{0, []int{
			10, 635, 0, 30, 635, 0, 40, 635, 37, 50, 681, 79, 80, 681, 169, 120, 686, 174,
			160, 660, 247, 200, 699, 324, 280, 698, 347, 350, 733, 341, 500, 735, 335, 580, 776, 342,
			670, 767, 327, 690, 715, 266, 720, 701, 233, 760, 744, 257, 780, 768, 145, 790, 768, 0,
			860, 768, 0,
		}},
		{1, []int{
			10, 1105, 0, 60, 1105, 0, 70, 1105, 209, 80, 1105, 331, 110, 1105, 1319, 160, 1105, 2886,
			650, 1105, 3100, 780, 1107, 362, 790, 1107, 254, 820, 1107, 37, 860, 1107, 0,
		}},
	},
	202: { // R2
		{0, []int{
			10, 1300, 0, 40, 1300, 18, 60, 1300, 101, 70, 1300, 244, 80, 1300, 418, 90, 1300, 715,
			120, 1301, 1865, 160, 1304, 3078, 670, 1304, 3100, 720, 1302, 2124, 750, 1304, 1180, 760, 1305, 970,
			790, 1306, 539, 800, 1305, 711, 810, 1305, 0, 950, 1305, 0,
		}},
		{1, []int{
			10, 733, 0, 60, 733, 0, 70, 733, 103, 80, 653, 75, 120, 659, 241, 160, 685, 308,
			700, 375, 355, 730, 377, 324, 750, 415, 259, 760, 418, 239, 800, 376, 257, 830, 412, 223,
			900, 421, 24, 910, 420, 60, 920, 420, 0, 950, 420, 0,
		}},
		{2, []int{
			10, 1697, 0, 150, 1697, 0, 160, 1697, 367, 200, 1667, 518, 240, 1606, 690, 250, 1606, 0,
			950, 1606, 0,
		}},
	},
	203: { // R3
		{0, []int{
			10, 1605, 0, 30, 1605, 0, 40, 1605, 95, 70, 1605, 451, 130, 1605, 2438, 660, 1606, 2828,
			800, 1607, 109, 810, 1606, 135, 820, 1606, 0, 960, 1606, 0,
		}},
		{1, []int{
			10, 636, 0, 30, 636, 0, 200, 694, 736, 570, 771, 320, 960, 706, 0,
		}},
		{2, []int{
			10, 345, 0, 170, 345, 0, 180, 345, 121, 650, 367, 436, 730, 296, 177, 820, 299, 62,
			830, 299, 0, 960, 299, 0,
		}},
	},
	204: { // R4
		{0, []int{
			10, 1901, 13, 30, 1901, 75, 40, 1901, 183, 50, 1901, 334, 140, 1906, 3100, 650, 1907, 2920,
			790, 1905, 130, 800, 1905, 0, 830, 1905, 0,
		}},
		{1, []int{
			10, 629, 0, 30, 629, 0, 40, 629, 59, 60, 603, 85, 90, 588, 312, 110, 555, 390,
			140, 549, 425, 170, 629, 475, 390, 863, 217, 540, 863, 106, 590, 857, 0, 830, 857, 0,
		}},
		{2, []int{
			10, 462, 0, 210, 462, 0, 220, 462, 521, 230, 463, 402, 590, 719, 358, 640, 737, 394,
			680, 702, 307, 720, 728, 292, 730, 728, 0, 830, 728, 0,
		}},
		{3, []int{
			10, 282, 0, 330, 282, 0, 340, 282, 197, 370, 279, 222, 470, 314, 503, 630, 367, 452,
			680, 365, 408, 690, 373, 355, 700, 441, 373, 710, 441, 0, 830, 441, 0,
		}},
		{4, []int{
			10, 1476, 0, 370, 1476, 0, 380, 1476, 177, 500, 1634, 298, 510, 1634, 0, 830, 1634, 0,
		}},
	},
	205: { // R5
		{0, []int{
			10, 347, 12, 50, 347, 75, 70, 347, 185, 100, 307, 258, 120, 302, 291, 130, 310, 306,
			150, 359, 372, 170, 379, 426, 640, 382, 419, 690, 349, 399, 760, 431, 419, 800, 492, 206,
			830, 492, 108, 850, 496, 103, 860, 496, 0, 890, 496, 0,
		}},
		{1, []int{
			10, 2206, 0, 30, 2206, 43, 50, 2206, 236, 70, 2206, 625, 120, 2207, 2483, 670, 2208, 2506,
			690, 2208, 1790, 770, 2207, 271, 790, 2208, 97, 830, 2209, 0, 890, 2209, 0,
		}},
		{2, []int{
			10, 810, 0, 90, 810, 0, 100, 810, 346, 170, 769, 414, 640, 757, 416, 670, 740, 404,
			680, 812, 0, 690, 812, 171, 720, 831, 212, 760, 787, 218, 790, 736, 175, 800, 736, 0,
			890, 736, 0,
		}},
	},
	206: { // R6
		{0, []int{
			10, 1668, 0, 40, 1668, 58, 80, 1646, 291, 130, 1489, 217, 140, 1707, 0, 250, 1707, 0,
			260, 1707, 200, 340, 1643, 426, 460, 1482, 312, 470, 542, 0, 680, 542, 0, 690, 542, 579,
			750, 518, 484, 780, 469, 279, 810, 442, 416, 850, 504, 335, 860, 504, 0, 930, 504, 0,
		}},
		{1, []int{
			10, 477, 0, 30, 477, 0, 40, 477, 139, 70, 471, 414, 110, 462, 526, 140, 377, 578,
			160, 358, 695, 220, 374, 720, 400, 307, 669, 520, 287, 199, 530, 1622, 0, 690, 1622, 0,
			700, 1622, 346, 760, 1670, 263, 830, 1688, 287, 860, 1681, 119, 900, 1678, 0, 930, 1678, 0,
		}},
		{2, []int{
			10, 2511, 0, 40, 2511, 0, 50, 2511, 75, 70, 2511, 326, 80, 2511, 529, 130, 2510, 3084,
			700, 2509, 3100, 780, 2508, 677, 810, 2508, 172, 830, 2506, 50, 840, 2506, 0, 930, 2506, 0,
		}},
		{3, []int{
			10, 760, 0, 130, 760, 0, 140, 760, 320, 150, 713, 466, 220, 735, 605, 610, 466, 794,
			620, 466, 0, 930, 466, 0,
		}},
		{4, []int{
			10, 878, 0, 250, 878, 0, 260, 878, 106, 440, 869, 332, 640, 643, 682, 680, 625, 746,
			690, 625, 0, 930, 625, 0,
		}},
	},
	207: { // R7
		{0, []int{
			10, 2805, 0, 20, 2805, 0, 30, 2805, 73, 40, 2805, 160, 110, 2809, 2416, 640, 2809, 2959,
			750, 2810, 485, 790, 2811, 0, 970, 2811, 0,
		}},
		{1, []int{
			10, 593, 0, 30, 593, 0, 40, 593, 112, 70, 565, 345, 90, 492, 334, 120, 475, 367,
			130, 493, 486, 140, 1002, 0, 150, 1002, 0, 160, 1002, 333, 280, 1133, 143, 290, 260, 0,
			380, 260, 0, 390, 260, 188, 670, 287, 444, 750, 403, 472, 820, 409, 97, 970, 410, 0,
		}},
		{2, []int{
			10, 639, 0, 100, 639, 0, 110, 639, 299, 180, 596, 566, 340, 697, 334, 350, 761, 0,
			400, 761, 0, 410, 761, 183, 500, 780, 128, 510, 1077, 0, 670, 1077, 0, 680, 1077, 255,
			770, 1138, 282, 820, 1141, 71, 880, 1141, 78, 910, 1144, 0, 970, 1144, 0,
		}},
		{3, []int{
			10, 424, 0, 200, 424, 0, 210, 424, 419, 610, 591, 366, 710, 572, 346, 720, 572, 0,
			970, 572, 0,
		}},
		{4, []int{
			10, 901, 0, 270, 901, 0, 280, 901, 283, 640, 1196, 383, 700, 1273, 139, 710, 1273, 0,
			970, 1273, 0,
		}},
	},
	208: { // R8
		{0, []int{
			10, 728, 0, 20, 728, 0, 30, 728, 89, 60, 706, 540, 120, 756, 1200, 130, 2783, 0,
			140, 2783, 411, 170, 2770, 583, 230, 2655, 627, 310, 2660, 270, 350, 2663, 177, 360, 2670, 222,
			370, 1910, 0, 420, 1910, 0, 430, 1910, 85, 480, 1905, 225, 510, 1903, 402, 660, 1828, 599,
			720, 1792, 456, 730, 1730, 385, 740, 689, 0, 747, 689, 0, 757, 689, 773, 784, 682, 768,
			801, 624, 798, 811, 614, 780, 836, 616, 365, 883, 619, 0, 913, 619, 0,
		}},
		{2, []int{
			10, 575, 0, 110, 575, 0, 120, 575, 903, 140, 621, 1143, 200, 679, 1670, 670, 614, 1670,
			720, 540, 1353, 762, 541, 1112, 772, 574, 1749, 779, 574, 0, 913, 574, 0,
		}},
		{3, []int{
			10, 3111, 0, 50, 3111, 0, 60, 3111, 175, 110, 3111, 1417, 160, 3110, 2868, 690, 3112, 2571,
			757, 3111, 925, 801, 3111, 215, 811, 3111, 129, 826, 3112, 126, 836, 3112, 0, 913, 3112, 0,
		}},
		{4, []int{
			10, 2311, 0, 190, 2311, 0, 200, 2311, 107, 270, 2317, 180, 340, 2316, 399, 410, 2228, 386,
			540, 2220, 193, 550, 2211, 242, 560, 2211, 0, 913, 2211, 0,
		}},
	},
	209: { // R9
		{0, []int{
			10, 680, 0, 20, 680, 0, 40, 678, 202, 70, 682, 622, 100, 689, 906, 150, 651, 1385,
			190, 678, 1498, 670, 612, 1436, 710, 534, 1169, 730, 531, 1075, 764, 526, 946, 774, 545, 1372,
			781, 545, 0, 917, 545, 0,
		}},
		{1, []int{
			10, 1693, 0, 40, 1693, 0, 50, 1693, 130, 70, 1710, 276, 120, 1800, 400, 130, 1677, 0,
			160, 1677, 0, 170, 1677, 326, 230, 1667, 526, 340, 1668, 518, 400, 1605, 516, 520, 1602, 497,
			600, 1538, 547, 730, 1548, 412, 752, 1512, 366, 798, 1587, 323, 805, 1587, 0, 917, 1587, 0,
		}},
		{3, []int{
			10, 3412, 0, 40, 3412, 0, 50, 3412, 150, 70, 3412, 524, 90, 3412, 1187, 170, 3413, 4266,
			660, 3413, 4074, 759, 3411, 1025, 781, 3410, 620, 798, 3410, 451, 805, 3410, 0, 917, 3410, 0,
		}},
		{4, []int{
			10, 334, 0, 180, 334, 0, 190, 334, 260, 210, 334, 149, 220, 350, 143, 250, 348, 129,
			260, 328, 119, 280, 347, 107, 290, 324, 101, 310, 344, 97, 320, 327, 98, 340, 360, 108,
			350, 331, 100, 380, 300, 110, 410, 312, 149, 430, 340, 184, 440, 323, 195, 450, 688, 0,
			752, 688, 0, 759, 688, 621, 786, 684, 636, 805, 630, 660, 817, 618, 658, 887, 628, 0,
			917, 628, 0,
		}},
	},
	210: { // A0
		{0, []int{
			10, 1615, 0, 120, 1615, 0, 130, 1615, 571, 2040, 2653, 443, 2050, 2574, 512, 2060, 415, 0,
			3170, 415, 0, 3180, 415, 1070, 3220, 437, 1205, 3350, 385, 616, 3370, 327, 631, 3380, 327, 0,
			3410, 327, 0,
		}},
		{1, []int{
			10, 305, 0, 320, 305, 0, 330, 305, 192, 340, 301, 132, 400, 294, 535, 470, 215, 289,
			520, 281, 605, 590, 242, 504, 650, 283, 802, 700, 262, 660, 1120, 340, 840, 1170, 314, 927,
			1210, 360, 816, 1270, 269, 845, 1280, 274, 749, 1290, 361, 921, 1300, 224, 0, 1360, 224, 0,
			1370, 224, 784, 1380, 266, 866, 1410, 291, 1242, 1460, 236, 1142, 1500, 280, 1538, 1540, 247, 1421,
			1580, 279, 1769, 1690, 263, 1830, 2930, 384, 3100, 3160, 305, 1694, 3170, 305, 0, 3410, 305, 0,
		}},
	},
	214: { // A4
		{0, []int{
			10, 247, 37, 110, 247, 1050, 140, 247, 2036, 210, 247, 1965, 320, 261, 3100, 930, 496, 3100,
			970, 534, 3100, 1451, 1122, 2799, 1611, 1712, 2041, 1811, 2464, 1874, 1851, 2689, 1439, 1971, 3578, 1041,
			2011, 3878, 1859, 2051, 3911, 2295, 2091, 3913, 2153, 2491, 3913, 417, 2501, 3913, 0, 2531, 3913, 0,
		}},
	},
	215: { // A5
		{0, []int{
			10, 644, 0, 30, 644, 0, 40, 644, 167, 50, 641, 167, 90, 641, 1267, 130, 636, 1894,
			1890, 405, 1913, 1920, 417, 1890, 1950, 389, 1980, 1990, 413, 1827, 2030, 358, 2181, 2060, 409, 1766,
			2070, 409, 1747, 2100, 345, 2417, 2110, 338, 2584, 2120, 341, 2461, 2160, 409, 1578, 2190, 341, 2388,
			2240, 336, 2487, 2265, 377, 1581, 2273, 410, 1524, 2278, 410, 0, 5590, 410, 0,
		}},
		{1, []int{
			10, 333, 0, 2273, 333, 0, 2278, 333, 1062, 2298, 333, 913, 2398, 331, 3100, 2568, 333, 631,
			2578, 393, 461, 2588, 409, 421, 2608, 390, 541, 2618, 333, 796, 2678, 331, 2645, 2758, 334, 1697,
			2768, 384, 1230, 2788, 411, 1039, 2798, 401, 1137, 2818, 333, 1958, 2858, 333, 2472, 2868, 335, 2098,
			2888, 405, 1364, 2898, 413, 1307, 2918, 384, 1535, 2928, 338, 1951, 2948, 335, 2271, 2958, 342, 2017,
			2968, 386, 1649, 2988, 414, 1460, 3008, 388, 1625, 3018, 350, 1884, 3028, 348, 1963, 3058, 414, 1563,
			3078, 408, 1585, 3098, 376, 1751, 3128, 418, 1615, 3158, 403, 1628, 3198, 423, 1621, 5002, 628, 1444,
			5510, 628, 64, 5590, 628, 0,
		}},
	},
	216: { // A6
		{0, []int{
			10, 308, 22, 70, 308, 171, 110, 308, 1200, 150, 457, 2815, 570, 1649, 1859, 1270, 1649, 72,
			1280, 1649, 0, 1970, 1649, 0,
		}},
		{1, []int{
			10, 262, 0, 160, 262, 0, 170, 262, 2141, 240, 304, 3057, 710, 799, 3100, 1650, 804, 1509,
			1900, 804, 135, 1940, 804, 0, 1970, 804, 0,
		}},
	},
	217: { // A7
		{0, []int{
			10, 1145, 0, 20, 1145, 0, 30, 1145, 67, 120, 1156, 2309, 1540, 1462, 1032, 2400, 1463, 226,
			3000, 1463, 111, 3460, 1463, 73, 4120, 1456, 85, 4130, 1456, 0, 5900, 1456, 0,
		}},
		{1, []int{
			10, 418, 0, 110, 418, 0, 120, 418, 48, 150, 418, 50, 940, 418, 1177, 2390, 515, 3093,
			4770, 867, 2850, 5650, 928, 203, 5750, 928, 0, 5900, 928, 0,
		}},
		{2, []int{
			10, 876, 0, 140, 876, 0, 150, 876, 48, 850, 874, 901, 2450, 1229, 2465, 4770, 1748, 2190,
			5870, 1815, 0, 5900, 1815, 0,
		}},
	},
	218: { // A8
		{0, []int{
			10, 348, 16, 70, 348, 98, 200, 348, 1080, 640, 999, 3100, 1300, 1003, 33, 1310, 284, 0,
			1620, 284, 0, 1730, 280, 400, 1800, 280, 1172, 2100, 498, 2966, 2690, 502, 39, 2760, 502, 0,
		}},
		{1, []int{
			10, 289, 0, 290, 289, 0, 300, 289, 1106, 350, 290, 1380, 650, 498, 2922, 1230, 501, 119,
			1240, 544, 0, 1460, 544, 11, 1610, 544, 553, 2070, 1966, 2370, 2680, 2008, 52, 2760, 2008, 0,
		}},
	},
	219: { // A9
		{0, []int{
			10, 1303, 0, 20, 1303, 0, 40, 1302, 179, 50, 1302, 362, 100, 1301, 1890, 560, 1295, 2068,
			2130, 304, 3100, 2250, 252, 1216, 2400, 251, 0, 2560, 251, 0,
		}},
		{1, []int{
			10, 2431, 0, 30, 2431, 0, 40, 2431, 129, 110, 2449, 1676, 810, 2431, 1913, 2420, 1334, 2296,
			2490, 1308, 393, 2520, 1308, 46, 2530, 1308, 0, 2560, 1308, 0,
		}},
	},
	220: { // B0
		{0, []int{
			10, 2603, 0, 20, 2603, 0, 30, 2603, 102, 110, 2606, 3076, 1150, 2607, 3100, 1740, 2605, 31,
			1780, 2605, 0,
		}},
	},
	221: { // B1
		{0, []int{
			10, 1421, 0, 30, 1421, 0, 40, 1421, 169, 80, 1416, 1565, 130, 1473, 2909, 1490, 3837, 891,
			1530, 3879, 996, 1570, 3822, 875, 2630, 1546, 2752, 2700, 1418, 3074, 3870, 1407, 51, 3940, 1407, 0,
		}},
	},
	222: { // B2
		{0, []int{
			10, 1406, 0, 30, 1406, 0, 40, 1406, 117, 50, 1406, 334, 120, 1412, 3032, 1220, 1414, 3018,
			1270, 1562, 1652, 1280, 1869, 0, 1310, 1869, 0, 1320, 1869, 2459, 1390, 2266, 2525, 1400, 2395, 0,
			1410, 2395, 2678, 1460, 2507, 3023, 1590, 2508, 3007, 1630, 2461, 1731, 1640, 1767, 0, 1650, 1767, 1698,
			1710, 1707, 2906, 2820, 1707, 2887, 3920, 1708, 118, 3990, 1709, 10, 4030, 1709, 0,
		}},
		{1, []int{
			10, 260, 0, 2040, 260, 0, 2070, 257, 32, 2160, 259, 251, 2230, 298, 509, 2610, 601, 510,
			3920, 600, 0, 4030, 600, 0,
		}},
	},
	223: { // B3
		{0, []int{
			10, 3103, 0, 20, 3103, 0, 30, 3103, 130, 120, 3114, 2135, 1100, 3112, 2072, 1150, 3076, 788,
			1170, 3075, 154, 1180, 3075, 0, 3100, 3075, 0,
		}},
		{1, []int{
			10, 615, 0, 30, 615, 0, 40, 615, 130, 120, 596, 815, 2280, 602, 815, 3020, 602, 0,
			3100, 602, 0,
		}},
		{2, []int{
			10, 1416, 0, 1100, 1416, 0, 1110, 1416, 67, 1140, 1422, 619, 1200, 1404, 2994, 2310, 1404, 3082,
			3070, 1404, 0, 3100, 1404, 0,
		}},
	},
	224: { // B4
		{0, []int{
			10, 1828, 0, 20, 1828, 0, 30, 1828, 216, 70, 1828, 1632, 140, 1855, 3020, 230, 1835, 1678,
			310, 1805, 3100, 360, 1841, 2495, 2730, 1805, 2951, 2780, 1853, 1682, 2860, 1807, 1498, 2920, 1858, 3074,
			3690, 1834, 166, 3740, 1834, 29, 3790, 1839, 0,
		}},
		{1, []int{
			10, 1620, 0, 400, 1620, 0, 410, 1620, 1524, 1470, 357, 2167, 1520, 325, 2115, 1560, 353, 2163,
			2600, 1574, 1487, 2640, 1617, 1919, 2650, 1617, 0, 3790, 1617, 0,
		}},
	},
	225: { // B5
		{0, []int{
			10, 982, 0, 30, 982, 0, 50, 981, 277, 120, 965, 2565, 780, 320, 3100, 870, 260, 1375,
			1030, 258, 33, 1040, 258, 0, 1660, 258, 0,
		}},
		{1, []int{
			10, 3508, 0, 30, 3508, 0, 40, 3508, 56, 50, 3502, 212, 130, 3511, 1656, 1100, 3514, 1599,
			1620, 3517, 132, 1630, 3517, 0, 1660, 3517, 0,
		}},
	},
	226: { // B6
		{0, []int{
			10, 3503, 0, 20, 3503, 0, 30, 3503, 141, 90, 3507, 2297, 270, 3452, 1305, 280, 3493, 0,
			480, 3493, 0, 490, 3493, 1154, 610, 3513, 2647, 680, 3491, 949, 690, 3469, 0, 820, 3469, 0,
			830, 3469, 1274, 890, 3509, 2586, 940, 3463, 1313, 950, 3463, 0, 1710, 3463, 0,
		}},
		{1, []int{
			10, 1821, 0, 230, 1821, 0, 240, 1821, 101, 260, 1821, 447, 310, 1805, 2904, 460, 1804, 3086,
			510, 1803, 1026, 520, 1803, 0, 610, 1803, 0, 620, 1803, 47, 630, 1802, 162, 640, 1802, 361,
			700, 1803, 3096, 800, 1809, 2578, 850, 1823, 298, 870, 1822, 0, 900, 1822, 0, 910, 1822, 117,
			930, 1822, 496, 970, 1808, 2565, 1120, 1804, 3100, 1620, 1805, 203, 1670, 1805, 33, 1710, 1805, 0,
		}},
	},
	227: { // B7
		{0, []int{
			10, 2908, 0, 20, 2908, 0, 30, 2908, 174, 60, 2910, 1164, 100, 2914, 2900, 410, 2912, 3100,
			440, 2910, 2556, 490, 2907, 434, 510, 2907, 70, 520, 3884, 0, 610, 3884, 0, 620, 3884, 115,
			630, 3903, 460, 660, 3897, 1353, 730, 3918, 2813, 1060, 3916, 2742, 1350, 3918, 1385, 1590, 3917, 202,
			1600, 3917, 0, 1630, 3917, 0,
		}},
	},
	228: { // B8
		{0, []int{
			10, 2085, 0, 20, 2085, 0, 30, 2085, 84, 40, 2086, 240, 110, 2082, 2804, 660, 1752, 2853,
			1250, 1501, 3015, 2010, 1330, 3100, 2740, 1305, 2417, 3150, 1305, 124, 3170, 1305, 49, 3220, 1306, 0,
		}},
		{1, []int{
			10, 3618, 0, 350, 3618, 0, 360, 3618, 455, 410, 3621, 1401, 580, 3619, 2237, 1240, 3005, 2894,
			2300, 2611, 2451, 2690, 2609, 2454, 2840, 2611, 1771, 3150, 2612, 97, 3170, 2613, 82, 3180, 2613, 0,
			3220, 2613, 0,
		}},
	},
	235: { // C5
		{0, []int{
			10, 716, 0, 85, 716, 0, 95, 716, 128, 150, 716, 268, 508, 716, 266, 658, 716, 948,
			939, 597, 2179, 1302, 586, 2165, 1597, 472, 2136, 1627, 421, 2188, 1647, 453, 1985, 1672, 441, 1909,
			1682, 339, 0, 1767, 339, 0, 1777, 339, 2131, 1832, 332, 3100, 1882, 324, 1153, 1888, 406, 0,
			1893, 406, 0, 1899, 406, 1757, 1909, 357, 1011, 1929, 318, 1300, 2009, 302, 2532, 2049, 323, 2383,
			2109, 289, 2322, 2259, 299, 2174, 2329, 305, 852, 2429, 305, 457, 2489, 305, 687, 2549, 305, 471,
			2609, 305, 680, 2659, 305, 467, 2719, 305, 679, 2779, 305, 466, 2829, 305, 679, 2889, 305, 473,
			2949, 305, 688, 3009, 305, 473, 3069, 305, 677, 3119, 305, 463, 3179, 305, 670, 3239, 305, 399,
			3299, 305, 405, 3379, 305, 68, 3389, 316, 123, 3399, 316, 0, 3429, 316, 0,
		}},
		{1, []int{
			10, 2705, 0, 95, 2705, 0, 100, 2705, 805, 110, 2708, 389, 180, 2710, 1278, 540, 2668, 1062,
			550, 2708, 1595, 560, 901, 0, 807, 901, 0, 812, 901, 1306, 912, 903, 2083, 1052, 902, 1799,
			1288, 898, 307, 1302, 898, 268, 1312, 899, 361, 1322, 281, 0, 1652, 281, 0, 1662, 281, 1755,
			1692, 361, 2622, 1712, 360, 2545, 1732, 417, 1680, 1752, 424, 1500, 1767, 406, 1628, 1777, 406, 0,
			3429, 406, 0,
		}},
		{2, []int{
			10, 302, 0, 862, 302, 0, 872, 302, 254, 939, 301, 493, 1112, 299, 1822, 1288, 300, 2004,
			1482, 300, 2228, 1492, 300, 0, 3429, 300, 0,
		}},
	},
	240: { // D0
		{0, []int{
			10, 965, 0, 30, 965, 0, 40, 965, 318, 50, 968, 764, 70, 963, 1775, 110, 950, 3100,
			1010, 950, 3065, 1190, 950, 1788, 1420, 949, 899, 1480, 949, 535, 1540, 949, 236, 1570, 949, 98,
			1590, 950, 79, 1600, 950, 0, 1640, 950, 0,
		}},
		{1, []int{
			10, 1334, 0, 70, 1334, 0, 80, 1334, 1754, 140, 1346, 2730, 980, 1341, 2711, 1190, 1342, 1853,
			1420, 1342, 641, 1480, 1342, 486, 1510, 1342, 302, 1550, 1342, 136, 1590, 1342, 26, 1640, 1344, 0,
		}},
	},
	241: { // D1
		{0, []int{
			10, 690, 0, 40, 690, 0, 50, 690, 377, 70, 697, 878, 90, 705, 2040, 130, 707, 3100,
			1030, 701, 3046, 1160, 701, 2016, 1360, 701, 1069, 1440, 701, 829, 1550, 701, 209, 1610, 700, 0,
			1640, 700, 0,
		}},
		{1, []int{
			10, 1231, 0, 70, 1231, 0, 80, 1231, 1247, 90, 1213, 1727, 130, 1214, 2589, 1030, 1215, 2047,
			1160, 1215, 1978, 1530, 1215, 250, 1600, 1215, 0, 1640, 1215, 0,
		}},
	},
	242: { // D2
		{0, []int{
			10, 714, 0, 40, 714, 0, 50, 714, 328, 60, 693, 411, 130, 704, 3098, 1040, 701, 3100,
			1590, 701, 57, 1620, 701, 0, 1660, 701, 0,
		}},
		{1, []int{
			10, 1350, 0, 60, 1350, 0, 90, 1343, 1405, 140, 1347, 2533, 1110, 1341, 2163, 1500, 1341, 426,
			1590, 1341, 48, 1660, 1342, 0,
		}},
	},
	243: { // D3
		{0, []int{
			10, 699, 0, 40, 699, 0, 60, 699, 424, 70, 699, 783, 80, 699, 1447, 130, 706, 3100,
			1020, 701, 3055, 1550, 701, 203, 1610, 701, 0, 1640, 701, 0,
		}},
		{1, []int{
			10, 1489, 0, 60, 1489, 0, 70, 1489, 1019, 130, 1479, 2366, 1110, 1480, 2004, 1540, 1480, 185,
			1600, 1480, 0, 1640, 1480, 0,
		}},
	},
	244: { // D4
		{0, []int{
			10, 794, 0, 40, 794, 0, 80, 780, 1568, 130, 773, 3098, 1000, 773, 3100, 1590, 773, 48,
			1650, 773, 0,
		}},
		{1, []int{
			10, 1202, 0, 80, 1202, 0, 90, 1202, 1839, 1150, 1215, 2036, 1460, 1215, 603, 1590, 1215, 0,
			1650, 1215, 0,
		}},
	},
	245: { // D5
		{0, []int{
			10, 765, 0, 40, 765, 0, 50, 765, 221, 70, 765, 772, 80, 765, 1441, 130, 772, 3100,
			1030, 773, 3052, 1590, 772, 54, 1610, 772, 0, 1640, 772, 0,
		}},
		{1, []int{
			10, 1354, 0, 70, 1354, 0, 80, 1354, 1460, 140, 1346, 2540, 1150, 1341, 1980, 1280, 1341, 1177,
			1540, 1341, 233, 1600, 1342, 0, 1640, 1342, 0,
		}},
	},
	246: { // D6
		{0, []int{
			10, 780, 0, 40, 780, 0, 50, 780, 449, 60, 758, 503, 70, 758, 782, 90, 757, 1893,
			130, 771, 3100, 1040, 773, 3028, 1320, 773, 1210, 1450, 773, 791, 1550, 773, 227, 1610, 773, 0,
			1640, 773, 0,
		}},
		{1, []int{
			10, 1476, 0, 60, 1476, 0, 70, 1476, 424, 80, 1470, 1089, 130, 1479, 2440, 1050, 1480, 2332,
			1540, 1481, 214, 1610, 1480, 0, 1640, 1480, 0,
		}},
	},
	247: { // D7
		{0, []int{
			10, 895, 0, 30, 895, 0, 40, 895, 577, 50, 862, 445, 130, 861, 3100, 1080, 856, 2759,
			1570, 855, 73, 1600, 856, 0, 1630, 856, 0,
		}},
		{1, []int{
			10, 1220, 0, 70, 1220, 0, 90, 1212, 1894, 130, 1214, 2718, 1050, 1214, 2570, 1580, 1215, 0,
			1630, 1215, 0,
		}},
	},
	248: { // D8
		{0, []int{
			10, 878, 0, 30, 878, 0, 50, 853, 307, 130, 860, 3100, 1030, 856, 3044, 1590, 854, 47,
			1640, 854, 0,
		}},
		{1, []int{
			10, 1360, 0, 70, 1360, 0, 80, 1360, 853, 130, 1346, 2607, 1070, 1342, 2380, 1590, 1342, 41,
			1600, 1342, 0, 1640, 1342, 0,
		}},
	},
	249: { // D9
		{0, []int{
			10, 865, 0, 40, 865, 0, 90, 850, 2007, 1050, 856, 2991, 1600, 856, 49, 1650, 856, 0,
		}},
		{1, []int{
			10, 1486, 0, 60, 1486, 0, 70, 1486, 737, 130, 1480, 2501, 1070, 1480, 2337, 1610, 1483, 0,
			1650, 1483, 0,
		}},
	},
	250: { // D10
		{0, []int{
			10, 936, 0, 40, 936, 0, 50, 936, 507, 130, 950, 3100, 1010, 949, 3065, 1570, 948, 93,
			1580, 949, 111, 1590, 949, 0, 1620, 949, 0,
		}},
		{1, []int{
			10, 1211, 0, 100, 1211, 0, 110, 1211, 2174, 140, 1215, 2827, 1010, 1215, 2786, 1500, 1216, 353,
			1520, 1216, 301, 1530, 1217, 375, 1540, 1217, 0, 1620, 1217, 0,
		}},
	},
	251: { // D11
		{0, []int{
			10, 953, 0, 40, 953, 0, 50, 953, 848, 130, 949, 3100, 1050, 950, 2899, 1560, 949, 124,
			1600, 949, 0, 1640, 949, 0,
		}},
		{1, []int{
			10, 1470, 0, 70, 1470, 0, 80, 1470, 1530, 990, 1479, 2579, 1590, 1478, 29, 1640, 1478, 0,
		}},
	},
	252: { // M0
		{0, []int{
			10, 1732, 0, 50, 1732, 0, 60, 1732, 3100, 70, 1784, 1971, 100, 1810, 3100, 2570, 1782, 3100,
			2680, 1844, 3100, 3360, 1782, 3100, 8590, 1783, 428, 9720, 1783, 357, 9850, 1783, 244, 10920, 1783, 253,
			11040, 1783, 133, 11880, 1783, 130, 11950, 1783, 41, 11960, 1794, 70, 11970, 1794, 0, 12000, 1794, 0,
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
		p[i].ms = float64(float32(float64(tm-prev) / 10))
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
			p[i].freq[t.osc] = float64(float32(f))
			p[i].vol[t.osc] = float64(float32(v / 100))
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
