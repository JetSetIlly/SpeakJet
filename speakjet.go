package speakjet

import "math"

// SampleRate is the rate at which samples are produced. the chip makes its own
// at about 8kHz and holds each until the next, which puts images of the sound
// up to 20kHz and more; the output is at a rate that carries them
const SampleRate = 48000

// the size of the input buffer, and the level above which the Buffer Half
// Full status line is set. from the manual
const (
	BufferSize     = 64
	bufferHalfFull = 32
)

// the rate the chip makes samples at, measured from the images of its tones on
// two units (8039 and 8013 Hz; the manual says 8192). the chip's own arithmetic
// assumes 8000: its tones come out rate/8000 sharp on both units
const (
	rate        = 8026.0
	nominal     = 8000.0
	nhpFC       = 400.0 // an allophone's noise high pass, Hz
	nsweepHP    = 150.0 // the swept noise's high pass (C8, C9), Hz
	fricRamp    = 3.0   // an unvoiced fricative's attack and release, ms
	thumpMs     = 16.0  // a sibilant's starting burst of low noise: it fades out over this, ms
	fxburstMs   = 5.0   // an effect's starting click: it fades out over this, ms (220 B0: the chip's is ~5 ms)
	thumpHz     = 800.0 // and is low-passed here
	nasalAttack = 25.0  // a nasal's attack, ms (the vowels': attackMs)
	barUp       = 0.12  // a voiced stop's bar: its upper formants at this fraction of the release's
	barClick    = 0.0   // and the click it opens on, ms
)

// the constants of the model, as the recordings gave them. the same values as
// the C library's struct tune
const (
	envRise       = 0.5    // voicing envelope: rise (under one tick: a step), ticks
	envFall       = 58.0   // and linear fall
	lpfFc         = 1609.0 // the output stage's two poles (both units: the 12 DTMF pairs' level ratios and VoxProbe 7's direct tones at 125, 1000 and 3000 Hz, all 14 within 0.45 dB)
	lpfQ          = 0.462
	lpfZero       = 14000.0 // and one real zero, Hz: the images above 8 kHz (every effect, both units)
	attackMs      = 35.0    // every sound's level ramp
	releaseMs     = 30.0
	fxAttackMs    = 1.0 // the effects' (both units: a tone starts within 1-3 ms, its splatter is the bed around it)
	fxReleaseMs   = 25.0
	floor         = 0.0                 // the voiced output noise floor, relative. 0: the chip's voice is exactly periodic (both units, every vowel and nasal at Pitch 88: the period holds to one 48 kHz sample, and the gaps between harmonics sit 30-60 dB under them, which a 0.05 floor filled to 15-30)
	jitter        = 0.0                 // the voiced oscillators' frequency wanders by this fraction each sample. 0, measured with the floor: it smeared the same gaps
	quant         = 0.0                 // the output\'s steps per full scale: the chip\'s output is coarse, about 5 bits across a vowel
	decayDB       = 35.0                // an fxDecay part falls this many dB
	transitionMs  = 25.0                // the voice moves to a new sound's target over this
	closureMs     = 30.0                // the closure before a stop's burst
	joinClosure   = 45.0                // a stop's closure after a sound, at least this, ms (Man Goes Down's GO)
	nasalBar      = 0.15848931924611134 // a voiced stop's bar after a nasal: 16 dB under its own level
	slowGlideOn   = 50.0                // ms: see the ASSUMPTION at its use
	joinA         = 66.9                // a vowel or glide into a vowel or glide: cut joinA - joinK * (F1+F2 ERB distance) ms
	joinK         = 6.9
	nasalFade     = 0.6513             // and the nasal fades into it over this share of the closure
	nasalClosure  = 38.0               // a voiced stop's closure after a nasal, ms (before Speed and Slow)
	nasalRamp     = 10.0               // and its release rises over this, ms
	nasalRampFrom = 0.316228           // from 10 dB down
	slew          = 0.078              // the effects' bed, measured: grows with the tone's frequency on both units
	bedPole       = 0.6909588478246612 // up to about 1.5 kHz, then with the tone's level alone (one pole)
	gateDepth     = 0.4                // an effect's gate: its off half's level, a factor
	overlapMs     = 15.0               // a sound followed at once by another is cut short by this
	overlapVMs    = 30.0               // more when both are voiced
	speedA        = 162.0              // length factor (speedA - Speed) / speedB
	speedB        = 48.0
	fxSpeedA      = 147.0 // the effects' (they shorten faster)
	fxSpeedB      = 33.0
	fastFactor    = 2.0 / 3.0 // Fast and Slow: factors on the played length
	slowFactor    = 4.0 / 3.0
	copyExtraMs   = 0.0  // a Repeat copy adds this to the manual's length
	copyClosure   = 20.0 // a stop's Repeat copy opens on this much closure, ms
	bendHz        = 44.0 // Bend step
)

// the pauses, ms (VoxProbe 1 section 5, AW Pn AW x3, both units, 2600 and Stella; Man Goes Down P1 and P2's gaps):
// P0 leaves ~20 ms, P3 P5 P6 run 52/30/31 ms under the manual's 700 60 90, P1 P2 P4 are the manual's (AW's own
// fade-in and -out leave every AW Pn AW gap ~19 ms under the chip's; the phrases' gaps between stops and hisses
// match these)
var pauseMs = [7]float64{20, 90, 200, 648, 15, 30, 59}

func ticks(ms float64) int { return int(ms*rate/1000.0 + 0.5) }

// one target the synthesizer moves toward: three voiced formants (Hz, level 0
// to 31) and one noise band (centre Hz, throw Hz, level 0 to 31)
type frame struct {
	f, a       [3]float32
	lv         float32 // the sound's level as a factor
	nf, nw, na float32
}

// a sound is a few segments; each holds or glides to a frame for some ticks
type segment struct {
	ticks    int
	from, to frame
	voiced   bool
	gate     int     // ticks of the on/off period for a pulsed effect, 0 = none
	flags    uint16  // fx*
	pa, pb   float32 // an effect part's pulse rate: Pitch plus pa Hz at its start, pb at its end
	lr       float32 // an effect part's level ramp: half-decibels gained (or lost) across the part, even in dB
}

// the sound effects: each is a few parts. a part plays up to three tones
// (plain sines; with fxVoiced the first two are the voice's formants and the
// third stays a plain tone), a noise band, for ms; the tones sweep from fa to
// fb (linearly, or fxExp geometrically, or fxExpDecay settling toward fb);
// fxDecay fades the level over the part, fxBump shapes it as a half sine,
// fxNSweep sweeps the noise band within each gate period; gate pulses it
const (
	fxVoiced   = 1
	fxExp      = 2
	fxDecay    = 4
	fxBump     = 8
	fxExpDecay = 16
	fxNSweep   = 32
	fxPulsed   = 64   // the plain tones pulse with the voice's envelope at Pitch
	fxFade     = 128  // the level falls straight to nothing over the part (the tones' measured ending)
	fxAll      = 256  // every tone and the noise of a voiced part pulse with the voice (C0, M2: silent between pulses on both units)
	fxWander   = 512  // the plain tones' pitch drifts (252 M0, both units: 1.45-1.59 % rms, spread to about 70 Hz)
	fxSine     = 1024 // the envelope oscillator's waveform (manual, register 8): saw (the voice's measured pulse),
	fxTri      = 2048 // sine, triangle or square
	fxSquare   = 3072
	fxOne      = 4096  // only the first tone pulses, the other two are plain tones beside it (A0, both units: a voice at Pitch under two smooth glides; register 8 envelopes oscillators 1-3 as a group, 4 and 5 run apart)
	fxSmooth   = 16384 // a tone switching on from silence or off to silence ramps over 3 ms (24 ticks) instead of stepping at a part boundary (202 R2, 218 A8: no click on the chip); per sound, 233 C3's tones switch sharply
	fxBuzz     = 8192  // the noise band driven by an impulse every period of the part's pulse rate instead of noise, at the same power (232 C2's banded burst)
	fxShape    = 3072
)

// the tables' units: frequencies in 16 Hz steps, lengths in 5 ms, so every
// field is a byte
const (
	hzStep = 16
	msStep = 5
)

type fxpart struct {
	ms       uint8    // msStep
	fa, fb   [3]uint8 // hzStep
	v        [3]uint8
	nf, nw   uint8 // hzStep
	nv, gate uint8 // gate in ms
	flags    uint16
	lv       int8     // the part's level, half-decibels (measured against the voice, both units)
	pa, pb   int16    // the part's pulse rate, Hz added to Pitch at its start and end (both units: the same offset at Pitch 60, 88 and 150)
	lr       int8     // the part's level ramp, half-decibels across it (both units: B1's tone falls and rises a steady 1.5 dB per 10 ms through its sweeps, the same up and down where the output stage alone can't)
	ve       [3]uint8 // each tone's level at the part's end, 0 to 31 (both units: A7's lowest tone fades in by 27 dB while its highest fades out by 30, inside one sweep; the tones' volumes are registers swept like their frequencies)
}

const maxSegments = 32

// the sound component database: per allophone, its kind, measured length and
// targets (tables.go, generated)
const (
	kVowel = iota
	kGlide
	kNasal
	kVFric
	kFric
	kStop
	kVStop
	kAffr
)

type allophone struct {
	kind   uint8
	ms     uint8    // the length played at Speed 114, measured, msStep
	f      [3]uint8 // formants, hzStep
	a      [3]uint8 // their levels, 0 to 31
	nf, nw uint8    // the noise band's centre and bandwidth, hzStep
	na     uint8
	cp     uint8 // the length of each further copy of a Repeat, msStep
	lv     int8  // the sound's level, half-decibels (measured against the vowels, both units)
	cb     uint8 // a stop's Repeat copies keep its voice bar through their closures
	ob     uint8 // its onset: the first ob ms play og half-decibels louder (0: none)
	og     uint8
	cl     uint8 // a stop's closure, ms (0: the common closureMs)
	rl     int8  // a stop's release after its first pulse, half-decibels re its bar (0: as the table)
	wf     uint8 // a white floor under its noise band, /256 of the band's level (0: none)
	rr     uint8 // its own release (fade-out) length, ms (0: the common one)
	tc     uint8 // its starting burst's low-pass corner, Hz/32 (0: thumpHz)
	th     uint8 // a low thump at its start: one cycle near 190 Hz, /256 of the noise band's level (0: none)
	ba     uint8 // a voiced stop's bar fade-in from silence, ms (0: fricRamp)
	bl     int8  // the bar's level re the release, half-decibels (0: the same)
	aa     uint8 // its own attack (fade-in) length, ms (0: the class's)
	hp     uint8 // its noise high pass corner, Hz/16 (0: nhpFC)
	nd     uint8 // a decaying release: the noise band's level at the end of its last segment, /256 (0: steady)
}

