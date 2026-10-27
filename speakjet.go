package speakjet

// the size of the input buffer, and the level above which the Buffer Half
// Full status line is set. from the manual
const (
	BufferSize     = 64
	bufferHalfFull = 32
)

type command struct {
	code uint8
	data uint8
	size int
}

// SpeakJet is the chip. Write() is called by whatever is sending it data (the
// AtariVox serial line) and Render() by whatever is playing its output. the
// two can be called from different goroutines
type SpeakJet struct {
	queue []command

	// number of bytes in the queue, including the command that is executing
	buffered int32

	// size of the command that is executing
	executing int

	// whether the chip is "speaking" at this instant
	speaking bool

	// accessed only by Render()
	msa *msa

	// the output (see chipRate): the sample being held, how far through it the
	// output is (in samples at chipRate), whether the chip was sounding when it
	// was made, and the low pass's state
	held         float64
	heldPos      float64
	heldSounding bool
	sosZ         [3][2]float64

	// the amount of volume increase to apply to each sample. defaults to 1.0
	Gain float32
}

// NewSpeakJet is the preferred method of initialisation for the SpeakJet type
func NewSpeakJet() *SpeakJet {
	return &SpeakJet{
		msa:  newMSA(),
		Gain: 1.0,
	}
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
	size := CommandSize(code)

	if int(sj.buffered)+size > BufferSize {
		return false
	}
	sj.queue = append(sj.queue, command{code: code, data: data, size: size})
	sj.buffered += int32(size)
	return true
}

// BufferHalfFull is the state of the Buffer Half Full status line: true when
// the buffer holds more than 32 bytes
func (sj *SpeakJet) BufferHalfFull() bool {
	return sj.buffered > bufferHalfFull
}

// Speaking is the state of the Speaking status line
func (sj *SpeakJet) Speaking() bool {
	return sj.speaking
}

// Reset the chip as though the reset line had been cycled: the buffer is
// cleared and the controls return to their defaults
func (sj *SpeakJet) Reset() {
	sj.queue = sj.queue[:0]
	sj.buffered = int32(sj.executing)
	sj.msa.resetControls()
}

// pop the next command from the buffer. the command still counts towards the
// buffer until it has finished executing
func (sj *SpeakJet) pop() (command, bool) {
	if len(sj.queue) == 0 {
		return command{}, false
	}
	c := sj.queue[0]
	sj.queue = sj.queue[1:]
	sj.executing = c.size
	return c, true
}

func (sj *SpeakJet) peek() (uint8, bool) {
	if len(sj.queue) == 0 {
		return 0, false
	}
	return sj.queue[0].code, true
}

// the executing command has finished
func (sj *SpeakJet) retire() {
	sj.buffered -= int32(sj.executing)
	sj.executing = 0
}

// chipSample is the chip's next sample, at chipRate
func (sj *SpeakJet) chipSample() float64 {
	m := sj.msa
	for !m.busy() {
		if sj.executing > 0 {
			sj.retire()
		}

		c, ok := sj.pop()
		if !ok {
			m.autoSilence()
			break
		}

		m.execute(c.code, c.data, sj.peek)
	}

	v, _ := m.next()
	sj.heldSounding = m.busy() || sj.buffered > 0
	return v
}

// Render fills buf with the next len(buf) samples of output, at SampleRate,
// in the range -1 to 1. returns true if the chip is producing sound
func (sj *SpeakJet) Render(buf []float32) bool {
	const step = chipRate / SampleRate
	sounding := sj.heldSounding || sj.buffered > 0 || sj.msa.busy()

	for i := range buf {
		// the average of the held samples over this output sample's time
		rest, sum := step, 0.0
		for sj.heldPos+rest >= 1 {
			sum += sj.held * (1 - sj.heldPos)
			rest -= 1 - sj.heldPos
			sj.held = sj.chipSample()
			sj.heldPos = 0
			sounding = sounding || sj.heldSounding
		}
		sum += sj.held * rest
		sj.heldPos += rest

		// the low pass after the chip (direct form II transposed)
		v := sum / step
		for k := range outputSOS {
			c := &outputSOS[k]
			z := &sj.sosZ[k]
			y := c[0]*v + z[0]
			z[0] = c[1]*v - c[4]*y + z[1]
			z[1] = c[2]*v - c[5]*y
			v = y
		}
		buf[i] = float32(v) * sj.Gain
	}

	sj.speaking = sounding
	return sounding
}
