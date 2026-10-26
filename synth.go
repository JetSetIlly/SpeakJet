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
)

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
// 1700Hz. the VoxRecord recording's third take measures it directly: the sound
// effects that sweep a single pure tone (B1 from 1.4 to 3.8kHz and back, and
// nine others) lose level with frequency alone, and solving for one response
// common to all of them gives it flat to 1.5kHz, then -2.2dB at 1.9kHz, -5.7
// at 2.4, -10.8 at 2.9, -15.8 at 3.6 and -17.2 at 3.9. that is the shape of a
// third-order butterworth at 2000Hz, but at this sample rate a recursive
// filter can't keep it that close to 4096Hz, so this is an 11-tap
// linear-phase FIR designed to it (within 0.9dB RMS of the measurements). it
// includes whatever filtering the AtariVox and the recorder add
const outputTaps = 11

var outputFIR = [outputTaps]float64{
	0.0070836153, -0.0046812003, -0.0276894693, -0.0319246056,
	0.2395192408, 0.6353848384, 0.2395192408, -0.0319246056,
	-0.0276894693, -0.0046812003, 0.0070836153,
}

// synth is the 5-channel synthesizer
type synth struct {
	// phase of oscillators 1 to 5 and of the envelope oscillator, in cycles
	phase    [5]float64
	envPhase float64

	// noise source for distortion (xorshift)
	noise uint32

	// output filter history (newest first) and dc blocker state
	hist        [outputTaps]float64
	dcIn, dcOut float64
}

func newSynth() *synth {
	return &synth{noise: 0x2545f491}
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
	env := 1.0
	if f.pitch > 0 {
		s.envPhase += f.pitch / SampleRate
		if s.envPhase >= 1 {
			s.envPhase -= math.Floor(s.envPhase)

			// restart the formant oscillators with each envelope period
			if f.voiced {
				s.phase[0] = 0
				s.phase[1] = 0
				s.phase[2] = 0
			}
		}
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
		// a pitch of zero does not vocalise
		env = 0
	}

	// mixer 1: oscillators 1 to 3
	var mix1, total float64
	for i := range 3 {
		fr := math.Min(f.freq[i], maxFrequency)
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

	// mixer 2: oscillators 4 and 5, with distortion spreading noise around
	// the oscillator frequency
	var mix2 float64
	spread := f.distortion / 255 * 1.5
	for i := 3; i < 5; i++ {
		fr := math.Min(f.freq[i], maxFrequency)
		s.phase[i] += fr * (1 + spread*s.random()) / SampleRate
		s.phase[i] -= math.Floor(s.phase[i])
		mix2 += math.Min(f.vol[i], maxOscVolume) * math.Sin(2*math.Pi*s.phase[i])
	}
	if f.halfEnv {
		mix2 *= 0.5 + 0.5*env
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