// SpeakJet is the chip. Write() is called by whatever is sending it data (the
// AtariVox serial line) and Render() by whatever is playing its output
type SpeakJet struct {
	// the input buffer: a FIFO of raw bytes, as the chip's is
	buf         [BufferSize]uint8
	head, count int
	executing   int // bytes of the command being executed, still counted

	// the controls
	volume, speed, pitch, bend, repeat uint8
	fast, slow, stress, relax          bool
	joined                             bool // the last sound ran straight into the next: no ramps at that join
	lastNasal                          bool // the last sound played was a nasal, MM NE NO NGE NGO (not the liquids that share kNasal)
	rampSeg                            int  // a voiced stop after a nasal: its release ramp's segment (-1: none)

	// what is playing
	seg             [maxSegments]segment
	nseg, cur, tick int
	stick, stotal   int  // ticks into the sound, and its whole length
	effect          bool // the sound is an effect (its Speed law)
	read            bool // the command after it has been read ahead
	ahead           int  // bytes of that command, executed early: retired with the sound
	attack, release int
	onset           int // the sound's louder onset: its length in ticks, where it starts (a stop's or an affricate's release, after the closure)
	onsetFrom       int
	releaseGain     float64 // a stop's release (from onsetFrom), relative
	wfloor          float64 // an allophone's white floor under its noise band
	thump           float64 // its starting burst of low noise, relative to the noise band
	thlp            float64 // that burst's low-pass state
	thk             float64 // and its low-pass coefficient
	nhk             float64 // an allophone's noise high pass coefficient
	fxburst         float64 // an effect's onset burst (fxs): level
	fxbms           float64 // its length, ms (fxs tl; fxburstMs when 0)
	fxhpk           float64 // the effect's own noise high pass (fxs hp), one stage; 0: none
	fxbk            float64 // its low-pass coefficient
	fxblp           float64 // and state
	onsetGain       float64 // and its gain
	copies          int     // copies of a Repeat still to play
	copyOf          int     // which allophone
	last            frame   // where the voice is, for transitions into the next sound
	sounding        bool

	// synthesizer state
	period, ptick int
	phase         [3]float64
	xa            [3]float32    // fxSmooth: the tone amplitudes as they ramp
	xv            float32       // fxSmooth: the voice's gain as it ramps in
	vbuzz         float64       // fxBuzz: this tick's band output, kept out of the bed's slope
	neq           int           // the playing effect's peaking filters (fxeq)
	eqc           [3][5]float64 // b0 b1 b2 a1 a2
	eqz           [3][2]float64
	bzbuf         [1024]float64 // fxBuzz: the source's recent past, for the comb
	bzpos, bzclk  int           // and ticks to the next click
	nz            [2]float64    // the noise resonator's two delays
	nhp           [4]float64    // an allophone's noise high pass: input and output of each of its two stages
	venv          float64       // the voicing envelope this tick, for an fxAll effect's noise
	wander        float64       // an fxWander part's pitch drift, a fraction
	rnd           uint32
	lastr         float64 // the previous noise sample, for the floor's high pass
	vprev         float64 // the output's last tick
	bed           float64 // the output's change per tick, smoothed, for the effects' bed
	am            float64 // the effects' slow level wobble

	// the serial control protocol (manual, SCP): "\\0" enters it, then the synthesizer's registers are written
	// directly ("<address>J<value>N"); its commands act at once, never through the buffer
	scp, esc     bool       // in control mode; a "\\" seen, its node character next
	sval         uint32     // the SCP input buffer's decimal value so far
	slen         int        // and its characters (16 at most)
	mtype, maddr uint8      // the Memory Type and Memory Address control registers
	reg          [16]uint16 // the synthesizer's registers as SCP last wrote them (0-5 Hz, 6 distortion, 8 envelope control, 11-15 volumes; 7, the master volume, is volume)
	hold         bool       // SCP "S": the MSA ignores the buffer until "T"
	sph          [5]float64 // the direct oscillators' phases
	eage         int        // ticks into the envelope oscillator's period

	// output
	held, heldpos float64
	z             [2]float64
	lpf           [6]float64 // the low pass: its two poles (per tick), their decays over an output sample, p2 / (p2 - p1), p2 / zero
	speaking      bool

	// the amount of volume increase to apply to each sample. defaults to 1.0
	Gain float32
}

// NewSpeakJet is the preferred method of initialisation for the SpeakJet type
func NewSpeakJet() *SpeakJet {
	sj := &SpeakJet{Gain: 1.0}
	sj.powerOn()
	return sj
}

func (sj *SpeakJet) resetControls() {
	sj.volume, sj.speed, sj.pitch, sj.bend, sj.repeat = 96, 114, 88, 5, 0 // the manual's Pitch 88: its period is one of the twelve one tick long (92 ticks, VoxProbe 2 and 10)
	sj.fast, sj.slow, sj.stress, sj.relax = false, false, false, false
}

func (sj *SpeakJet) powerOn() {
	gain := sj.Gain
	*sj = SpeakJet{Gain: gain}
	sj.resetControls()
	sj.rnd = 0x12345678
	// the output low pass: two real poles (Q under 0.5) per chip tick, their
	// decays over one output sample, and the zero
	w := 2.0 * math.Pi * lpfFc / rate
	h := 0.5 / math.Min(lpfQ, 0.499)
	r := math.Sqrt(h*h - 1.0)
	sj.lpf[0] = w * (h - r)
	sj.lpf[1] = w * (h + r)
	sj.lpf[2] = math.Exp(-sj.lpf[0] * rate / SampleRate)
	sj.lpf[3] = math.Exp(-sj.lpf[1] * rate / SampleRate)
	sj.lpf[4] = sj.lpf[1] / (sj.lpf[1] - sj.lpf[0])
	sj.lpf[5] = sj.lpf[1] / (2.0 * math.Pi * lpfZero / rate)
}

// settle drives the two poles with x held for a stretch, exactly (e1, e2:
// their decays over it)
func (sj *SpeakJet) settle(x, e1, e2 float64) {
	u1 := sj.z[0] - x
	sj.z[0] = x + u1*e1
	sj.z[1] = x + (sj.z[1]-x)*e2 + u1*sj.lpf[4]*(e1-e2)
}

// CommandSize returns the number of bytes a command takes, including its data
// byte
func CommandSize(code uint8) int {
	switch code {
	case 20, 21, 22, 23, 24, 25, 26, 28, 29, 30:
		return 2
	}
	return 1
}

// Write a command to the input buffer. the data byte is used only by commands
// that take one. returns false if the buffer is too full to take the command,
// in which case it is lost
func (sj *SpeakJet) Write(code uint8, data uint8) bool {
	if CommandSize(code) == 1 && sj.scpByte(code) {
		return true
	}
	n := CommandSize(code)
	if sj.count+n > BufferSize {
		return false
	}
	sj.buf[(sj.head+sj.count)%BufferSize] = code
	if n == 2 {
		sj.buf[(sj.head+sj.count+1)%BufferSize] = data
	}
	sj.count += n
	return true
}

// scpByte gives one serial byte to the SCP module, which watches every byte: true if it took the byte. Registers 0-5
// take 0 to 3999 Hz, 6 and 8 0 to 255, 7 (the master volume, the MSA's Volume) 0 to 127, 11-15 0 to 31. "V" (say
// "Ready") is not played: the chip's phrase for it is unmeasured. Memory type 32 (the EEPROM) is ignored
func (sj *SpeakJet) scpByte(c uint8) bool {
	if sj.esc {
		sj.esc = false
		if c >= '0' && c <= '9' {
			// node 0 is every chip's; the AtariVox's own node is 0
			sj.scp = c == '0'
			sj.sval, sj.slen = 0, 0
			sj.mtype, sj.maddr = 0, 0
			return true
		}
		if sj.scp {
			sj.scp = false // an escape and a non-numeric character leave control mode
			return true
		}
		return false // outside control mode the character goes on to the MSA
	}
	if c == '\\' {
		sj.esc = true
		return true
	}
	if !sj.scp {
		return false
	}
	if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') {
		if sj.slen < 16 {
			sj.slen++
			if c <= '9' {
				sj.sval = sj.sval*10 + uint32(c-'0')
			}
		}
		return true
	}
	v := sj.sval
	sj.sval, sj.slen = 0, 0
	switch c {
	case 'H':
		sj.mtype = uint8(v)
	case 'J':
		sj.maddr = uint8(v)
	case 'N':
		if sj.mtype == 0 && sj.maddr < 16 {
			a := sj.maddr
			switch {
			case a == 7:
				sj.volume = uint8(min(v, 127))
			case a <= 5:
				sj.reg[a] = uint16(min(v, 3999))
			case a >= 11:
				sj.reg[a] = uint16(min(v, 31))
			case a == 6 || a == 8:
				sj.reg[a] = uint16(min(v, 255))
			}
		}
	case 'R': // clears the buffer: what is playing plays on
		sj.count = sj.executing + sj.ahead
	case 'S':
		sj.hold = true
	case 'T':
		sj.hold = false
	case 'W':
		sj.powerOn()
	case 'X':
		sj.scp = false
	}
	return true
}

// BufferHalfFull is the state of the Buffer Half Full status line: true when
// the buffer holds more than 32 bytes
func (sj *SpeakJet) BufferHalfFull() bool { return sj.count > bufferHalfFull }

// Speaking is the state of the Speaking status line
func (sj *SpeakJet) Speaking() bool { return sj.speaking }

// Reset the chip as though the reset line had been cycled: the buffer is
// cleared and the controls return to their defaults. the sound already
// playing is not cut short
func (sj *SpeakJet) Reset() {
	sj.head, sj.count, sj.executing = 0, 0, 0
	sj.resetControls()
}

// ------------------------------------------------------------- the MSA ----

// the length of a sound at the current Speed, as a factor on its length at
// Speed 114. linear: measured 95/197/287 ms for AW at Speed 127/90/60; the
// effects change more: (147 - Speed) / 33 on every part (fitted on every effect
// recording at a Speed other than 114; alone they read about (145 - Speed) / 31,
// steeper still at low Speeds than near 114). Speech below Speed 114 follows
// the effects' law (VoxProbe 12, both units: AY and IYUW x1.21 at Speed 108,
// the effects' law x1.18, the old speech law x1.125; Man Goes Down P1 at
// Speed 108 E 8.28 -> 6.01, P 1.78 -> 0.94); above 114 the speech law stays
// (P2 at Speed 118 is worse on the effects' law)
func (sj *SpeakJet) speedFactor(effect bool) float64 {
	if effect || sj.speed < 114 {
		return (fxSpeedA - float64(sj.speed)) / fxSpeedB
	}
	return (speedA - float64(sj.speed)) / speedB
}

// Fast x2/3 and Slow x4/3 of the played length (VoxProbe 12 section 2,
// 2026-11-05, at -20 dB, both units, 2600 and Stella: Fast AY IYUW x0.63-0.70,
// Slow AY IYUW OWWW NE AW UX x1.27-1.36; was x0.77 and x1.22). The unvoiced
// stops and affricates don't change (TT is 45-50 ms plain, fast or slow); the
// voiced stops do under Slow (VoxProbe 12: Slow GO OWWW +12 ms)
func (sj *SpeakJet) modifiedMs(idx int, ms float64) float64 {
	k := allophones[idx].kind
	if k == kStop || k == kAffr || (k == kVStop && !sj.slow) {
		return ms
	}
	if k == kVStop {
		return ms * slowFactor
	}
	if sj.fast {
		ms *= fastFactor
	}
	if sj.slow {
		ms *= slowFactor
	}
	if ms < 10 {
		return 10
	}
	return ms
}

