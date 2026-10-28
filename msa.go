package speakjet

import "math"

// control defaults and ranges, from the manual
const (
	defaultVolume = 96  // 0 to 127
	defaultSpeed  = 114 // 0 to 127
	defaultPitch  = 88  // 0 to 255, in Hz
	defaultBend   = 5   // 0 to 15
)

// pause lengths in milliseconds for codes 0 to 6, measured as the silence the
// chip leaves between two AWs. pause 0 is not nothing
var pauseMs = [...]float64{14, 107, 203, 645, 25, 43, 70}

// Delay is longer than its 10ms intervals: on the chip (the VoxProbe recording)
// delays of 30, 60 and 90ms leave 44, 75 and 116ms between two AWs, against
// 26ms for pause 4. it doesn't change with Speed (500ms is 500 to 530ms at
// speeds 40, 114 and 127)
const delayExtraMs = 12.0

// how much Fast and Slow change a sound's length
const (
	fastScale = 0.65
	slowScale = 1.35
)

// how quickly the volume ramps down at the start of a pause, and at auto
// silence when the input buffer has emptied. not given by the manual
const (
	rampMs        = 10
	autoSilenceMs = 15
)

// segment is a part with its timing converted to samples
type segment struct {
	samples int
	glide   int
	target  frame

	// the frequencies are set at the start of the segment rather than glided
	jump bool
}

func samples(ms float64) int {
	return int(math.Round(ms * chipRate / 1000))
}

// msa sequences commands into segments and plays them on the synthesizer
type msa struct {
	synth *synth

	volume int
	speed  int
	pitch  int
	bend   int

	// modifiers for the next sound
	fast, slow    bool
	stress, relax bool
	repeat        int
	jumpNextFreq  bool
	autoSilenced  bool

	// the length of the pauses since the last sound, in ms (see closure())
	pausedMs float64

	segs     []segment
	segPos   int
	segStart frame
	current  frame
}

func newMSA() *msa {
	m := &msa{synth: newSynth()}
	m.resetControls()

	// start silent but at a neutral vowel and the default pitch, so that the
	// first sound fades in rather than sweeping up from 0Hz
	m.current = silentFrame(fAX)
	m.current.pitch = defaultPitch

	return m
}

func (m *msa) resetControls() {
	m.volume = defaultVolume
	m.speed = defaultSpeed
	m.pitch = defaultPitch
	m.bend = defaultBend
}

// Bend adds to the frequencies of oscillators 1 to 3 (the formants and tones),
// not multiplies them: 10 of the chip's frequency steps (freqStepHz) a Bend, 5
// from 4 to 5. the hiss on oscillators 4 and 5 doesn't move
func (m *msa) bendHz() float64 {
	if m.bend >= defaultBend {
		return 10 * freqStepHz * float64(m.bend-defaultBend)
	}
	return freqStepHz * float64(10*m.bend-45)
}

// a tone Bend took under nothing (see expand())
const (
	wrapHz     = 5680.0
	wrapMinVol = 8.0
)

// foldHz is a frequency played at the chip's rate, folded under half of it
func foldHz(hz float64) float64 {
	hz = math.Mod(hz, chipRate)
	if hz < 0 {
		hz += chipRate
	}
	if hz > chipRate/2 {
		return chipRate - hz
	}
	return hz
}

// the voicing frequency for a Pitch setting: the chip counts each voicing
// period in whole ticks of a voiceTickHz clock, 8000 / Pitch ticks (whole
// numbers) and one more, and at 12 settings one more again. not emulated: at
// very low pitch the chip starts each sound with a pulse; here pitch 0 is
// silent
const voiceTickHz = 7986.0

func voiceHz(pitch float64) float64 {
	p := int(pitch)
	if p <= 0 {
		return 0
	}
	ticks := 8000/p + 1
	switch p {
	case 44, 51, 53, 56, 63, 77, 87, 88, 92, 104, 127, 151:
		ticks++
	}
	return voiceTickHz / float64(ticks)
}

// the duration multiplier from the Speed control: lengths fall in a straight
// line with speed, reaching zero at speedZero, so against the default it is
// (146 - speed) / 32 (4.6 at speed 0, 0.59 at 127). pauses are lengthened by it
// too (see execute())
const speedZero = 146.0

func (m *msa) speedFactor() float64 {
	return (speedZero - float64(m.speed)) / (speedZero - defaultSpeed)
}

// first frame of a sound after the controls in effect now. used to glide the
// formants towards the next sound during pauses 1 to 3
func (m *msa) firstTarget(code uint8) (frame, bool) {
	s, ok := lookup(code)
	if !ok || len(s.parts) == 0 {
		return frame{}, false
	}
	return s.parts[0].target, true
}

func lookup(code uint8) (sound, bool) {
	switch {
	case code >= 128 && code < 128+uint8(len(allophones)):
		s := allophones[code-128]
		s = soundShaping(s, int(code))
		s.copyLength = copyLength[code-128]
		s.noSlow = code >= 182 // the voiceless consonants (see expand())
		return s, true
	case code >= 200 && int(code) < 200+len(effects):
		s := soundShaping(sharpOnset(effects[code-200], int(code)), int(code))
		s.copyLength = 1
		s.noBend = code == 254 // M2 (see expand())
		return s, true
	}
	return sound{}, false
}

