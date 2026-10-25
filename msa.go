package speakjet

import "math"

// control defaults and ranges, from the manual
const (
	defaultVolume = 96  // 0 to 127
	defaultSpeed  = 114 // 0 to 127
	defaultPitch  = 88  // 0 to 255, in Hz
	defaultBend   = 5   // 0 to 15
)

// pause lengths in milliseconds for codes 0 to 6. the manual gives 0, 100,
// 200, 700, 30, 60 and 90. two takes of the VoxRecord recording (see
// speedFactor()) measure the silence between two AWs, against the two delays
// (whose length is exact) as a reference: pauses 3, 5 and 6 are shorter than
// the manual says, about 650, 45 and 70ms. the takes differ by up to 35ms on a
// pause, so the others, within 10ms of the manual, keep it
var pauseMs = [...]float64{0, 100, 200, 650, 30, 45, 70}

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
	return int(math.Round(ms * SampleRate / 1000))
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

	// the sound playing is speech rather than a sound effect: Bend works
	// differently on the two (see effectBendHz())
	speech bool

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
	m.speech = true

	return m
}

func (m *msa) resetControls() {
	m.volume = defaultVolume
	m.speed = defaultSpeed
	m.pitch = defaultPitch
	m.bend = defaultBend
}

// bendFactor is the multiplier the Bend control applies to oscillator
// frequencies in speech. the manual only says that bend goes from a
// "deep-hollow" to a "High-metallic" sounding voice, so the curve is a guess: a
// semitone per step either side of the default. the recording's clipped vowels
// can't confirm it. sound effects are bent by effectBendHz() instead
func (m *msa) bendFactor() float64 {
	if !m.speech {
		return 1.0
	}
	return math.Pow(2, float64(m.bend-defaultBend)/12)
}

// on a sound effect Bend doesn't multiply the frequencies but adds to them.
// measured from two takes of the VoxRecord recording (see speedFactor()), which
// plays D1 and A5 at Bends 0, 1, 5, 10 and 15: D1's two tones move together and
// keep their 512Hz spacing, by -198, -154, 0, +220 and +439Hz, and A5 by the
// same at Bends 0 and 1. that is 44Hz a step from 5 up, and from 1 down;
// between 1 and 5 it is 22Hz short, which these five Bends can't place, so the
// shortfall is taken up at 5. the offset is applied to oscillators 1 to 3, the
// tones; whether it moves the noise oscillators isn't known
func (m *msa) effectBendHz() float64 {
	if m.bend >= defaultBend {
		return 44.0 * float64(m.bend-defaultBend)
	}
	return 44.0*float64(m.bend) - 198.0
}

// the duration multiplier from the Speed control. the manual gives the range
// and default but not the curve. measured from a recording of a real AtariVox
// playing the VoxRecord ROM (TT AW EYIY SE at speeds 0 to 127): the length
// falls in a straight line with speed, 2115ms at speed 0 and 955ms at 80, and
// the line predicts speeds 20, 40 and 60 to within 2%. it reaches zero at
// about 146, so relative to the default speed the multiplier is
// (146 - speed) / 32: 4.6 at speed 0, 0.59 at 127. pauses are lengthened by it
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
		return allophones[code-128], true
	case code >= 200 && int(code) < 200+len(effects):
		return effects[code-200], true
	}
	return sound{}, false
}

// expand converts a sound into segments, applying the controls and modifiers
func (m *msa) expand(s sound) []segment {
	scale := 1.0
	pitch := float64(m.pitch)
	f1, f2 := 1.0, 1.0

	if s.speech {
		scale = m.speedFactor() * s.length

		// fast and slow, measured from the vox_test recording (see
		// steadyLength): fast plays a sound at 0.53 to 0.67 of its length (NE,
		// JH, AXRR), and slow at 1.14 to 1.67 times (SE, NE, TH, IY)
		if m.fast {
			scale *= 0.6
		}
		if m.slow {
			scale *= 1.5
		}

		// "STRESS, IH sounds more like (but not quite) the IY sound". stress
		// tenses the vowel (lower F1, higher F2) and raises the tone; relax
		// does the opposite. the amounts are not given by the manual
		if m.stress {
			pitch *= 1.12
			f1, f2 = 0.94, 1.06
		} else if m.relax {
			pitch *= 0.9
			f1, f2 = 1.06, 0.94
		}
	}

	m.speech = s.speech
	bendHz := 0.0
	if !s.speech {
		bendHz = m.effectBendHz()
	}

	var segs []segment
	for _, p := range s.parts {
		ms := p.ms * scale
		glide := p.glideMs * scale
		if p.glideMs == autoGlide {
			glide = math.Min(math.Max(ms*0.35, 20), 50)
		}
		glide = math.Min(glide, ms)

		t := p.target
		if s.speech {
			t.pitch = pitch
			t.freq[0] *= f1
			t.freq[1] *= f2
		} else {
			for j := range 3 {
				if t.freq[j] > 0 {
					t.freq[j] = math.Max(t.freq[j]+bendHz, 0)
				}
			}
		}
		segs = append(segs, segment{samples: samples(ms), glide: samples(glide), target: t})
	}

	m.fast, m.slow, m.stress, m.relax = false, false, false, false

	if len(segs) > 0 && m.jumpNextFreq {
		segs[0].jump = true
		m.jumpNextFreq = false
	}

	if m.repeat > 1 {
		one := segs
		for i := 1; i < m.repeat; i++ {
			segs = append(segs, one...)
		}
	}
	m.repeat = 0

	return segs
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
		m.segs = m.silence(float64(data)*10, nil)

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
		m.segs = m.expand(s)
	}

	m.segPos = 0
	m.segStart = m.current
	if len(m.segs) > 0 && m.segs[0].jump {
		m.segStart.freq = m.segs[0].target.freq
	}

	return true
}

// busy is true while there are segments still to play
func (m *msa) busy() bool {
	return len(m.segs) > 0
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
		return m.synth.sample(m.current, m.bendFactor(), float64(m.volume)), false
	}

	sg := &m.segs[0]
	if sg.glide > 0 && m.segPos < sg.glide {
		m.current = lerp(m.segStart, sg.target, float64(m.segPos+1)/float64(sg.glide))
	} else {
		m.current = sg.target
	}

	v := m.synth.sample(m.current, m.bendFactor(), float64(m.volume))

	m.segPos++
	if m.segPos >= sg.samples {
		m.segs = m.segs[1:]
		m.segPos = 0
		m.segStart = m.current
		if len(m.segs) > 0 && m.segs[0].jump {
			m.segStart.freq = m.segs[0].target.freq
		}
	}

	return v, len(m.segs) > 0
}