// Bend: every frequency moves by the same amount, 44 Hz a step (D1's 697 and
// 1209 Hz both step by 44). below 5 there is a 22 Hz jump
func (sj *SpeakJet) bendHz() float64 {
	b := bendHz * (float64(sj.bend) - 5)
	if sj.bend < 5 {
		b += bendHz / 2
	}
	return b
}

// the master volume: measured on AW, 16 to 127 is only 6 dB
// the library's own sin, exp and log for everything done per sample: a
// platform's last bits differ, and the output quantiser turns a last bit on
// the wrong side of a step into a whole step. plain series in float64, the
// same as the C library's. sin2pi takes cycles
func sin2pi(x float64) float64 {
	x -= math.Floor(x + 0.5) // [-0.5, 0.5)
	if x > 0.25 {
		x = 0.5 - x
	} else if x < -0.25 {
		x = -0.5 - x // [-0.25, 0.25]
	}
	y := 2.0 * math.Pi * x
	y2 := y * y
	term, sum := y, y
	for n := 1; n <= 9; n++ { // to y^19: under 1e-15 at pi/2
		term *= -y2 / ((2.0 * float64(n)) * (2.0*float64(n) + 1.0))
		sum += term
	}
	return sum
}

func exp(x float64) float64 {
	if x < -700.0 {
		return 0.0
	}
	k := math.Floor(x/math.Ln2 + 0.5)
	r := x - k*math.Ln2 // |r| <= ln2 / 2
	term, sum := 1.0, 1.0
	for n := 1; n <= 15; n++ {
		term *= r / float64(n)
		sum += term
	}
	return math.Ldexp(sum, int(k))
}

func log(x float64) float64 {
	m, e := math.Frexp(x) // m in [0.5, 1)
	s := (m - 1.0) / (m + 1.0)
	s2 := s * s // |s| <= 1/3
	term, sum := s, s
	for n := 1; n <= 15; n++ {
		term *= s2
		sum += term / (2.0*float64(n) + 1.0)
	}
	return 2.0*sum + float64(e)*math.Ln2
}

func (sj *SpeakJet) volumeGain() float64 {
	// measured on both units at Pitch 20 (VoxProbe 6): Volume 16 is 15 dB
	// under 32, and from 32 up the level barely moves
	db := [5]float64{-15.5, -1.3, -1.1, 0.0, 2.2}
	at := [5]float64{16, 32, 64, 96, 127}
	v := float64(sj.volume)
	if v <= 0 {
		return 0.0
	}
	var d float64
	if v < at[0] {
		d = db[0] - (at[0]-v)*0.6
	} else {
		i := 0
		for i < 3 && v > at[i+1] {
			i++
		}
		d = db[i] + (db[i+1]-db[i])*(v-at[i])/(at[i+1]-at[i])
	}
	return exp(d / 20.0 * math.Ln10)
}

func bent(hz float64, bend float64) float32 {
	if hz > 0 {
		return float32(hz + bend)
	}
	return 0
}

func (sj *SpeakJet) addSegment(ticks int, from, to frame, voiced bool) {
	if sj.nseg >= maxSegments || ticks <= 0 {
		return
	}
	sj.seg[sj.nseg] = segment{ticks: ticks, from: from, to: to, voiced: voiced}
	sj.nseg++
}

// start playing the segments built so far
func (sj *SpeakJet) start(effect bool) {
	if !effect {
		sj.neq = 0 // an allophone has no peaking filters
	}
	if effect {
		sj.attack, sj.release = ticks(fxAttackMs), ticks(fxReleaseMs)
	} else {
		sj.attack, sj.release = ticks(attackMs), ticks(releaseMs)
	}
	sj.fxburst = 0
	sj.fxbms = fxburstMs
	sj.fxhpk = 0
	sj.onset = 0
	sj.onsetFrom = 0
	sj.releaseGain = 1.0
	sj.wfloor = 0
	sj.thump = 0
	sj.thlp = 0
	sj.cur, sj.tick, sj.stick = 0, 0, 0
	sj.copies = 0
	sj.stotal = 0
	for i := 0; i < sj.nseg; i++ {
		sj.stotal += sj.seg[i].ticks
	}
	sj.sounding = sj.nseg > 0
	// the MSA drives the synthesizer itself and leaves the oscillator volumes at 0 (manual: its pauses ramp them out)
	for k := 11; k <= 15; k++ {
		sj.reg[k] = 0
	}
	sj.effect = effect
	sj.read = false
	sj.ahead = 0
}

// fxDecode reads one effect part from the table's byte stream (fxdata, from fxat[code-200]): its length, a byte
// saying which fields follow (bits 0-1 the number of tones; 4 noise, 8 gate, 16 pulse offsets, 32 level ramp, 64 end
// levels, 128 a high flags byte), its flags and level, then three bytes a tone and only the fields it uses
func fxDecode(at int, f *fxpart) int {
	*f = fxpart{}
	f.ms = fxdata[at]
	h := fxdata[at+1]
	f.flags = uint16(fxdata[at+2])
	at += 3
	if h&128 != 0 {
		f.flags |= uint16(fxdata[at]) << 8
		at++
	}
	f.lv = int8(fxdata[at])
	at++
	n := int(h & 3)
	for k := 0; k < n; k++ {
		f.fa[k], f.fb[k], f.v[k] = fxdata[at], fxdata[at+1], fxdata[at+2]
		at += 3
	}
	if h&4 != 0 {
		f.nf, f.nw, f.nv = fxdata[at], fxdata[at+1], fxdata[at+2]
		at += 3
	}
	if h&8 != 0 {
		f.gate = fxdata[at]
		at++
	}
	if h&16 != 0 {
		f.pa = int16(uint16(fxdata[at]) | uint16(fxdata[at+1])<<8)
		f.pb = int16(uint16(fxdata[at+2]) | uint16(fxdata[at+3])<<8)
		at += 4
	}
	if h&32 != 0 {
		f.lr = int8(fxdata[at])
		at++
	}
	f.ve = f.v
	if h&64 != 0 {
		for k := 0; k < n; k++ {
			f.ve[k] = fxdata[at+k]
		}
		at += n
	}
	return at
}

func (sj *SpeakJet) playEffect(code int) {
	sj.lastNasal = false
	// its peaking filters (fxeq, at most 3): a fixed shaping of its spectrum toward the chip's, set after its
	// structure was tuned (2026-11-04); RBJ peaking biquads at the chip's rate
	sj.neq = 0
	for i := 0; i < len(fxeq) && sj.neq < 3; i++ {
		if int(fxeq[i][0]) != code-200 {
			continue
		}
		f, g, qq := float64(fxeq[i][1])*hzStep, float64(int8(fxeq[i][2]))/2.0, float64(fxeq[i][3])/10.0
		A := math.Pow(10.0, g/40.0)
		w := 2.0 * math.Pi * f / rate
		al := math.Sin(w) / (2.0 * qq)
		cw := math.Cos(w)
		a0 := 1.0 + al/A
		c := &sj.eqc[sj.neq]
		c[0] = (1.0 + al*A) / a0
		c[1] = -2.0 * cw / a0
		c[2] = (1.0 - al*A) / a0
		c[3] = -2.0 * cw / a0
		c[4] = (1.0 - al/A) / a0
		sj.eqz[sj.neq] = [2]float64{}
		sj.neq++
	}
	at, end := int(fxat[code-200]), int(fxat[code-199])
	bend := sj.bendHz()
	sj.nseg = 0
	for at < end {
		var part fxpart
		at = fxDecode(at, &part)
		p := &part
		var a, b frame
		a.lv = float32(exp(float64(p.lv) * (0.5 * math.Ln10 / 20.0)))
		b.lv = a.lv
		for k := 0; k < 3; k++ {
			a.f[k] = bent(float64(p.fa[k])*hzStep, bend)
			b.f[k] = bent(float64(p.fb[k])*hzStep, bend)
			a.a[k] = float32(p.v[k])
			b.a[k] = float32(p.ve[k])
		}
		// D0 to D11 are the DTMF pairs, measured on both units within 3 Hz
		// of the standard once each unit's rate is allowed for (keypad order:
		// D1 to D9, then 0, * and #), at equal level; the table holds their
		// length
		if code >= 240 && code <= 251 {
			row := [4]float64{697, 770, 852, 941}
			col := [3]float64{1209, 1336, 1477}
			d := code - 240
			r, c := 3, 1
			switch {
			case d == 0:
			case d == 10:
				c = 0
			case d == 11:
				c = 2
			default:
				r, c = (d-1)/3, (d-1)%3
			}
			a.f[0], b.f[0] = bent(row[r], bend), bent(row[r], bend)
			a.f[1], b.f[1] = bent(col[c], bend), bent(col[c], bend)
			a.f[2], b.f[2] = 0, 0
			a.a[0], b.a[0], a.a[1], b.a[1] = 31, 31, 31, 31
			a.a[2], b.a[2] = 0, 0
		}
		a.nf, b.nf = float32(p.nf)*hzStep, float32(p.nf)*hzStep
		a.nw, b.nw = float32(p.nw)*hzStep, float32(p.nw)*hzStep
		a.na, b.na = float32(p.nv), float32(p.nv)
		n := sj.nseg
		sj.addSegment(ticks(float64(p.ms)*msStep*sj.speedFactor(true)), a, b, p.flags&(fxVoiced|fxPulsed) != 0)
		if sj.nseg > n {
			sj.seg[n].gate = ticks(float64(p.gate))
			sj.seg[n].flags = p.flags
			sj.seg[n].pa, sj.seg[n].pb = float32(p.pa), float32(p.pb)
			sj.seg[n].lr = float32(p.lr)
		}
	}
	sj.start(true)
	// its own structure (fxs, the structure loop's): fade-in and fade-out lengths, and a burst of low noise at its start
	// (2026-11-06: the DTMF tones open on a broadband column ~15 ms long that the chip has and the library lacked)
	x := fxs[code-200]
	if x[0] > 0 {
		sj.attack = ticks(float64(x[0]))
	}
	if x[1] > 0 {
		sj.release = ticks(float64(x[1]))
	}
	sj.fxburst = float64(x[2]) / 64.0 // /64 since 2026-11-07 (was /256; values divided by four): 253 M1's 50 ms burst needs the range
	bc := thumpHz
	if x[3] > 0 {
		bc = float64(x[3]) * 32.0
	}
	sj.fxbk = 1.0 - math.Exp(-2.0*math.Pi*bc/8000.0)
	sj.fxblp = 0
	// 253 M1 (2026-11-07, both units, every take): a low block 100-500 Hz for the first ~50 ms under a hiss that has little
	// under 500 Hz. An effect's noise is otherwise white to DC: its own high pass (hp, Hz/16, one stage) and its burst's own
	// length (tl, ms)
	sj.fxhpk = 0
	if x[4] > 0 {
		sj.fxhpk = 1.0 / (1.0 + 2.0*math.Pi*(float64(x[4])*16.0)/8000.0)
	}
	sj.fxbms = fxburstMs
	if x[5] > 0 {
		sj.fxbms = float64(x[5])
	}
}

