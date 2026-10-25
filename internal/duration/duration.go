// MIT License; Copyright Wijnand Modderman-Lenstra (maze.io). See LICENSE.
// Copied from github.com/flanksource/commons v1.59.0:
// https://github.com/flanksource/commons/blob/v1.59.0/duration/duration.go
// Originally forked from https://git.maze.io/go/duration/src/branch/master/duration.go
// Only Duration.String and its required helpers and units are retained.
package duration

import (
	"strings"
	"time"
)

type Duration time.Duration

func (d Duration) String() string {
	if d.Hours() > 24*7 {
		d = d - d%Duration(time.Hour)
	} else if d.Hours() > 24 {
		d = d - d%Duration(time.Minute)
	} else if d.Minutes() > 2 {
		d = d - d%Duration(time.Second)
	} else if d.Nanoseconds() > 2*1000*1000 {
		d = d - d%Duration(time.Millisecond)
	}
	var buf [32]byte
	w := len(buf)
	u := uint64(d)
	neg := d < 0
	if neg {
		u = -u
	}

	if u < uint64(Second) {
		var prec int
		w--
		buf[w] = 's'
		w--
		switch {
		case u == 0:
			return "0s"
		case u < uint64(Microsecond):
			prec = 0
			buf[w] = 'n'
		case u < uint64(Millisecond):
			prec = 3
			w--
			copy(buf[w:], "µ")
		default:
			prec = 6
			buf[w] = 'm'
		}
		w, u = fmtFrac(buf[:w], u, prec)
		w = fmtInt(buf[:w], u)
	} else if u > uint64(Week) {
		w--
		buf[w] = 'h'
		u /= uint64(Hour)
		w = fmtInt(buf[:w], u%24)
		u /= 24
		if u > 0 {
			w--
			buf[w] = 'd'
			w = fmtInt(buf[:w], u%7)
			u /= 7
			if u > 0 {
				w--
				buf[w] = 'w'
				w = fmtInt(buf[:w], u)
			}
		}
	} else if u > uint64(Day) {
		w--
		buf[w] = 'm'
		u /= uint64(Minute)
		w = fmtInt(buf[:w], u%60)
		u /= 60
		if u > 0 {
			w--
			buf[w] = 'h'
			w = fmtInt(buf[:w], u%24)
			u /= 24
			if u > 0 {
				w--
				buf[w] = 'd'
				w = fmtInt(buf[:w], u)
			}
		}
	} else {
		w--
		buf[w] = 's'
		w, u = fmtFrac(buf[:w], u, 9)
		w = fmtInt(buf[:w], u%60)
		u /= 60
		if u > 0 {
			w--
			buf[w] = 'm'
			w = fmtInt(buf[:w], u%60)
			u /= 60
			if u > 0 {
				w--
				buf[w] = 'h'
				w = fmtInt(buf[:w], u)
			}
		}
	}

	if neg {
		w--
		buf[w] = '-'
	}

	result := string(buf[w:])
	// Strip standalone zero minutes and seconds, preserving units such as 30m and 320ms.
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }
	if i := strings.Index(result, "0m"); i >= 0 {
		if (i+2 >= len(result) || result[i+2] != 's') && (i == 0 || !isDigit(result[i-1])) {
			result = result[:i] + result[i+2:]
		}
	}
	if strings.HasSuffix(result, "0s") && len(result) > 2 && !isDigit(result[len(result)-3]) {
		result = result[:len(result)-2]
	}
	return result
}

func fmtFrac(buf []byte, v uint64, prec int) (nw int, nv uint64) {
	w := len(buf)
	print := false
	for i := 0; i < prec; i++ {
		digit := v % 10
		print = print || digit != 0
		if print {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if print {
		w--
		buf[w] = '.'
	}
	return w, v
}

func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

func (d Duration) Nanoseconds() int64 { return int64(d) }

func (d Duration) Hours() float64 {
	hour := d / Hour
	nsec := d % Hour
	return float64(hour) + float64(nsec)*(1e-9/60/60)
}

func (d Duration) Minutes() float64 {
	min := d / Minute
	nsec := d % Minute
	return float64(min) + float64(nsec)*(1e-9/60)
}

var (
	Nanosecond  = Duration(time.Nanosecond)
	Microsecond = Duration(time.Microsecond)
	Millisecond = Duration(time.Millisecond)
	Second      = Duration(time.Second)
	Minute      = Duration(time.Minute)
	Hour        = Duration(time.Hour)
	Day         = Hour * 24
	Week        = Day * 7
)
