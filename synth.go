package speakjet

import "math"

// SampleRate is the rate at which samples are produced. the chip makes its own
// at about 8kHz (the manual: "The frequency is produced at a rate of 8192
// samples per second"; measured, 8015 to 8040), but holds each until the next,
// which puts images of the sound up to 20kHz and more; the output is at a rate
// that carries them (see chipRate)
const SampleRate = 48000

// limits of the synthesizer registers, from the manual
const (
	maxFrequency = 3999
	maxOscVolume = 31
	maxMixer1    = 63
	maxMaster    = 127
)

// envelope wave shapes, as selected by the first two bits of the envelope
// control register
type envelopeWave int

const (
	envSaw envelopeWave = iota
	envSine
	envTriangle
	envSquare
	envVoice
)

// the chip's oscillators run on a grid of frequencies FREQ_STEP_HZ apart: on
// the second VoxProbe recording all seven DTMF frequencies are whole multiples
// of it (697.17Hz is 159 steps, 1468.75Hz 335, each within 0.12Hz), a step of
// Bend is 10 of them and the short step between Bends 4 and 5 is 5. that is the
// step of a phase accumulator. onGrid() puts a frequency on it
const freqStepHz = 4.3845

func onGrid(hz float64) float64 { return math.Floor(hz/freqStepHz+0.5) * freqStepHz }

// the chip's voicing fade: the level of the formant oscillators after each
// voicing pulse, every voiceFadeStepMs, straight lines in between, nothing
// after the last (see envVoice in synth.sample()). measured from single pulses
// at pitch 40 on two units: full level, a dip to 0.7 within half a millisecond,
// back to 0.8, held to 2ms, then a near straight fall to nothing by 7.5ms.
// Volume, Bend and Pitch don't change it
const voiceFadeStepMs = 0.25

var voiceFade = [...]float64{
	1.000, 0.807, 0.701, 0.892, 0.787, 0.800, 0.756, 0.734,
	0.753, 0.687, 0.660, 0.612, 0.598, 0.547, 0.513, 0.469,
	0.426, 0.386, 0.331, 0.288, 0.252, 0.246, 0.193, 0.134,
	0.150, 0.085, 0.096, 0.054, 0.040, 0.043, 0.003, 0.008,
	0.012, 0.000,
}

// how long the chip takes to bring voicing up from silence, in a straight line:
// vowels, liquids and diphthongs onsetMs, nasals nasalOnsetMs. a time, not a
// number of periods (the same at every pitch). voiced stops and fricatives have
// none
const (
	onsetMs      = 40.0
	nasalOnsetMs = 25.0
)

// onsetFor is the onset each allophone has (see onsetMs): the vowels, liquids
// and diphthongs onsetMs, the nasals nasalOnsetMs, the rest none
func onsetFor(code int) float64 {
	switch {
	case code > 164:
		return 0
	case code >= 140 && code <= 144:
		return nasalOnsetMs
	}
	return onsetMs
}