func alloFrame(a *allophone, second bool, bend float64) frame {
	// a glide runs from one allophone's frame to another's, levels included
	// (measured on all 16: IYRR starts on IY and ends on RR); a glide's f[0] and
	// f[1] hold the two allophones' indices and it has no formants of its own
	if a.kind == kGlide {
		if second {
			return alloFrame(&allophones[a.f[1]], false, bend)
		}
		return alloFrame(&allophones[a.f[0]], false, bend)
	}
	// a voiced stop's release ends on formants of its own (allog: the few that have them)
	fs := a.f
	if second {
		for _, g := range allog {
			if &allophones[g[0]] == a {
				fs = [3]uint8{g[1], g[2], g[3]}
			}
		}
	}
	var f frame
	for k := 0; k < 3; k++ {
		f.f[k] = bent(float64(fs[k])*hzStep, bend)
		f.a[k] = float32(a.a[k])
	}
	f.lv = float32(exp(float64(a.lv) * (0.5 * math.Ln10 / 20.0)))
	f.nf, f.nw, f.na = float32(a.nf)*hzStep, float32(a.nw)*hzStep, float32(a.na)
	return f
}

// the segments of one copy of a Repeat: the sound's body for its own copy
// length (about the manual's length for most sounds), no transition (the
// copies run on seamlessly, measured on ZZ x10 and EYIY x5). a stop's copy is
// the stop again, closure and release (VoxProbe 4 and 7, both units: the 12
// voiced stops repeated 2 to 5 times, each copy a gap then a burst)
func (sj *SpeakJet) copySegments(idx int) int {
	a := &allophones[idx]
	bend := sj.bendHz()
	tk := ticks((float64(a.cp)*msStep + copyExtraMs) * sj.speedFactor(false))
	tgt := alloFrame(a, false, bend)
	sj.nseg = 0
	switch {
	case a.kind == kStop || a.kind == kVStop:
		// the copy opens on a closure, then the release; cp is the whole copy, measured from one copy's onset to the
		// next. the closure is silent (VoxProbe 4 and 7, both units: GE, GO and the final stops dip 27-35 dB between
		// copies, 33-50 dB in the voice bar's band) or, for BE, BO, DE and DO, holds the voice bar (they dip only about 22 dB)
		closure := ticks(copyClosure * sj.speedFactor(false))
		if closure > tk-1 {
			closure = tk - 1
		}
		var gone frame
		gone.lv = 1
		if a.cb != 0 {
			gone.f[0] = tgt.f[0]
			gone.a[0] = tgt.a[0]
			gone.lv = tgt.lv
		}
		sj.addSegment(closure, gone, gone, a.cb != 0)
		sj.addSegment(tk-closure, tgt, alloFrame(a, true, bend), a.kind == kVStop)
	case a.kind == kGlide:
		tgt2 := alloFrame(a, true, bend)
		hold, move := tk/5, tk*7/10
		// the copy glides back from the last copy's end to the first frame (VoxProbe 4, both units: IYRR x5 rises
		// 1383 -> 2083 Hz over its first 60 ms each time, not a jump)
		sj.addSegment(hold, tgt2, tgt, true)
		sj.addSegment(move, tgt, tgt2, true)
		sj.addSegment(tk-hold-move, tgt2, tgt2, true)
	default:
		sj.addSegment(tk, tgt, tgt, int(a.a[0])+int(a.a[1])+int(a.a[2]) > 0)
	}
	return tk
}

// expand an allophone into segments: its first play
// the frame a sound's transition starts from: the last frame, or, after
// silence, the sound's own frame at no level (the formants don't sweep up
// from 0 Hz at the start of a phrase or after a pause)
func (sj *SpeakJet) fromFrame(tgt frame) frame {
	if sj.last.a[0] > 0 || sj.last.a[1] > 0 || sj.last.a[2] > 0 || sj.last.na > 0 {
		return sj.last
	}
	tgt.a = [3]float32{}
	tgt.na = 0
	return tgt
}

func lerpFrame(a, b frame, t float32) frame {
	for q := 0; q < 3; q++ {
		a.f[q] += (b.f[q] - a.f[q]) * t
		a.a[q] += (b.a[q] - a.a[q]) * t
	}
	a.lv += (b.lv - a.lv) * t
	a.nf += (b.nf - a.nf) * t
	a.nw += (b.nw - a.nw) * t
	a.na += (b.na - a.na) * t
	return a
}

// anticipate: the sound's last at ticks bend F2 and F3 to (f2, f3), its levels and F1 as they are (split where the
// span starts)
func (sj *SpeakJet) anticipate(at int, f2, f3 float32) {
	total := 0
	for k := 0; k < sj.nseg; k++ {
		total += sj.seg[k].ticks
	}
	if at <= 0 || at >= total-1 || sj.nseg >= maxSegments {
		return
	}
	start := total - at
	acc, k := 0, 0
	for acc+sj.seg[k].ticks <= start {
		acc += sj.seg[k].ticks
		k++
	}
	if acc < start {
		sg := sj.seg[k]
		t1 := start - acc
		mid := lerpFrame(sg.from, sg.to, float32(t1)/float32(sg.ticks))
		copy(sj.seg[k+2:sj.nseg+1], sj.seg[k+1:sj.nseg])
		sj.seg[k].ticks = t1
		sj.seg[k].to = mid
		sj.seg[k+1] = sg
		sj.seg[k+1].ticks = sg.ticks - t1
		sj.seg[k+1].from = mid
		sj.nseg++
		k++
	}
	f2a, f3a := sj.seg[k].from.f[1], sj.seg[k].from.f[2]
	el := 0
	for ; k < sj.nseg; k++ {
		g := &sj.seg[k]
		g.from.f[1] = f2a + (f2-f2a)*float32(el)/float32(at)
		g.from.f[2] = f3a + (f3-f3a)*float32(el)/float32(at)
		el += g.ticks
		g.to.f[1] = f2a + (f2-f2a)*float32(el)/float32(at)
		g.to.f[2] = f3a + (f3-f3a)*float32(el)/float32(at)
	}
}

