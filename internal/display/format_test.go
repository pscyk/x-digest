package display

import (
	"testing"
	"time"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{10000, "10.0K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{123456789, "123.5M"},
	}
	for _, tt := range tests {
		got := FormatNumber(tt.input)
		if got != tt.expected {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected []string
	}{
		{
			name:     "short text",
			input:    "hello world",
			width:    50,
			expected: []string{"hello world"},
		},
		{
			name:     "wraps at width",
			input:    "hello world foo bar",
			width:    11,
			expected: []string{"hello world", "foo bar"},
		},
		{
			name:     "strips empty lines",
			input:    "line one\n\nline two",
			width:    50,
			expected: []string{"line one", "line two"},
		},
		{
			name:     "truncates beyond 5 lines",
			input:    "a\nb\nc\nd\ne\nf\ng",
			width:    50,
			expected: []string{"a", "b", "c", "d", "e…"},
		},
		{
			name:     "handles CRLF",
			input:    "line one\r\nline two",
			width:    50,
			expected: []string{"line one", "line two"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WordWrap(tt.input, tt.width)
			if len(got) != len(tt.expected) {
				t.Fatalf("WordWrap(%q, %d) returned %d lines, want %d\ngot:  %v\nwant: %v",
					tt.input, tt.width, len(got), len(tt.expected), got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestRelativeTime_Zero(t *testing.T) {
	got := RelativeTime(time.Time{})
	if got != "" {
		t.Errorf("RelativeTime(zero) = %q, want empty", got)
	}
}

func FuzzWordWrap(f *testing.F) {
	seeds := []string{"", "a", "hello world", "one\ntwo\nthree", "a b c d e f g h i j k l m"}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		lines := WordWrap(input, 40)

		if len(lines) > MaxTextLines {
			t.Errorf("WordWrap returned %d lines, max is %d", len(lines), MaxTextLines)
		}

		for i, line := range lines {
			if line == "" {
				t.Errorf("line %d is empty", i)
			}
		}
	})
}
