package speakjet

import (
	"math"
	"testing"
)

// durations in milliseconds from the manual's allophone and sound effect
// tables, codes 128 to 254
var manualMs = map[uint8]float64{
	128: 70, 129: 70, 130: 70, 131: 70, 132: 70, 133: 70, 134: 70, 135: 70, 136: 70, 137: 70,
	138: 70, 139: 70, 140: 70, 141: 70, 142: 70, 143: 70, 144: 70, 145: 70, 146: 70, 147: 70,
	148: 70, 149: 200, 150: 200, 151: 190, 152: 200, 153: 185, 154: 165, 155: 200, 156: 225,
	157: 185, 158: 170, 159: 140, 160: 180, 161: 170, 162: 170, 163: 200, 164: 131, 165: 70,
	166: 70, 167: 70, 168: 70, 169: 70, 170: 45, 171: 45, 172: 10, 173: 10, 174: 45, 175: 45,
	176: 10, 177: 10, 178: 55, 179: 55, 180: 55, 181: 55, 182: 70, 183: 70, 184: 70, 185: 70,
	186: 70, 187: 40, 188: 40, 189: 50, 190: 40, 191: 50, 192: 70, 193: 170, 194: 55, 195: 55,
	196: 55, 197: 45, 198: 99, 199: 99,
	200: 80, 201: 80, 202: 80, 203: 80, 204: 80, 205: 80, 206: 80, 207: 80, 208: 80, 209: 80,
	210: 300, 211: 101, 212: 102, 213: 540, 214: 530, 215: 500, 216: 135, 217: 600, 218: 300,
	219: 250, 220: 200, 221: 270, 222: 280, 223: 260, 224: 300, 225: 100, 226: 104, 227: 100,
	228: 270, 229: 262, 230: 160, 231: 300, 232: 182, 233: 120, 234: 175, 235: 350, 236: 160,
	237: 260, 238: 95, 239: 75, 240: 95, 241: 95, 242: 95, 243: 95, 244: 95, 245: 95, 246: 95,
	247: 95, 248: 95, 249: 95, 250: 95, 251: 95, 252: 125, 253: 250, 254: 530,
}

// the allophones' parts add up to the manual's length (the chip's longer
// lengths come from sound.length). most of the sound effects, and the
// allophones played from the chip's own frames (see vfPoints), are traced from
// recordings of the chip and have its lengths instead, so for them only check
// that they exist
func TestDurations(t *testing.T) {
	for code, want := range manualMs {
		s, ok := lookup(code)
		if !ok {
			t.Errorf("code %d: no sound", code)
			continue
		}
		if code >= 200 || s.framed {
			if s.ms() <= 0 {
				t.Errorf("code %d (%s): no length", code, s.name)
			}
			continue
		}
		if math.Abs(s.ms()-want) > 0.01 {
			t.Errorf("code %d (%s): %.2fms, manual says %.0fms", code, s.name, s.ms(), want)
		}
	}
}

// render runs the chip until it falls silent and returns the samples
func render(sj *SpeakJet) []float32 {
	var out []float32
	buf := make([]float32, 256)
	for range 10000 {
		sounding := sj.Render(buf)
		out = append(out, buf...)
		if !sounding && sj.buffered == 0 {
			break
		}
	}
	return out
}

func TestPhraseLength(t *testing.T) {
	sj := NewSpeakJet()
	phrase := []uint8{183, 135, 199} // HO OH PO
	for _, c := range phrase {
		sj.Write(c, 0)
	}
	// count samples one at a time until the chip stops sounding
	buf := make([]float32, 1)
	n := 0
	for sj.Render(buf) {
		n++
	}
	got := float64(n) * 1000 / SampleRate
	// the last sound is phrase-final (see finalLength)
	want := 70.0*allophoneLength[183-128] + 70.0*allophoneLength[135-128] +
		99*allophoneLength[199-128]*finalLength[199-128] + autoSilenceMs
	if math.Abs(got-want) > 1 {
		t.Errorf("phrase played for %.0fms, expected about %.0fms", got, want)
	}
}

func TestBuffer(t *testing.T) {
	sj := NewSpeakJet()
	for i := range BufferSize {
		if !sj.Write(128, 0) {
			t.Fatalf("buffer refused byte %d", i)
		}
		if half := sj.BufferHalfFull(); half != (i+1 > bufferHalfFull) {
			t.Errorf("after %d bytes buffer half full is %v", i+1, half)
		}
	}
	if sj.Write(128, 0) {
		t.Errorf("buffer accepted a 65th byte")
	}
	if sj.Write(21, 100) {
		t.Errorf("buffer accepted a two byte command when full")
	}

	// draining the buffer clears the status line
	render(sj)
	if sj.BufferHalfFull() || sj.buffered != 0 {
		t.Errorf("buffer not empty after rendering: %d bytes", sj.buffered)
	}
}

// Soft (18) is accepted and ignored: a sound after it plays exactly as it
// does without it
func TestSoft(t *testing.T) {
	plain := NewSpeakJet()
	plain.Write(170, 0) // BE
	soft := NewSpeakJet()
	soft.Write(18, 0)
	soft.Write(170, 0)

	a := render(plain)
	b := render(soft)
	if len(a) != len(b) {
		t.Fatalf("Soft changed the length: %d samples, %d without", len(b), len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Soft changed sample %d: %v, %v without", i, b[i], a[i])
		}
	}
}

func TestOutputRange(t *testing.T) {
	sj := NewSpeakJet()
	for c := 128; c <= 254; c++ {
		sj.Write(uint8(c), 0)
		for _, v := range render(sj) {
			if math.IsNaN(float64(v)) || v > 1.5 || v < -1.5 {
				t.Fatalf("code %d: sample out of range: %v", c, v)
			}
		}
	}
}