func (sj *SpeakJet) playAllophone(idx int) {
	a := &allophones[idx]
	bend := sj.bendHz()
	tk := ticks(sj.modifiedMs(idx, float64(a.ms)*msStep) * sj.speedFactor(false))
	tgt := alloFrame(a, false, bend)
	from := sj.fromFrame(tgt)
	sj.nseg = 0
	sj.rampSeg = -1
	var silent frame
	silent.lv = 1
	switch a.kind {
	case kVowel, kNasal, kVFric:
		tr := ticks(transitionMs)
		if tr > tk {
			tr = tk
		}
		// a sound with an onset, from silence, starts on its own frame (its onset at full level)
		if a.kind == kVFric && a.ob > 0 && from.a[0]+from.a[1]+from.a[2]+from.na == 0 {
			tr = 0
		}
		sj.addSegment(tr, from, tgt, true)
		sj.addSegment(tk-tr, tgt, tgt, true)
	case kGlide:
		// measured on the chip's glides: the first frame holds for a fifth of
		// the sound, the move takes the next seven tenths, the second frame
		// holds the last tenth
		tgt2 := alloFrame(a, true, bend)
		// a glide's ba: its first frame's F1 in percent (161 AXUW)
		tgt0 := tgt
		tgt := tgt
		if a.ba > 0 {
			tgt.f[0] = tgt.f[0] * float32(a.ba) / 100.0
		}
		hold, move := tk/5, tk*7/10
		tr := ticks(transitionMs)
		if tr > hold {
			tr = hold
		}
		// with its own F1 (ba) the hold starts on the first allophone's and rises to it (161 AXUW)
		sj.addSegment(tr, from, tgt0, true)
		sj.addSegment(hold-tr, tgt0, tgt, true)
		// a glide's cl: its F1 holds until cl percent of the sound, then moves with the rest (161 AXUW)
		if k := int(a.cl) * tk / 100; a.cl > 0 && k > hold && k < hold+move {
			k1 := k - hold
			fr := float32(k1) / float32(move)
			m := tgt
			for q := 0; q < 3; q++ {
				m.f[q] = tgt.f[q] + (tgt2.f[q]-tgt.f[q])*fr
				m.a[q] = tgt.a[q] + (tgt2.a[q]-tgt.a[q])*fr
			}
			m.lv = tgt.lv + (tgt2.lv-tgt.lv)*fr
			m.nf = tgt.nf + (tgt2.nf-tgt.nf)*fr
			m.nw = tgt.nw + (tgt2.nw-tgt.nw)*fr
			m.na = tgt.na + (tgt2.na-tgt.na)*fr
			m.f[0] = tgt.f[0]
			sj.addSegment(k1, tgt, m, true)
			sj.addSegment(move-k1, m, tgt2, true)
		} else {
			sj.addSegment(move, tgt, tgt2, true)
		}
		sj.addSegment(tk-hold-move, tgt2, tgt2, true)
	case kFric:
		sj.addSegment(tk, tgt, tgt, false)
	case kStop, kVStop:
		// the closure ("silence already added" says the manual; a voiced stop
		// hums through it on its first formant alone) then the burst
		// and release, which is the measured length
		// the sound before fades away through the closure (both units: a vowel
		// before a stop falls 30 dB over 50 to 60 ms at Speed 114, twice that
		// at Speed 60), not a cut to silence
		bar, gone := from, silent
		if a.kind == kVStop {
			gone.f[0] = tgt.f[0]
			gone.a[0] = tgt.a[0]                                                    // the first formant's level: both units' voiced stops open on a voice bar 2-9 dB under the release (BE, BO, DE, DO, GE, GO)
			gone.lv = tgt.lv                                                        // at the sound's own level (VoxProbe 6, slowed: DE's bar 3-4 dB under its release, BE's 8)
			gone.lv = float32(float64(tgt.lv) * math.Pow(10.0, float64(a.bl)/40.0)) // or its own (the structure loop's bl)
			// and its upper formants, weak (both units, the bar alone: 6-10 dB more than F1's tail at 1.8-5 kHz, 12-24 on GE GO)
			gone.f[1] = tgt.f[1]
			gone.f[2] = tgt.f[2]
			gone.a[1] = float32(float64(tgt.a[1]) * barUp)
			gone.a[2] = float32(float64(tgt.a[2]) * barUp)
		}
		// from silence, a voiced stop's bar is there at once, at its level (both units, the VoxRecord solo takes lined up
		// on the release: BE BO DE DO GE GO's bar 2-11 dB under the peak through the closure, not a fade up from nothing)
		if a.kind == kVStop && bar.a[0]+bar.a[1]+bar.a[2]+bar.na == 0 {
			bar = gone
		}
		// each stop's own closure (both units, lined up on the release: BE BO DE DO's bar 50-55 ms, the final stops' 5-16)
		cl := closureMs
		if a.cl > 0 {
			cl = float64(a.cl)
		}
		// ASSUMPTION (2026-11-07, VoxProbe 12 DO AYWW, both units: 284 ms where DO's 44 ms closure gave 315; DO alone
		// needs the 44): a voiced stop from silence, not slowed, running into a sound with no Slow on it closes for at
		// most 30 ms. "man goes down"'s DO (Slow on its AYWW) keeps its closure, as the recordings want
		if a.kind == kVStop && from.a[0]+from.a[1]+from.a[2]+from.na == 0 && sj.nextSound(sj.executing) >= 128 && !sj.slowBeforeNext(sj.executing) && !sj.slow && cl > 30.0 {
			cl = 30.0
		}
		// after a sound the chip closes for at least joinClosure (Man Goes Down's NE GO, both units)
		if from.a[0]+from.a[1]+from.a[2]+from.na > 0 && cl < joinClosure {
			cl = joinClosure
		}
		// a voiced stop after a nasal closes quietly, and Slow lengthens that closure (Man Goes Down's NE GO, VoxProbe 12
		// section 4, all four takes: the join falls to -38..-41 dB above 300 Hz, under -30 dB for 30-40 ms as sent and
		// 20-30 without its Slow; ours stayed above -22 dB and "goes" lost its /g/. EH DE, a vowel into a voiced stop,
		// has no such gap on either unit. P1: E 6.01 -> 5.02, P 0.94 -> 0.53; P2 and EH DE unchanged)
		nasalJoin := a.kind == kVStop && sj.lastNasal && from.a[0]+from.a[1]+from.a[2]+from.na > 0
		// the gap's width, band by band on both units' Man Goes Down NE GO (2026-11-07): 20-32 ms under the row's peak
		// -45 dB at 700-3000 Hz, closing at 473-485 ms; 45 ms read 28-38 and closed 480-488
		if nasalJoin {
			cl = nasalClosure
		}
		if nasalJoin && sj.slow {
			cl *= slowFactor
		}
		closure := ticks(cl * sj.speedFactor(false))
		// from silence, the bar opens on a click: the release's full frame falling into the bar over barClick ms (both
		// units, every voiced stop's first 10 ms 11-33 dB over the library's at 1-10 kHz)
		click := 0
		if a.kind == kVStop && from.a[0]+from.a[1]+from.a[2]+from.na == 0 && barClick > 0 {
			click = ticks(barClick)
			if click > closure-1 {
				click = closure - 1
			}
			sj.addSegment(click, tgt, gone, true)
		}
		if nasalJoin {
			quiet := gone
			quiet.lv = float32(float64(gone.lv) * nasalBar)
			// and the whole band dips evenly (both units: 300-700 Hz 22-25 ms under its peak -30 dB, where the bar's F1
			// hummed through; 1300-1700 at -38..-47 dB, where eight-fold F2 drew a line through the gap)
			quiet.a[0] *= 0.300
			quiet.a[1] *= 2.500
			quiet.a[2] *= 2.500
			// the nasal's F2 and F3 fall into the closure rather than rising to the stop's (both units, NE GO: F2 1640 ->
			// 1510, F3 2615 -> 2140 Hz; ours rose to GO's 1900 and 3200)
			quiet.f[1] = bar.f[1] * 0.92
			quiet.f[2] = bar.f[2] * 0.82
			fade := int(float64(closure) * nasalFade)
			sj.addSegment(fade, bar, quiet, true)
			sj.addSegment(closure-fade, quiet, quiet, true)
		} else {
			sj.addSegment(closure-click, bar, gone, a.kind == kVStop || bar.a[0]+bar.a[1]+bar.a[2] > 0)
		}
		// the release glides to the stop's second frame: the first formant falls into a labial or alveolar
		// closure (both units: EB, OB, ED, OD); a stop without a measured target holds
		// after a nasal the release comes in over nasalRamp from 10 dB down (both units: 5-15 dB under ours in its first
		// 10 ms, full by ~15 ms; ours started at full level with a 5 ms onset boost on top)
		if nasalJoin && tk > ticks(nasalRamp)+1 {
			low := tgt
			low.lv = float32(float64(tgt.lv) * nasalRampFrom)
			sj.rampSeg = sj.nseg
			sj.addSegment(ticks(nasalRamp), low, tgt, true)
			sj.addSegment(tk-ticks(nasalRamp), tgt, alloFrame(a, true, bend), true)
		} else {
			sj.addSegment(tk, tgt, alloFrame(a, true, bend), a.kind == kVStop)
		}
	case kAffr:
		// a stop's closure, then the fricative (voiced for JH)
		sj.addSegment(ticks(closureMs), silent, silent, false)
		sj.addSegment(tk, tgt, tgt, int(a.a[0])+int(a.a[1])+int(a.a[2]) > 0)
	}
	// a release that decays: the noise band ends its last segment at nd/256 of its level (TS, both units, every take:
	// -13 dB over its 95 ms, the low band holding - the white floor - while the band falls)
	if a.nd > 0 && sj.nseg > 0 {
		sj.seg[sj.nseg-1].to.na *= float32(a.nd) / 256.0
	}
	sj.start(false)
	sj.lastNasal = idx >= 12 && idx <= 16
	nasalRampOn := sj.rampSeg >= 0
	// an unvoiced fricative starts and stops at once (HE HO WH FF SE SO SH TH, both units, background allowed for: rise
	// and fall 0-4 ms, median 2, where the vowels' 35 ms ramp gave 22-38)
	if a.kind == kFric {
		sj.attack = ticks(fricRamp)
		sj.release = ticks(fricRamp)
	}
	if a.kind == kVStop && from.a[0]+from.a[1]+from.a[2]+from.na == 0 {
		sj.attack = ticks(fricRamp) // its bar starts at once too
	}
	if a.kind == kNasal {
		sj.attack = ticks(nasalAttack) // a nasal rises faster than a vowel (both units, lined up at -20 dB: full in 15 ms, not 25)
	}
	// a louder onset (VoxRecord solo, both units: the voiced stops' and fricatives' first 1-8 ms, 3-19 dB over the rest)
	sj.onset = ticks(float64(a.ob))
	sj.onsetGain = math.Pow(10.0, float64(a.og)/40.0)
	if nasalRampOn {
		sj.onset = 0 // its ramp replaces the onset boost
	}
	if sj.onset > 0 {
		sj.attack = 0 // a sound with an onset starts at once, the onset at full level
	}
	if a.kind == kVStop && a.ba > 0 && from.a[0]+from.a[1]+from.a[2]+from.na == 0 {
		sj.attack = ticks(float64(a.ba)) // the bar fades in (both units: 175 DO's bar quiet for its first ~30 ms)
	}
	if a.aa > 0 {
		sj.attack = ticks(float64(a.aa)) // a sound's own fade-in (the structure loop's aa: e.g. 185 WH starting ~5 ms early)
	}
	if (a.kind == kStop || a.kind == kVStop || a.kind == kAffr) && sj.nseg > 1 {
		for k := 0; k < sj.nseg-1; k++ {
			sj.onsetFrom += sj.seg[k].ticks // the release: everything before its segment
		}
	}
	// a stop's release level of its own (both units, lined up on the release: DE's and DO's voicing after the first
	// pulse 7-11 dB under the library's, their bars and bursts as the library's)
	if a.rl != 0 && sj.onsetFrom > 0 {
		sj.releaseGain = math.Pow(10.0, float64(a.rl)/40.0)
	}
	// a white floor under the band (WH, both units: noise 30 dB under its 700 Hz band out to 12 kHz)
	sj.wfloor = float64(a.wf) / 256.0
	hpc := nhpFC
	if a.hp > 0 {
		hpc = float64(a.hp) * 16.0
	}
	sj.nhk = 1.0 / (1.0 + 2.0*math.Pi*hpc/8000.0) // its own corner (PE, PO: their noise's bottom end)
	// a sibilant starts on a thump: one cycle near 190 Hz, downward (both units: SE SO SH 12-18 dB over their body's
	// 100-700 Hz band for 4-6 ms at the onset, every take)
	sj.thump = float64(a.th) / 64.0 // /64 since 2026-11-07 (was /256): SE's thump was pinned at the byte's top; values divided by four
	tcHz := thumpHz
	if a.tc > 0 {
		tcHz = float64(a.tc) * 32.0
	}
	sj.thk = 1.0 - math.Exp(-2.0*math.Pi*tcHz/8000.0)
	if a.rr > 0 {
		sj.release = ticks(float64(a.rr)) // a sound's own fade-out (SE, SO: the chip fades over most of the sound)
	}
	// the copies of a Repeat follow on; the level ramps once, over the whole
	sj.copies = 0
	if sj.repeat > 1 {
		sj.copies = int(sj.repeat) - 1
	}
	sj.copyOf = idx
	if sj.copies > 0 {
		nseg, total := sj.nseg, sj.stotal
		saved := sj.seg
		sj.stotal = total + sj.copies*sj.copySegments(idx)
		sj.seg = saved
		sj.nseg = nseg
	}
}

func (sj *SpeakJet) playPause(ms float64) {
	sj.nseg = 0
	var silent frame
	silent.lv = 1
	sj.addSegment(ticks(ms), silent, silent, false)
	sj.start(false)
}

func (sj *SpeakJet) peek(i int) uint8 { return sj.buf[(sj.head+i)%BufferSize] }