// frame is a set of synthesizer register values. the MSA sequencer moves the
// synthesizer from one frame to the next over time
type frame struct {
	// frequency and volume of oscillators 1 to 5
	freq [5]float64
	vol  [5]float64

	// distortion of oscillators 4 and 5 (0 to 255)
	distortion float64

	// frequency of the envelope oscillator. for speech this is the voicing
	// pitch
	pitch float64

	// envelope control: the wave shape, whether oscillators 1 to 3 are
	// enveloped and whether oscillators 4 and 5 are half enveloped
	wave    envelopeWave
	voiced  bool
	halfEnv bool

	// the phase (in cycles) oscillators 2 and 3 restart at with each voicing
	// pulse, against oscillator 1 (see formantPhases)
	phase0 [3]float64

	// how long the voicing takes to fade in when it starts from silence (see
	// onsetMs). 0 is no fade
	onsetMs float64

	// how deeply the envelope gates oscillators 4 and 5 (0 to 1), when they
	// aren't half enveloped
	noiseDepth float64

	// how far Bend moves oscillators 4 and 5 of a sound effect, in hertz, added
	// after their distortion spread (see expand())
	noiseShift float64

	// how long each random value of the distortion is held, in ms; 0 for a new
	// one every sample (see cNoise())
	noiseHoldMs float64

	// a gated effect that pulses as the voicing does (see fxPulse): its rate is
	// pulseOffset hertz from the voicing, and its pulses fade along voiceFade
	// stretched fadeScale times (0 is 1)
	pulsed      bool
	pulseOffset float64
	fadeScale   float64

	// oscillators 1 to 3 that the voicing doesn't envelope: not restarted by
	// its pulses nor faded after them (bit 0 is oscillator 1). the manual has
	// the envelope control choose which oscillators it envelopes (see c1())
	freeOsc int

	// the shaping of the consonants (see soundEQ): the gain in dB of each
	// of the eqBands peaking filters on the output, broadband noise riding on
	// oscillators 4 and 5 (as a fraction of their volume), and an overall gain
	// in dB
	eq     [eqBands]float64
	wide   float64
	eqGain float64
}

// lerp returns the frame part way between a and b. the envelope control
// settings are taken from b
func lerp(a, b frame, t float64) frame {
	r := b
	for i := range a.freq {
		r.freq[i] = a.freq[i] + (b.freq[i]-a.freq[i])*t
		r.vol[i] = a.vol[i] + (b.vol[i]-a.vol[i])*t
	}
	r.distortion = a.distortion + (b.distortion-a.distortion)*t
	r.pitch = a.pitch + (b.pitch-a.pitch)*t
	for i := range r.eq {
		r.eq[i] = a.eq[i] + (b.eq[i]-a.eq[i])*t
	}
	r.wide = a.wide + (b.wide-a.wide)*t
	r.eqGain = a.eqGain + (b.eqGain-a.eqGain)*t
	return r
}

// the chip's output. it makes its samples at chipRate (measured from the mirror
// images of steady tones; the units' clocks differ by 0.3%), holds each until
// the next, and an analogue low pass shapes the result, images and all. here
// the samples are held the same way, each output sample is the average of what
// is held over its time, and outputSOS is that low pass (measured from the
// voicing pulses below 4kHz and from the images above: -5.4dB at 2kHz, -14 at
// 3.5kHz, -18 at 5.5kHz, -38 at 23.5kHz), whatever the AtariVox and recorder
// add included
const chipRate = 8028.0

var outputSOS = [3][6]float64{
	{0.0636279260299, 0, 0, 1, 0.0618381585253, 0},
	{1, -0.0974446347576, 0, 1, -1.63378468591, 0.680472611685},
	{1, 0.985616684932, 0, 1, 1.0014122889, 0.0588508646104},
}

// synth is the 5-channel synthesizer
type synth struct {
	// phase of oscillators 1 to 5 and of the envelope oscillator, in cycles
	phase    [5]float64
	envPhase float64

	// samples since the last voicing pulse
	envAge int64

	// samples since oscillators 1 to 3 last came up from silence
	onsetAge int64

	// noise source for distortion (xorshift)
	noise uint32

	// the held random values of oscillators 4 and 5, and the samples left until
	// the next (see noiseHoldMs)
	held     [2]float64
	holdLeft int64

	// the consonants' peaking filters (see soundEQ): the gains their
	// coefficients were made for, the coefficients (b0, b1, b2, a1, a2) and
	// the last two inputs and outputs of each
	eqDB    [eqBands]float64
	eqCo    [eqBands][5]float64
	eqSt    [eqBands][4]float64
	eqReady bool

	// the noise source of the consonants' broadband noise
	wideNoise uint32

	// the level the noise floor follows, and its own noise source (see
	// skirtDB)
	skirtLevel float64
	skirtNoise uint32

	// dc blocker state
	dcIn, dcOut float64
}