// the sounds shaped further than the oscillators can: each is played through
// peaking filters half an octave apart (see synth.eq()), the voiceless ones
// whose hiss is broader on the chip with broadband noise as well, and an
// overall gain that keeps each at its level. the gains glide with the frames
// from one sound into the next. the code, the filters' gains in dB, the
// broadband noise and the overall gain
type consonantShape struct {
	db           [eqBands]float64
	wide, gainDB float64
}

var soundEQ = map[int]consonantShape{
	128: {[eqBands]float64{0.59, 0.92, 0.8, 0.56, -0.11, -4.27, 0.73, 1.78, 0.69, -0.66}, 0.00, 0.00},         // IY
	129: {[eqBands]float64{0.81, 0.3, 0.53, 2.09, 0.61, -2.84, 1.16, 1.54, -0.81, -0.67}, 0.00, 0.00},         // IH
	137: {[eqBands]float64{0.04, 0.79, 0.38, -0.44, 0.11, 0.29, -0.74, -4.16, 2.52, -1.08}, 0.00, 0.00},       // OW
	138: {[eqBands]float64{-0.34, -0.23, -0.25, -0.4, -0.56, -0.33, -0.5, -0.63, 0.31, 0.34}, 0.00, 0.00},     // UH
	141: {[eqBands]float64{0.52, 0.71, -1.77, -3.12, 5.48, 3.0, -2.16, 0.66, -0.4, -0.04}, 0.00, 0.00},        // NE
	142: {[eqBands]float64{0.21, 0.79, 0.9, 0.91, 4.69, -0.18, -3.12, 4.81, -1.82, -0.32}, 0.00, 0.00},        // NO
	143: {[eqBands]float64{1.5, 1.7, 0.15, -1.93, -1.67, 3.09, 0.07, -1.45, -0.89, 0.18}, 0.00, 0.00},         // NGE
	144: {[eqBands]float64{1.45, 1.99, -0.97, -2.64, 3.27, -0.0, -1.75, 0.49, -1.86, -1.18}, 0.00, 0.00},      // NGO
	145: {[eqBands]float64{-0.59, -0.05, -0.28, -0.76, -0.61, 0.04, -0.92, -3.34, -0.26, 1.11}, 0.00, 0.00},   // LE
	146: {[eqBands]float64{1.83, 1.91, 1.51, 0.27, 0.22, 2.77, 0.36, -4.49, 3.8, -2.56}, 0.00, 0.00},          // LO
	147: {[eqBands]float64{0.14, 0.38, 0.07, -0.12, 0.13, -0.45, -1.43, -1.13, 0.17, 1.21}, 0.00, 0.00},       // WW
	148: {[eqBands]float64{0.94, 1.07, 0.61, -0.13, 0.39, 1.28, 0.59, 0.32, -1.79, -0.49}, 0.00, 0.00},        // RR
	149: {[eqBands]float64{-0.45, 0.73, 1.15, 0.42, -0.13, -0.32, -0.37, 2.8, -0.82, -0.32}, 0.00, 0.00},      // IYRR
	150: {[eqBands]float64{0.59, 2.02, 5.4, 0.93, -8.79, 1.4, 4.8, 0.5, -2.03, -0.57}, 0.00, 0.00},            // EYRR
	152: {[eqBands]float64{1.23, 1.52, 1.51, 0.73, -0.09, 0.22, 1.24, 2.67, -2.02, -1.5}, 0.00, 0.00},         // AWRR
	153: {[eqBands]float64{1.42, 1.71, 1.45, 0.41, 0.01, 0.32, -0.48, 7.66, -5.81, -0.96}, 0.00, 0.00},        // OWRR
	155: {[eqBands]float64{-1.09, -0.11, 2.06, 2.7, -1.59, -3.47, 2.26, 2.17, -0.05, -0.28}, 0.00, 0.00},      // OHIY
	157: {[eqBands]float64{0.5, 0.01, 1.2, 2.19, -1.63, -2.45, 2.01, 0.61, -0.21, -0.57}, 0.00, 0.00},         // OHIH
	160: {[eqBands]float64{-0.13, 0.39, 1.03, 1.7, 0.03, -1.4, -0.15, 2.37, -0.35, -0.58}, 0.00, 0.00},        // IYUW
	162: {[eqBands]float64{0.73, 0.35, 0.3, 1.06, 1.27, 0.19, 0.06, 1.95, -1.32, -0.88}, 0.00, 0.00},          // IHWW
	163: {[eqBands]float64{1.06, 1.35, 1.5, 0.64, -0.0, 0.8, 0.65, 0.79, -2.09, -0.57}, 0.00, 0.00},           // AYWW
	164: {[eqBands]float64{2.24, 2.59, 2.55, 0.43, 0.44, 2.74, -0.61, 2.59, -4.1, -1.81}, 0.00, 0.00},         // OWWW
	165: {[eqBands]float64{-6.4, -0.3, 6.2, -5.1, 3.7, -1.6, -1.5, -0.1, 1.3, 3.8}, 0.00, 1.19},               // JH
	166: {[eqBands]float64{5.09, 0.01, -1.01, 2.03, -2.75, -1.79, 0.32, 0.11, 1.7, -2.45}, 0.00, 0.23},        // VV
	167: {[eqBands]float64{1.9, -0.3, -0.6, 2.1, 0.4, -3.7, 0.4, -1.8, -1.1, 2.8}, 0.00, -0.45},               // ZZ
	168: {[eqBands]float64{0.45, 0.35, -1.52, -1.06, 3.8, -2.04, -2.27, 4.69, 0.18, -1.18}, 0.00, -0.14},      // ZH
	169: {[eqBands]float64{2.7, -0.9, -0.9, -1.2, 8.4, -5.4, -1.1, 1.4, -2.3, -0.5}, 0.00, -0.45},             // DH
	170: {[eqBands]float64{-8.2, 3.3, 3.2, 2.2, -4.5, -3.2, 5.5, 0.1, 2.1, -0.5}, 0.00, 0.56},                 // BE
	171: {[eqBands]float64{2.0, 2.4, 0.3, -0.8, -1.7, 3.6, -0.3, -0.1, -2.2, -3.3}, 0.00, -0.97},              // BO
	173: {[eqBands]float64{-0.5, 4.5, -3.5, 3.5, 3.9, -11.5, 15.0, -7.0, 3.1, -7.8}, 0.00, -2.81},             // OB
	174: {[eqBands]float64{-5.91, -0.32, 8.46, -0.96, -4.0, 0.66, -0.66, -3.21, 0.68, 4.1}, 0.00, 1.38},       // DE
	175: {[eqBands]float64{1.4, 0.6, -1.0, -1.1, 4.0, 0.8, -0.9, 1.0, -1.5, -3.4}, 0.00, -0.86},               // DO
	176: {[eqBands]float64{-4.94, 0.68, 1.7, -4.78, 14.76, -3.74, -0.04, 3.1, -4.58, -1.06}, 0.00, 1.11},      // ED
	177: {[eqBands]float64{-1.5, 0.8, -2.8, -1.4, 10.0, -0.8, -1.7, 1.2, -3.2, -0.5}, 0.00, 0.78},             // OD
	178: {[eqBands]float64{-1.02, 0.92, -1.23, -3.58, 6.81, 7.37, -14.04, 5.31, -3.39, 2.85}, 0.00, 0.67},     // GE
	179: {[eqBands]float64{-1.2, 1.0, -1.1, -2.1, 3.5, 14.4, -15.0, 7.0, -10.8, 4.4}, 0.00, 0.12},             // GO
	180: {[eqBands]float64{-0.42, 0.97, -0.53, -1.12, 10.61, -4.92, -1.04, 3.87, -6.69, -1.2}, 0.00, -1.34},   // EG
	181: {[eqBands]float64{1.2, 1.0, -0.5, 1.9, 4.9, -3.7, 0.3, 1.3, -7.0, 0.5}, 0.00, -1.73},                 // OG
	184: {[eqBands]float64{1.26, 0.91, -0.86, -1.22, -0.77, -0.39, -1.21, -0.71, -1.05, 1.16}, 0.00, 0.00},    // HO
	185: {[eqBands]float64{-2.25, -0.18, -2.37, -1.16, 0.61, -2.23, -5.49, -1.02, 1.7, 2.2}, 0.00, 0.00},      // WH
	187: {[eqBands]float64{7.36, 2.8, -7.71, -6.11, 2.58, 0.46, -1.0, -3.48, -0.41, 2.45}, 0.00, 0.00},        // SE
	188: {[eqBands]float64{6.58, 3.83, -7.1, -6.78, 4.02, 1.38, -4.01, -1.59, -0.85, -1.86}, 0.00, 0.00},      // SO
	189: {[eqBands]float64{5.95, -0.24, -7.18, -2.18, 3.08, -2.64, -1.37, 1.32, 1.53, -2.11}, 0.00, 0.00},     // SH
	190: {[eqBands]float64{-2.02, 2.83, 1.28, -0.81, -0.62, -0.05, 2.49, -3.29, 1.66, 1.15}, 0.00, 0.00},      // TH
	191: {[eqBands]float64{3.1, 1.19, -1.25, -1.05, -0.89, -1.47, -2.59, -1.85, -1.0, 0.8}, 0.00, 0.00},       // TT
	192: {[eqBands]float64{-2.71, 0.29, -3.38, -0.61, -1.42, -3.45, -6.33, -0.11, 4.07, 10.45}, 0.60, -6.18},  // TU
	193: {[eqBands]float64{0.63, -0.01, 0.11, 0.32, 0.19, 0.04, -1.1, -2.48, -0.11, -0.53}, 0.00, 0.00},       // TS
	194: {[eqBands]float64{-0.16, -1.78, 0.21, 2.53, -1.67, 0.27, 2.53, -0.37, 3.75, -2.81}, 0.00, 0.00},      // KE
	195: {[eqBands]float64{2.1, 2.8, -6.9, -3.0, 7.0, -2.7, 1.1, 2.7, 3.6, -6.7}, 0.30, -2.29},                // KO
	196: {[eqBands]float64{0.27, 1.59, 2.0, -4.02, -2.63, -0.96, 6.46, -2.63, 5.84, -5.57}, 0.30, -1.01},      // EK
	197: {[eqBands]float64{3.04, 0.01, -0.06, 0.63, -3.1, 0.44, 1.48, 1.91, 0.98, -2.91}, 0.00, 0.00},         // OK
	198: {[eqBands]float64{7.82, 2.52, -2.54, -1.67, 1.63, -1.47, 0.05, -2.11, -6.27, -0.34}, 0.00, 0.00},     // PE
	199: {[eqBands]float64{6.42, 0.76, -2.16, -2.84, -0.61, -0.25, -3.13, -2.09, -1.04, 0.89}, 0.00, 0.00},    // PO
	200: {[eqBands]float64{3.74, 1.61, -0.69, -1.09, -0.23, -1.19, -0.44, -2.49, -0.93, -0.09}, 0.00, 0.00},   // R0
	201: {[eqBands]float64{0.07, 0.7, 2.55, 1.9, -1.87, -2.68, -1.61, -0.32, -1.36, 2.31}, 0.00, 0.00},        // R1
	202: {[eqBands]float64{4.66, 7.04, -3.66, -5.05, -0.88, -2.55, -1.45, -2.56, 2.41, 0.24}, 0.00, 0.00},     // R2
	204: {[eqBands]float64{4.92, -1.88, -4.48, -3.07, -3.88, -0.18, -2.6, -3.04, 2.21, -0.09}, 0.00, 0.00},    // R4
	205: {[eqBands]float64{13.1, 2.71, -5.99, 0.14, -0.25, -0.73, 3.51, 1.5, -8.89, -2.01}, 0.00, 0.00},       // R5
	206: {[eqBands]float64{5.65, -2.19, -6.24, -3.53, -4.5, -1.78, 2.62, -15.0, 0.88, 6.59}, 0.00, 0.00},      // R6
	207: {[eqBands]float64{5.92, -1.73, -4.59, -2.57, -1.37, -3.12, -2.36, 1.65, -5.11, 0.16}, 0.00, 0.00},    // R7
	208: {[eqBands]float64{3.7, -2.13, -2.51, 2.0, -0.93, -3.17, 0.58, -1.3, -1.96, 0.51}, 0.00, 0.00},        // R8
	209: {[eqBands]float64{3.52, -0.19, -1.61, 0.37, -0.5, 0.2, 1.53, -3.03, -0.81, -1.44}, 0.00, 0.00},       // R9
	210: {[eqBands]float64{12.11, -2.22, -7.96, -3.48, -2.47, -3.68, -2.74, -10.51, 9.28, -3.84}, 0.00, 0.00}, // A0
	211: {[eqBands]float64{1.09, 0.93, 0.82, 0.97, 1.04, 0.45, 0.42, 0.81, -0.31, -1.21}, 0.00, 0.00},         // A1
	212: {[eqBands]float64{3.04, 1.32, -0.5, -0.7, -0.38, -0.42, -0.82, -1.3, 1.62, -1.14}, 0.00, 0.00},       // A2
	213: {[eqBands]float64{1.32, 1.67, 1.04, 0.27, 1.07, 1.18, 1.89, -0.25, -1.63, -0.92}, 0.00, 0.00},        // A3
	214: {[eqBands]float64{8.15, -1.43, -3.03, -0.77, -0.58, -1.6, -0.15, -1.96, -0.2, -0.92}, 0.00, 0.00},    // A4
	215: {[eqBands]float64{1.63, 0.78, 0.09, -0.06, -0.69, -1.06, -1.94, -0.89, 0.42, -0.12}, 0.00, 0.00},     // A5
	218: {[eqBands]float64{10.76, 0.42, -5.11, -2.3, -1.69, -2.8, -1.19, -3.63, -4.09, -0.41}, 0.00, 0.00},    // A8
	219: {[eqBands]float64{15.0, 1.3, -4.51, -1.64, -0.22, -0.69, -0.38, 0.58, -4.75, -5.65}, 0.00, 0.00},     // A9
	222: {[eqBands]float64{15.0, -0.56, -13.01, -8.09, -2.82, 2.34, -5.52, -7.31, -8.71, 0.68}, 0.00, 0.00},   // B2
	223: {[eqBands]float64{-0.4, -0.34, -0.44, -1.21, -0.72, 0.05, -1.05, 2.81, 0.93, -0.32}, 0.00, 0.00},     // B3
	224: {[eqBands]float64{0.43, 0.36, -1.78, -1.83, -1.41, -2.46, -3.21, 3.4, 0.25, 1.32}, 0.00, 0.00},       // B4
	225: {[eqBands]float64{15.0, -3.34, -9.05, -4.62, -5.61, -1.21, 3.55, -9.98, -5.94, -4.63}, 0.00, 0.00},   // B5
	226: {[eqBands]float64{0.59, 0.32, -1.45, -2.38, -2.22, -1.1, 1.12, 1.89, -3.64, 1.38}, 0.00, 0.00},       // B6
	228: {[eqBands]float64{0.16, 1.66, 0.77, -1.57, -1.43, -0.24, -3.08, -1.66, -3.54, -1.43}, 0.00, 0.00},    // B8
	229: {[eqBands]float64{8.37, 3.98, 4.05, 4.64, 1.85, 1.34, -3.02, -6.62, -7.25, -4.84}, 0.00, 0.00},       // B9
	230: {[eqBands]float64{1.3, 1.13, 1.19, 1.05, 0.17, 0.17, 0.43, 0.45, -0.4, -1.2}, 0.00, 0.00},            // C0
	231: {[eqBands]float64{-0.06, 0.8, 0.29, -0.66, -0.65, -0.69, -0.75, -1.03, 0.87, 0.69}, 0.00, 0.00},      // C1
	232: {[eqBands]float64{-1.92, -0.33, 0.08, -0.79, 0.16, 1.24, -2.94, -0.58, 0.77, 1.03}, 0.00, 0.00},      // C2
	233: {[eqBands]float64{-0.84, -2.06, 0.02, 1.51, -0.34, -0.26, 0.96, -2.55, 1.29, 0.49}, 0.00, 0.00},      // C3
	234: {[eqBands]float64{0.46, 1.66, 1.66, -0.05, 1.20, 5.89, -5.14, 0.12, 1.53, -1.19}, 0.00, 0.00},        // C4
	235: {[eqBands]float64{14.23, -0.52, -5.23, -0.72, -1.0, -3.16, 1.87, 5.76, -5.96, -5.58}, 0.00, 0.00},    // C5
	236: {[eqBands]float64{-0.09, -0.09, -0.13, -0.9, -1.22, -0.65, -0.38, 0.09, 0.68, 0.45}, 0.00, 0.00},     // C6
	237: {[eqBands]float64{4.9, 2.98, 2.32, 3.2, 1.91, 0.46, -1.07, -0.65, -1.23, -2.81}, 0.00, 0.00},         // C7
	238: {[eqBands]float64{-0.15, 0.58, 0.09, 2.65, 3.27, 1.16, -1.61, -0.58, 4.66, -0.71}, 0.00, 0.00},       // C8
	239: {[eqBands]float64{-1.68, -0.34, 0.61, 1.43, 1.84, 0.32, -1.07, -0.16, 6.3, -1.0}, 0.00, 0.00},        // C9
	241: {[eqBands]float64{1.15, 1.36, 4.22, 0.33, -9.03, -2.0, -3.76, -3.33, 4.01, 0.59}, 0.00, 0.00},        // D1
	242: {[eqBands]float64{-0.81, 2.07, 6.08, 0.4, -6.12, 9.4, -2.16, -3.26, -0.95, -0.09}, 0.00, 0.00},       // D2
	243: {[eqBands]float64{0.91, 2.6, 6.22, 1.53, -8.6, 5.37, -1.34, -3.53, -0.56, -0.06}, 0.00, 0.00},        // D3
	244: {[eqBands]float64{2.35, 5.68, 6.22, 4.34, -6.09, -4.88, -2.97, -1.15, -0.27, -0.03}, 0.00, 0.00},     // D4
	245: {[eqBands]float64{1.83, 4.72, 7.91, 6.8, -7.78, -1.88, -2.92, -3.77, -0.71, -0.07}, 0.00, 0.00},      // D5
	246: {[eqBands]float64{4.84, 5.63, 7.8, 5.49, -9.07, -8.74, -4.01, -2.37, -0.57, -0.06}, 0.00, 0.00},      // D6
	247: {[eqBands]float64{0.89, 1.74, 2.95, 6.65, -0.17, -10.42, 0.22, 2.41, -1.66, -0.14}, 0.00, 0.00},      // D7
	249: {[eqBands]float64{1.05, 2.31, 4.07, 5.54, -2.72, -5.38, -2.54, -1.91, 0.07, 0.0}, 0.00, 0.00},        // D9
	250: {[eqBands]float64{-0.83, -1.71, -1.96, 3.01, 3.96, -8.87, -3.89, 5.1, 1.45, 0.13}, 0.00, 0.00},       // D10
	251: {[eqBands]float64{-1.81, 1.94, 2.94, 2.69, 1.37, -4.02, -2.06, -1.33, 1.51, 0.13}, 0.00, 0.00},       // D11
	252: {[eqBands]float64{-0.22, 0.52, 1.21, -1.12, 0.12, -2.37, -0.8, 1.93, -2.42, 2.06}, 0.00, 0.00},       // M0
}