// take the next command from the buffer and start it. the command stays
// counted in the buffer until it has finished
func (sj *SpeakJet) nextCommand() {
	for sj.count > 0 && !sj.sounding && !sj.hold {
		code := sj.peek(0)
		n := CommandSize(code)
		var data uint8
		if n == 2 {
			data = sj.peek(1)
		}
		sj.executing = n
		switch {
		case code <= 6:
			sj.playPause(pauseMs[code])
			sj.joined = false
		case code >= 128 && code <= 199:
			// a sound played straight after the same sound (joined, controls between allowed) plays as a Repeat copy
			// of it (VoxProbe 12, 2026-11-05, both units, 2600 and Stella: AY AY 208 ms, AW AW AW 255, UX UX UX 260 -
			// what Repeat 2 / 3 makes, 214 / 268 / 259 - not two whole sounds, 272 / 316 / 329; a Pitch or Speed between
			// the copies doesn't change it: AW Pitch 88 AW Pitch 88 AW 256)
			again := sj.joined && sj.copyOf == int(code)-128 && sj.repeat <= 1
			sj.playAllophone(int(code) - 128)
			if again {
				sj.stotal = sj.copySegments(int(code) - 128)
				sj.cur = 0
				sj.tick = 0
			}
			sj.repeat = 0
			// the level ramps were measured on sounds played alone. inside a
			// phrase the chip's level never drops more than 14-30 dB between
			// sounds (Man Goes Down, both units) where ramping every sound out
			// and in drops it 40-65 dB: a sound that follows straight on starts
			// without its attack, and one followed straight on ends without its
			// release
			if sj.joined {
				sj.attack = 0
			}
			sj.joined = false
			// followed at once by another sound: the next one's start covers
			// this one's end
			next := sj.nextSound(n)
			// ASSUMPTION (2026-11-07, Man Goes Down P4 only, both units: its Slow IYUW ends ~40-50 ms after ours; the same
			// Slow IYUW alone matches, VoxProbe 12): a slowed glide followed by modifiers that include a Slow, before the
			// next sound, plays slowGlideOn ms longer. P4 944 ms on the chip, 875 -> 925 here; the cause is unknown
			// (VoxProbe 13 tests a Slow after a sound)
			if allophones[code-128].kind == kGlide && sj.slow && next >= 128 && sj.slowBeforeNext(n) && sj.nseg > 0 {
				sj.seg[sj.nseg-1].ticks += ticks(slowGlideOn * sj.speedFactor(false))
				sj.stotal += ticks(slowGlideOn * sj.speedFactor(false))
			}
			// a voiced sound straight into SE bends F2 and F3 to ZZ's, SE's voiced partner, over its last 40 ms (Man Goes
			// Down P4, both units: IYUW's F2 falls to ~1090 Hz and rises to ~1500 before the hiss; ZZ's F2 is 1469)
			if next == 187 && sj.nseg > 0 && sj.seg[sj.nseg-1].voiced {
				zz := &allophones[167-128]
				sj.anticipate(ticks(40.0*sj.speedFactor(false)), bent(float64(zz.f[1])*hzStep, sj.bendHz()), bent(float64(zz.f[2])*hzStep, sj.bendHz()))
			}
			// GO followed at once by a sound starting on OW (OW OWRR OWIY OWWW) releases on F2 1430 and F3 1920 Hz, F3 the
			// stronger by 4-5 dB (Man Goes Down's GO OWWW, both units, the user's drawn lines: two level bands that then
			// split into OW's F2 and F3); alone it keeps its table formants, which its solo takes need (moving them there:
			// spectrum 2.28 -> 4.12). Only before OW until another context is recorded (VoxProbe 13 section 7)
			if code == 179 && (next == 137 || next == 153 || next == 156 || next == 164) && sj.nseg > 0 {
				r := &sj.seg[sj.nseg-1]
				bh := sj.bendHz()
				r.from.f[1] = bent(1430.0, bh)
				r.to.f[1] = r.from.f[1]
				r.from.f[2] = bent(1920.0, bh)
				r.to.f[2] = r.from.f[2]
				r.to.a[1] = r.to.a[1] * 0.5
				r.from.a[1] = r.to.a[1]
				r.from.a[2] = r.to.a[2] * 6.0
				r.to.a[2] = r.from.a[2]
				if sj.rampSeg >= 0 && sj.rampSeg < sj.nseg-1 {
					q := &sj.seg[sj.rampSeg]
					q.from.f[1], q.to.f[1] = r.from.f[1], r.from.f[1]
					q.from.f[2], q.to.f[2] = r.from.f[2], r.from.f[2]
					q.from.a[1], q.to.a[1] = r.from.a[1], r.from.a[1]
					q.from.a[2], q.to.a[2] = r.from.a[2], r.from.a[2]
				}
			}
			if next >= 0 && sj.nseg > 0 {
				l := &sj.seg[sj.nseg-1]
				// a stop on either side takes the short overlap (the comment's own "a stop on either side"; the code only
				// looked at the next sound, so GO lost 51 ms of its release to OWWW and turned 15-25 ms early)
				kc := allophones[code-128].kind
				both := l.voiced && kc != kStop && kc != kVStop && next <= 199 && allophones[next-128].kind != kFric && allophones[next-128].kind != kStop
				cut := ticks(overlapMs)
				if both {
					cut = ticks(overlapVMs)
				}
				// no overlap before the same sound again: it plays as a copy, which runs on from this one as a Repeat's do
				// no overlap after an unvoiced stop: its burst and aspiration play whole (the TT AW pairs, VoxProbe 4, 8, 9,
				// both units: 166-176 ms where ours read 145; 160 with the whole release)
				if next == int(code) || kc == kStop {
					cut = 0
				}
				// ASSUMPTION (2026-11-07, fitted on VoxProbe 4's vowel pairs, both units, 2600 and Stella; replaces the fixed
				// 30 ms for these joins, an earlier finding these recordings don't support): a vowel or glide running into a
				// vowel or glide is cut by joinA - joinK * d ms, d the ERB distance of F1 and F2 between its last frame and
				// the next sound's; below zero it runs on by that much (the chip takes longer between sounds further apart:
				// AW IY and IY AW 261/260 ms where the fixed cut gave 215, AW UW 204 where it gave 212). Nasals, liquids and
				// voiced hisses keep the fixed cut: their table formants aren't a vowel position, and the rule put "man goes
				// down" and Man Goes Down P2 51-145 ms long
				kn := -1
				if next <= 199 {
					kn = int(allophones[next-128].kind)
				}
				if both && next != int(code) && (kc == kVowel || kc == kGlide) && (kn == int(kVowel) || kn == int(kGlide)) {
					b := &allophones[next-128]
					if b.kind == kGlide {
						b = &allophones[b.f[0]] // a glide's f[0] f[1] name the allophones it runs between: it starts on the first's
					}
					bf := [2]float64{float64(int(b.f[0]) * hzStep), float64(int(b.f[1]) * hzStep)}
					d := 0.0
					for q := 0; q < 2; q++ {
						ea := 21.4 * log(1.0+0.00437*float64(l.to.f[q])) / math.Ln10
						eb := 21.4 * log(1.0+0.00437*bf[q]) / math.Ln10
						if ea > eb {
							d += ea - eb
						} else {
							d += eb - ea
						}
					}
					ms := (joinA - joinK*d) * sj.speedFactor(false)
					if ms >= 0.0 {
						cut = ticks(ms)
					} else {
						cut = 0
						l.ticks += ticks(-ms)
						sj.stotal += ticks(-ms)
					}
				}
				// ASSUMPTION (2026-11-07, every recorded voiced-stop pair, both units): a voiced stop into a vowel or glide
				// with no Slow on it loses 30 - 10 * d ms more off its end, d the ERB distance between its release F2 and
				// the vowel's (BE EH 168 ms where we gave 198, DO AYWW 284 / 315; BE AW and GO OWWW, far apart, matched)
				if kc == kVStop && next != int(code) && (kn == int(kVowel) || kn == int(kGlide)) && !sj.slowBeforeNext(n) {
					b := &allophones[next-128]
					if b.kind == kGlide {
						b = &allophones[b.f[0]]
					}
					ea := 21.4 * log(1.0+0.00437*float64(l.to.f[1])) / math.Ln10
					eb := 21.4 * log(1.0+0.00437*float64(int(b.f[1])*hzStep)) / math.Ln10
					d := eb - ea
					if ea > eb {
						d = ea - eb
					}
					if extra := 30.0 - 10.0*d; extra > 0.0 {
						sj.trimTail(cut + ticks(extra*sj.speedFactor(false)))
						l = &sj.seg[sj.nseg-1]
						cut = 0
					}
				}
				if cut > 0 && l.ticks > cut+1 {
					l.ticks -= cut
					sj.stotal -= cut
				}
				sj.release = 0
				sj.joined = true
			}
		case code >= 200 && code <= 254:
			sj.playEffect(int(code))
			sj.repeat = 0
			sj.joined = false
		default:
			switch code {
			case 7:
				sj.fast = true
			case 8:
				sj.slow = true
			case 14:
				sj.stress = true
			case 15:
				sj.relax = true
			case 20:
				sj.volume = min(data, 127)
			case 21:
				sj.speed = min(data, 127)
			case 22:
				sj.pitch = data
			case 23:
				sj.bend = min(data, 15)
			case 26:
				sj.repeat = data
			case 30:
				sj.playPause(float64(data) * 10.0)
			case 31:
				sj.resetControls()
			default: // 16 wait, 24/25 ports, 28/29 eeprom, 255: nothing
			}
		}
		if !sj.sounding {
			// the command is done at once
			sj.head = (sj.head + n) % BufferSize
			sj.count -= n
			sj.executing = 0
		}
	}
}

// the next sound in the buffer after offset i, looking past the modifiers and
// settings that sit between sounds (Fast, Slow, Stress, Relax, Pitch, Speed,
// Volume, Bend, Repeat): a phrase from text puts a pitch or a stress before
// most sounds. a pause, a delay, a reset or the end of the buffer means the
// sound is not followed at once: -1
// slowBeforeNext: a Slow among the modifiers between offset i and the next sound
func (sj *SpeakJet) slowBeforeNext(i int) bool {
	for i < sj.count {
		c := int(sj.peek(i))
		if c >= 128 {
			return false
		}
		switch c {
		case 8:
			return true
		case 7, 14, 15, 16:
			i++
		case 20, 21, 22, 23, 26:
			i += 2
		default:
			return false
		}
	}
	return false
}

// trimTail: the sound's last span ticks taken off, across its segments
func (sj *SpeakJet) trimTail(span int) {
	for span > 0 && sj.nseg > 1 {
		g := &sj.seg[sj.nseg-1]
		if g.ticks > span+1 {
			g.ticks -= span
			sj.stotal -= span
			return
		}
		span -= g.ticks
		sj.stotal -= g.ticks
		sj.nseg--
	}
}

func (sj *SpeakJet) nextSound(i int) int {
	for i < sj.count {
		c := int(sj.peek(i))
		if c >= 128 && c <= 254 {
			return c
		}
		switch c {
		case 7, 8, 14, 15, 16:
			i++
		case 20, 21, 22, 23, 26:
			i += 2
		default:
			return -1
		}
	}
	return -1
}

// the sound has ended: retire its command, clear the one-shot modifiers
func (sj *SpeakJet) finishSound() {
	if sj.peek(0) >= 128 {
		sj.fast, sj.slow, sj.stress, sj.relax = false, false, false, false
	}
	sj.head = (sj.head + sj.executing + sj.ahead) % BufferSize
	sj.count -= sj.executing + sj.ahead
	sj.executing = 0
	sj.ahead = 0
	sj.sounding = false
}

// --------------------------------------------------------- synthesizer ----

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func (sj *SpeakJet) random() float64 {
	sj.rnd = sj.rnd*1664525 + 1013904223
	return float64(sj.rnd>>8)*(1.0/16777216.0) - 0.5
}

// readAheadMs: the chip reads the command after a sound while the sound is still playing, at its release: an
// effect's closing fade, or the last readAheadMs (VoxProbe 8, both units: "Speed v, D1, Speed 114" fades at Speed
// 114's length whatever v; "Volume 16, IY, Volume 96" jumps to Volume 96 in IY's last 25-30 ms; VoxProbe 7 and 4: a
// Pitch after a sound doesn't move it, and only the one command right after is read). A Speed or Volume read so is
// carried out at once: the rest of the sound plays at the new rate and level
const readAheadMs = 30

func (sj *SpeakJet) readAhead() {
	sj.read = true
	i := sj.executing
	if i+2 > sj.count {
		return
	}
	code, data := sj.peek(i), sj.peek(i+1)
	switch code {
	case 20:
		if data > 127 {
			data = 127
		}
		sj.volume = data
	case 21:
		before := sj.speedFactor(sj.effect)
		if data > 127 {
			data = 127
		}
		sj.speed = data
		scale := sj.speedFactor(sj.effect) / before
		total := 0
		for k := 0; k < sj.nseg; k++ {
			g := &sj.seg[k]
			if k == sj.cur {
				g.ticks = sj.tick + 1 + int(float64(g.ticks-sj.tick-1)*scale+0.5)
			} else if k > sj.cur {
				if float64(g.ticks)*scale < 1.0 {
					g.ticks = 1
				} else {
					g.ticks = int(float64(g.ticks)*scale + 0.5)
				}
			}
			total += g.ticks
		}
		sj.stotal = total
	default:
		return
	}
	sj.ahead = 2
}

