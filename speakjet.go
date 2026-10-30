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
	rate    = 8026.0
	nominal = 8000.0
)

// the constants of the model, as the recordings gave them. the same values as
// the C library's struct tune
const (
	envRise      = 0.5    // voicing envelope: rise (under one tick: a step), ticks
	envFall      = 58.0   // and linear fall
	lpfFc        = 2396.0 // the output stage's two poles: real, near 2 kHz and 2.9 kHz
	lpfQ         = 0.491
	attackMs     = 35.0 // every sound's level ramp
	releaseMs    = 30.0
	fxAttackMs   = 10.0 // the effects'
	fxReleaseMs  = 15.0
	floor        = 0.05  // the output noise floor, relative (both units: the bed between an effect's lines is -30 to -36 dB per band re the line, whatever its level)
	jitter       = 0.03  // every oscillator\'s frequency wanders by this fraction each sample (the skirts around the chip\'s tones)
	quant        = 0.0   // the output\'s steps per full scale: the chip\'s output is coarse, about 5 bits across a vowel
	decayDB      = 35.0  // an fxDecay part falls this many dB
	transitionMs = 25.0  // the voice moves to a new sound's target over this
	closureMs    = 30.0  // the closure before a stop's burst
	slew         = 0.06  // the effects' bed, measured: grows with the tone's frequency on both units
	gateDepth    = 0.4   // an effect's gate: its off half's level, a factor
	overlapMs    = 15.0  // a sound followed at once by another is cut short by this
	overlapVMs   = 30.0  // more when both are voiced
	speedA       = 162.0 // length factor (speedA - Speed) / speedB
	speedB       = 48.0
	fxSpeedA     = 149.0 // the effects' (they shorten faster)
	fxSpeedB     = 35.0
	fastFactor   = 0.77 // Fast and Slow: factors on the played length
	slowFactor   = 1.22
	copyExtraMs  = 0.0  // a Repeat copy adds this to the manual's length
	bendHz       = 44.0 // Bend step
)

var pauseMs = [7]float64{0, 100, 200, 700, 30, 60, 90}

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
	gate     int    // ticks of the on/off period for a pulsed effect, 0 = none
	flags    uint16 // fx*
	pa, pb   float32 // an effect part's pulse rate: Pitch plus pa Hz at its start, pb at its end
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
	fxPulsed   = 64  // the plain tones pulse with the voice's envelope at Pitch
	fxFade     = 128 // the level falls straight to nothing over the part (the tones' measured ending)
	fxAll      = 256 // every tone and the noise of a voiced part pulse with the voice (C0, M2: silent between pulses on both units)
	fxWander   = 512 // the plain tones' pitch drifts (252 M0, both units: 1.45-1.59 % rms, spread to about 70 Hz)
	fxSine     = 1024 // the envelope oscillator's waveform (manual, register 8): saw (the voice's measured pulse),
	fxTri      = 2048 // sine, triangle or square
	fxSquare   = 3072
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
	lv       int8 // the part's level, half-decibels (measured against the voice, both units)
	pa, pb   int16 // the part's pulse rate, Hz added to Pitch at its start and end (both units: the same offset at Pitch 60, 88 and 150)
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
	g      [3]uint8 // kGlide: the second target
	nf, nw uint8    // the noise band's centre and bandwidth, hzStep
	na     uint8
	cp     uint8 // the length of each further copy of a Repeat, msStep
	lv     int8  // the sound's level, half-decibels (measured against the vowels, both units)
}