// the chip starts most tones at once, where the traces (5ms frames) ramp up. so
// these effects start sharply: the parts before the first whose summed volume
// reaches the given fraction of the sound's loudest are silent, and that part
// is reached at once, the sound's length unchanged
var sharpOnsets = map[int]float64{
	201: 0.5,   // R1
	202: 0.125, // R2
	203: 0.5,   // R3
	204: 0.2,   // R4
	205: 0.1,   // R5
	206: 0.3,   // R6
	207: 0.2,   // R7
	208: 0.2,   // R8
	211: 0.05,  // A1
	212: 0.5,   // A2
	213: 0.7,   // A3
	214: 0.7,   // A4
	216: 0.3,   // A6
	217: 0.2,   // A7
	218: 0.05,  // A8
	219: 0.2,   // A9
	220: 0.2,   // B0
	221: 0.7,   // B1
	222: 0.5,   // B2
	223: 0.7,   // B3
	224: 0.5,   // B4
	225: 0.7,   // B5
	226: 0.3,   // B6
	227: 0.5,   // B7
	228: 0.2,   // B8
	229: 0.3,   // B9
	235: 0.05,  // C5
	236: 0.025, // C6
	239: 0.5,   // C9
	240: 0.2,   // D0
	242: 0.7,   // D2
	243: 0.475, // D3
	244: 0.05,  // D4
	245: 0.025, // D5
	246: 0.05,  // D6
	247: 0.2,   // D7
	248: 0.5,   // D8
	249: 0.7,   // D9
	250: 0.1,   // D10
	251: 0.2,   // D11
	254: 0.7,   // M2
}

