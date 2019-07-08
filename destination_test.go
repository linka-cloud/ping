package ping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeStats(t *testing.T) {
	assert := assert.New(t)
	const (
		z  = time.Duration(0)
		ms = time.Millisecond
		µs = time.Microsecond
		ns = time.Nanosecond
	)

	testcases := []struct {
		title    string
		results  []time.Duration
		received int
		lost     int

		best   time.Duration
		worst  time.Duration
		mean   time.Duration
		stddev time.Duration
		loss   float64
	}{
		{
			title:    "simplest case",
			results:  []time.Duration{},
			received: 0,
			best:     z, worst: z, mean: z, stddev: z,
		},
		{
			title:    "another simple case",
			results:  []time.Duration{ms},
			received: 1,
			best:     ms, worst: ms, mean: ms, stddev: z,
		},
		{
			title:    "different numbers, manually calculated",
			results:  []time.Duration{ms, 2 * ms},
			received: 2,
			best:     ms,
			worst:    2 * ms,
			mean:     1500 * µs,
			stddev:   500 * µs,
		},
		{
			title:    "wilder numbers",
			results:  []time.Duration{6 * ms, 2 * ms, 14 * ms, 11 * ms},
			received: 6,
			lost:     2,
			best:     2 * ms,
			worst:    14 * ms,
			mean:     8250 * µs, // (6000+2000+14000+11000)/4
			stddev:   4602988,   // 4602988.15988
			loss:     25,        // sent = 6+2
		},
		{
			title:    "verifying captured data",
			received: 50,
			lost:     7,
			loss:     12.28, // 7 / 57

			best:   451327200,
			worst:  492082650,
			mean:   487287379,
			stddev: 9356133,

			results: []time.Duration{
				478427841, 486727913, 489902185, 490369676, 489957386,
				490784152, 491390728, 491012043, 491313203, 489869560,
				488634310, 451590351, 480933928, 451431418, 491046095,
				492017348, 488906398, 490187284, 490733777, 490418928,
				490627269, 490710944, 491339118, 491300740, 490320794,
				489706066, 487735713, 488153523, 490988560, 490293234,
				492082650, 490784586, 488731408, 488008147, 487630508,
				490190288, 490712289, 489931645, 490608008, 490625639,
				491721463, 451327200, 491615584, 490238328, 489234608,
				488510694, 488807517, 489176334, 488981822, 488619758,
			},
		},
	}

	for i, tc := range testcases {
		h := history{received: tc.received, results: tc.results, lost: tc.lost}
		subject := h.compute()

		assert.Equal(tc.best, subject.MinRtt, "test case #%d (%s): best", i, tc.title)
		assert.Equal(tc.worst, subject.MaxRtt, "test case #%d (%s): worst", i, tc.title)
		assert.Equal(tc.mean, subject.AvgRtt, "test case #%d (%s): mean", i, tc.title)
		assert.Equal(tc.stddev, subject.StdDevRtt, "test case #%d (%s): stddev", i, tc.title)
		assert.Equal(tc.received+tc.lost, subject.PacketsSent, "test case #%d (%s): pktSent", i, tc.title)
		assert.InDelta(tc.loss, subject.PacketLoss, 0.01, "test case #%d (%s): pktLoss", i, tc.title)
	}
}

func Test_history_addResult(t *testing.T) {
	h := &history{
		results: make([]time.Duration, 10),
	}
	h.addResult(1, nil)
	assert.Equal(t, []time.Duration{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}, h.results)

	h.addResult(2, nil)
	assert.Equal(t, []time.Duration{2, 1, 0, 0, 0, 0, 0, 0, 0, 0}, h.results)

	h.addResult(3, nil)
	assert.Equal(t, []time.Duration{3, 2, 1, 0, 0, 0, 0, 0, 0, 0}, h.results)

	h.addResult(4, nil)
	assert.Equal(t, []time.Duration{4, 3, 2, 1, 0, 0, 0, 0, 0, 0}, h.results)

	h.addResult(5, nil)
	assert.Equal(t, []time.Duration{5, 4, 3, 2, 1, 0, 0, 0, 0, 0}, h.results)

	h.addResult(6, nil)
	assert.Equal(t, []time.Duration{6, 5, 4, 3, 2, 1, 0, 0, 0, 0}, h.results)

	h.addResult(7, nil)
	assert.Equal(t, []time.Duration{7, 6, 5, 4, 3, 2, 1, 0, 0, 0}, h.results)

	h.addResult(8, nil)
	assert.Equal(t, []time.Duration{8, 7, 6, 5, 4, 3, 2, 1, 0, 0}, h.results)

	h.addResult(9, nil)
	assert.Equal(t, []time.Duration{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, h.results)

	h.addResult(10, nil)
	assert.Equal(t, []time.Duration{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}, h.results)

	h.addResult(11, nil)
	assert.Equal(t, []time.Duration{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}, h.results)

	h.addResult(0, nil)
	assert.Equal(t, []time.Duration{0, 11, 10, 9, 8, 7, 6, 5, 4, 3}, h.results)
}