// the manual's lengths, for the copies of a Repeat
var manualMs = [72]uint8{
	70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70,
	70, 70, 70, 200, 200, 190, 200, 185, 165, 200, 225, 185, 170, 140, 180, 170, 170, 200,
	131, 70, 70, 70, 70, 70, 45, 45, 10, 10, 45, 45, 10, 10, 55, 55, 55, 55,
	70, 70, 70, 70, 70, 40, 40, 50, 40, 50, 70, 170, 55, 55, 55, 45, 99, 99}

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

	// what is playing
	seg             [maxSegments]segment
	nseg, cur, tick int
	stick, stotal   int // ticks into the sound, and its whole length
	attack, release int
	copies          int   // copies of a Repeat still to play
	copyOf          int   // which allophone
	last            frame // where the voice is, for transitions into the next sound
	sounding        bool

	// synthesizer state
	period, ptick int
	phase         [3]float64
	nz            [2]float64 // the noise resonator's two delays
	venv          float64    // the voicing envelope this tick, for an fxAll effect's noise
	wander        float64    // an fxWander part's pitch drift, a fraction
	rnd           uint32
	lastr         float64 // the previous noise sample, for the floor's high pass
	vprev         float64 // the output's last tick
	am            float64 // the effects' slow level wobble

	// output
	held, heldpos float64
	z             [2]float64
	lpf           [5]float64 // the low pass's coefficients: b0 b1 b2 a1 a2
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
	sj.volume, sj.speed, sj.pitch, sj.bend, sj.repeat = 96, 114, 88, 5, 0
	sj.fast, sj.slow, sj.stress, sj.relax = false, false, false, false
}