// sharpOnset starts an effect sharply (see sharpOnsets). the parts are copied,
// since the table's sound shares them
func sharpOnset(s sound, code int) sound {
	fraction, ok := sharpOnsets[code]
	if !ok {
		return s
	}
	s.parts = append([]part(nil), s.parts...)
	loudest := 0.0
	for _, p := range s.parts {
		v := 0.0
		for j := 0; j < 5; j++ {
			v += p.target.vol[j]
		}
		loudest = math.Max(loudest, v)
	}
	for i := range s.parts {
		v := 0.0
		for j := 0; j < 5; j++ {
			v += s.parts[i].target.vol[j]
		}
		s.parts[i].glideMs = 0
		if v >= fraction*loudest {
			break
		}
		for j := 0; j < 5; j++ {
			s.parts[i].target.vol[j] = 0
		}
	}
	return s
}

// soundShaping sets a sound's shaping on each of its parts (see soundEQ). the
// parts are copied, since the table's sound shares them
func soundShaping(s sound, code int) sound {
	c, ok := soundEQ[code]
	if !ok {
		return s
	}
	s.parts = append([]part(nil), s.parts...)
	for i := range s.parts {
		f := &s.parts[i].target
		f.eq = c.db
		f.wide = c.wide
		f.eqGain = c.gainDB
	}
	return s
}

