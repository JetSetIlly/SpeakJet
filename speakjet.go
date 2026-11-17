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

// C library's SJ_VERSION is version 0.5.4

// the rate the chip makes samples at, measured from the images of its tones on
// two units (8039 and 8013 Hz; the manual says 8192). the chip's own arithmetic
// assumes 8000: its tones come out rate/8000 sharp on both units
const (
	rate        = 8026.0
	fxOutHP     = 40.0 // Hz: 0.5.4, the effects' output high-pass
	lpRise      = 25.0 // ms: the low pulse's rise (PE PO; 0.5.4: 25, by picture)
	nominal     = 8000.0
	nhpFC       = 400.0  // an allophone's noise high pass, Hz
	nsweepHP    = 150.0  // the swept noise's high pass (C8, C9), Hz
	fricRamp    = 3.0    // an unvoiced fricative's attack and release, ms
	thumpMs     = 16.0   // a sibilant's starting burst of low noise: it fades out over this, ms
	fxburstMs   = 5.0    // an effect's starting click: it fades out over this, ms (220 B0: the chip's is ~5 ms)
	thumpHz     = 800.0  // and is low-passed here
	nasalAttack = 25.0   // a nasal's attack, ms (the vowels': attackMs)
	barUp       = 0.12   // a voiced stop's bar: its upper formants at this fraction of the release's
	barNF       = 1000.0 // 0.5.3: an alveolar's bar is partly noise: a band, Hz (both units: DE DO's bar pulses match one another 0.40-0.59 above 700 Hz, ours 0.76-0.81)
	barNW       = 800.0  // its width, Hz
	barNA       = 4.0    // its level (the velars' bar is periodic on the chip)
	lbarNF      = 1800.0 // 0.5.3: BE BO's bar above F1 is noise, not weak formants: a band, Hz (both units: above 700 Hz the part that differs pulse to pulse is within 0-3 dB of the whole)
	lbarNW      = 1400.0 // its width, Hz
	lbarNA      = 6.0    // its level
	barClick    = 0.0    // and the click it opens on, ms
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
	joinA         = 66.9                // a vowel or glide into a vowel or glide: cut joinA - joinK * (F1+F2 ERB distance) ms
	joinK         = 6.9
	nasalFade     = 0.6513   // and the nasal fades into it over this share of the closure
	nasalClosure  = 38.0     // a voiced stop's closure after a nasal, ms (before Speed and Slow)
	nasalRamp     = 10.0     // and its release rises over this, ms
	nasalRampFrom = 0.316228 // from 10 dB down
	xjA           = 5.0      // a vowel or glide into a vowel: its formants move over xjA + xjK * (F1+F2 ERB distance) ms, ...
	xjK           = 5.5
	xjMin         = 20.0   // ... at least this, ...
	xjEnd         = 31.0   // ... ending this far into the vowel (crossJoin)
	kvKE          = 2100.0 // a vowel after KE starts with F2 here (Hz), F1 at kvF1, and moves to its own (the velar locus)
	kvF1          = 300.0
	kvMs          = 8.0                // ... over this many ms per ERB of F2 distance
	kvAdd         = 4.0                // ... and is this many ms per ERB longer
	slew          = 0.078              // the effects' bed, measured: grows with the tone's frequency on both units
	bedPole       = 0.6909588478246612 // up to about 1.5 kHz, then with the tone's level alone (one pole)
	gateDepth     = 0.4                // an effect's gate: its off half's level, a factor (unused since 0.5.0)
	fxSlew        = 0.0468             // 0.5.0: the effects' bed, flatter (2026-11-14)
	fxBedPole     = 0.5                // 0.5.0
	fxDist2       = 0.07               // 0.5.0: the effects' output stage, second order
	glideBed      = 0.0                // 0.5.0: the bed's power with the tones' glide (off)
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
// P2 188 ms, not 200 (2026-11-08, VoxProbe 13 section 9, both units, placement fixed: the second AW of AW P2 AW
// starts 10-20 ms after the chips' at every Speed and Pitch; section 9 12.0 -> 9.4 ms)
var pauseMs = [7]float64{20, 90, 188, 648, 15, 30, 59}

func ticks(ms float64) int { return int(ms*rate/1000.0 + 0.5) }

// one target the synthesizer moves toward: three voiced formants (Hz, level 0
// to 31) and one noise band (centre Hz, throw Hz, level 0 to 31)
// nTone: an effect part's tones (2026-11-09, see the C library's NTONE); the voice keeps three
const nTone = 5

// fxMore: the part has more than three tones, a count byte after its level
const fxMore = 0x8000

type frame struct {
	f, a       [nTone]float32
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
	hzStep   = 16
	msStep   = 5
	fxMsStep = 0.25 // an effect part's length unit, ms (see the C library)
	fxHzStep = 1    // and its frequencies' unit, Hz
)

type fxpart struct {
	ms       uint16        // fxMsStep (2026-11-09: 16-bit, 0.25 ms; was a byte of 5 ms)
	fa, fb   [nTone]uint16 // fxHzStep (16-bit, 1 Hz; was a byte of 16 Hz)
	v        [nTone]uint16 // 1/2048 of a level step (2026-11-09: 16-bit; was a byte of whole steps)
	nf, nw   uint16        // fxHzStep
	nv, gate uint8         // gate in ms
	flags    uint16
	lv       int8          // the part's level, half-decibels (measured against the voice, both units)
	pa, pb   int16         // the part's pulse rate, Hz added to Pitch at its start and end (both units: the same offset at Pitch 60, 88 and 150)
	lr       int8          // the part's level ramp, half-decibels across it (both units: B1's tone falls and rises a steady 1.5 dB per 10 ms through its sweeps, the same up and down where the output stage alone can't)
	ve       [nTone]uint16 // each tone's level at the part's end, 1/2048 steps of 0 to 31 (both units: A7's lowest tone fades in by 27 dB while its highest fades out by 30, inside one sweep; the tones' volumes are registers swept like their frequencies)
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
	og     int8  // signed since 0.5.0: a negative og is a quiet onset
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
	gh, gm uint8 // a glide's first hold and its move, percent of the sound (0: a fifth and seven tenths)
	pa, pt uint8 // a low pulse (PE, PO, both units: a damped swing near 62 Hz, 9-10 % of full scale, 48-53 ms in, decaying
	pf, pd uint8 // over 23-29 ms): its size /1024 of full scale, start ms, frequency Hz, decay ms,
	pp     uint8 // starting phase /256 turn (pa 0: none)
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
	prevSlow                           bool   // the sound before was slowed
	prevCode                           int    // and which it was
	joined                             bool   // the last sound ran straight into the next: no ramps at that join
	afterPause                         bool   // a pause code played since the last sound
	afterVelar                         bool   // the last sound played was GE
	aidx                               int    // 0.5.1: the playing allophone
	afterVstop, afterVelar2            bool   // the last sound played was a voiced stop; GE or GO
	afterKVelar                        bool   // the last sound played was KE
	lastNGE                            bool   // the last sound played was NGE
	lastNasal                          bool   // the last sound played was a nasal, MM NE NO NGE NGO (not the liquids that share kNasal)
	rampSeg                            int    // a voiced stop after a nasal: its release ramp's segment (-1: none)
	xon                                bool   // the next vowel's formants finish a move begun in this sound's tail (crossJoin)
	xpost                              [3]int // each formant's ticks to its target after the join

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
	lpa, lpr        float64 // its low pulse: size, decay per tick
	lpc, lps        float64 // the pulse's phasor
	lpcr, lpsr      float64 // and its rotation per tick
	lpt             int     // the pulse's start, ticks into the sound
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
	// 0.5.5: an effect tone's leap passes through its midpoint: the frequency played, the target, the level, ticks left on
	// the midpoint, the last part voiced
	fsl, ftg, fsa [nTone]float32
	fsn           [nTone]int
	fslon, fsv    bool
	phase         [nTone]float64
	xa            [nTone]float32 // fxSmooth: the tone amplitudes as they ramp
	xv            float32        // fxSmooth: the voice's gain as it ramps in
	vbuzz         float64        // fxBuzz: this tick's band output, kept out of the bed's slope
	neq           int            // the playing effect's peaking filters (fxeq)
	eqc           [3][5]float64  // b0 b1 b2 a1 a2
	eqz           [3][2]float64
	bzbuf         [1024]float64 // fxBuzz: the source's recent past, for the comb
	bzpos, bzclk  int           // and ticks to the next click
	nz            [2]float64    // the noise resonator's two delays
	nz2           [2]float64    // 0.5.4: its second stage (M0)
	fxhpz         float64       // 0.5.4: the effects' output high-pass state
	nhp           [4]float64    // an allophone's noise high pass: input and output of each of its two stages
	venv          float64       // the voicing envelope this tick, for an fxAll effect's noise
	vstarted      bool          // 0.5.0: voicing has run since the last silence
	vcount        int           // 0.5.4: voicing periods since the voice started
	vafter        bool          // 0.5.4: this sound follows a voiced stop or a nasal: no long period
	fhold         [3]float32    // mechanism: where the formants are, held through hisses and pauses (0: not set)
	prepause      frame         // 0.5.4: the last sound's frame before a short pause
	ppok          bool          // 0.5.4: and whether it carries over
	paused        bool          // 0.5.4: a pause since the last sound
	aplain        bool          // 0.5.0: the playing allophone is a plain vowel (IY..UW)
	wander        float64       // an fxWander part's pitch drift, a fraction
	rnd           uint32
	rnj           uint32  // the jitter's, wander's and drift's own sequence (2026-11-11)
	lastr         float64 // the previous noise sample, for the floor's high pass
	vprev         float64 // the output's last tick
	bed           float64 // the output's change per tick, smoothed, for the effects' bed
	vbed          bool    // 0.5.4: a voiced effect part (not all-tones, not smoothed) is playing: the bed doubles
	bedhold       float64 // 0.5.0: the signal's magnitude, peak-held (20 ms): the effects' flat bed follows it
	bedk          float64 // 0.5.0: this tick's bed against the fades (0: 1)
	bscale        float64 // the playing effect's own bed scale (fxs bs; 1: the common)
	wobr, wobd    float64 // the playing effect's wobble: rate (Hz) and depth (fxs wr, wd)
	wobph         float64
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
	output        int         // OutputClean or OutputAtariVox
	ring          [64]float64 // clean output: the last chip samples, for the interpolator
	ringw         int64       // chip samples produced so far
	cpos          float64     // the chip time (in chip samples) of the next clean output sample
	lastsnd       int64       // the newest chip sample at which the chip was sounding (the clean output drains to it)
	lpf           [6]float64  // the low pass: its two poles (per tick), their decays over an output sample, p2 / (p2 - p1), p2 / zero
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
	sj.prevSlow, sj.prevCode = false, -1
	sj.fhold = [3]float32{}
}

func (sj *SpeakJet) powerOn() {
	gain := sj.Gain
	*sj = SpeakJet{Gain: gain}
	sj.resetControls()
	sj.rnd = 0x12345678
	sj.rnj = 0x9E3779B9
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
// The output stage (2026-11-13). OutputClean (the default): the chip's 8026 Hz samples interpolated band-limited to
// SampleRate, then the AtariVox's measured in-band response - no clock images. OutputAtariVox: the samples held as the
// AtariVox holds them (it has no reconstruction filter), so every tone carries its mirror images round 8 kHz, as the real
// thing's speaker does; the output as the library made it before this. Clean adds 3 ms of latency (the interpolator's
// window). Either way the synthesis is the same.
const (
	OutputClean    = 0
	OutputAtariVox = 1
)

// SetOutput selects the output stage: OutputClean or OutputAtariVox.
func (sj *SpeakJet) SetOutput(mode int) {
	if mode == OutputAtariVox {
		sj.output = OutputAtariVox
	} else {
		sj.output = OutputClean
	}
}

// Output returns the output stage in use.
func (sj *SpeakJet) Output() int { return sj.output }

// the clean output's interpolator: a windowed sinc, rsHalf chip samples each side, cut off at rsCut of the chip's Nyquist,
// tabulated at rsPh phases and read with linear interpolation between them (the same table in C and Go)
const (
	rsHalf = 24
	rsPh   = 128
	rsCut  = 0.95
)

var rsTab [rsPh + 1][2 * rsHalf]float64
var rsReady bool

func rsInit() {
	if rsReady {
		return
	}
	for p := 0; p <= rsPh; p++ {
		fr := float64(p) / rsPh
		for k := 0; k < 2*rsHalf; k++ {
			d := float64(k-rsHalf+1) - fr // tap k at chip sample floor(t) - rsHalf + 1 + k, offset from t
			x := math.Pi * rsCut * d
			sinc := 1.0
			if x != 0 {
				sinc = math.Sin(x) / x
			}
			w := 0.42 + 0.5*math.Cos(math.Pi*d/rsHalf) + 0.08*math.Cos(2.0*math.Pi*d/rsHalf) // Blackman over +-rsHalf
			if math.Abs(d) >= rsHalf {
				w = 0
			}
			rsTab[p][k] = rsCut * sinc * w
		}
	}
	rsReady = true
}

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

// Seed starts the noise's random sequence and the tones' oscillator phases
// from the given seed (without it the chip starts from fixed ones): several
// seeds are several plays of the same sound, as the real chip's differ
func (sj *SpeakJet) Seed(seed uint32) {
	// the chip's tones run freely and start wherever their oscillators are (2026-11-11: B0, B1 and B6 start with
	// either sign on different takes of the same unit), and its noise is a long sequence: a seed gives the oscillators
	// random starting phases as well as the noise its own start
	r := seed
	for k := 0; k < nTone; k++ {
		r = r*1664525 + 1013904223
		sj.phase[k] = float64(r>>8) * (1.0 / 16777216.0)
	}
	sj.rnd = r
}

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
// ASSUMPTION (2026-11-08, VoxProbe 13 sections 3 and 9 and Man Goes Down P2 at Speed 118, both units): above Speed
// 114 an unslowed vowel or glide (and its Repeat copy) and a pause shrink by (155.9 - Speed) / 47.4, a step down from
// 114 (118: x0.80, 127: x0.61, as measured; the effects' law gives x0.88 at 118), and the other speech sounds don't
// shrink at all. P2 1700 ms against 1695/1700, timing inside it 12.9 -> 8.5 ms; with the step on the vowels alone the
// consonants ran 15-40 ms short each. Below 114 nothing changes. The step is at 116, not 115 (2026-11-09, VoxProbe
// 14 section 2, both units: at Speed 115 IYUW is 225 ms as at 114, AW P2 AW 435 against 455 at 113; with the step
// at 115 ours read IYUW -30, AW P2 AW -35 ms), so Speed 115 plays as 114
// ASSUMPTION (2026-11-09, VoxProbe 4, 13 and 14's vowel and liquid joins, both units, the 2600 takes): besides the
// distance law, a few sounds carry a join term of their own (alloj), added to the cut of every vowel or glide join
// they are on either side of: IH +15, AW +6, OW -10, RR -16 ms. The law alone left AW IH, IH AW IH and AW IH AW
// 22-45 ms long and AW RR, RR AW 10-13 ms short; with the terms the 26 joins without AY read rms 16.8 -> 5.0 ms, max
// 42 -> 10. Fitted on VoxProbe 4 and 13 alone, the terms took VoxProbe 14 from rms 30.1 to 10.6 ms. AY IY is still
// ~30 ms long: no term on AY or IY fits it with AY EH and IY's other joins (open)
func joinTerm(idx int) float64 {
	for _, j := range alloj {
		if int(j[0]) == idx {
			return float64(j[1]) - 128.0
		}
	}
	return 0.0
}

func (sj *SpeakJet) speedStep() float64 { return (155.9 - float64(sj.speed)) / 47.4 }

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
	// SE SO SH take half of Slow's stretch (2026-11-08, VoxProbe 13 section 10, all four takes: every slowed SE 16-34
	// ms long at x4/3; VoxProbe 12 had Slow SE +20 ms; Man Goes Down P4's Slow SE Slow SE)
	if sj.slow {
		if idx >= 59 && idx <= 61 {
			ms *= 1.0 + (slowFactor-1.0)/2.0
		} else if allophones[idx].kind == kVowel {
			ms *= 1.29 // 0.5.0: a plain vowel stretches x1.29 under Slow (VoxProbe 8 section 2, both units: Slow AW IY 149-150 ms where x4/3 gave 154-157); glides and nasals keep x4/3
		} else {
			ms *= slowFactor
		}
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
	sj.bscale = 1
	sj.wobd = 0
	sj.wobph = 0
	sj.onset = 0
	sj.onsetFrom = 0
	sj.releaseGain = 1.0
	sj.wfloor = 0
	sj.thump = 0
	sj.thlp = 0
	sj.lpa = 0
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
	f.ms = uint16(fxdata[at]) | uint16(fxdata[at+1])<<8
	h := fxdata[at+2]
	f.flags = uint16(fxdata[at+3])
	at += 4
	if h&128 != 0 {
		f.flags |= uint16(fxdata[at]) << 8
		at++
	}
	f.lv = int8(fxdata[at])
	at++
	n := int(h & 3)
	if f.flags&fxMore != 0 {
		n += int(fxdata[at]) // tones past three (gen_tables: the count after the level)
		at++
	}
	if n > nTone {
		n = nTone
	}
	for k := 0; k < n; k++ {
		f.fa[k] = uint16(fxdata[at]) | uint16(fxdata[at+1])<<8
		f.fb[k] = uint16(fxdata[at+2]) | uint16(fxdata[at+3])<<8
		f.v[k] = uint16(fxdata[at+4]) | uint16(fxdata[at+5])<<8
		at += 6
	}
	if h&4 != 0 {
		f.nf = uint16(fxdata[at]) | uint16(fxdata[at+1])<<8
		f.nw = uint16(fxdata[at+2]) | uint16(fxdata[at+3])<<8
		f.nv = fxdata[at+4]
		at += 5
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
			f.ve[k] = uint16(fxdata[at+2*k]) | uint16(fxdata[at+2*k+1])<<8
		}
		at += 2 * n
	}
	return at
}

func (sj *SpeakJet) playEffect(code int) {
	sj.lastNasal = false
	sj.afterPause = false
	sj.afterVelar = false
	sj.afterVstop = false
	sj.afterVelar2 = false
	sj.afterKVelar = false
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
		for k := 0; k < nTone; k++ {
			a.f[k] = bent(float64(p.fa[k])*fxHzStep, bend)
			b.f[k] = bent(float64(p.fb[k])*fxHzStep, bend)
			a.a[k] = float32(p.v[k]) / 2048.0
			b.a[k] = float32(p.ve[k]) / 2048.0
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
		// a tone absent from this part holds the frequency it ended the last part on (2026-11-11, 219 A9: an absent tone
		// was put at 0 Hz and not sounded, so the smooth ramp for a tone switching off at a part boundary never played)
		if sj.nseg > 0 && code < 240 && p.flags&fxSmooth != 0 {
			for k := 0; k < nTone; k++ {
				if p.v[k] == 0 && p.ve[k] == 0 {
					a.f[k] = sj.seg[sj.nseg-1].to.f[k]
					b.f[k] = a.f[k]
				}
			}
		}
		a.nf, b.nf = float32(p.nf)*fxHzStep, float32(p.nf)*fxHzStep
		a.nw, b.nw = float32(p.nw)*fxHzStep, float32(p.nw)*fxHzStep
		a.na, b.na = float32(p.nv), float32(p.nv)
		// the oscillator bank (2026-11-11, 219 A9 over eight random plays): a line carrying on into the next part stays on
		// its oscillator, the rest take oscillators left silent, an idle oscillator holds its last frequency; voiced and
		// pulsed parts, and the part after one, keep their order (see the C library for the measurements)
		if sj.nseg > 0 && code < 240 && p.flags&(fxVoiced|fxPulsed) == 0 && sj.seg[sj.nseg-1].flags&(fxVoiced|fxPulsed) == 0 {
			pv := &sj.seg[sj.nseg-1].to
			var slot [nTone]int
			var used [nTone]bool
			for j := range slot {
				slot[j] = -1
			}
			for j := 0; j < nTone; j++ {
				if !(pv.a[j] > 0 && pv.f[j] > 0) {
					continue
				}
				best, bd := -1, math.Log(1.15)
				for k := 0; k < nTone; k++ {
					if used[k] || !(a.a[k] > 0 || b.a[k] > 0) || !(a.f[k] > 0) {
						continue
					}
					if d := math.Abs(math.Log(float64(a.f[k]) / float64(pv.f[j]))); d < bd {
						bd, best = d, k
					}
				}
				if best >= 0 {
					slot[j], used[best] = best, true
				}
			}
			for k := 0; k < nTone; k++ {
				if used[k] || !(a.a[k] > 0 || b.a[k] > 0) {
					continue
				}
				j := -1
				// 0.5.5: an oscillator whose tone just stopped first, then an idle one (226 B6: the old tone's oscillator carries
				// the new one at a part change; B6 1.063 -> 1.046, B5 0.877 -> 0.860, no effect worse)
				for i := 0; i < nTone && j < 0; i++ {
					if slot[i] < 0 && pv.a[i] > 0 && pv.f[i] > 0 {
						j = i
					}
				}
				for i := 0; i < nTone && j < 0; i++ {
					if slot[i] < 0 {
						j = i
					}
				}
				if j >= 0 {
					slot[j], used[k] = k, true
				}
			}
			na, nb := a, b
			for j := 0; j < nTone; j++ {
				if slot[j] >= 0 {
					na.f[j], na.a[j], nb.f[j], nb.a[j] = a.f[slot[j]], a.a[slot[j]], b.f[slot[j]], b.a[slot[j]]
				} else {
					na.f[j], nb.f[j] = pv.f[j], pv.f[j]
					na.a[j], nb.a[j] = 0, 0
				}
			}
			a, b = na, nb
		}
		n := sj.nseg
		sj.addSegment(ticks(float64(p.ms)*fxMsStep*sj.speedFactor(true)), a, b, p.flags&(fxVoiced|fxPulsed) != 0)
		if sj.nseg > n {
			sj.seg[n].gate = ticks(float64(p.gate))
			sj.seg[n].flags = p.flags
			sj.seg[n].pa, sj.seg[n].pb = float32(p.pa), float32(p.pb)
			sj.seg[n].lr = float32(p.lr)
		}
	}
	sj.start(true)
	sj.attack = 0 // 0.5.0: effects start hard
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
	// its own bed scale (bs, percent; 0: the common): see the C library's play_effect
	sj.bscale = 1
	if x[6] > 0 {
		sj.bscale = float64(x[6]) / 100.0
	}
	// its wobble (wr Hz, wd /256; 0: none): see the C library's play_effect
	sj.wobr = float64(x[7])
	sj.wobd = float64(x[8]) / 256.0
	sj.wobph = 0
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
func (sj *SpeakJet) copySegments(idx int, slowed bool) int {
	a := &allophones[idx]
	bend := sj.bendHz()
	cf := sj.speedFactor(false)
	if (a.kind == kVowel || a.kind == kGlide) && sj.speed > 115 && !sj.slow {
		cf = sj.speedStep()
	}
	if slowed { // a slowed vowel joined after itself (see play's again)
		cf *= 1.29
	}
	tk := ticks((float64(a.cp)*msStep + copyExtraMs) * cf)
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
		// per glide where measured (gh, gm; 2026-11-09, 158 IYEH, both units)
		if a.gh > 0 {
			hold = tk * int(a.gh) / 100
		}
		if a.gm > 0 {
			move = tk * int(a.gm) / 100
		}
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
	// after a hiss (noise only) a sound starts as from silence: its formants and noise band don't sweep from the
	// hiss's (2026-11-08, VoxProbe 13 SE TT AW and AW Slow SE Slow SE UX, both units: the hiss stops and the next sound
	// starts on its own frame; ours swept a dark line from ~6 kHz down into TT's closure and UX's onset)
	if sj.last.a[0] > 0 || sj.last.a[1] > 0 || sj.last.a[2] > 0 {
		return sj.last
	}
	for q := 0; q < 3; q++ {
		if sj.fhold[q] > 0 {
			tgt.f[q] = sj.fhold[q]
		}
	}
	tgt.a = [nTone]float32{}
	tgt.na = 0
	return tgt
}

func lerpFrame(a, b frame, t float32) frame {
	for q := 0; q < nTone; q++ {
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

// splitAt: split the sound's segments at tick at from its start (no split at an existing edge)
func (sj *SpeakJet) splitAt(at int) {
	acc, k := 0, 0
	for k < sj.nseg && acc+sj.seg[k].ticks <= at {
		acc += sj.seg[k].ticks
		k++
	}
	if k >= sj.nseg || acc == at || sj.nseg >= maxSegments {
		return
	}
	sg := sj.seg[k]
	t1 := at - acc
	mid := lerpFrame(sg.from, sg.to, float32(t1)/float32(sg.ticks))
	copy(sj.seg[k+2:sj.nseg+1], sj.seg[k+1:sj.nseg])
	sj.seg[k].ticks = t1
	sj.seg[k].to = mid
	sj.seg[k+1] = sg
	sj.seg[k+1].ticks = sg.ticks - t1
	sj.seg[k+1].from = mid
	sj.nseg++
}

// crossJoin: ASSUMPTION (2026-11-08, VoxProbe 13 section 5 and VoxProbe 4 section 5, both units): a vowel, glide or
// liquid into a vowel, glide or liquid moves its formants on one line across the join, over xjA + xjK * d ms (d the
// F1+F2 ERB distance, at least xjMin), ending xjEnd ms into the next sound. See the C library's cross_join
func (sj *SpeakJet) crossJoin(tgt frame) {
	total := 0
	for k := 0; k < sj.nseg; k++ {
		total += sj.seg[k].ticks
	}
	a := sj.seg[sj.nseg-1].to
	sf := sj.speedFactor(false)
	d := 0.0
	for q := 0; q < 2; q++ {
		ea := 21.4 * log(1.0+0.00437*float64(a.f[q])) / math.Ln10
		eb := 21.4 * log(1.0+0.00437*float64(tgt.f[q])) / math.Ln10
		if ea > eb {
			d += ea - eb
		} else {
			d += eb - ea
		}
	}
	ln := xjA + xjK*d
	if ln < xjMin {
		ln = xjMin
	}
	var s, e [3]int
	for q := 0; q < 3; q++ {
		s[q] = ticks((xjEnd - ln) * sf)
		e[q] = ticks(xjEnd * sf)
		if -s[q] > total*3/4 {
			s[q] = -(total * 3 / 4)
			if e[q] <= s[q] {
				e[q] = s[q] + 1
			}
		}
		if s[q] < 0 {
			sj.splitAt(total + s[q])
			if e[q] < 0 {
				sj.splitAt(total + e[q])
			}
		}
	}
	var v0 [3]float32
	for q := 0; q < 3; q++ {
		v0[q] = a.f[q]
	}
	el := 0
	for k := 0; k < sj.nseg; k++ {
		g := &sj.seg[k]
		for q := 0; q < 3; q++ {
			if s[q] >= 0 {
				continue
			}
			t0, t1 := el-total, el+g.ticks-total
			if t1 <= s[q] {
				continue
			}
			f0 := v0[q]
			if t0 > s[q] {
				f0 = a.f[q] + (tgt.f[q]-a.f[q])*float32(t0-s[q])/float32(e[q]-s[q])
			}
			te := t1
			if te > e[q] {
				te = e[q]
			}
			f1 := a.f[q] + (tgt.f[q]-a.f[q])*float32(te-s[q])/float32(e[q]-s[q])
			if t0 >= e[q] {
				g.from.f[q] = tgt.f[q]
			} else {
				g.from.f[q] = f0
			}
			g.to.f[q] = f1
		}
		el += g.ticks
	}
	for q := 0; q < 3; q++ {
		sj.xpost[q] = 0
		if e[q] > 0 {
			sj.xpost[q] = e[q]
		}
	}
	sj.xon = true
}

// crossSegments: the next sound's side of crossJoin: each formant reaches the target xpost ticks in, the rest of the
// frame at tr, all of it within n ticks (the sound's length, or a glide's first hold)
func (sj *SpeakJet) crossSegments(from, tgt frame, tr, n int, xpost [3]int) {
	bp := [4]int{tr, xpost[0], xpost[1], xpost[2]}
	nb, prev := 0, 0
	for i := 0; i < 4; i++ {
		if bp[i] > 0 && bp[i] < n {
			bp[nb] = bp[i]
			nb++
		}
	}
	for i := 1; i < nb; i++ {
		for j := i; j > 0 && bp[j] < bp[j-1]; j-- {
			bp[j], bp[j-1] = bp[j-1], bp[j]
		}
	}
	fp := from
	for i := 0; i <= nb; i++ {
		t := n
		if i < nb {
			t = bp[i]
		}
		if t <= prev {
			continue
		}
		fr := float32(1)
		if tr > 0 {
			tt := t
			if tt > tr {
				tt = tr
			}
			fr = float32(tt) / float32(tr)
		}
		fn := lerpFrame(from, tgt, fr)
		for q := 0; q < 3; q++ {
			if xpost[q] > 0 {
				tt := t
				if tt > xpost[q] {
					tt = xpost[q]
				}
				fn.f[q] = from.f[q] + (tgt.f[q]-from.f[q])*float32(tt)/float32(xpost[q])
			} else {
				fn.f[q] = tgt.f[q]
			}
		}
		sj.addSegment(t-prev, fp, fn, true)
		fp = fn
		prev = t
	}
}

func (sj *SpeakJet) playAllophone(idx int) {
	sj.aplain = idx <= 11 // 0.5.0
	sj.aidx = idx
	a := &allophones[idx]
	bend := sj.bendHz()
	// vowels and glides take the effects' Speed law above 114 too (2026-11-08, VoxProbe 13 section 3, all four takes:
	// AW AY EYIY IYUW OWWW UX alone at Speed 118, 122, 127 10-30 ms long on the speech law, within a few ms on this one;
	// Man Goes Down P2 at Speed 118 1698 ms against 1695/1700 with the liquid joins below)
	sf := sj.speedFactor(false)
	// 0.5.5: a nasal above Speed 118 takes the vowels' Speed step (VoxProbe 16, both units: Speed 127 MM AW's handover 38-56 ms
	// where ours came at 80-102)
	if a.kind == kVowel || a.kind == kGlide || (a.kind == kNasal && sj.speed > 118) {
		sf = sj.speedFactor(true)
		if sj.speed > 115 && !sj.slow {
			sf = sj.speedStep()
		}
	} else if sj.speed > 114 {
		sf = 1.0
	}
	tk := ticks(sj.modifiedMs(idx, float64(a.ms)*msStep) * sf)
	// ASSUMPTION (2026-11-15, VoxProbe 7's 'Pitch 150 then AW/IY/OW', both units: 133-134 ms where the library gave 113-117, the same
	// sounds 114-123 at Pitch 88): a vowel or glide above Pitch 88 runs longer, by 16 % at 150, taken as linear in Pitch; one Pitch measured
	if (a.kind == kVowel || a.kind == kGlide) && sj.pitch > 88 {
		tk = int(float64(tk)*(1.0+0.16*float64(int(sj.pitch)-88)/62.0) + 0.5)
	}
	tgt := alloFrame(a, false, bend)
	from := sj.fromFrame(tgt)
	if sj.afterVstop && !sj.afterVelar2 && (a.kind == kVowel || a.kind == kGlide) { // 0.5.0: after a voiced stop (not GE GO) the vowel is at its own level and amplitudes from the start; its formants still glide from the stop's release
		from.lv = tgt.lv
		from.a = tgt.a
	}
	xon, xpost := sj.xon, sj.xpost
	sj.xon = false
	sj.nseg = 0
	sj.rampSeg = -1
	var silent frame
	silent.lv = 1
	switch a.kind {
	case kVowel, kNasal, kVFric:
		// 0.5.4: a true nasal (MM NE NO NGE NGO) after a sound takes 5 ms to move, not 25 (VoxProbe 4, both units: AW MM and
		// IY MM, the vowel's F1 gone from the nasal at once on the chip; ours carried AW's ~700 Hz 10-29 dB into MM)
		tr := ticks(transitionMs)
		if idx >= 12 && idx <= 16 && from.a[0]+from.a[1]+from.a[2]+from.na > 0 {
			tr = ticks(5.0)
		}
		// 0.5.0: a voiced fricative after a pause opens with 40 ms of its noise alone before the voicing (Man Goes Down P2's
		// EYIY P4 DH, both units: ~40 ms of faint frication before the voicing, 10 alone)
		if a.kind == kVFric && sj.afterPause {
			nz := tgt
			nz.a[0], nz.a[1], nz.a[2] = 0, 0, 0
			sj.addSegment(ticks(40.0), nz, nz, false)
			lead := ticks(40.0)
			if lead > tk/2 {
				lead = tk / 2
			}
			tk -= lead
		}
		// ASSUMPTION (2026-11-08, VoxProbe 13 section 7, all four takes: after GE's burst F2 glides from ~2000 Hz to the
		// vowel's over ~60 ms before OW; GE OW, GE UW 39-49 ms short, GE IY EH AY 15-20): a vowel after GE moves from
		// GE's frame over 8 ms more per ERB of F2 distance (10 put GE OW's end 10 ms late on the picture), and is that much longer
		if sj.afterVelar && a.kind == kVowel {
			ea := 21.4 * log(1.0+0.00437*float64(from.f[1])) / math.Ln10
			eb := 21.4 * log(1.0+0.00437*float64(tgt.f[1])) / math.Ln10
			d := ea - eb
			if d < 0 {
				d = -d
			}
			ex := ticks(8.0 * d * sj.speedFactor(true))
			tr += ex
			tk += ex
		}
		// mechanism 2: a vowel's move takes 5 ms per ERB of F1/F2 distance (never under the common transition), stretched by
		// Speed and Slow - it glides at a rate, not over a fixed time (replaces 0.5.5's slowed vowel after a nasal)
		if a.kind == kVowel {
			de := 0.0
			for q := 0; q < 2; q++ {
				d := math.Abs(21.4*log(1.0+0.00437*float64(from.f[q]))/math.Ln10 - 21.4*log(1.0+0.00437*float64(tgt.f[q]))/math.Ln10)
				if d > de {
					de = d
				}
			}
			ms := transitionMs
			if 5.0*de > ms {
				ms = 5.0 * de
			}
			sl := 1.0
			if sj.slow {
				sl = slowFactor
			}
			tr = ticks(ms * sj.speedFactor(false) * sl)
		}
		if tr > tk {
			tr = tk
		}
		// a sound with an onset, from silence, starts on its own frame (its onset at full level)
		if a.ob > 0 && from.a[0]+from.a[1]+from.a[2]+from.na == 0 {
			tr = 0
		}
		if xon && (a.kind == kVowel || (idx >= 17 && idx <= 20)) {
			sj.crossSegments(from, tgt, tr, tk, xpost)
			break
		}
		// ASSUMPTION (2026-11-09, VoxProbe 14 section 7, both units by picture: after KE the vowel's F2 starts
		// ~2000-2500 Hz and falls to AW OW UW's over 60-80 ms, after KO it starts ~1000 and rises to IY's over ~50; KE IY
		// and KO OW, where the locus is the vowel's, have no glide; ours started on the vowel and KE AW OW UW ran 32-50 ms
		// short, KO IY 20): KE, which has no formants of its own, hands the vowel a locus to start from; the glide takes
		// ~8 ms per ERB of F2 distance (the pictures' ~60 ms into AW) but the vowel grows only half that. Fitted on the ten
		// KE/KO items: KE AW -30 -> -5, KE OW -45 -> -15, KE UW -30 -> -5 ms, KE IY EH unchanged. KO is left alone: no one
		// locus fits both KO IY (-20) and KO EH (+5), which went to +15 and +20 (open)
		if sj.afterKVelar && a.kind == kVowel {
			lf := tgt
			lf.f[0] = bent(kvF1, bend)
			lf.f[1] = bent(kvKE, bend)
			ea := 21.4 * log(1.0+0.00437*float64(lf.f[1])) / math.Ln10
			eb := 21.4 * log(1.0+0.00437*float64(tgt.f[1])) / math.Ln10
			de := ea - eb
			if de < 0 {
				de = -de
			}
			gl := ticks(kvMs * de * sj.speedFactor(true))
			tk += ticks(kvAdd * de * sj.speedFactor(true))
			if gl > tk-tr-1 {
				gl = tk - tr - 1
			}
			if gl > 0 {
				sj.addSegment(tr, from, lf, true)
				sj.addSegment(gl, lf, tgt, true)
				sj.addSegment(tk-tr-gl, tgt, tgt, true)
				break
			}
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
		// per glide where measured (gh, gm; 2026-11-09, 158 IYEH, both units)
		if a.gh > 0 {
			hold = tk * int(a.gh) / 100
		}
		if a.gm > 0 {
			move = tk * int(a.gm) / 100
		}
		tr := ticks(transitionMs)
		if tr > hold {
			tr = hold
		}
		// with its own F1 (ba) the hold starts on the first allophone's and rises to it (161 AXUW)
		if xon {
			// a crossing join into a glide (crossJoin) finishes within its first hold
			var xp [3]int
			for q := 0; q < 3; q++ {
				xp[q] = xpost[q]
				if xp[q] > hold {
					xp[q] = hold
				}
			}
			sj.crossSegments(from, tgt, tr, hold, xp)
		} else {
			sj.addSegment(tr, from, tgt0, true)
			sj.addSegment(hold-tr, tgt0, tgt, true)
		}
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
			m.f[2] = tgt.f[2] // 0.5.3: F3 holds with F1 (161 AXUW, both units: F3 ~2350 Hz until the late fall; 1.21 -> 1.18)
			sj.addSegment(k1, tgt, m, true)
			sj.addSegment(move-k1, m, tgt2, true)
		} else {
			sj.addSegment(move, tgt, tgt2, true)
		}
		sj.addSegment(tk-hold-move, tgt2, tgt2, true)
	case kFric:
		// ASSUMPTION (2026-11-08, VoxProbe 13 section 10, all four takes: TH between a vowel and a sound ~75 ms on the
		// chip, ~40 here; AW TH AW, IHWW TH RR 25-39 ms short): TH after a voiced sound and before another plays 25 ms
		// longer. Not before a pause (Man Goes Down P2's Slow TH P4: timing 12.7 -> 16.0 ms with it)
		if idx == 62 && from.a[0]+from.a[1]+from.a[2]+from.na > 0 && sj.nextSound(sj.executing) >= 128 {
			tk += ticks(25.0 * sj.speedFactor(false))
		}
		sj.addSegment(tk, tgt, tgt, false)
	case kStop, kVStop:
		// 0.5.4: TT before a vowel, pauses and controls between allowed, is 3 dB louder (both units: VoxProbe 4 and 8, TT AW
		// TT P4 AW Slow TT AW, the burst at -24..-29 dB against the vowel where ours was -30..-34; TT alone unchanged)
		ttG := float32(1.0)
		if idx == 63 {
			i, c := sj.executing, -1
			for i < sj.count {
				c = int(sj.peek(i))
				if c >= 128 {
					break
				}
				if c <= 6 || c == 7 || c == 8 {
					i++
					c = -1
					continue
				}
				if c == 20 || c == 21 || c == 22 || c == 23 || c == 26 {
					i += 2
					c = -1
					continue
				}
				break
			}
			if c >= 128 && c <= 199 && (allophones[c-128].kind == kVowel || allophones[c-128].kind == kGlide) {
				ttG = 1.4125375 // +3 dB
			}
		}
		tgt := tgt
		tgt.lv *= ttG
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
			if idx >= 46 && idx <= 49 { // DE DO ED OD
				gone.nf = barNF
				gone.nw = barNW
				gone.na = barNA
			}
			if idx == 42 || idx == 43 { // BE BO: noise in place of F2 F3
				gone.a[1], gone.a[2] = 0, 0
				gone.nf = lbarNF
				gone.nw = lbarNW
				gone.na = lbarNA
			}
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
		// Not after a pause code (2026-11-08, Man Goes Down P2's TT P1 BE: BE 45 ms early, P2 timing 14.1 -> 12.7 ms)
		if a.kind == kVStop && from.a[0]+from.a[1]+from.a[2]+from.na == 0 && sj.nextSound(sj.executing) >= 128 && !sj.slowBeforeNext(sj.executing) && !sj.slow && !sj.afterPause && cl > 30.0 {
			cl = 30.0
		}
		// after a sound the chip closes for at least joinClosure (Man Goes Down's NE GO, both units)
		if from.a[0]+from.a[1]+from.a[2]+from.na > 0 && cl < joinClosure {
			cl = joinClosure
		}
		// ASSUMPTION (2026-11-08, VoxProbe 13 section 6, all four takes): after a sound DE DO close 25 ms longer and GE
		// GO 15 ms (AW EH LO MM NE NO NGE OW RR before DO were 15-59 ms short, before GO 4-58, before BO 0-11)
		alveolar := a == &allophones[46] || a == &allophones[47]
		velar := a == &allophones[50] || a == &allophones[51]
		if a.kind == kVStop && from.a[0]+from.a[1]+from.a[2]+from.na > 0 {
			if a == &allophones[47] { // 0.5.0: DO and GO only - the VoxProbe 13 measurements were before DO and GO; DE after EH (VoxProbe 4, both units) closes for 45 ms, its own 40
				cl += 25.0
			} else if a == &allophones[51] {
				cl += 15.0
			}
		}
		// ASSUMPTION (2026-11-08): an unvoiced stop straight after a pause code closes 30 ms longer, in silence, so it
		// can't be seen on a stop played alone (VoxProbe 13 UX P1 EK 22 ms short; Man Goes Down P4's UX P1 EK: P4 timing
		// 9.7 -> 4.9 ms, length 910 -> 940 against 942/945)
		if a.kind == kStop && sj.afterPause {
			cl += 30.0
		}
		// ASSUMPTION (2026-11-08, VoxProbe 13 NE KO OW, both units by picture: NE stops at 0.105 s and KO bursts at 0.15,
		// 30 ms after ours; NE TT and NE PO already land): KE and KO after a nasal close longer. 15 ms, not 30
		// (2026-11-09, the closure's quiet run against both units' 2600 takes: with 30 it was 11-30 ms long on VoxProbe
		// 14's NE NO MM NGE KO AW and 15 on VoxProbe 13's NE KO OW itself)
		if (a == &allophones[66] || a == &allophones[67]) && sj.lastNasal && from.a[0]+from.a[1]+from.a[2]+from.na > 0 {
			cl += 15.0
		}
		// ASSUMPTION (2026-11-08): a slowed unvoiced stop after a sound closes 20 ms longer (VoxProbe 13 section 10, all
		// four takes: AW Slow TT AW and Slow AYWW Slow TT 60 ms short; Man Goes Down P2's UW Slow TT: P2 timing 17.0 ->
		// 12.3)
		// only before a pause (2026-11-08): before a sound the vowel ahead runs on instead (see the join). That is TT's
		// way; a slowed KE KO PE PO closes longer before a sound too (2026-11-09, VoxProbe 14 section 4, both units: AW
		// Slow KO AW's closure 56 ms against 44 unslowed, AW Slow PO AW's 101 against 85, where the AW ahead doesn't run on)
		if a.kind == kStop && sj.slow && from.a[0]+from.a[1]+from.a[2]+from.na > 0 && (sj.nextSound(sj.executing) < 128 || idx != 63) {
			cl += 20.0
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
		// ASSUMPTION (2026-11-08, VoxProbe 13 section 6; Man Goes Down P1 1488 -> 1502 against 1495/1502): after a
		// nasal DE DO GE GO close 10 ms longer than BE BO; not under Slow (Man Goes Down P1's Slow GO: timing 18.3 ->
		// 12.3 ms without it)
		if nasalJoin && (alveolar || velar) && !sj.slow {
			cl += 10.0
		}
		// ASSUMPTION (2026-11-08, VoxProbe 13 section 6, all four takes: NGE before BO DO GO 38-48 ms short): after NGE
		// the closure is 40 ms longer
		if nasalJoin && sj.lastNGE {
			cl += 40.0
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
		} else if a.kind == kStop && bar.a[0]+bar.a[1]+bar.a[2]+bar.na > 0 && closure > ticks(10.0)+1 {
			// an unvoiced stop's closure is silent after a 10 ms fade (2026-11-08, VoxProbe 13, both units: every vowel,
			// nasal or hiss before TT KO PO stops and leaves a 20-40 ms gap before the burst; ours faded across the whole
			// closure and never went quiet, so the runs of sound didn't match the chips' on 11 items)
			sj.addSegment(ticks(10.0), bar, gone, bar.a[0]+bar.a[1]+bar.a[2] > 0)
			sj.addSegment(closure-ticks(10.0), gone, gone, false)
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
			t2 := alloFrame(a, true, bend)
			t2.lv *= ttG
			sj.addSegment(tk, tgt, t2, a.kind == kVStop)
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
	if idx == 59 || idx == 63 { // SE, TT: the alveolar locus
		sj.fhold[0], sj.fhold[1] = 400.0, 1600.0
	}
	sj.lastNGE = idx == 15
	sj.afterVelar = idx == 50
	sj.afterVstop = a.kind == kVStop
	sj.afterVelar2 = idx == 50 || idx == 51
	sj.afterKVelar = idx == 66
	wasPause := sj.afterPause
	sj.afterPause = false
	// 0.5.4: a vowel straight after a voiced stop or a nasal has no long second period
	sj.vafter = !wasPause && sj.prevCode >= 128 && (allophones[sj.prevCode-128].kind == kVStop || (sj.prevCode >= 140 && sj.prevCode <= 144))
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
	// 0.5.4: AW starts at once before an unvoiced sound or LE (both units, VoxProbe 4: AW TT and AW LE 7-18 dB under the
	// chip at 400-800 Hz over AW's first 20-40 ms with its 45 ms fade-in; before a vowel, RR or MM the fade-in fits)
	nx := sj.nextSound(sj.executing)
	awFast := sj.aidx == 8 && nx >= 128 && nx <= 199 && (allophones[nx-128].kind == kFric || allophones[nx-128].kind == kStop || allophones[nx-128].kind == kAffr || nx == 145)
	if awFast {
		sj.attack = ticks(20.0) // 0.5.4: a 20 ms fade-in (AW LE 17-31 %, AW SE 16 % closer)
	}
	if a.aa > 0 && !awFast {
		f := 1.0
		if sj.pitch > 88 {
			f = 88.0 / float64(sj.pitch)
		}
		sj.attack = ticks(float64(a.aa) * f) // a sound's own fade-in: aa ms at Pitch 88, shortened above it in pitch periods, never lengthened below (0.5.0)
	}
	// 0.5.5: a vowel after a short pause (P4, P5) fades in over 20 ms at most (both units, NE P4 AW and AW P4 AW: the AW
	// 8-12 dB under the chip over its first 30 ms; phrase items 249.4 -> 247.5, none worse)
	if sj.ppok && a.kind == kVowel && sj.attack > ticks(20.0) {
		sj.attack = ticks(20.0)
	}
	// 0.5.4: a slowed nasal comes in almost at once (Slow NE AW, both units)
	if sj.slow && sj.aidx >= 12 && sj.aidx <= 16 {
		sj.attack = int(float64(sj.attack) * 0.15)
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
	// no starting burst straight after a hiss (2026-11-08, VoxProbe 13 AW SE SE and AW Slow SE Slow SE UX, both units:
	// the second SE has no low column; ours had a dark blob under 1 kHz there, Man Goes Down P4's too)
	sj.thump = float64(a.th) / 64.0
	if sj.last.na > 0 && sj.last.a[0]+sj.last.a[1]+sj.last.a[2] == 0 {
		sj.thump = 0
	} // /64 since 2026-11-07 (was /256): SE's thump was pinned at the byte's top; values divided by four
	tcHz := thumpHz
	if a.tc > 0 {
		tcHz = float64(a.tc) * 32.0
	}
	sj.thk = 1.0 - math.Exp(-2.0*math.Pi*tcHz/8000.0)
	// a low pulse (2026-11-11, the takes' slow part from 40 Hz: PE PO a damped swing near 62 Hz, the loudest thing in them,
	// ours 22 dB short): a decaying phasor, turned and shrunk once a tick from its start
	sj.lpa = float64(a.pa) / 1024.0
	if a.pa > 0 {
		pf, pd := 62.0, 25.0
		if a.pf > 0 {
			pf = float64(a.pf)
		}
		if a.pd > 0 {
			pd = float64(a.pd)
		}
		w, ph := 2.0*math.Pi*pf/rate, 2.0*math.Pi*float64(a.pp)/256.0
		sj.lpt = ticks(float64(a.pt))
		sj.lpr = math.Exp(-1000.0 / (pd * rate))
		sj.lpcr = math.Cos(w)
		sj.lpsr = math.Sin(w)
		sj.lpc = math.Cos(ph)
		sj.lps = math.Sin(ph)
	}
	if a.rr > 0 {
		sj.release = ticks(float64(a.rr)) // a sound's own fade-out (SE, SO: the chip fades over most of the sound)
	}
	// 0.5.4: AW's own fade-out is 35 ms; after a pause, a slowed sound, LE or AYWW the chip ends it over 25
	if sj.aidx == 8 && (wasPause || sj.prevSlow || sj.prevCode == 145 || sj.prevCode == 163) {
		sj.release = ticks(25.0)
	}
	// 0.5.4: AW straight before a short pause (P4-P6) ends over 10 ms (AW P4 AW, both units)
	if c0 := sj.peek(sj.executing); sj.aidx == 8 && c0 >= 4 && c0 <= 6 {
		sj.release = ticks(10.0)
	}
	// 0.5.4: IY straight after another sound runs 5 ms shorter than from silence (solo 130 ms, in phrases 125)
	if sj.aidx == 0 && !wasPause && sj.prevCode >= 128 && sj.nseg > 0 && sj.seg[sj.nseg-1].ticks > ticks(5.0) {
		sj.seg[sj.nseg-1].ticks -= ticks(5.0)
		sj.stotal -= ticks(5.0)
	}
	// 0.5.5: a vowel straight after MM runs 10 ms shorter (both units, MM AW: AW 110 ms on the chip, ours 125; phrase
	// items 258.1 -> 251.4, none worse; after NE any cut was worse, so MM only)
	if a.kind == kVowel && !wasPause && sj.prevCode == 140 && sj.nseg > 0 && sj.seg[sj.nseg-1].ticks > ticks(10.0)+2 {
		sj.seg[sj.nseg-1].ticks -= ticks(10.0)
		sj.stotal -= ticks(10.0)
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
		sj.stotal = total + sj.copies*sj.copySegments(idx, false)
		sj.seg = saved
		sj.nseg = nseg
	}
}

func (sj *SpeakJet) playPause(ms float64) {
	// 0.5.4: across a short pause (under 70 ms) the next sound starts from the last one's formants (NE P4 AW, both units)
	sj.paused = true
	sj.ppok = false
	if ms >= 70.0 {
		sj.fhold = [3]float32{}
	}
	if ms < 70.0 && sj.last.a[0]+sj.last.a[1]+sj.last.a[2] > 0 {
		sj.prepause = sj.last
		sj.ppok = true
	}
	sj.nseg = 0
	var silent frame
	silent.lv = 1
	sj.addSegment(ticks(ms), silent, silent, false)
	sj.start(false)
	sj.afterPause = true
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
			pf := (fxSpeedA - float64(sj.speed)) / fxSpeedB
			if sj.speed > 115 {
				pf = sj.speedStep()
			}
			pms := pauseMs[code] * pf
			// 0.5.0: after a short pause (P4-P6), a voiced stop's bar opens 12 ms into the pause (Man Goes Down P1's ZZ P4 DO,
			// both units: the bar 11 ms after ZZ's end, ours 23)
			if nx := sj.nextSound(n); pms <= 20.0 && nx >= 128 && nx <= 199 && allophones[nx-128].kind == kVStop {
				if pms > 12.0 {
					pms -= 12.0
				} else {
					pms = 0
				}
			}
			sj.playPause(pms)
			sj.joined = false
		case code >= 128 && code <= 199:
			// a sound played straight after the same sound (joined, controls between allowed) plays as a Repeat copy
			// of it (VoxProbe 12, 2026-11-05, both units, 2600 and Stella: AY AY 208 ms, AW AW AW 255, UX UX UX 260 -
			// what Repeat 2 / 3 makes, 214 / 268 / 259 - not two whole sounds, 272 / 316 / 329; a Pitch or Speed between
			// the copies doesn't change it: AW Pitch 88 AW Pitch 88 AW 256)
			// a Slow on the second breaks it, two whole sounds (2026-11-08, VoxProbe 13 section 10, all four takes: AW
			// Slow SE Slow SE UX 464 ms, 435 as a copy), except on a vowel: that stays a copy, stretched as a slowed vowel
			// (2026-11-16, VoxProbe 15 section 4, all four takes: AW AW Pitch n Slow AW AW 20-35 ms over AW AW Pitch n AW
			// AW; as two whole sounds ours read +105)
			again := sj.joined && sj.copyOf == int(code)-128 && sj.repeat <= 1 && (!sj.slow || allophones[int(code)-128].kind == kVowel)
			prevNasal := sj.lastNasal
			prevKindVowel := sj.last.a[0] > 0 && !sj.lastNasal
			sj.playAllophone(int(code) - 128)
			if again {
				sj.stotal = sj.copySegments(int(code)-128, sj.slow)
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
			// 0.5.5: no run-on for a slowed glide before a Slow (0.4.x added 25 ms for Man Goes Down P4; with 0.5.4's rules
			// ours ran 15 ms long there and in VoxProbe 13's Slow IYUW Slow SE, both units: phrase items 264.0 -> 258.1, none worse)
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
				// 0.5.5: a slowed vowel into a nasal moves F1 to the nasal's over its last 60 ms (P1's Slow AY NE, both units)
				if next >= 140 && next <= 144 && sj.slow && allophones[code-128].kind == kVowel && sj.stotal > ticks(60.0)+2 {
					sj.splitAt(sj.stotal - ticks(60.0))
					l = &sj.seg[sj.nseg-1]
					l.to.f[0] = float32(bent(float64(allophones[next-128].f[0])*hzStep, sj.bendHz()))
				}
				// 0.5.4: a vowel or glide with F2 under 1500 Hz ends 15 ms sooner into a nasal (AW MM, both units)
				if k := allophones[code-128].kind; next >= 140 && next <= 144 && sj.seg[sj.nseg-1].to.f[1] < 1500.0 && (k == kVowel || k == kGlide) && sj.stotal > ticks(15.0)+ticks(20.0) {
					sj.trimTail(ticks(15.0))
					l = &sj.seg[sj.nseg-1]
				}
				// 0.5.4: an unslowed nasal before a vowel or glide hands over 5 ms sooner (MM AW, both units)
				if code >= 140 && code <= 144 && !sj.slow && next < 200 && (allophones[next-128].kind == kVowel || allophones[next-128].kind == kGlide) && l.ticks > ticks(5.0)+2 {
					l.ticks -= ticks(5.0)
					sj.stotal -= ticks(5.0)
				}
				// 0.5.5: a slowed nasal before a vowel or glide runs 10 ms longer (Slow NE AW, both units)
				if code >= 140 && code <= 144 && sj.slow && next < 200 && (allophones[next-128].kind == kVowel || allophones[next-128].kind == kGlide) {
					l.ticks += ticks(10.0)
					sj.stotal += ticks(10.0)
				}
				// 0.5.4: a vowel or glide into TT moves F2 to 1600 Hz over its last 20 ms (the alveolar locus; AW TT, both units)
				// 0.5.4: an unslowed TT before a vowel or glide releases 10 ms longer (TT AW, both units)
				if code == 191 && !sj.slow && next < 200 && (allophones[next-128].kind == kVowel || allophones[next-128].kind == kGlide) {
					l.ticks += ticks(10.0)
					sj.stotal += ticks(10.0)
				}
				if k := allophones[code-128].kind; next == 191 && (k == kVowel || k == kGlide) && sj.stotal > ticks(20.0)+2 {
					sj.splitAt(sj.stotal - ticks(20.0))
					l = &sj.seg[sj.nseg-1]
					l.to.f[1] = 1600.0
				}
				// 0.5.4: LE before another sound runs 5 ms longer than alone (solo 110 ms, in phrases 115)
				if code == 145 && next < 200 {
					l.ticks += ticks(5.0)
					sj.stotal += ticks(5.0)
				}
				// ASSUMPTION (2026-11-08, VoxProbe 13 section 6, both units, the runs of sound): an unslowed NE or NO straight
				// into GO or GE runs on 18 ms, and 25 more when the GO is slowed (the Slow read ahead); not with a Fast on the
				// GO. NE GO OW's NE 90 -> 110 against 110-120, NO GO 100 -> 115 against 112, NE Slow GO 90 -> 135 against 130,
				// NE Fast GO 90 against 90. Man Goes Down P1's NE is slowed and keeps its length (fitted to the drawn lines)
				if (code == 141 || code == 142) && (next == 178 || next == 179) && !sj.slow {
					fastNext := false
					for i := n; i < sj.count && sj.peek(i) < 128; {
						c := sj.peek(i)
						if c == 7 {
							fastNext = true
						}
						if c >= 20 && c <= 26 {
							i += 2
						} else {
							i++
						}
					}
					if !fastNext {
						ex := 18.0
						if sj.slowBeforeNext(n) {
							ex += 25.0
						}
						l.ticks += ticks(ex)
						sj.stotal += ticks(ex)
					}
				}
				// ASSUMPTION (2026-11-08, VoxProbe 13 AW Slow TT AW and Slow AYWW Slow TT, both units, the runs of sound: the
				// sound before a slowed unvoiced stop runs on ~40 ms - the Slow is read ahead - unless it is a nasal or the
				// stop is followed by a pause (Man Goes Down P2's UW Slow TT P1): AW 130 -> 170 ms against 170). Before TT
				// only (2026-11-09, VoxProbe 14 section 4, both units by picture: before a slowed KO or PO the AW is ~130 ms, as
				// unslowed, where ours ran ~160; AW Slow KO AW +30 -> ~0, AW Slow PO AW +25)
				if next == 191 && sj.slowBeforeNext(n) && allophones[code-128].kind != kStop && !(code >= 140 && code <= 144) {
					j := n
					for j < sj.count && sj.peek(j) < 128 {
						if c := sj.peek(j); c >= 20 && c <= 26 {
							j += 2
						} else {
							j++
						}
					}
					if !(j+1 < sj.count && sj.peek(j+1) <= 6) {
						l.ticks += ticks(40.0)
						sj.stotal += ticks(40.0)
					}
				}
				// ASSUMPTION (2026-11-08, VoxProbe 13 AW Slow TT AW, both units by picture: aspiration ~50 ms where ours had
				// ~25, the AW after it 30-40 ms later than ours; Slow TT alone doesn't change, NE Slow TT IHWW is already
				// long): a slowed unvoiced stop after a sound other than a nasal, followed at once by a voiced sound,
				// releases 40 ms longer
				// TT only (2026-11-09, VoxProbe 14 section 4, both units: after a slowed KO or PO the release is as
				// unslowed, 145 against 142 ms; the Slow goes into their closure)
				if code == 191 && sj.slow && !prevNasal && next <= 199 && allophones[next-128].kind != kStop && allophones[next-128].kind != kFric {
					l.ticks += ticks(15.0) // 15, not 40, now the vowel ahead runs on: the release after it 185 ms against 185
					sj.stotal += ticks(15.0)
				}
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
				// ASSUMPTIONS (2026-11-08, VoxProbe 13, both units, the runs of sound against the silent closure): no overlap
				// from a vowel, glide or hiss into TT (AW 130 -> 145 against 152, SE 80 -> 95 against 102), nor from NE or NO
				// into KE KO PE PO BE BO (NE 90 -> 105 against 105-110); PO into a voiced sound after a vowel overlaps 20 ms
				// (AW PO AW's release 195 -> 175 against 175)
				if next == 191 && !(code >= 140 && code <= 144) && !sj.slowBeforeNext(n) {
					cut = 0
				}
				if (code == 141 || code == 142) && (next == 194 || next == 195 || next == 198 || next == 199 || next == 170 || next == 171) {
					cut = 0
				}
				if code == 199 && prevKindVowel && next <= 199 && allophones[next-128].kind != kStop && allophones[next-128].kind != kFric {
					cut = ticks(20.0)
				}
				// ASSUMPTION (2026-11-08, VoxProbe 13 section 10, all four takes: AW TH AW, IHWW TH RR 25-54 ms short): no
				// overlap into TH (Man Goes Down P2's IH TH lines up with it)
				if next == 190 {
					cut = 0
				}
				// ASSUMPTION (2026-11-08, VoxProbe 13 section 5, all four takes: IY MM, MM IY, IY NE, NE IY 26-38 ms short):
				// a nasal (MM NE NO NGE NGO) and a vowel or glide join with no overlap, unless either is slowed (Man Goes
				// Down P1's three nasal joins are all slowed and keep it: P1 1502 against 1495/1502)
				if both && next <= 199 && !sj.slow && !sj.slowBeforeNext(n) {
					kn2 := allophones[next-128].kind
					if (code >= 140 && code <= 144 && kn2 == kVowel) || (next >= 140 && next <= 144 && kc == kVowel) {
						cut = 0
					}
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
				crossing := false // the cut came from the crossing-join rule
				// the liquids LE LO WW RR join vowels by the same rule (2026-11-08, Man Goes Down P2's IY WW IH 30 ms short:
				// P2 timing 23.4 -> 17.0 ms, 1665 -> 1702; VoxProbe 13 section 5 10.2 -> 8.6)
				liqC, liqN := code >= 145 && code <= 148, next >= 145 && next <= 148
				if both && next != int(code) && (kc == kVowel || kc == kGlide || liqC) && (kn == int(kVowel) || kn == int(kGlide) || liqN) {
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
					ms := (joinA - joinK*d + joinTerm(int(code)-128) + joinTerm(next-128)) * sj.speedFactor(false)
					if ms >= 0.0 {
						cut = ticks(ms)
						crossing = true
					} else {
						cut = 0
						l.ticks += ticks(-ms)
						sj.stotal += ticks(-ms)
					}
				}
				// ASSUMPTION (2026-11-07, every recorded voiced-stop pair, both units): a voiced stop into a vowel or glide
				// with no Slow on it loses 30 - 10 * d ms more off its end, d the ERB distance between its release F2 and
				// the vowel's (BE EH 168 ms where we gave 198, DO AYWW 284 / 315; BE AW and GO OWWW, far apart, matched)
				if kc == kVStop && code != 178 && code != 179 && next != int(code) && (kn == int(kVowel) || kn == int(kGlide)) {
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
					} else if code != 171 {
						// not after BO (section 6's X BO OW 9.9 -> 9.6 ms mean)
						// ASSUMPTION (2026-11-08, VoxProbe 13 section 7, both units by picture: the burst lands at the same
						// time on every row and the vowel after it runs longer when F2 has further to go; BE OW 25 ms short
						// -> 0, DO IY 30 -> 25; section 7 10.9 -> 9.2 ms, section 6 11.3 -> 10.7): below zero the rule
						// lengthens the release
						l.ticks += ticks(-extra * sj.speedFactor(false))
						sj.stotal += ticks(-extra * sj.speedFactor(false))
					}
				}
				if cut > 0 && l.ticks > cut+1 {
					l.ticks -= cut
					sj.stotal -= cut
				} else if crossing && cut > 0 && sj.nseg > 1 && sj.stotal > cut+ticks(20.0) {
					// 0.5.0: a crossing cut longer than the last segment trims across the segments (a glide ends on a short hold:
					// VoxProbe 8 section 10's AYWW AW 38-42 ms long on both units, its 37 ms cut skipped)
					sj.trimTail(cut)
					l = &sj.seg[sj.nseg-1]
				}
				liq := func(c int) bool { return c >= 145 && c <= 148 }
				if both && (kc == kVowel || kc == kGlide || liq(int(code))) && int(next) != int(code) && next <= 199 && (allophones[next-128].kind == kVowel || allophones[next-128].kind == kGlide || liq(int(next))) {
					sj.crossJoin(alloFrame(&allophones[next-128], false, sj.bendHz()))
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
	if c := int(sj.peek(0)); c >= 128 {
		sj.prevSlow, sj.prevCode = sj.slow, c
		sj.ppok = false
		sj.paused = false
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

// randomJ is the jitter's, wander's and drift's own sequence (2026-11-11): with one sequence for everything, a structural
// edit shifted every later noise sample and the visual score moved by a third on it
func (sj *SpeakJet) randomJ() float64 {
	sj.rnj = sj.rnj*1664525 + 1013904223
	return float64(sj.rnj>>8)*(1.0/16777216.0) - 0.5
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
	// the output stage's nonlinearity (0.5.0: the effects' second-order term; the speech sounds have none)
	if sj.effect {
		v += fxDist2 * v * v
	}
	// the output stage's step size
	if quant > 0 {
		v = math.Floor(v*quant+0.5) / quant
	}
	// the output noise floor: white under the effects' tones; none under the
	// voice (floor 0, the chip's voice is exactly periodic)
	r := sj.random()
	if voice {
		v *= 1.0 + floor*2.0*(r-sj.lastr)
		// 0.5.4: a noise floor with the signal itself, peak-held 20 ms, under voiced speech too (AX UX: the chip's valleys
		// between harmonics 5-19 dB shallower; 72 allophones X 0.867 -> 0.848)
		if math.Abs(v) > sj.bedhold {
			sj.bedhold = math.Abs(v)
		} else {
			sj.bedhold = sj.bedhold * 0.993770
		}
		v += 0.024 * 2.0 * r * sj.bedhold
	} else {
		// the effects' bed (both units): timing error, proportional to the
		// signal's change per tick smoothed by one pole - it grows with the
		// tone's frequency to about 1.5 kHz and only with its level above
		// the buzz's clicks kept out of the slope (each is a huge step: the bed turned 227 B7's bands into noise)
		if sj.effect { // 0.5.0: the effects' bed; the speech sounds keep the one below
			sj.bed += fxBedPole * (v - sj.vbuzz - sj.vprev - sj.bed)
			bk := 1.0
			if sj.bedk > 0.0 {
				bk = sj.bedk
			}
			v += fxSlew * sj.bscale * 2.0 * r * sj.bed * bk
			sj.bedk = 0.0
			if math.Abs(v) > sj.bedhold {
				sj.bedhold = math.Abs(v)
			} else {
				sj.bedhold = sj.bedhold * 0.993770
			}
			vb := 0.024
			if sj.vbed {
				vb = 0.048 // 0.5.4: doubled under a voiced part that pulses only some tones (the robots)
			}
			v += vb * 2.0 * r * sj.bedhold
		} else {
			sj.bed += bedPole * (v - sj.vbuzz - sj.vprev - sj.bed)
			v += slew * sj.bscale * 2.0 * r * sj.bed
		}
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
	// oscillators 4 and 5 sit 6 dB under 1-3 (VoxProbe 7 section 5, 1000 Hz at volume 31: 5.99 dB unit A, 5.96 unit B)
	v += half * s45 / 128.0 * 0.3
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
	for k := 0; k < nTone; k++ {
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
	lrg := 1.0 // 0.5.0: the level ramp's gain, for the bed
	if s.lr != 0 {
		lrg = exp(float64(s.lr) * t * (0.5 * math.Ln10 / 20.0))
		f.lv *= float32(lrg)
	}
	f.nf = float32(lerp(float64(s.from.nf), float64(s.to.nf), t))
	f.nw = float32(lerp(float64(s.from.nw), float64(s.to.nw), t))
	f.na = float32(lerp(float64(s.from.na), float64(s.to.na), t))
	if s.flags&fxSmooth != 0 {
		if sj.stick == 0 {
			for k := 0; k < nTone; k++ {
				sj.xa[k] = f.a[k]
			}
		}
		for k := 0; k < nTone; k++ {
			onoff := f.a[k] < 0.5 || sj.xa[k] < 0.5 || (sj.xa[k] < f.a[k]*0.5 && sj.xa[k] < 2.0)
			if onoff {
				sj.xa[k] = sj.xa[k] + (f.a[k]-sj.xa[k])/24.0
			} else {
				sj.xa[k] = f.a[k]
			}
			f.a[k] = sj.xa[k]
		}
	} else {
		for k := 0; k < nTone; k++ {
			sj.xa[k] = f.a[k]
		}
	}
	// 0.5.5: an effect oscillator already sounding whose target frequency leaps (by more than 25 % in a tick) plays the
	// midpoint of old and new for 12 ticks (1.5 ms), then the new (226 B6, all four takes: the 2.2-3.1 kHz hump between its
	// 1.8 and 3.5 kHz tones). Not from silence, not into or out of a voiced part, not on a small step
	if sj.effect {
		if !sj.fslon {
			for k := 0; k < nTone; k++ {
				sj.fsl[k], sj.ftg[k], sj.fsn[k] = f.f[k], f.f[k], 0
			}
			sj.fslon = true
		}
		vz := s.flags&(fxVoiced|fxPulsed) != 0
		plain := !vz && !sj.fsv
		sj.fsv = vz
		for k := 0; k < nTone; k++ {
			tg := f.f[k]
			was := sj.fsa[k] > 0
			sj.fsa[k] = f.a[k]
			if f.a[k] <= 0 || !was {
				sj.fsl[k], sj.ftg[k], sj.fsn[k] = tg, tg, 0
				continue
			}
			if plain && sj.ftg[k] > 0 && tg > 0 && (tg > 1.25*sj.ftg[k] || sj.ftg[k] > 1.25*tg) {
				sj.fsl[k] = 0.5 * (sj.fsl[k] + tg)
				sj.fsn[k] = 12
			} else if sj.fsn[k] > 0 {
				sj.fsn[k]--
			} else {
				sj.fsl[k] = tg
			}
			sj.ftg[k] = tg
			f.f[k] = sj.fsl[k]
		}
	} else {
		sj.fslon = false
	}
	sj.last = f
	if f.a[0]+f.a[1]+f.a[2] > 0 {
		for q := 0; q < 3; q++ {
			sj.fhold[q] = f.f[q]
		}
	}

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
			// 0.5.4: a plain vowel's long period is its second after the voice starts, not a delay before the first
			if !sj.vstarted {
				sj.vcount = 0
			}
			if !sj.effect && sj.aplain && !sj.vafter {
				sj.vcount++
				if sj.vcount == 3 {
					sj.ptick += 12
				}
			}
			// 0.5.4: a vowel or glide ending into silence slows its last periods (OH ... 11.48 11.96 13.25 ms); not AW
			// AW only alone: from silence with nothing after it
			nleft := 0
			for q := sj.executing; q < sj.count; q++ {
				if c := sj.peek(q); c >= 128 && c <= 254 {
					nleft++
				}
			}
			if k := allophones[sj.aidx].kind; !sj.effect && !sj.joined && (sj.aidx != 8 || (nleft == 0 && sj.prevCode < 128)) && (k == kVowel || k == kGlide || (sj.aidx >= 17 && sj.aidx <= 20)) && sj.stotal-sj.stick < ticks(25.0) {
				sj.ptick += int(14.0*(1.0-float64(sj.stotal-sj.stick)/float64(ticks(25.0))) + 0.5)
			}
			sj.vstarted = true
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
			fall := envFall
			// 0.5.4: the 58-tick fall for every vowel again (0.5.1's 72 ticks on EY UX UH AW was chosen on X's period-locked alignment)
			env = 1.0 - (age-envRise)/fall
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
			sj.phase[k] += float64(f.f[k]) * (1.0 + jitter*2.0*sj.randomJ()) / nominal
		}
		if s.flags&fxPulsed != 0 {
			env *= 0.6
		}
		v -= float64(sj.xv) * env * sum / 128.0 * float64(f.lv) // the pulse starts downward (both units, every vowel at Pitch 20)
		sj.venv = env
		for k := nf; k < nTone; k++ {
			if k >= 3 && f.a[k] <= 0 && f.f[k] <= 0 {
				continue // a silent extra tone draws no jitter: the random sequence stays as it was with three tones
			}
			v += float64(f.a[k]) / 128.0 * 0.6 * float64(f.lv) * sin2pi(sj.phase[k])
			sj.phase[k] += float64(f.f[k]) * (1.0 + jitter*2.0*sj.randomJ()) / nominal
		}
		if sj.effect && sj.wobd > 0 {
			v *= 1.0 + sj.wobd*sin2pi(sj.wobph)
			sj.wobph += sj.wobr / rate
			if sj.wobph >= 1 {
				sj.wobph -= 1
			}
		}
		sj.ptick--
	} else {
		sj.ptick = 0
		sj.vstarted = false
		sj.vcount = 0
		sj.venv = 1.0
		sj.wander += (0.015/(0.57735*math.Sqrt(0.055/2.0))*sj.randomJ() - sj.wander) * 0.055
		// the effects' tones hold their pitch (both units) and the energy round
		// their lines is a slow wobble in level, 10 % rms, spread to about 64 Hz
		sj.am += (0.1/(0.57735*math.Sqrt(0.05/2.0))*sj.randomJ() - sj.am) * 0.05
		wander := 0.0
		if s.flags&fxWander != 0 {
			wander = sj.wander
		}
		// plain tones (the effects)
		for k := 0; k < nTone; k++ {
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
		if sj.effect && sj.wobd > 0 {
			v *= 1.0 + sj.wobd*sin2pi(sj.wobph)
			sj.wobph += sj.wobr / rate
			if sj.wobph >= 1 {
				sj.wobph -= 1
			}
		}
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
			sw := 1.0 - (p-0.5)/0.5 // 0.5.0: the chip climbs for 50% of the cycle (C8 C9, 4 takes)
			if p < 0.5 {
				sw = p / 0.5
			}
			nf *= 0.5 + 1.35*sw                     // 0.5.3: linear in Hz, 0.5 to 1.85 times the centre (238 C8, both units: straight-line track)
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
		// 0.5.4: a wandering part's noise (252 M0) through the resonance twice: steeper skirts
		if sj.effect && s.flags&fxWander != 0 {
			y2 := g*y + 2.0*r*c*sj.nz2[0] - r2*sj.nz2[1]
			sj.nz2[1] = sj.nz2[0]
			sj.nz2[0] = y2
			y = y2
		}
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

	if s.gate > 0 && s.flags&fxNSweep == 0 && sj.tick%s.gate >= (s.gate*69)/100 {
		v *= 0.02 // 0.5.0: the gate's off part is the last 31 % of the period, 34 dB down (229 B9)
	}
	fg := lrg
	gw, ga := 0.0, 0.0 // the tones' glide rate, Hz/ms, level-weighted
	for k := 0; k < nTone; k++ {
		if f.a[k] > 0 {
			gw += float64(f.a[k]) * math.Abs(float64(s.to.f[k])-float64(s.from.f[k]))
			ga += float64(f.a[k])
		}
	}
	gr := 0.0
	if ga > 0.0 && s.ticks > 1 {
		gr = gw / ga / (float64(s.ticks) / 8.026)
	}
	if s.flags&fxDecay != 0 {
		fg *= exp(-decayDB * t / 20.0 * math.Ln10)
	}
	if s.flags&fxFade != 0 {
		fg *= 1.0 - t
	}
	v *= fg / lrg
	if s.flags&fxBump != 0 {
		v *= sin2pi(0.5 * t)
	}

	// the sound's attack and release
	if sj.stick < sj.attack {
		v *= float64(sj.stick) / float64(sj.attack)
	}
	if sj.stotal-sj.stick < sj.release {
		v *= float64(sj.stotal-sj.stick) / float64(sj.release)
		fg *= float64(sj.stotal-sj.stick) / float64(sj.release)
	}
	fgc := fg
	if fgc <= 0.1 {
		fgc = 0.1
	}
	sj.bedk = math.Sqrt(1.0+glideBed*gr) / fgc
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
	if sj.lpa > 0 && sj.stick >= sj.lpt { // the low pulse: lpa times the phasor's sine part, then turned and shrunk
		v += sj.lpa * sj.lps * (1.0 - math.Exp(-float64(sj.stick-sj.lpt)/(lpRise*rate/1000.0))) // 0.5.0: it swells in (PE PO)
		c := sj.lpc*sj.lpcr - sj.lps*sj.lpsr
		sj.lps = (sj.lps*sj.lpcr + sj.lpc*sj.lpsr) * sj.lpr
		sj.lpc = c * sj.lpr
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
	sj.vbed = s.flags&fxVoiced != 0 && s.flags&(fxAll|fxSmooth) == 0
	v = sj.outputStage(v, s.voiced && s.flags&(fxVoiced|fxPulsed) == 0)
	// 0.5.4: the effects' output through a 40 Hz high-pass (B2 A0 C7 R5 R7: nothing under ~50 Hz on the chip)
	if sj.effect {
		k := math.Exp(-2.0 * math.Pi * fxOutHP / rate)
		sj.fxhpz = k*sj.fxhpz + (1.0-k)*v
		v -= sj.fxhpz
	}

	sj.tick++
	if sj.tick >= s.ticks {
		sj.tick = 0
		sj.cur++
		if sj.cur >= sj.nseg {
			if sj.copies > 0 {
				sj.copies--
				sj.copySegments(sj.copyOf, false)
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
	if sj.output == OutputClean {
		// each output sample is the chip's samples interpolated at its chip time, rsHalf samples behind the newest (the
		// window's look-ahead as latency, so a command still lands at its own chip time), then the same two poles and zero
		rsInit()
		for i := range buf {
			base := int64(math.Floor(sj.cpos)) // the sample just before the output time
			for sj.ringw <= base+rsHalf {
				sj.ring[sj.ringw&63] = sj.chipSample()
				if sj.sounding || sj.count > 0 || sj.scpAudible() {
					sj.lastsnd = sj.ringw
				}
				sj.ringw++
			}
			sounding = sounding || base <= sj.lastsnd // sounding until the output has reached the last sounding tick
			fr := sj.cpos - float64(base)
			p := int(fr * rsPh)
			q := fr*rsPh - float64(p)
			x := 0.0
			for k := 0; k < 2*rsHalf; k++ {
				j := base - rsHalf + 1 + int64(k)
				if j >= 0 {
					x += sj.ring[j&63] * (rsTab[p][k] + q*(rsTab[p+1][k]-rsTab[p][k]))
				}
			}
			sj.settle(x, sj.lpf[2], sj.lpf[3])
			buf[i] = float32(sj.z[1]+sj.lpf[5]*(sj.z[0]-sj.z[1])) * sj.Gain
			sj.cpos += step
		}
		sj.speaking = sounding
		return sounding
	}
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
