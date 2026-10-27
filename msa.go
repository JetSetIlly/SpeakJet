package speakjet

import "math"

// control defaults and ranges, from the manual
const (
	defaultVolume = 96  // 0 to 127
	defaultSpeed  = 114 // 0 to 127
	defaultPitch  = 88  // 0 to 255, in Hz
	defaultBend   = 5   // 0 to 15
)

// pause lengths in milliseconds for codes 0 to 6. the manual gives 0, 100, 200,
// 700, 30, 60 and 90. the VoxProbe recording plays AW, a pause and AW three
// times for each pause, and these lengths give the same silence between the AWs
// as the chip, within about 5ms (the AWs measured the same way on both): 21,
// 109, 212, 661, 26, 47 and 73ms. pause 0 is not nothing: the chip still leaves
// a gap between the sounds
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

// Bend doesn't multiply the frequencies but adds to them. the manual only says
// that bend goes from a "deep-hollow" to a "High-metallic" sounding voice.
// measured from three takes of the VoxRecord recording (see speedFactor()):
// D1's two tones move together and keep their 512Hz spacing, by -198, -154, 0,
// +220 and +439Hz at Bends 0, 1, 5, 10 and 15, and A5 by the same at 0 and 1;
// AW's three formants all move by about 40Hz a step (F3 from 2480 to 2912Hz at
// Bend 15, where a multiplier would put it near 4400). that is 44Hz a step from
// 5 up, and from 1 down; between 1 and 5 it is 22Hz short, which these Bends
// can't place, so the shortfall is taken up at 5. it moves oscillators 1 to 3,
// the formants and tones; SE's hiss (oscillators 4 and 5) doesn't move at all.
// in the chip's frequency steps (freqStepHz) a Bend is 10 steps, and the short
// step from 4 to 5 is 5
func (m *msa) bendHz() float64 {
	if m.bend >= defaultBend {
		return 10 * freqStepHz * float64(m.bend-defaultBend)
	}
	return freqStepHz * float64(10*m.bend-45)
}

// the voicing frequency for a Pitch setting. the chip counts each voicing
// period in whole ticks of a VOICE_TICK_HZ clock: 8000 / Pitch ticks (dividing
// in whole numbers) and one more. on the second VoxProbe recording, which plays
// every Pitch from 0 to 254, the frequency holds flat over runs of Pitch
// settings and steps exactly where that count changes (201 to 205 all give
// 199.7Hz), and at 12 settings the count is one more again, measured
// consistently on both halves of each test; no simple arithmetic gives those,
// so they are listed.
//
// not emulated: at very low pitch the chip also starts each sound (and each
// copy of a repeated one) with a pulse, so pitch 0 gives a pulse a sound; but
// at pitch 33 and 38 the pulses run on evenly across a repeat, and what decides
// it isn't known. here pitch 0 is silent
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
		s := effects[code-200]
		s.noBend = code == 254 // M2 (see expand())
		return s, true
	}
	return sound{}, false
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

		// fast and slow, measured from the VoxProbe recording: fast plays OWIY
		// and AXRR at 0.66 and 0.63 of their length, and slow at 1.42 and 1.31
		// times
		if m.fast {
			scale *= fastScale
		}
		if m.slow {
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
		} else if t.wave == envVoice {
			// a voiced effect follows the Pitch control: on the chip A1 and M2
			// are voiced at 60.3, 87.9 and 149.5Hz at pitch 60, 88 and 150.
			// their traced pitch is the chip's at the default pitch
			t.pitch *= voiceHz(float64(m.pitch)) / voiceHz(defaultPitch)
		}
		// M2 doesn't follow Bend: on both units of the fourth VoxProbe recording
		// it is the same at bend 0 and bend 15, its formants within half a hertz
		// of each other over its whole length
		if !s.noBend {
			for j := range 3 {
				if t.freq[j] > 0 {
					t.freq[j] = math.Max(t.freq[j]+bendHz, 0)
				}
			}
		}
		// Bend moves an effect's noise as well: on both units of the fourth
		// VoxProbe recording C8's and C9's noise moves about 200Hz down at bend 0
		// and 440Hz up at bend 15, as their tones do. it moves as a block, its
		// width unchanged: at bend 15 nothing is left under 560Hz within 20dB,
		// where moving the oscillators, whose spread is in proportion to their
		// frequency, widened it and left the noise reaching down to 300Hz. so the
		// Bend is added after the spread (see sample()): C8 and C9 at bend 15 10.0
		// to 8.8 and 10.0 to 7.5dB, and on the first VoxProbe recording, left
		// out, C8 at bend 0 3.2 to 3.0 and at bend 15 9.8 to 8.6dB. whether the
		// hiss of speech moves too isn't measured, so it doesn't here
		if !s.speech {
			t.noiseShift = bendHz
		}
		segs = append(segs, segment{samples: samples(ms), glide: samples(glide), target: t})

		copyMs := p.ms * copyScale
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
		m.segs = m.expand(s)
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