// how far Stress raises F2 and F3 (see expand()): about 85Hz on the chip, which
// is 19 of its frequency steps
var stressHz = 19 * freqStepHz

// expand converts a sound into segments, applying the controls and modifiers
func (m *msa) expand(s sound) []segment {
	// Speed stretches every sound, the effects too: on the chip (the VoxProbe
	// recording) A1, B1 and D1 take 2.6 to 2.7 times as long at speed 60 and
	// 0.56 to 0.59 times at speed 127, as the speech does
	scale := m.speedFactor()
	stress := 0.0

	// the copies after the first, when the sound is repeated, take the
	// manual's length (scaled by Speed) rather than the chip's longer first
	// one: on the chip each extra AW adds 70ms, and repeating IY, UW and MM six
	// times takes one of them plus five times 70ms, to within 6ms
	copyScale := scale

	if s.speech {
		scale *= s.length

		// Fast and Slow (fastScale, slowScale). Slow doesn't lengthen the voiceless
		// consonants
		if m.fast {
			scale *= fastScale
		}
		if m.slow && !s.noSlow {
			scale *= slowScale
		}

		// "STRESS, IH sounds more like (but not quite) the IY sound". on the
		// chip (the VoxRecord recording's third take) stress raises F2 and F3
		// by 65 to 100Hz in AW, EYIY and NE and leaves the pitch and F1 alone.
		// relax changes nothing that can be measured, and so does nothing here
		if m.stress {
			stress = stressHz
		}
	}

	bendHz := m.bendHz()
	var wrapped [3]bool

	var segs, copies []segment
	for _, p := range s.parts {
		ms := p.ms * scale
		glide := p.glideMs * scale
		if p.glideMs == autoGlide {
			glide = math.Min(math.Max(ms*0.35, 20), 50)
		}
		glide = math.Min(glide, ms)

		t := p.target
		if s.speech {
			t.pitch = voiceHz(float64(m.pitch))
			t.freq[1] += stress
			t.freq[2] += stress
		} else if t.pulsed {
			// a pulsed effect's gate keeps its distance from the voicing (see
			// fxPulses)
			t.pitch = math.Max(voiceHz(float64(m.pitch))+t.pulseOffset, 1)
		} else if t.wave == envVoice && !s.ownPitch {
			// a voiced effect follows the Pitch control: on the chip A1 and M2
			// are voiced at 60.3, 87.9 and 149.5Hz at pitch 60, 88 and 150.
			// their traced pitch is the chip's at the default pitch
			t.pitch *= voiceHz(float64(m.pitch)) / voiceHz(defaultPitch)
		}
		// M2 doesn't follow Bend: on both units of the fourth VoxProbe recording
		// it is the same at bend 0 and bend 15, its formants within half a hertz
		// of each other over its whole length
		// C8's and C9's gate keeps time with Speed (see cNoise)
		if s.speedGate {
			t.pitch /= scale
		}
		// a tone that Bend takes under nothing wraps round: it plays wrapHz higher,
		// folded under half the chip's rate, and stays there. a tone quieter than
		// wrapMinVol (one its trace couldn't read) doesn't wrap
		if !s.noBend {
			for j := range 3 {
				if t.freq[j] > 0 {
					if !s.speech && t.vol[j] > wrapMinVol && t.freq[j]+bendHz < 0 {
						wrapped[j] = true
					}
					if wrapped[j] {
						t.freq[j] = foldHz(t.freq[j] + bendHz + wrapHz)
					} else {
						t.freq[j] = math.Max(t.freq[j]+bendHz, 0)
					}
				}
			}
		}
		// Bend moves an effect's noise as a block, its width unchanged (the Bend is
		// added after the spread: see synth.sample()). the hiss of speech doesn't
		// move
		if !s.speech {
			t.noiseShift = bendHz
		}
		segs = append(segs, segment{samples: samples(ms), glide: samples(glide), target: t})

		copyMs := p.ms * copyScale * s.copyLength
		copyGlide := p.glideMs * copyScale
		if p.glideMs == autoGlide {
			copyGlide = math.Min(math.Max(copyMs*0.35, 20), 50)
		}
		copies = append(copies, segment{samples: samples(copyMs), glide: samples(math.Min(copyGlide, copyMs)), target: t})
	}

	m.fast, m.slow, m.stress, m.relax = false, false, false, false

	if len(segs) > 0 && m.jumpNextFreq {
		segs[0].jump = true
		copies[0].jump = true
		m.jumpNextFreq = false
	}

	if m.repeat > 1 {
		for i := 1; i < m.repeat; i++ {
			segs = append(segs, copies...)
		}
	}
	m.repeat = 0

	return segs
}

