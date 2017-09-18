package httpmock

import (
	"bytes"
	"fmt"
	"github.com/montanaflynn/stats"
	"github.com/stretchr/testify/assert"
	"io"
	"math"
	"testing"
	"time"
)

func makeHistogram(buckets, samples []time.Duration) []int {
	histo := make([]int, len(buckets)+1, len(buckets)+1)
	for _, s := range samples {
		counted := false
		for i := range buckets {
			if s < buckets[i] {
				histo[i] += 1
				counted = true
				break
			}
		}
		if !counted {
			histo[len(histo)-1] += 1
		}
	}
	return histo
}

func makeBuckets(size int, max, start time.Duration) []time.Duration {
	interval := time.Duration(math.Floor(float64(max-start)/float64(size) + 0.5))
	size = int((max - start) / interval)
	buckets := make([]time.Duration, size, size)
	var v time.Duration = start
	for i := 0; i < size; i += 1 {
		v += interval
		buckets[i] = v
	}
	return buckets
}

var fractionalBlocks = []rune(" \u258F\u258E\u258D\u258C\u258B\u258A\u2589\u2588")

func printHistogram(w io.Writer, buckets []time.Duration, histo []int, maxWidth int) {
	var max int
	for _, h := range histo {
		if h > max {
			max = h
		}
	}
	scale := float64(maxWidth) / float64(max)
	for i := range histo {
		if i < len(buckets) {
			io.WriteString(w, fmt.Sprintf(" %-8s: ", buckets[i].String()))
		} else {
			io.WriteString(w, fmt.Sprintf(">%-8s: ", buckets[len(buckets)-1].String()))
		}
		blocksCalc := scale * float64(histo[i])
		blocks := int(math.Floor(blocksCalc))
		blocksFractionValue := int(math.Ceil((blocksCalc - float64(blocks)) / 0.125))
		for i = 0; i < blocks; i++ {
			io.WriteString(w, "\u2588")
		}
		io.WriteString(w, string(fractionalBlocks[blocksFractionValue]))
		io.WriteString(w, "\n")
	}
}

func TestNormalDelay(t *testing.T) {
	nd := normalDelay{mean: 0.001, stdDev: 0.0002, max: 0.002}
	samples := make([]time.Duration, 0, 100000)
	for i := 0; i < 100000; i++ {
		samples = append(samples, nd.NextWaitTime())
	}
	population := stats.LoadRawData(samples)

	mean, err := population.Mean()
	assert.NoError(t, err)
	assert.InDelta(t, float64(time.Millisecond), mean, 10*float64(time.Microsecond))

	median, err := population.Median()
	assert.NoError(t, err)
	assert.InDelta(t, float64(time.Millisecond), median, 100*float64(time.Microsecond))

	p95, err := population.Percentile(85.0)
	assert.NoError(t, err)
	assert.InDelta(t, float64(time.Millisecond+200*time.Microsecond), p95, 10*float64(time.Microsecond))

	p975, err := population.Percentile(97.5)
	assert.NoError(t, err)
	assert.InDelta(t, float64(time.Millisecond+400*time.Microsecond), p975, 10*float64(time.Microsecond))

	p9985, err := population.Percentile(99.85)
	assert.NoError(t, err)
	assert.InDelta(t, float64(time.Millisecond+600*time.Microsecond), p9985, 10*float64(time.Microsecond))

	buckets := makeBuckets(100, 2*time.Millisecond, 0*time.Microsecond)
	histogram := makeHistogram(buckets, samples)
	sw := bytes.NewBuffer(make([]byte, 0, 512))
	printHistogram(sw, buckets, histogram, 40)
	t.Logf("\n%s", sw.String())
}

func TestNormalDelay2(t *testing.T) {
	nd := normalDelay{mean: 0.0001, stdDev: 0.0002, max: 0.001}
	samples := make([]time.Duration, 0, 100000)
	for i := 0; i < 100000; i++ {
		samples = append(samples, nd.NextWaitTime())
	}
	population := stats.LoadRawData(samples)

	mean, err := population.Mean()
	assert.NoError(t, err)
	assert.InDelta(t, float64(100*time.Microsecond), mean, 10*float64(time.Microsecond))

	median, err := population.Median()
	assert.NoError(t, err)
	assert.InDelta(t, float64(100*time.Microsecond), median, 100*float64(time.Microsecond))

	p95, err := population.Percentile(85.0)
	assert.NoError(t, err)
	assert.InDelta(t, float64(170*time.Microsecond), p95, 10*float64(time.Microsecond))

	p975, err := population.Percentile(97.5)
	assert.NoError(t, err)
	assert.InDelta(t, float64(540*time.Microsecond), p975, 10*float64(time.Microsecond))

	p9985, err := population.Percentile(99.85)
	assert.NoError(t, err)
	assert.InDelta(t, float64(1960*time.Microsecond), p9985, 10*float64(time.Microsecond))

	buckets := makeBuckets(200, 400*time.Microsecond, 0*time.Millisecond)
	histogram := makeHistogram(buckets, samples)
	sw := bytes.NewBuffer(make([]byte, 0, 512))
	printHistogram(sw, buckets, histogram, 40)
	t.Logf("\n%s", sw.String())
}