// outputStage applies the master volume and the output stage: the voice (an allophone's voicing) gets the floor,
// everything else the bed
func (sj *SpeakJet) outputStage(v float64, voice bool) float64 {
	v *= sj.volumeGain()
	// the output stage's step size
	if quant > 0 {
		v = math.Floor(v*quant+0.5) / quant
	}
	// the output noise floor: white under the effects' tones; none under the
	// voice (floor 0, the chip's voice is exactly periodic)
	r := sj.random()
	if voice {
		v *= 1.0 + floor*2.0*(r-sj.lastr)
	} else {
		// the effects' bed (both units): timing error, proportional to the
		// signal's change per tick smoothed by one pole - it grows with the
		// tone's frequency to about 1.5 kHz and only with its level above
		// the buzz's clicks kept out of the slope (each is a huge step: the bed turned 227 B7's bands into noise)
		sj.bed += bedPole * (v - sj.vbuzz - sj.vprev - sj.bed)
		v += slew * 2.0 * r * sj.bed
	}
	sj.lastr = r
	sj.vprev = v - sj.vbuzz
	sj.vbuzz = 0.0
	return v
}

// scpDB is a register tone's level: VoxProbe 9, both units, the 1000 Hz tone at volume 31 stands 3.1-3.6 dB over 235
// C5 and 241 D1, which put it 4.9 and 6.9 dB above the effects' tone scale; the mean (written out as a factor below,
// so every compiler gets the same)
const scpDB = 5.9

func (sj *SpeakJet) scpAudible() bool {
	return sj.volume > 0 && (sj.reg[11]|sj.reg[12]|sj.reg[13]|sj.reg[14]|sj.reg[15]) != 0
}

// scpSample is the synthesizer as SCP set it, between the MSA's sounds: five sines at registers 1-5 and 11-15;
// register 8's bits 0-1 the envelope's shape (as the effects': saw is the voice's pulse), bit 6 envelopes oscillators
// 1-3, bit 7 half envelopes 4 and 5, and no bed (the effects' own); the envelope runs at register 0 Hz, ceil(8000 / rate) ticks (the effects' rule);
// register 6 spreads 4 and 5 into noise round their frequencies. Unmeasured: the distortion's law (here a random
// frequency error of up to distortion / 255 a tick), and the envelope at 0 Hz (here silent)
func (sj *SpeakJet) scpSample() float64 {
	ctl, rt := int(sj.reg[8]), int(sj.reg[0])
	env := 1.0
	if ctl&0xc0 != 0 {
		if rt <= 0 {
			env = 0.0
		} else {
			period := int(math.Ceil(8000.0 / float64(rt)))
			switch rt {
			case 44, 51, 53, 56, 63, 77, 87, 88, 92, 104, 127, 151:
				period++
			}
			if sj.eage >= period {
				sj.eage = 0
			}
			if sj.eage == 0 && ctl&0x40 != 0 {
				for k := 0; k < 3; k++ {
					sj.sph[k] = 0.0 // the enveloped tones restart with each pulse, as the voice's
				}
			}
			age := sj.eage
			sj.eage++
			x := float64(age) / float64(period)
			switch ctl & 3 {
			case 1:
				env = 0.5 + 0.5*sin2pi(x+0.25)
			case 2:
				env = x
			case 3:
				if x < 0.5 {
					env = 1.0
				} else {
					env = 0.0
				}
			default:
				fa := float64(age)
				env = 1.0 - (fa-envRise)/envFall
				if fa < envRise {
					env = fa / envRise
				}
				if env < 0.0 {
					env = 0.0
				}
			}
		}
	}
	s13, s45 := 0.0, 0.0
	for k := 0; k < 5; k++ {
		a, f := float64(sj.reg[11+k]), float64(sj.reg[1+k])
		if a > 0.0 && f > 0.0 {
			if k < 3 {
				s13 += a * sin2pi(sj.sph[k])
			} else {
				s45 += a * sin2pi(sj.sph[k])
			}
		}
		step := f / nominal
		if k >= 3 && sj.reg[6] > 0 {
			step *= 1.0 + float64(sj.reg[6])/255.0*2.0*sj.random()
		}
		sj.sph[k] += step
		sj.sph[k] -= math.Floor(sj.sph[k])
	}
	v := 0.0
	if ctl&0x40 != 0 {
		v -= env * s13 / 128.0 // enveloped, as the voice: the pulse starts downward
	} else {
		v += s13 / 128.0 * 0.6 // plain, as the effects' tones
	}
	half := 1.0
	if ctl&0x80 != 0 {
		half = 0.5*env + 0.5
	}
	v += half * s45 / 128.0 * 0.6
	v *= 1.9724227361148539
	return sj.outputStage(v, true) // no bed: under a register tone it is 57 dB down (VoxProbe 10, all three takes)
}