// a pause before a stop takes the place of that much of its closure, which
// keeps minClosureMs. after a sound the closure is all there
const minClosureMs = 20.0

func (m *msa) closure() {
	if len(m.segs) < 2 {
		return
	}
	keep := samples(minClosureMs)
	c := &m.segs[0]
	if c.samples <= keep {
		return
	}
	c.samples = max(c.samples-samples(m.pausedMs), keep)
	c.glide = min(c.glide, c.samples)
}

// the sounds played from the chip's frames (see vfPoints) were measured alone:
// their first and last frames are the voicing coming up from silence and dying
// away. between two sounds the chip doesn't do that, so after a voiced sound
// the frames before the first at framedJoin of the sound's loudest take its
// volumes, and before another sound the frames after the last take that one's
const framedJoin = 0.7

func frameVolume(f frame) float64 { return f.vol[0] + f.vol[1] + f.vol[2] }

func (m *msa) join(s sound, more bool) sound {
	// the parts are shared with the table the sound came from
	s.parts = append([]part(nil), s.parts...)
	loudest := 0.0
	for _, p := range s.parts {
		loudest = math.Max(loudest, frameVolume(p.target))
	}
	if m.current.voiced && frameVolume(m.current) > 0 {
		j := 0
		for j < len(s.parts)-1 && frameVolume(s.parts[j].target) < framedJoin*loudest {
			j++
		}
		for i := 0; i < j; i++ {
			copy(s.parts[i].target.vol[:3], s.parts[j].target.vol[:3])
		}
	}
	if more {
		j := len(s.parts) - 1
		for j > 0 && frameVolume(s.parts[j].target) < framedJoin*loudest {
			j--
		}
		for i := j + 1; i < len(s.parts); i++ {
			copy(s.parts[i].target.vol[:3], s.parts[j].target.vol[:3])
		}
	}
	return s
}