func (sj *SpeakJet) powerOn() {
	gain := sj.Gain
	*sj = SpeakJet{Gain: gain}
	sj.resetControls()
	sj.rnd = 0x12345678
	// the output low pass: two real poles, bilinear, prewarped
	w := 2.0 * SampleRate * math.Tan(math.Pi*lpfFc/SampleRate)
	k := 2.0 * SampleRate
	d := k*k + w/lpfQ*k + w*w
	sj.lpf[0] = w * w / d
	sj.lpf[1] = 2.0 * sj.lpf[0]
	sj.lpf[2] = sj.lpf[0]
	sj.lpf[3] = (2.0*w*w - 2.0*k*k) / d
	sj.lpf[4] = (k*k - w/lpfQ*k + w*w) / d
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
// effects shorten faster (0.63 at 127, 1.55 at 90, 2.6 at 62)
func (sj *SpeakJet) speedFactor(effect bool) float64 {
	if effect {
		return (fxSpeedA - float64(sj.speed)) / fxSpeedB
	}
	return (speedA - float64(sj.speed)) / speedB
}

// Fast and Slow: measured (VoxRecord section 7, both units) as x0.77 and
// x1.22 of the played length on vowels, nasals and fricatives; the stops
// don't change (TT is 45-50 ms plain, fast or slow)
func (sj *SpeakJet) modifiedMs(idx int, ms float64) float64 {
	k := allophones[idx].kind
	if k == kStop || k == kVStop || k == kAffr {
		return ms
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
	if effect {
		sj.attack, sj.release = ticks(fxAttackMs), ticks(fxReleaseMs)
	} else {
		sj.attack, sj.release = ticks(attackMs), ticks(releaseMs)
	}
	sj.cur, sj.tick, sj.stick = 0, 0, 0
	sj.copies = 0
	sj.stotal = 0
	for i := 0; i < sj.nseg; i++ {
		sj.stotal += sj.seg[i].ticks
	}
	sj.sounding = sj.nseg > 0
}

func (sj *SpeakJet) playEffect(code int) {
	first, count := int(fxindex[code-200][0]), int(fxindex[code-200][1])
	bend := sj.bendHz()
	sj.nseg = 0
	for i := 0; i < count; i++ {
		p := &fxparts[first+i]
		var a, b frame
		a.lv = float32(exp(float64(p.lv) * (0.5 * math.Ln10 / 20.0)))
		b.lv = a.lv
		for k := 0; k < 3; k++ {
			a.f[k] = bent(float64(p.fa[k])*hzStep, bend)
			b.f[k] = bent(float64(p.fb[k])*hzStep, bend)
			a.a[k] = float32(p.v[k])
			b.a[k] = a.a[k]
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
		}
	}
	sj.start(true)
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
	var f frame
	for k := 0; k < 3; k++ {
		if second {
			f.f[k] = bent(float64(a.g[k])*hzStep, bend)
		} else {
			f.f[k] = bent(float64(a.f[k])*hzStep, bend)
		}
		f.a[k] = float32(a.a[k])
	}
	f.lv = float32(exp(float64(a.lv) * (0.5 * math.Ln10 / 20.0)))
	f.nf, f.nw, f.na = float32(a.nf)*hzStep, float32(a.nw)*hzStep, float32(a.na)
	return f
}

// the segments of one copy of a Repeat: the sound's body for its own copy
// length (about the manual's length for most sounds, not for the stops), no
// transition (the copies run on seamlessly, measured on ZZ x10 and EYIY x5;
// a stop's copies repeat its burst, BE x5 is 260 ms)
func (sj *SpeakJet) copySegments(idx int) int {
	a := &allophones[idx]
	bend := sj.bendHz()
	tk := ticks((float64(a.cp)*msStep + copyExtraMs) * sj.speedFactor(false))
	tgt := alloFrame(a, false, bend)
	sj.nseg = 0
	if a.kind == kGlide {
		tgt2 := alloFrame(a, true, bend)
		hold, move := tk/5, tk*7/10
		sj.addSegment(hold, tgt, tgt, true)
		sj.addSegment(move, tgt, tgt2, true)
		sj.addSegment(tk-hold-move, tgt2, tgt2, true)
	} else {
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

func (sj *SpeakJet) playAllophone(idx int) {
	a := &allophones[idx]
	bend := sj.bendHz()
	tk := ticks(sj.modifiedMs(idx, float64(a.ms)*msStep) * sj.speedFactor(false))
	tgt := alloFrame(a, false, bend)
	from := sj.fromFrame(tgt)
	sj.nseg = 0
	var silent frame
	silent.lv = 1
	switch a.kind {
	case kVowel, kNasal, kVFric:
		tr := ticks(transitionMs)
		if tr > tk {
			tr = tk
		}
		sj.addSegment(tr, from, tgt, true)
		sj.addSegment(tk-tr, tgt, tgt, true)
	case kGlide:
		// measured on the chip's glides: the first frame holds for a fifth of
		// the sound, the move takes the next seven tenths, the second frame
		// holds the last tenth
		tgt2 := alloFrame(a, true, bend)
		hold, move := tk/5, tk*7/10
		tr := ticks(transitionMs)
		if tr > hold {
			tr = hold
		}
		sj.addSegment(tr, from, tgt, true)
		sj.addSegment(hold-tr, tgt, tgt, true)
		sj.addSegment(move, tgt, tgt2, true)
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
			gone.a[0] = tgt.a[0] // at full level: both units' voiced stops open on a voice bar 2-9 dB under the release (BE, BO, DE, DO, GE, GO)
		}
		sj.addSegment(ticks(closureMs*sj.speedFactor(false)), bar, gone, a.kind == kVStop || bar.a[0]+bar.a[1]+bar.a[2] > 0)
		// the release glides to the stop's second frame: the first formant falls into a labial or alveolar
		// closure (both units: EB, OB, ED, OD); a stop without a measured target holds
		sj.addSegment(tk, tgt, alloFrame(a, true, bend), a.kind == kVStop)
	case kAffr:
		// a stop's closure, then the fricative (voiced for JH)
		sj.addSegment(ticks(closureMs), silent, silent, false)
		sj.addSegment(tk, tgt, tgt, int(a.a[0])+int(a.a[1])+int(a.a[2]) > 0)
	}
	sj.start(false)
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
	for sj.count > 0 && !sj.sounding {
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
			sj.playAllophone(int(code) - 128)
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
			if next >= 0 && sj.nseg > 0 {
				l := &sj.seg[sj.nseg-1]
				both := l.voiced && next <= 199 && allophones[next-128].kind != kFric && allophones[next-128].kind != kStop
				cut := ticks(overlapMs)
				if both {
					cut = ticks(overlapVMs)
				}
				if l.ticks > cut+1 {
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
	sj.head = (sj.head + sj.executing) % BufferSize
	sj.count -= sj.executing
	sj.executing = 0
	sj.sounding = false
}

// --------------------------------------------------------- synthesizer ----

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func (sj *SpeakJet) random() float64 {
	sj.rnd = sj.rnd*1664525 + 1013904223
	return float64(sj.rnd>>8)*(1.0/16777216.0) - 0.5
}

// one chip sample
func (sj *SpeakJet) chipSample() float64 {
	if !sj.sounding {
		sj.nextCommand()
	}
	if !sj.sounding {
		return 0.0
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
	f.nf = float32(lerp(float64(s.from.nf), float64(s.to.nf), t))
	f.nw = float32(lerp(float64(s.from.nw), float64(s.to.nw), t))
	f.na = float32(lerp(float64(s.from.na), float64(s.to.na), t))
	sj.last = f

	v := 0.0

	// voicing
	if s.voiced && sj.pitch > 0 {
		if sj.ptick <= 0 {
			// measured: 92 ticks at Pitch 88 on both units. an effect's pulses
			// run at Pitch plus the part's offset (C1, C2, C3, C6, C7: the same
			// offset at Pitch 60, 88 and 150 on both units, gliding within the part)
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
			sj.period = int(rate/f) + 1
			sj.ptick = sj.period
		}
		// an effect's third tone is a plain tone beside the voice
		nf := 3
		if s.flags&fxVoiced != 0 && s.flags&fxAll == 0 {
			nf = 2
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
			if x < 0.5 {
				env = 1.0 - 2.0*x
			} else {
				env = 2.0*x - 1.0
			}
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
		v -= env * sum / 128.0 * float64(f.lv) // the pulse starts downward (both units, every vowel at Pitch 20)
		sj.venv = env
		if nf == 2 {
			v += float64(f.a[2]) / 128.0 * 0.6 * float64(f.lv) * sin2pi(sj.phase[2])
			sj.phase[2] += float64(f.f[2]) * (1.0 + jitter*2.0*sj.random()) / nominal
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
		v *= 1.0 + sj.am
	}

	// noise: white noise through a two-pole resonance at nf with bandwidth nw
	// (the chip's hiss is random noise - Gaussian to the sample, both units -
	// not a tone; FF, TH and HE are flat to 4 kHz, SH and CH peak near 2.3
	// kHz, WH at 800 Hz, SE and SO at the top of the band). the gain holds
	// the output power at the input's whatever the band (the AR(2) variance)
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
			nf *= exp((2.0*sw - 1.0) * 0.6)
			lvl = exp(-5.0 * sw / 20.0 * math.Ln10)
		}
		nw := float64(f.nw)
		if nw < 10 {
			nw = 10
		}
		r, c := exp(-math.Pi*nw/rate), sin2pi(nf/rate+0.25)
		r2 := r * r
		g := math.Sqrt((1.0 - r2) * ((1.0+r2)*(1.0+r2) - 4.0*r2*c*c) / (1.0 + r2))
		y := g*sj.random() + 2.0*r*c*sj.nz[0] - r2*sj.nz[1]
		sj.nz[1] = sj.nz[0]
		sj.nz[0] = y
		if s.flags&fxAll != 0 {
			lvl *= sj.venv // both units: C0 and M2 are silent between their pulses
		}
		v += lvl * float64(f.lv) * float64(f.na) / 128.0 * 0.86 * y
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
	sj.stick++

	v *= sj.volumeGain()
	// the output stage's step size
	if quant > 0 {
		v = math.Floor(v*quant+0.5) / quant
	}
	// the output noise floor: white under the effects' tones, rising with
	// frequency under the voice (the valley between the first two formants is
	// deeper on the chip than a white floor leaves it)
	r := sj.random()
	if s.voiced && s.flags&(fxVoiced|fxPulsed) == 0 {
		v *= 1.0 + floor*2.0*(r-sj.lastr)
	} else {
		// the effects' bed grows with the tone's frequency (both units): timing
		// error, proportional to the signal's change per tick
		v += slew * 2.0 * r * (v - sj.vprev)
	}
	sj.lastr = r
	sj.vprev = v

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
	b0, b1, b2, a1, a2 := sj.lpf[0], sj.lpf[1], sj.lpf[2], sj.lpf[3], sj.lpf[4]
	sounding := sj.sounding || sj.count > 0
	for i := range buf {
		// the average of the held chip samples over this output sample
		rest, sum := step, 0.0
		for sj.heldpos+rest >= 1.0 {
			sum += sj.held * (1.0 - sj.heldpos)
			rest -= 1.0 - sj.heldpos
			sj.held = sj.chipSample()
			sj.heldpos = 0.0
			sounding = sounding || sj.sounding || sj.count > 0
		}
		sum += sj.held * rest
		sj.heldpos += rest
		x := sum / step
		// direct form II transposed
		y := b0*x + sj.z[0]
		sj.z[0] = b1*x - a1*y + sj.z[1]
		sj.z[1] = b2*x - a2*y
		buf[i] = float32(y) * sj.Gain
	}
	sj.speaking = sounding
	return sounding
}