// the sounds' shaping (see soundEQ): eqBands peaking filters half an octave
// apart, then an overall gain. a filter is made again whenever its gain changes
// at all, not when it has moved some step, so that compilers that do and don't
// fuse a multiply and an add give the same samples
const (
	eqBands = 10
	eqQ     = 1.0
)

var eqHz = [eqBands]float64{180, 250, 355, 500, 710, 1000, 1410, 2000, 2830, 3600}

func (s *synth) eq(f frame, x float64) float64 {
	for k := range eqBands {
		c := &s.eqCo[k]
		if !s.eqReady || f.eq[k] != s.eqDB[k] {
			s.eqDB[k] = f.eq[k]
			a := math.Pow(10, s.eqDB[k]/40)
			w := 2 * math.Pi * eqHz[k] / chipRate
			al := math.Sin(w) / (2 * eqQ)
			a0 := 1 + al/a
			c[0] = (1 + al*a) / a0
			c[1] = -2 * math.Cos(w) / a0
			c[2] = (1 - al*a) / a0
			c[3] = -2 * math.Cos(w) / a0
			c[4] = (1 - al/a) / a0
		}
		z := &s.eqSt[k]
		y := c[0]*x + c[1]*z[0] + c[2]*z[1] - c[3]*z[2] - c[4]*z[3]
		z[1] = z[0]
		z[0] = x
		z[3] = z[2]
		z[2] = y
		x = y
	}
	s.eqReady = true
	return x * math.Pow(10, f.eqGain/20)
}

func newSynth() *synth {
	return &synth{noise: 0x2545f491, skirtNoise: 0x9e3779b9, wideNoise: 0x85ebca6b, envAge: 1 << 30}
}

// the chip's output has a floor of noise under whatever it plays, 40 to 55dB
// under a pure tone. here white noise follows the output's level (over about
// 25ms) skirtDB under it
const (
	skirtDB     = -24.0
	skirtFollow = 0.995
)

func (s *synth) random() float64 {
	s.noise ^= s.noise << 13
	s.noise ^= s.noise >> 17
	s.noise ^= s.noise << 5
	return float64(int32(s.noise)) / math.MaxInt32
}

