package main

import (
	"github.com/kierdavis/gosound/frontend"
	"github.com/kierdavis/gosound/sound"
	"github.com/kierdavis/gosound/sound/filter"
	"time"
)

// Copy input to output
func pipe(input, output chan float64) {
	for x := range input {
		output <- x
	}
	close(output)
}

// Delay line suitable for use in feedback systems without causing deadlock.
func delay(ctx sound.Context, input chan float64, length uint) (output chan float64) {
	output = make(chan float64, ctx.StreamBufferSize)
	
	go func() {
		buffer := make([]float64, length)
		pos := uint(0)
		
		for {
			output <- buffer[pos]
			buffer[pos] = <-input
			pos = (pos + 1) % length
		}
	}()
	
	return output
}

// input should be finite and preferably short (e.g. one cycle of a triangle wave)
func KarplusStrong(ctx sound.Context, input chan float64, delaySamples uint, cutoff float64, decay float64) (output chan float64) {
	feedback := make(chan float64, ctx.StreamBufferSize)
	
	// Mix the input with the feedback.
	output = ctx.Add(input, feedback)
	
	// Fork off a copy of the output.
	output, outputCopy := ctx.Fork2(output)
	
	// The copy is first passed through a delay line...
	outputCopy = delay(ctx, outputCopy, delaySamples)
	
	// ...then filtered...
	//outputCopy = filter.Chebyshev(ctx, outputCopy, filter.LowPass, cutoff, 0.5, 2)
	outputCopy = filter.RC(ctx, outputCopy, filter.LowPass, cutoff)
	
	// ...and finally attenuated slightly.
	outputCopy = ctx.Mul(outputCopy, ctx.Const(decay))
	
	// The filtered output copy is fed back into the system.
	go pipe(outputCopy, feedback)
	
	return output
}

func KarplusStrongTriangle(ctx sound.Context, frequency, cutoff, decay float64) (output chan float64) {
	delaySamples := uint((1.0 / frequency) * ctx.SampleRate)
	wave := ctx.Triangle(ctx.Const(frequency))
	input := ctx.Take(wave, delaySamples, true)
	return KarplusStrong(ctx, input, delaySamples, cutoff, decay)
}

func KarplusStrongSaw(ctx sound.Context, frequency, cutoff, decay float64) (output chan float64) {
	delaySamples := uint((1.0 / frequency) * ctx.SampleRate)
	wave := ctx.Saw(ctx.Const(frequency))
	input := ctx.Take(wave, delaySamples, true)
	return KarplusStrong(ctx, input, delaySamples, cutoff, decay)
}

func KarplusStrongNoise(ctx sound.Context, seed int64, length uint, cutoff, decay float64) (output chan float64) {
	input := ctx.Take(ctx.RandomNoise(seed), length, true)
	return KarplusStrong(ctx, input, length, cutoff, decay)
}

func FilterFormant(ctx sound.Context, input chan float64, centre, q, gain float64) (output chan float64) {
	lowerCutoff := (centre - (centre / q)) / 0.772
	upperCutoff := (centre + (centre / q)) / 1.29
	
	input, formant := ctx.Fork2(input)
	formant = filter.Chebyshev(ctx, formant, filter.HighPass, lowerCutoff, 0.5, 2)
	formant = filter.Chebyshev(ctx, formant, filter.LowPass, upperCutoff, 0.5, 2)
	formant = ctx.Mul(formant, ctx.Const(gain - 1.0))
	return ctx.Add(input, formant)
}

func main() {
	ctx := sound.DefaultContext
	
	/*
	seq := sound.NewSequencer(ctx)
	f0 := 110.0
	seq.Add(0,                 KarplusStrongSaw(ctx, f0*1.00, f0*1.00 * 16, 0.98))
	seq.Add(time.Second*12/40, KarplusStrongSaw(ctx, f0*1.25, f0*1.25 * 16, 0.98))
	seq.Add(time.Second*24/40, KarplusStrongSaw(ctx, f0*1.50, f0*1.50 * 16, 0.98))
	seq.Add(time.Second*36/40, KarplusStrongSaw(ctx, f0*2.00, f0*2.00 * 16, 0.98))
	seq.Add(time.Second*37/40, KarplusStrongSaw(ctx, f0*2.50, f0*2.50 * 16, 0.98))
	seq.Add(time.Second*38/40, KarplusStrongSaw(ctx, f0*3.00, f0*3.00 * 16, 0.98))
	seq.Add(time.Second*39/40, KarplusStrongSaw(ctx, f0*4.00, f0*4.00 * 16, 0.98))
	stream := seq.Play()
	*/
	
	stream := KarplusStrongSaw(ctx, 100.0, 1000.0, 0.98)
	stream = filter.RC(ctx, stream, filter.LowPass, 300.0)
	
	stream = ctx.TakeDuration(stream, time.Second * 3/2, false)
	stream = ctx.MulInf(stream, ctx.Const(0.1))
	
	frontend.Main(ctx, stream)
}