// silence ramps the volume down and holds silence. if towards is not nil the
// formant frequencies glide to it while the volume ramps
func (m *msa) silence(ms float64, towards *frame) []segment {
	t := m.current
	t.vol = [5]float64{}
	if towards != nil {
		t.freq = towards.freq
		t.freq[3], t.freq[4] = m.current.freq[3], m.current.freq[4]
	}

	segs := []segment{{samples: samples(rampMs), glide: samples(rampMs), target: t}}
	if ms > rampMs {
		segs = append(segs, segment{samples: samples(ms - rampMs), target: t})
	}
	return segs
}

// execute a single command. peek returns the next command in the buffer, if
// there is one. the return value is false if the command is not recognised
func (m *msa) execute(code uint8, data uint8, peek func() (uint8, bool)) bool {
	m.autoSilenced = false

	switch {
	case code <= 6:
		// pauses 1 to 3 "ramp the volume while the formant frequencies are
		// being changed". pauses 4 to 6 "wait for silence before changing the
		// formant frequencies"
		var towards *frame
		if code >= 1 && code <= 3 {
			if next, ok := peek(); ok {
				if f, ok := m.firstTarget(next); ok {
					towards = &f
				}
			}
		} else if code >= 4 {
			m.jumpNextFreq = true
		}
		// on the chip a pause stretches with the Speed control just as the
		// sounds do: at speed 0 two pause 2s are 1845ms of silence
		m.segs = m.silence(pauseMs[code]*m.speedFactor(), towards)
		m.pausedMs += pauseMs[code] * m.speedFactor()

	case code == 7:
		m.fast = true
	case code == 8:
		m.slow = true
	case code == 14:
		m.stress = true
	case code == 15:
		m.relax = true

	case code == 20:
		m.volume = min(int(data), maxMaster)
	case code == 21:
		m.speed = min(int(data), 127)
	case code == 22:
		m.pitch = int(data)
	case code == 23:
		m.bend = min(int(data), 15)

	case code == 26:
		m.repeat = int(data)

	case code == 30:
		// "the number of 10ms intervals to delay before continuing"
		m.segs = m.silence(float64(data)*10+delayExtraMs, nil)

	case code == 31:
		m.resetControls()

	case code == 16, code == 18, code == 24, code == 25, code == 28, code == 29:
		// wait, port control, port output and the EEPROM phrase commands have
		// no effect on the sound here: there is no serial control protocol to
		// restart after a wait, the output pins are not connected, and the
		// contents of the AtariVox's SpeakJet EEPROM are not known
		//
		// code 18 isn't in the manual, which goes from Wait (16) to Volume
		// (20). Magnevation's PhraseALator software sends it as "Soft" ("Play
		// the Next Sound Soft") and its dictionary uses it, mostly before BE
		// and BO, so it is a real command. nothing describes what it does to
		// the sound, so it is accepted and ignored
		return false

	default:
		s, ok := lookup(code)
		if !ok {
			return false
		}
		if s.framed {
			next, ok := peek()
			s = m.join(s, ok && next >= 128 && next < 200)
		}
		// the last sound before a pause or the end (see finalLength)
		if code >= 128 && code < 128+uint8(len(allophones)) && m.repeat <= 1 {
			if next, ok := peek(); !ok || next <= 6 {
				s.length *= finalLength[code-128]
			}
		}
		m.segs = m.expand(s)
		if s.stop {
			m.closure()
		}
		m.pausedMs = 0
	}

	m.segPos = 0
	m.segStart = m.current
	if len(m.segs) > 0 && m.segs[0].jump {
		m.segStart.freq = m.segs[0].target.freq
	}
	m.noiseOnset()

	return true
}

