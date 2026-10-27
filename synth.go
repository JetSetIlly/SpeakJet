package speakjet

import "math"

// SampleRate is the rate at which the synthesizer produces samples. From the
// manual: "The frequency is produced at a rate of 8192 samples per second"
const SampleRate = 8192

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
// voicing pulse, every voiceFadeStepMs from the pulse, in between in a
// straight line, and after the last nothing (see envVoice in sample()).
// measured from the waveforms of the third VoxProbe recording, where at pitch
// 40 each pulse stands alone and about 24 of them were averaged, on 18 vowels,
// nasals and liquids on two AtariVoxes. each pulse was fitted as three sines
// starting together, times one envelope shared by every sound, through one
// output filter (see outputFIR); the envelope and the filter were solved for
// along with each sound's frequencies and amplitudes. that leaves 1/470 of the
// pulses' energy unexplained (-26.7dB), against 1/28 for the straight 8ms ramp
// first used. the fade starts at full level, dips to about 0.7 within half a
// millisecond and comes back to about 0.8 (the onset shapes the whole spectrum:
// smoothing it into a flat start costs the single pulses 1.8dB), holds to 2ms,
// falls almost in a straight line and is gone by 7.5ms. fitted to either unit
// alone it fits the other within 0.2dB, and fitted the same way to pulses this
// library makes it gives back this library's own envelope, so the method
// doesn't shape it. Volume and Bend don't change it, and nor does Pitch from 25
// to 60
const voiceFadeStepMs = 0.25

var voiceFade = [...]float64{
	1.000, 0.807, 0.701, 0.892, 0.787, 0.800, 0.756, 0.734,
	0.753, 0.687, 0.660, 0.612, 0.598, 0.547, 0.513, 0.469,
	0.426, 0.386, 0.331, 0.288, 0.252, 0.246, 0.193, 0.134,
	0.150, 0.085, 0.096, 0.054, 0.040, 0.043, 0.003, 0.008,
	0.012, 0.000,
}

// how long the chip takes to bring a vowel, nasal, liquid or diphthong's
// voicing up from silence: in a straight line, from nothing to full level. on
// the VoxRecord recording, both units, the first three voicing periods of the
// 37 sounds played alone stand at -17, -6.3 and -1.5dB against the rest
// (medians), which 40ms gives to within 1.4dB; ramping the volumes alone gave
// -8, -2.7 and 0, because the mixer's limit (maxMixer1) holds the sum of the
// formants at full level long before the volumes get there. AW MM at eight
// pitches from 60 to 250 takes about 30ms throughout on both units, so it is a
// time rather than a number of periods. the voiced stops and fricatives start
// as they did: the ramp on them made both units match them worse. the nasals
// come up faster: measured from where they rise out of the recording's
// background (which, being quieter sounds, they sit nearer to), both units have
// them within 3dB of full level by the second period. 25ms fits them best (40ms
// 90.7%, 25ms 91.4%)
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
	return r
}

// the low pass after the PWM output (the manual places a "two pole" filter
// there). the openFPGA-7800 AtariVox engine (MIT licence) fitted a 2-pole
// 2200Hz corner to recordings of games, and Stella's SpeakJet engine measured
// 1700Hz. the VoxRecord recording's third take measured its level directly,
// from the sound effects that sweep a single pure tone: flat to 1.5kHz, then
// -2.2dB at 1.9kHz, -5.7 at 2.4, -10.8 at 2.9, -15.8 at 3.6. the voicing pulses
// of the third VoxProbe recording (see voiceFade) measure it again, level and
// phase, on both units: -1.2dB at 1kHz, -2.8 at 1.5, -5.4 at 2, -8.9 at 2.5,
// -11.8 at 3 and -14.3 at 3.5kHz against 500Hz, with no delay that changes with
// frequency. the two measurements agree within 2 to 3dB; this is the second, as
// an 11-tap FIR at this sample rate (within 0.75dB RMS and 5 degrees of it). it
// includes whatever filtering the AtariVox and the recorder add
const outputTaps = 11

var outputFIR = [outputTaps]float64{
	0.0195866114, -0.0027854151, 0.0363309004, -0.0453457714,
	0.3233301119, 0.5632340831, 0.2283437277, 0.0506567790,
	-0.0181499488, 0.0017538421, -0.0052263971,
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

	// output filter history (newest first) and dc blocker state
	hist        [outputTaps]float64
	dcIn, dcOut float64
}

func newSynth() *synth {
	return &synth{noise: 0x2545f491, envAge: 1 << 30}
}

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
		s.envPhase += f.pitch / SampleRate
		if s.envPhase >= 1 {
			s.envPhase -= math.Floor(s.envPhase)
			pulse = true
		}
	}
	if pulse {
		s.envAge = 0
		// restart the formant oscillators with each pulse
		if f.voiced {
			s.phase[0] = f.phase0[0]
			s.phase[1] = f.phase0[1]
			s.phase[2] = f.phase0[2]
		}
	}
	if f.wave == envVoice {
		// the chip's voicing: each pulse restarts the formant oscillators at full
		// level and they fade to nothing along voiceFade in under 10ms, whatever
		// the pitch or volume. at low pitch the rest of the period is silent; at
		// high pitch the next pulse cuts the fade short
		k := float64(s.envAge) * 1000 / SampleRate / voiceFadeStepMs
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
	var mix1, total float64
	for i := range 3 {
		fr := math.Min(onGrid(f.freq[i]), maxFrequency)
		s.phase[i] += fr / SampleRate
		s.phase[i] -= math.Floor(s.phase[i])
		v := math.Min(f.vol[i], maxOscVolume)
		total += v
		mix1 += v * math.Sin(2*math.Pi*s.phase[i])
	}
	if total > maxMixer1 {
		mix1 *= maxMixer1 / total
	}
	if f.voiced {
		mix1 *= env
	}
	// the voicing coming up from silence (see onsetMs)
	if total == 0 {
		s.onsetAge = 0
	} else if f.onsetMs > 0 && float64(s.onsetAge) < math.Floor(f.onsetMs*SampleRate/1000) {
		mix1 *= float64(s.onsetAge) / (f.onsetMs * SampleRate / 1000)
		s.onsetAge++
	}

	// mixer 2: oscillators 4 and 5, with distortion spreading noise around
	// the oscillator frequency
	var mix2 float64
	spread := f.distortion / 255 * 1.5
	for i := 3; i < 5; i++ {
		fr := math.Min(onGrid(f.freq[i]), maxFrequency)
		s.phase[i] += fr * (1 + spread*s.random()) / SampleRate
		s.phase[i] -= math.Floor(s.phase[i])
		mix2 += math.Min(f.vol[i], maxOscVolume) * math.Sin(2*math.Pi*s.phase[i])
	}
	if f.halfEnv {
		mix2 *= 0.5 + 0.5*env
	} else if f.noiseDepth > 0 {
		mix2 *= 1 - f.noiseDepth*(1-env)
	}

	// mixer 5: summation and master volume
	x := (mix1 + mix2) / (maxMixer1 + 2*maxOscVolume) * math.Min(master, maxMaster) / maxMaster

	// dc blocker
	y := x - s.dcIn + 0.995*s.dcOut
	s.dcIn = x
	s.dcOut = y

	// output filter
	for i := outputTaps - 1; i > 0; i-- {
		s.hist[i] = s.hist[i-1]
	}
	s.hist[0] = y
	var out float64
	for i := range outputTaps {
		out += outputFIR[i] * s.hist[i]
	}

	return out
}