// one chip sample
func (sj *SpeakJet) chipSample() float64 {
	if !sj.sounding {
		sj.nextCommand()
	}
	if !sj.sounding {
		if sj.scpAudible() {
			return sj.scpSample()
		}
		return 0.0
	}
	if !sj.read && sj.copies == 0 && sj.peek(0) >= 128 { // sounds only: no recording shows a pause reading ahead
		fade := sj.seg[sj.nseg-1].flags&fxFade != 0
		if (fade && sj.cur == sj.nseg-1) || (!fade && sj.stotal-sj.stick <= ticks(readAheadMs)) {
			sj.readAhead()
		}
	}

	s := &sj.seg[sj.cur]
	t := 1.0
	if s.ticks > 1 {
		t = float64(sj.tick) / float64(s.ticks-1)
	}
	var f frame
	for k := 0; k < 3; k++ {
		fa, fb := float64(s.from.f[k]), float64(s.to.f[k])
		switch {
		case s.flags&fxExp != 0 && fa > 0 && fb > 0:
			f.f[k] = float32(fa * exp(t*log(fb/fa)))
		case s.flags&fxExpDecay != 0:
			f.f[k] = float32(fb + (fa-fb)*exp(-4.0*t))
		default:
			f.f[k] = float32(lerp(fa, fb, t))
		}
		f.a[k] = float32(lerp(float64(s.from.a[k]), float64(s.to.a[k]), t))
	}
	f.lv = float32(lerp(float64(s.from.lv), float64(s.to.lv), t))
	if s.lr != 0 {
		f.lv *= float32(exp(float64(s.lr) * t * (0.5 * math.Ln10 / 20.0)))
	}
	f.nf = float32(lerp(float64(s.from.nf), float64(s.to.nf), t))
	f.nw = float32(lerp(float64(s.from.nw), float64(s.to.nw), t))
	f.na = float32(lerp(float64(s.from.na), float64(s.to.na), t))
	if s.flags&fxSmooth != 0 {
		if sj.stick == 0 {
			for k := 0; k < 3; k++ {
				sj.xa[k] = f.a[k]
			}
		}
		for k := 0; k < 3; k++ {
			onoff := f.a[k] < 0.5 || sj.xa[k] < 0.5 || (sj.xa[k] < f.a[k]*0.5 && sj.xa[k] < 2.0)
			if onoff {
				sj.xa[k] = sj.xa[k] + (f.a[k]-sj.xa[k])/24.0
			} else {
				sj.xa[k] = f.a[k]
			}
			f.a[k] = sj.xa[k]
		}
	} else {
		for k := 0; k < 3; k++ {
			sj.xa[k] = f.a[k]
		}
	}
	sj.last = f

	v := 0.0

	// voicing
	// fxSmooth: where voicing starts after unvoiced sound, the voice comes in over 3 ms (24 ticks)
	{
		want := float32(0.0)
		if s.voiced && sj.pitch > 0 {
			want = 1.0
		}
		if s.flags&fxSmooth == 0 || sj.stick == 0 {
			sj.xv = want
		} else if want > sj.xv {
			sj.xv += (want - sj.xv) / 24.0
		} else {
			sj.xv = want
		}
	}
	if s.voiced && sj.pitch > 0 {
		if sj.ptick <= 0 {
			// the period in ticks, from the chip's own nominal 8000: the voice runs
			// floor(8000 / Pitch) + 1 (VoxProbe 2; VoxProbe 9: 41 at Pitch 200, 67 at
			// 120), an effect's pulse ceil(8000 / rate), one tick shorter wherever
			// 8000 / rate is whole (VoxProbe 9, both units: C5 at 80, 67, 54, 50, 40,
			// 32 ticks for Pitch 100, 120, 150, 160, 200, 250). an effect's pulses run
			// at Pitch plus the part's offset (C1, C2, C3, C6, C7: the same offset at
			// Pitch 60, 88 and 150 on both units, gliding within the part)
			// the offset from integers and one division, so every compiler gets the same period
			f := float64(sj.pitch) + float64(s.pa)
			if s.ticks > 1 {
				f += float64((s.pb-s.pa)*float32(sj.tick)) / float64(s.ticks-1)
			} else {
				f += float64(s.pb - s.pa)
			}
			if f < 10.0 {
				f = 10.0
			}
			if s.flags&(fxVoiced|fxPulsed) != 0 {
				sj.period = int(math.Ceil(8000.0 / f))
			} else {
				// the voice counts a 16-bit phase up by floor(rate * 65536 / 8000) a tick: floor(65536 / step) + 1 ticks
				// (VoxProbe 2's Pitch sweep 20-255, both units: every Pitch to measuring precision, where
				// floor(8000 / Pitch) + 1 missed 27; it is why the reset default, Pitch 88, runs 92 ticks)
				sj.period = 65536/int(f*65536.0/8000.0) + 1
			}
			// an effect's pulse: one tick more at twelve rates (VoxProbe 10's register sweep, both units)
			if s.flags&(fxVoiced|fxPulsed) != 0 && f == float64(int(f)) {
				switch int(f) {
				case 44, 51, 53, 56, 63, 77, 87, 88, 92, 104, 127, 151:
					sj.period++
				}
			}
			sj.ptick = sj.period
		}
		// an effect's third tone is a plain tone beside the voice
		nf := 3
		if s.flags&fxVoiced != 0 && s.flags&fxAll == 0 {
			nf = 2
			if s.flags&fxOne != 0 {
				nf = 1
			}
		}
		if sj.ptick == sj.period && s.flags&fxPulsed == 0 { // pulsed tones run on; the enveloped ones restart, a plain tone beside them runs on
			for k := 0; k < nf; k++ {
				sj.phase[k] = 0.0
			}
		}
		age := float64(sj.period - sj.ptick)
		x := age / float64(sj.period)
		var env float64
		switch s.flags & fxShape {
		case fxSine:
			env = 0.5 + 0.5*sin2pi(x+0.25)
		case fxTri:
			env = x // VoxProbe 7, both units: a rising ramp, then straight back to nothing
		case fxSquare:
			if x < 0.5 {
				env = 1.0
			} else {
				env = 0.0
			}
		default:
			env = 1.0 - (age-envRise)/envFall
			if age < envRise {
				env = age / envRise
			}
			if env < 0.0 {
				env = 0.0
			}
		}
		// the offset: measured on every voiced sound's pulse (both units), an
		// envelope-shaped term beside the sines, against their starting direction
		sum := 0.0
		for k := 0; k < nf; k++ {
			sum += float64(f.a[k]) * sin2pi(sj.phase[k])
			sj.phase[k] += float64(f.f[k]) * (1.0 + jitter*2.0*sj.random()) / nominal
		}
		if s.flags&fxPulsed != 0 {
			env *= 0.6
		}
		v -= float64(sj.xv) * env * sum / 128.0 * float64(f.lv) // the pulse starts downward (both units, every vowel at Pitch 20)
		sj.venv = env
		for k := nf; k < 3; k++ {
			v += float64(f.a[k]) / 128.0 * 0.6 * float64(f.lv) * sin2pi(sj.phase[k])
			sj.phase[k] += float64(f.f[k]) * (1.0 + jitter*2.0*sj.random()) / nominal
		}
		sj.ptick--
	} else {
		sj.ptick = 0
		sj.venv = 1.0
		sj.wander += (0.015/(0.57735*math.Sqrt(0.055/2.0))*sj.random() - sj.wander) * 0.055
		// the effects' tones hold their pitch (both units) and the energy round
		// their lines is a slow wobble in level, 10 % rms, spread to about 64 Hz
		sj.am += (0.1/(0.57735*math.Sqrt(0.05/2.0))*sj.random() - sj.am) * 0.05
		wander := 0.0
		if s.flags&fxWander != 0 {
			wander = sj.wander
		}
		// plain tones (the effects)
		for k := 0; k < 3; k++ {
			if f.f[k] > 0 && f.a[k] > 0 {
				v += float64(f.a[k]) / 128.0 * 0.6 * float64(f.lv) * sin2pi(sj.phase[k])
			}
			sj.phase[k] += float64(f.f[k]) * (1.0 + wander) / nominal
		}
		amScale := 0.006 / 0.1 // a plain tone's level holds to 0.01-0.09 dB rms in its middle (VoxRecord, both units: the DTMF tones, B0, B3, R7, R9)
		if s.flags&fxWander != 0 {
			amScale = 1.0 // the full wobble only on a wandering part (M0)
		}
		v *= 1.0 + sj.am*amScale
	}

	// noise: white noise through a two-pole resonance at nf with bandwidth nw
	// (the chip's hiss is random noise - Gaussian to the sample, both units -
	// not a tone; FF is flat to 4 kHz, the others resonate from WH's 800 Hz
	// to SE's 4 kHz). the gain holds the output power at the input's whatever the band (the AR(2) variance)
	if f.na > 0 {
		nf, lvl := float64(f.nf), 1.0
		// fxNSweep: over each gate period the band climbs from 0.4 to 2 times
		// its centre and back, quieter at the top (measured on C8 and C9: 650
		// Hz to 3 kHz and back every 20 ms with a 13 dB dip, both units)
		if s.flags&fxNSweep != 0 && s.gate > 0 {
			// both units, C8 and C9: up in 8 ms, down in 12, by 1.8x either side
			// of the centre, and 5 dB quieter at the top beyond the output stage
			p := float64(sj.tick%s.gate) / float64(s.gate)
			sw := 1.0 - (p-0.4)/0.6
			if p < 0.4 {
				sw = p / 0.4
			}
			nf *= exp((2.0*sw - 1.0) * 0.75)        // 2026-11-06: the chip spans ~650 Hz - 2.4 kHz
			lvl = exp(-5.0 * sw / 20.0 * math.Ln10) // 5 dB quieter at the top (2026-11-06: 15 dB scored worse)
		}
		nw := float64(f.nw)
		if nw < 10 {
			nw = 10
		}
		r, c := exp(-math.Pi*nw/rate), sin2pi(nf/rate+0.25)
		r2 := r * r
		g := math.Sqrt((1.0 - r2) * ((1.0+r2)*(1.0+r2) - 4.0*r2*c*c) / (1.0 + r2))
		// fxBuzz: the noise minus itself one period of the part's pulse rate ago at 0.6: notches every rate Hz in an
		// otherwise flat noise, the power kept (232 C2's late burst)
		ex := sj.random()
		if s.flags&fxBuzz != 0 {
			// clicks, not noise: an impulse every voice period (Pitch), an even broadband colour; the comb cuts narrow
			// notches every rate Hz (user, C2: plain orange with narrow, regular purple stripes)
			vper := 92
			if sj.pitch > 0 {
				vper = int(8000.0/float64(sj.pitch)) + 1
			}
			if sj.bzclk <= 0 {
				sj.bzclk = vper
			}
			if sj.bzclk == vper {
				ex = math.Sqrt(float64(vper) / 12.0)
			} else {
				ex = 0.0
			}
			sj.bzclk--
			rate := float64(sj.pitch) + float64(s.pa)
			if s.ticks > 1 {
				rate += float64((s.pb-s.pa)*float32(sj.tick)) / float64(s.ticks-1)
			} else {
				rate += float64(s.pb - s.pa)
			}
			if rate < 10.0 {
				rate = 10.0
			}
			per := int(math.Ceil(8000.0 / rate))
			if per > 1023 {
				per = 1023
			}
			old := sj.bzbuf[(sj.bzpos-per)&1023]
			sj.bzbuf[sj.bzpos&1023] = ex
			sj.bzpos++
			ex = (ex - 0.6*old) * 0.8574929257125441 * 0.25
		}
		y := g*ex + 2.0*r*c*sj.nz[0] - r2*sj.nz[1]
		sj.nz[1] = sj.nz[0]
		sj.nz[0] = y
		if s.flags&fxAll != 0 {
			lvl *= sj.venv // both units: C0 and M2 are silent between their pulses
		}
		// an allophone's noise is high-passed twice at nhpFC (every unvoiced fricative, both units, the VoxRecord solo
		// takes with the background taken off: 6-21 dB too much under 400 Hz without it)
		yo := y
		// and the swept noise (C8, C9) at nsweepHP (2026-11-06: ~10 dB heavy under 300 Hz without it)
		if !sj.effect || s.flags&fxNSweep != 0 || sj.fxhpk > 0 {
			k := sj.nhk
			if sj.effect {
				k = 1.0 / (1.0 + 2.0*math.Pi*nsweepHP/8000.0)
				if sj.fxhpk > 0 {
					k = sj.fxhpk // the effect's own high pass (fxs hp)
				}
			}
			o1 := k * (sj.nhp[1] + y - sj.nhp[0])
			sj.nhp[0] = y
			sj.nhp[1] = o1
			o2 := k * (sj.nhp[3] + o1 - sj.nhp[2])
			sj.nhp[2] = o1
			sj.nhp[3] = o2
			yo = o2 + sj.wfloor*ex
		}
		if sj.thump > 0 && sj.stick < ticks(thumpMs) { // low noise (one pole at thumpHz) fading over thumpMs
			sj.thlp += sj.thk * (ex - sj.thlp)
			yo += sj.thump * sj.thlp * (1.0 - float64(sj.stick)/float64(ticks(thumpMs)))
		}
		v += lvl * float64(f.lv) * float64(f.na) / 128.0 * 0.86 * yo
		if s.flags&fxBuzz != 0 {
			sj.vbuzz = lvl * float64(f.lv) * float64(f.na) / 128.0 * 0.86 * y
		}
	}

	if s.gate > 0 && s.flags&fxNSweep == 0 && sj.tick%s.gate >= s.gate/2 {
		v *= gateDepth
	}
	if s.flags&fxDecay != 0 {
		v *= exp(-decayDB * t / 20.0 * math.Ln10)
	}
	if s.flags&fxFade != 0 {
		v *= 1.0 - t
	}
	if s.flags&fxBump != 0 {
		v *= sin2pi(0.5 * t)
	}

	// the sound's attack and release
	if sj.stick < sj.attack {
		v *= float64(sj.stick) / float64(sj.attack)
	}
	if sj.stotal-sj.stick < sj.release {
		v *= float64(sj.stotal-sj.stick) / float64(sj.release)
	}
	if sj.stick >= sj.onsetFrom && sj.stick < sj.onsetFrom+sj.onset {
		v *= sj.onsetGain
	}
	if sj.onsetFrom > 0 && sj.stick >= sj.onsetFrom {
		v *= sj.releaseGain
	}
	if sj.fxburst > 0 && sj.stick < ticks(sj.fxbms) {
		sj.fxblp += sj.fxbk * (2.0*sj.random() - sj.fxblp)
		v += sj.fxburst * sj.fxblp * (1.0 - float64(sj.stick)/float64(ticks(sj.fxbms)))
	}
	sj.stick++

	// the effect's peaking filters (transposed direct form II)
	for k := 0; k < sj.neq; k++ {
		c := &sj.eqc[k]
		y := c[0]*v + sj.eqz[k][0]
		sj.eqz[k][0] = c[1]*v - c[3]*y + sj.eqz[k][1]
		sj.eqz[k][1] = c[2]*v - c[4]*y
		v = y
	}
	v = sj.outputStage(v, s.voiced && s.flags&(fxVoiced|fxPulsed) == 0)

	sj.tick++
	if sj.tick >= s.ticks {
		sj.tick = 0
		sj.cur++
		if sj.cur >= sj.nseg {
			if sj.copies > 0 {
				sj.copies--
				sj.copySegments(sj.copyOf)
				sj.cur = 0
			} else {
				sj.finishSound()
			}
		}
	}
	return v
}

// Render fills buf with the next len(buf) samples of output, at SampleRate,
// in the range -1 to 1. returns true if the chip is producing sound
func (sj *SpeakJet) Render(buf []float32) bool {
	const step = rate / SampleRate
	sounding := sj.sounding || sj.count > 0 || sj.scpAudible()
	for i := range buf {
		// the held chip samples through the AtariVox's low pass, sampled at
		// the end of this output sample (the zero adds the second pole's input
		// in proportion: y2 + y2' / zero)
		if sj.heldpos+step < 1.0 {
			sj.settle(sj.held, sj.lpf[2], sj.lpf[3])
			sj.heldpos += step
		} else {
			a := 1.0 - sj.heldpos
			b := step - a
			sj.settle(sj.held, math.Exp(-sj.lpf[0]*a), math.Exp(-sj.lpf[1]*a))
			sj.held = sj.chipSample()
			sounding = sounding || sj.sounding || sj.count > 0 || sj.scpAudible()
			sj.settle(sj.held, math.Exp(-sj.lpf[0]*b), math.Exp(-sj.lpf[1]*b))
			sj.heldpos = b
		}
		buf[i] = float32(sj.z[1]+sj.lpf[5]*(sj.z[0]-sj.z[1])) * sj.Gain
	}
	sj.speaking = sounding
	return sounding
}
