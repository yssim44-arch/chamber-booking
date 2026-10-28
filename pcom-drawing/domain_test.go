package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func setupPresets(t *testing.T) {
	t.Helper()
	data, e := assets.ReadFile("data/presets.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = loadPresets(data); e != nil {
		t.Fatal(e)
	}
}
func TestSourcePresetsRoundTrip(t *testing.T) {
	setupPresets(t)
	if len(presets) != 19 {
		t.Fatalf("presets %d", len(presets))
	}
	for _, p := range presets {
		t.Run(p.ID, func(t *testing.T) {
			m, e := makeModel(p.Config)
			if e != nil {
				t.Fatal(e)
			}
			if p.Source != nil {
				want := strings.Replace(p.Source.DrawingNumber, "CTS-pCOM-08-", "CTS-pCOM-A8-", 1)
				if m.Code != want {
					t.Fatalf("%s != %s", m.Code, want)
				}
			}
			if len(m.Pins) < 32 {
				t.Fatal("missing body signals")
			}
			if !strings.Contains(renderSVG(m), m.Code) {
				t.Fatal("drawing has no code")
			}
		})
	}
}
func findPin(t *testing.T, m Model, conn string, pin int) Pin {
	t.Helper()
	for _, p := range m.Pins {
		if p.Connector == conn && p.Pin == pin {
			return p
		}
	}
	t.Fatalf("missing %s pin%d", conn, pin)
	return Pin{}
}
func TestCriticalPinProfiles(t *testing.T) {
	setupPresets(t)
	c := defaultConfig()
	a, _ := makeModel(c)
	if p := findPin(t, a, "②", 12); p.Target != "IN:9" {
		t.Fatal(p)
	}
	c.IOConnector = "L"
	b, _ := makeModel(c)
	if p := findPin(t, b, "①", 2); !strings.Contains(p.Target, "IN:9") {
		t.Fatal(p)
	}
	if p := findPin(t, b, "①", 6); p.Target != "OUT:9" {
		t.Fatal(p)
	}
	c.IOConnector = "F"
	c.Role = "S"
	c.BoardPinOrder = "gnd-24v"
	f, _ := makeModel(c)
	if p := findPin(t, f, "②", 12); p.Target != "I/O:24" {
		t.Fatal(p)
	}
	if p := findPin(t, f, "BOARD", 1); p.Signal != "GND" {
		t.Fatal(p)
	}
	if p := findPin(t, f, "BOARD", 2); p.Signal != "24V" {
		t.Fatal(p)
	}
}
func TestSerialAndOpenEnds(t *testing.T) {
	setupPresets(t)
	c := defaultConfig()
	c.MaintConnector = "3"
	c.UserConnector = "4"
	m, _ := makeModel(c)
	if findPin(t, m, "①", 4).Target != "MAINT:2" || findPin(t, m, "⑤", 2).Target != "USER:3" {
		t.Fatal("DSUB mapping incorrect")
	}
	c.IOConnector = "0"
	c.MaintConnector = "0"
	c.UserConnector = "0"
	c.BoardConnector = "0"
	c.BeltConnector = "0"
	c.Lengths.User = 0
	m, _ = makeModel(c)
	for _, x := range m.Connectors {
		if x.Status != "open" {
			t.Fatal(x)
		}
	}
	if findPin(t, m, "⑤", 1).Target != "USER 케이블 없음" {
		t.Fatal("no-cable missing")
	}
	for _, x := range m.BOM {
		if x.Category == "외부 커넥터" {
			t.Fatal("no connector must not emit external housing")
		}
	}
}
func TestInvalidInputs(t *testing.T) {
	setupPresets(t)
	cases := []struct {
		name string
		f    func(*Config)
	}{{"16bit", func(c *Config) { c.BitCode = "AF" }}, {"step", func(c *Config) { c.Lengths.IO = 405 }}, {"negative", func(c *Config) { c.Lengths.PLC = -20 }}, {"no-cable-with-connector", func(c *Config) { c.Lengths.User = 0 }}, {"unsupported", func(c *Config) { c.IOConnector = "C" }}, {"code-overflow", func(c *Config) { c.Lengths.IO = 10000 }}, {"markup-maker", func(c *Config) { c.Maker = "<script>" }}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := defaultConfig()
			tt.f(&c)
			if _, e := makeModel(c); e == nil {
				t.Fatal("invalid spec accepted")
			}
		})
	}
}
func TestDecodeAndKnownPolarityAmbiguity(t *testing.T) {
	setupPresets(t)
	code := "CTS-pCOM-08-S-02-SF-03-F27-41-054-054-054-024"
	c, w, e := decodeCode(code)
	if e != nil {
		t.Fatal(e)
	}
	if c.BitCode != "A8" || c.BoardPinOrder != "gnd-24v" || len(w) == 0 {
		t.Fatal(c, w)
	}
	_, w, e = decodeCode("CTS-pCOM-A8-S-05-LW-06-F00-11-035-050-050-020")
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, x := range w {
		if x.Code == "polarity-from-code" {
			found = true
		}
	}
	if !found {
		t.Fatal("code cannot determine LW polarity; must report ambiguity")
	}
}
func TestConfigJSONRoundTrip(t *testing.T) {
	setupPresets(t)
	for _, p := range presets {
		b, e := json.Marshal(p.Config)
		if e != nil {
			t.Fatal(e)
		}
		var c Config
		if e = json.Unmarshal(b, &c); e != nil {
			t.Fatal(e)
		}
		if configCode(c) != configCode(p.Config) {
			t.Fatal("config roundtrip")
		}
	}
}

func TestSourceBlackModelsAndNC(t *testing.T) {
	setupPresets(t)
	for _, p := range presets {
		if p.ID == "d04-s1" {
			m, e := makeModel(p.Config)
			if e != nil {
				t.Fatal(e)
			}
			foundIn, foundMaint := false, false
			for _, x := range m.Connectors {
				if x.Function == "I/O IN" {
					foundIn = x.Model == "SMR-12V-B"
				}
				if x.Function == "MAINT" {
					foundMaint = x.Model == "SMR-3V-B"
				}
			}
			if !foundIn || !foundMaint {
				t.Fatal("black part number must match source", m.Connectors)
			}
		}
		if p.ID == "d02-s2" || p.ID == "d03-s2" || p.ID == "d04-s2" || p.ID == "d05-s1" || p.ID == "d10-s2" {
			m, e := makeModel(p.Config)
			if e != nil {
				t.Fatal(e)
			}
			target := findPin(t, m, "⑤", 1).Target
			want := p.ID != "d10-s2"
			if strings.Contains(target, "NC 표기") != want {
				t.Fatal(p.ID, target)
			}
		}
	}
}