// sample produces the next output sample, in the range -1 to 1. master is the
// master volume (0 to 127)
func (s *synth) sample(f frame, master float64) float64 {
	// envelope oscillator
	// envelope oscillator: a voicing pulse at the end of each period
	env := 1.0
	pulse := false
	if f.pitch > 0 {
		s.envPhase += f.pitch / chipRate
		if s.envPhase >= 1 {
			s.envPhase -= math.Floor(s.envPhase)
			pulse = true
		}
	}
	if pulse {
		s.envAge = 0
		// restart the formant oscillators with each pulse
		if f.voiced {
			for i := range 3 {
				if f.freeOsc>>i&1 == 0 {
					s.phase[i] = f.phase0[i]
				}
			}
		}
	}
	if f.wave == envVoice {
		// the chip's voicing: each pulse restarts the formant oscillators at full
		// level and they fade to nothing along voiceFade in under 10ms, whatever
		// the pitch or volume. at low pitch the rest of the period is silent; at
		// high pitch the next pulse cuts the fade short. a pulsed effect's fade
		// can be longer (see fxPulse)
		scale := 1.0
		if f.fadeScale > 0 {
			scale = f.fadeScale
		}
		k := float64(s.envAge) * 1000 / chipRate / voiceFadeStepMs / scale
		i := int(k)
		if i >= len(voiceFade)-1 {
			env = 0
		} else {
			env = voiceFade[i] + (voiceFade[i+1]-voiceFade[i])*(k-float64(i))
		}
	} else if f.pitch > 0 {
		p := s.envPhase
		switch f.wave {
		case envSaw:
			env = 1 - p
		case envSine:
			env = 0.5 + 0.5*math.Cos(2*math.Pi*p)
		case envTriangle:
			env = 1 - 2*math.Abs(p-0.5)
		case envSquare:
			if p >= 0.5 {
				env = 0
			}
		}
	} else if f.voiced {
		// a pitch of zero gives no periodic envelope
		env = 0
	}
	if s.envAge < 1<<30 {
		s.envAge++
	}

	// mixer 1: oscillators 1 to 3
	var mix1, free1, total float64
	for i := range 3 {
		fr := math.Min(onGrid(f.freq[i]), maxFrequency)
		s.phase[i] += fr / chipRate
		s.phase[i] -= math.Floor(s.phase[i])
		v := math.Min(f.vol[i], maxOscVolume)
		total += v
		if f.freeOsc>>i&1 == 1 {
			free1 += v * math.Sin(2*math.Pi*s.phase[i])
		} else {
			mix1 += v * math.Sin(2*math.Pi*s.phase[i])
		}
	}
	if f.voiced {
		mix1 *= env
	}
	mix1 += free1
	if total > maxMixer1 {
		mix1 *= maxMixer1 / total
	}
	// the voicing coming up from silence (see onsetMs)
	if total == 0 {
		s.onsetAge = 0
	} else if f.onsetMs > 0 && float64(s.onsetAge) < math.Floor(f.onsetMs*chipRate/1000) {
		mix1 *= float64(s.onsetAge) / (f.onsetMs * chipRate / 1000)
		s.onsetAge++
	}

	// mixer 2: oscillators 4 and 5, with distortion spreading noise around
	// the oscillator frequency, a new random value every sample or held for
	// noiseHoldMs, and then moved by noiseShift
	var mix2 float64
	spread := f.distortion / 255 * 1.5
	if f.noiseHoldMs > 0 {
		s.holdLeft--
		if s.holdLeft <= 0 {
			s.held[0] = s.random()
			s.held[1] = s.random()
			s.holdLeft = int64(f.noiseHoldMs*chipRate/1000 + 0.5)
		}
	}
	for i := 3; i < 5; i++ {
		fr := math.Min(onGrid(f.freq[i]), maxFrequency)
		var r float64
		if f.noiseHoldMs > 0 {
			r = s.held[i-3]
		} else {
			r = s.random()
		}
		hz := fr*(1+spread*r) + f.noiseShift
		if spread == 0 {
			hz = math.Max(math.Min(hz, maxFrequency), 0)
		}
		s.phase[i] += hz / chipRate
		s.phase[i] -= math.Floor(s.phase[i])
		mix2 += math.Min(f.vol[i], maxOscVolume) * math.Sin(2*math.Pi*s.phase[i])
	}
	// the consonants' broadband noise (see soundEQ)
	if f.wide > 0 {
		s.wideNoise ^= s.wideNoise << 13
		s.wideNoise ^= s.wideNoise >> 17
		s.wideNoise ^= s.wideNoise << 5
		mix2 += f.wide * (math.Min(f.vol[3], maxOscVolume) + math.Min(f.vol[4], maxOscVolume)) *
			float64(int32(s.wideNoise)) / math.MaxInt32
	}
	if f.halfEnv {
		mix2 *= 0.5 + 0.5*env
	} else if f.noiseDepth > 0 {
		mix2 *= 1 - f.noiseDepth*(1-env)
	}

	// mixer 5: summation and master volume
	x := (mix1 + mix2) / (maxMixer1 + 2*maxOscVolume) * math.Min(master, maxMaster) / maxMaster

	x = s.eq(f, x)

	// the noise floor (see skirtDB)
	s.skirtLevel = skirtFollow*s.skirtLevel + (1-skirtFollow)*math.Abs(x)
	s.skirtNoise ^= s.skirtNoise << 13
	s.skirtNoise ^= s.skirtNoise >> 17
	s.skirtNoise ^= s.skirtNoise << 5
	x += math.Pow(10, skirtDB/20) * s.skirtLevel * float64(int32(s.skirtNoise)) / math.MaxInt32

	// dc blocker
	y := x - s.dcIn + 0.995*s.dcOut
	s.dcIn = x
	s.dcOut = y

	return y
}