// busy is true while there are segments still to play
func (m *msa) busy() bool {
	return len(m.segs) > 0
}

// noiseOnset makes noise that starts (oscillators 4 and 5 silent before,
// sounding in the segment ahead) start at its own frequencies rather than
// gliding up to them: on the chip SE and SO start as hiss, where gliding made
// them sweep up from the previous sound's frequencies
func (m *msa) noiseOnset() {
	if len(m.segs) == 0 {
		return
	}
	t := &m.segs[0].target
	if m.segStart.vol[3]+m.segStart.vol[4] == 0 && t.vol[3]+t.vol[4] > 0 {
		m.segStart.freq[3] = t.freq[3]
		m.segStart.freq[4] = t.freq[4]
	}
}

// autoSilence starts the ramp to silence once the input buffer is empty
func (m *msa) autoSilence() {
	if m.autoSilenced {
		return
	}
	m.autoSilenced = true
	m.segs = m.silence(autoSilenceMs, nil)
	m.segPos = 0
	m.segStart = m.current
}

// next produces one sample from the segments currently queued. it returns
// false once the segments have all been played
func (m *msa) next() (float64, bool) {
	if len(m.segs) == 0 {
		return m.synth.sample(m.current, float64(m.volume)), false
	}

	sg := &m.segs[0]
	if sg.glide > 0 && m.segPos < sg.glide {
		m.current = lerp(m.segStart, sg.target, float64(m.segPos+1)/float64(sg.glide))
	} else {
		m.current = sg.target
	}

	v := m.synth.sample(m.current, float64(m.volume))

	m.segPos++
	if m.segPos >= sg.samples {
		m.segs = m.segs[1:]
		m.segPos = 0
		m.segStart = m.current
		if len(m.segs) > 0 && m.segs[0].jump {
			m.segStart.freq = m.segs[0].target.freq
		}
		m.noiseOnset()
	}

	return v, len(m.segs) > 0
}
