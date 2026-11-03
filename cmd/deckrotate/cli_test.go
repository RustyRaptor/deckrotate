package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	cases := []struct {
		args   string
		method string
		params []any
	}{
		{"lock", "SetLocked", []any{true}},
		{"unlock", "SetLocked", []any{false}},
		{"toggle-lock", "ToggleLock", nil},
		{"mode auto", "SetMode", []any{"auto"}},
		{"mode manual", "SetMode", []any{"manual"}},
		{"mode toggle", "ToggleMode", nil},
		{"rotate cw", "RotateBy", []any{int32(1)}},
		{"rotate ccw", "RotateBy", []any{int32(-1)}},
		{"rotate flip", "RotateBy", []any{int32(2)}},
		{"rotate landscape", "RotateTo", []any{"landscape"}},
		{"rotate portrait-cw", "RotateTo", []any{"portrait-cw"}},
		{"rotate portrait-ccw", "RotateTo", []any{"portrait-ccw"}},
		{"rotate landscape-flipped", "RotateTo", []any{"landscape-flipped"}},
		{"settle 700ms", "SetSettleMs", []any{int32(700)}},
		{"settle 1.5s", "SetSettleMs", []any{int32(1500)}},
		{"prompt-timeout 8s", "SetPromptTimeoutMs", []any{int32(8000)}},
		{"accept", "Accept", nil},
		{"dismiss", "Dismiss", nil},
	}
	for _, c := range cases {
		m, p, err := parseCommand(strings.Fields(c.args))
		if err != nil || m != c.method || !reflect.DeepEqual(p, c.params) {
			t.Errorf("parseCommand(%q) = %q %v %v; want %q %v", c.args, m, p, err, c.method, c.params)
		}
	}
}

func TestParseCommandErrors(t *testing.T) {
	for _, args := range []string{"", "spin", "mode", "mode sometimes", "rotate", "rotate diagonal", "lock now",
		"settle", "settle fast", "settle 1s 2s", "prompt-timeout", "settle 10000h"} {
		if _, _, err := parseCommand(strings.Fields(args)); err == nil {
			t.Errorf("parseCommand(%q) should fail", args)
		}
	}
}

func TestFormatStatus(t *testing.T) {
	got := formatStatus(statusFixture)
	want := "mode:      manual\nlocked:    yes\ndevice:    portrait-cw\nscreen:    landscape\nsuggested: portrait-cw\nimu:       ok\n" +
		"settle:    700ms\nbutton:    8s\n"
	if got != want {
		t.Errorf("formatStatus =\n%s\nwant\n%s", got, want)
	}
}
