package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var options = map[string][]Option{
	"role":            {{Value: "M", Label: "MASTER"}, {Value: "S", Label: "SLAVE"}},
	"user_serial":     {{Value: "2", Label: "RS-232"}, {Value: "5", Label: "RS-485"}},
	"maker":           {{Value: "SE", Label: "SE · SEMES"}, {Value: "SF", Label: "SF · SynusTech"}, {Value: "DA", Label: "DA · DAIFUKU"}, {Value: "MU", Label: "MU · Muratec"}, {Value: "SA", Label: "SA · SFA"}, {Value: "GE", Label: "GE · 기존 도면"}, {Value: "MF", Label: "MF · 기존 도면"}, {Value: "GL", Label: "GL · 기존 도면"}, {Value: "LW", Label: "LW · 기존 도면"}},
	"filter_current":  {{Value: "03", Label: "3A"}, {Value: "06", Label: "6A"}},
	"io_connector":    {{Value: "0", Label: "0 · 커넥터 없음 (개방선)"}, {Value: "1", Label: "1 · HIROSE HIF6 26핀"}, {Value: "2", Label: "2 · JST SMR 12핀 ×2 / A Type"}, {Value: "F", Label: "F · MOLEX 501646-2600 / 26핀"}, {Value: "L", Label: "L · JST SMR 12핀 ×2 / B Type"}},
	"maint_connector": {{Value: "0", Label: "0 · 커넥터 없음"}, {Value: "1", Label: "1 · MOLEX 51103-0310 / 3핀"}, {Value: "2", Label: "2 · JST SMR-03V-N / 3핀"}, {Value: "3", Label: "3 · D-SUB 9핀 Female"}},
	"user_connector":  {{Value: "0", Label: "0 · 커넥터 없음"}, {Value: "1", Label: "1 · MOLEX 51103-0310 / 3핀"}, {Value: "2", Label: "2 · JST SMR-03V-N / 3핀"}, {Value: "4", Label: "4 · D-SUB 9핀 Female"}, {Value: "7", Label: "7 · MOLEX 51163-0300 / 3핀"}},
	"board_connector": {{Value: "0", Label: "0 · 커넥터 없음"}, {Value: "1", Label: "1 · MOLEX 5557-02 / 2핀"}, {Value: "4", Label: "4 · MOLEX 51067-0200 / 2핀"}, {Value: "5", Label: "5 · MOLEX 5559-02 / 2핀"}, {Value: "6", Label: "6 · MOLEX 5557-04 / 4핀"}},
	"belt_connector":  {{Value: "0", Label: "0 · 커넥터 없음"}, {Value: "1", Label: "1 · MOLEX 5559-02 / 2핀"}, {Value: "4", Label: "4 · MOLEX 5557-02 / 2핀"}, {Value: "5", Label: "5 · MOLEX 5559-04 / 4핀"}},
	"board_pin_order": {{Value: "24v-gnd", Label: "1번 24V / 2번 GND (일반형)"}, {Value: "gnd-24v", Label: "1번 GND / 2번 24V (SYNUS·LW SLAVE)"}},
}
var presets []Preset

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
type ValidationError struct {
	Detail []FieldError `json:"detail"`
}

func (e ValidationError) Error() string {
	if len(e.Detail) > 0 {
		return e.Detail[0].Message
	}
	return "사양을 확인해 주세요."
}
func defaultConfig() Config {
	return Config{BitCode: "A8", Role: "M", FunctionCode: "0", UserSerial: "2", Maker: "SE", FilterCurrent: "03", IOConnector: "2", MaintConnector: "2", UserConnector: "2", BoardConnector: "1", BeltConnector: "1", BoardPinOrder: "24v-gnd", Lengths: Lengths{IO: 400, Maint: 100, User: 300, PLC: 300, Belt: 110, Board: 110}, Revision: "V1.0", Date: time.Now().In(time.FixedZone("KST", 9*3600)).Format("2006-01-02"), PresetID: "spec-example"}
}
func loadPresets(data []byte) error { return json.Unmarshal(data, &presets) }
func allowed(field, value string) bool {
	for _, o := range options[field] {
		if o.Value == value {
			return true
		}
	}
	return false
}
func validateConfig(c Config) (Config, error) {
	c.Maker = strings.ToUpper(strings.TrimSpace(c.Maker))
	c.Notes = strings.TrimSpace(c.Notes)
	c.Revision = strings.TrimSpace(c.Revision)
	if c.BoardPinOrder == "" {
		c.BoardPinOrder = "24v-gnd"
	}
	if c.Lengths.Belt == 0 {
		c.Lengths.Belt = 110
	}
	if c.Lengths.Board == 0 {
		c.Lengths.Board = 110
	}
	errors := []FieldError{}
	add := func(f, m string) { errors = append(errors, FieldError{f, m}) }
	if c.BitCode != "A8" {
		add("bit_code", "이 프로그램은 A8 · 8비트 도면만 지원합니다.")
	}
	if c.FunctionCode != "0" {
		add("function_code", "지원하는 기능 코드는 0 (I/O + UserSerial + Motion)입니다.")
	}
	fields := []struct{ f, v string }{{"role", c.Role}, {"user_serial", c.UserSerial}, {"filter_current", c.FilterCurrent}, {"io_connector", c.IOConnector}, {"maint_connector", c.MaintConnector}, {"user_connector", c.UserConnector}, {"board_connector", c.BoardConnector}, {"belt_connector", c.BeltConnector}, {"board_pin_order", c.BoardPinOrder}}
	for _, p := range fields {
		if !allowed(p.f, p.v) {
			add(p.f, "지원하지 않는 선택값입니다: "+p.v)
		}
	}
	if !regexp.MustCompile(`^[A-Z]{2}$`).MatchString(c.Maker) {
		add("maker", "VHL Maker 코드는 영문 대문자 두 자리여야 합니다.")
	}
	for _, x := range []struct {
		f           string
		v, min, max int
	}{{"io", c.Lengths.IO, 200, 9990}, {"maint", c.Lengths.Maint, 100, 9990}, {"plc", c.Lengths.PLC, 100, 9990}, {"belt", c.Lengths.Belt, 10, 9990}, {"board", c.Lengths.Board, 10, 9990}} {
		if x.v < x.min || x.v > x.max || x.v%10 != 0 {
			add("lengths."+x.f, fmt.Sprintf("길이는 %d~%dmm, 10mm 단위로 입력해 주세요.", x.min, x.max))
		}
	}
	if c.Lengths.User != 0 && (c.Lengths.User < 100 || c.Lengths.User > 9990 || c.Lengths.User%10 != 0) {
		add("lengths.user", "USER 길이는 0(케이블 없음) 또는 100~9990mm, 10mm 단위입니다.")
	}
	if c.Lengths.User == 0 && c.UserConnector != "0" {
		add("lengths.user", "USER 커넥터를 사용할 때는 케이블 길이를 100mm 이상 입력해 주세요.")
	}
	if len([]rune(c.Revision)) == 0 || len([]rune(c.Revision)) > 24 {
		add("revision", "개정은 1~24자로 입력해 주세요.")
	}
	if _, e := time.Parse("2006-01-02", c.Date); e != nil {
		add("date", "날짜는 YYYY-MM-DD 형식으로 입력해 주세요.")
	}
	if len([]rune(c.Notes)) > 1000 {
		add("notes", "도면 메모는 1,000자 이내로 입력해 주세요.")
	}
	for k, v := range c.ConnectorColors {
		if !contains([]string{"io_in", "io_out", "maint", "user", "belt", "board"}, k) || !contains([]string{"natural", "black"}, v) {
			add("connector_colors", "지원하지 않는 커넥터 색상입니다.")
		}
	}
	if c.PresetID != "" {
		found := false
		for _, p := range presets {
			if p.ID == c.PresetID {
				found = true
			}
		}
		if !found {
			add("preset_id", "존재하지 않는 기준 도면입니다.")
		}
	}
	if len(errors) > 0 {
		return c, ValidationError{errors}
	}
	return c, nil
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func configCode(c Config) string {
	return fmt.Sprintf("CTS-pCOM-A8-%s-%s%s-%s-%s-%s%s%s-%s%s-%03d-%03d-%03d-%03d", c.Role, c.FunctionCode, c.UserSerial, c.Maker, c.FilterCurrent, c.IOConnector, c.MaintConnector, c.UserConnector, c.BoardConnector, c.BeltConnector, c.Lengths.IO/10, c.Lengths.Maint/10, c.Lengths.User/10, c.Lengths.PLC/10)
}
func decodeCode(code string) (Config, []Warning, error) {
	a := strings.Split(strings.TrimSpace(code), "-")
	c := defaultConfig()
	c.PresetID = ""
	warnings := []Warning{}
	fail := func() (Config, []Warning, error) {
		return c, warnings, ValidationError{[]FieldError{{"code", "도면번호 형식은 CTS-pCOM-A8-M-02-SE-03-222-11-040-010-030-030 입니다."}}}
	}
	if len(a) != 13 || a[0] != "CTS" || a[1] != "pCOM" || len(a[4]) != 2 || len(a[7]) != 3 || len(a[8]) != 2 {
		return fail()
	}
	if a[2] == "08" {
		warnings = append(warnings, Warning{"legacy-bit-code", "기존 코드 08은 이번 프로그램의 A8로 변환했습니다. 원본 도면번호는 별도로 보존해 주세요."})
		a[2] = "A8"
	}
	// Exact original presets carry attributes not encoded in the order code:
	// reverse BOARD pins, colors and auxiliary power lengths.
	var matched *Preset
	normalizedCode := strings.Join(a, "-")
	for i := range presets {
		p := &presets[i]
		if configCode(p.Config) == normalizedCode && (matched == nil || p.Config.Date > matched.Config.Date) {
			matched = p
		}
	}
	if matched != nil {
		warnings = append(warnings, Warning{"preset-restored", "기존 도면과 일치하여 전원 핀 순서·색상·보조 치수를 포함한 프리셋을 복원했습니다: " + matched.Label})
		return matched.Config, warnings, nil
	}
	c.BitCode = a[2]
	c.Role = a[3]
	c.FunctionCode = a[4][:1]
	c.UserSerial = a[4][1:]
	c.Maker = a[5]
	c.FilterCurrent = a[6]
	c.IOConnector = a[7][:1]
	c.MaintConnector = a[7][1:2]
	c.UserConnector = a[7][2:]
	c.BoardConnector = a[8][:1]
	c.BeltConnector = a[8][1:]
	vals := []int{}
	for _, s := range a[9:] {
		if !regexp.MustCompile(`^[0-9]{3}$`).MatchString(s) {
			return fail()
		}
		v, _ := strconv.Atoi(s)
		vals = append(vals, v*10)
	}
	c.Lengths.IO = vals[0]
	c.Lengths.Maint = vals[1]
	c.Lengths.User = vals[2]
	c.Lengths.PLC = vals[3]
	warnings = append(warnings, Warning{"polarity-from-code", "도면번호만으로 BOARD 핀 극성과 보조 치수를 확정할 수 없습니다. 기본값 1=24V, 2=GND 및 BELT/BOARD 110mm를 확인해 주세요."})
	c, e := validateConfig(c)
	return c, warnings, e
}
func profileLabel(c Config) string {
	switch c.IOConnector {
	case "0":
		return "개방선 · I/O 커넥터 없음"
	case "1":
		return "HIROSE 26핀 통합"
	case "F":
		return "MOLEX 26핀 통합"
	case "L":
		return "SMR 12핀 ×2 · B Type"
	default:
		return "SMR 12핀 ×2 · A Type"
	}
}
func makeModel(in Config) (Model, error) {
	c, err := validateConfig(in)
	if err != nil {
		return Model{}, err
	}
	m := Model{Code: configCode(c), Config: c, Title: "pCOM 8BIT 케이블", RoleLabel: map[string]string{"M": "MASTER", "S": "SLAVE"}[c.Role], ProfileLabel: profileLabel(c), Warnings: []Warning{}, Summary: []SummaryItem{}, Connectors: []Connector{}, Pins: []Pin{}, BOM: []BOMItem{}}
	warn := func(code, msg string) { m.Warnings = append(m.Warnings, Warning{code, msg}) }
	for _, p := range presets {
		if p.ID == c.PresetID {
			m.Source = p.Source
			for _, n := range p.Notes {
				warn("source-reference", n)
			}
			if p.Source != nil && m.Code != strings.Replace(p.Source.DrawingNumber, "CTS-pCOM-08-", "CTS-pCOM-A8-", 1) {
				warn("modified-source", "기준 도면에서 사양을 변경했습니다. 현재 선택 사양으로 새 개략 연결도를 생성합니다.")
			}
			break
		}
	}
	if !contains([]string{"SE", "DA", "MU", "SF", "SA"}, c.Maker) {
		warn("maker-extension", "VHL Maker "+c.Maker+"는 기본사양 표에 없는 기존 도면/확장 코드입니다.")
	}
	extended := []string{}
	for _, p := range []struct {
		n string
		v int
	}{{"I/O", c.Lengths.IO}, {"MAINT", c.Lengths.Maint}, {"USER", c.Lengths.User}, {"PLC", c.Lengths.PLC}} {
		if p.v > 900 {
			extended = append(extended, p.n)
		}
	}
	if len(extended) > 0 {
		warn("extended-length", strings.Join(extended, ", ")+" 길이가 기본사양의 900mm를 초과합니다. 기존 장척 도면 사례를 지원하는 확장 입력입니다.")
	}
	if c.MaintConnector == "1" || c.UserConnector == "1" {
		warn("molex-part-number", "기본사양의 51103-0310을 사용했습니다. 기존 PPT BOM에는 51103-0300이 있어 실제 채택 품번 확인이 필요합니다.")
	}
	if c.IOConnector == "1" {
		warn("hirose-part-number", "기존 PPT의 HIF6-26D-1.27R을 표시합니다. 기본사양에는 HIF-26D-1.27R로 표기되어 있습니다.")
	}
	if c.IOConnector == "0" || c.MaintConnector == "0" || c.UserConnector == "0" || c.BoardConnector == "0" || c.BeltConnector == "0" {
		warn("open-ends", "코드 0은 완성품 커넥터 없음입니다. 이 도면은 개방선을 표시하며 검사·제거용 임시 커넥터는 출력하지 않습니다.")
	}
	if c.UserSerial == "5" {
		warn("serial-labels", "RS-485 선택은 USER 통신 코드입니다. TX/RX 신호명은 제공된 8비트 도면 표기를 유지하며 A/B 전기적 대응은 새로 추정하지 않습니다.")
	}
	if c.BoardConnector == "6" || c.BeltConnector == "5" {
		warn("four-pin-power", "4핀 전원 하우징의 핀표는 원도면에 정의된 1·2번만 표시합니다. 3·4번은 미정의입니다.")
	}
	warn("drawing-scope", "편집 가능한 개략 연결도와 주요 자재표입니다. 원본 기구 형상 및 전체 제작 BOM을 그대로 복제하지 않습니다.")
	m.Summary = []SummaryItem{{"비트수", "8 BIT"}, {"역할", m.RoleLabel}, {"USER 통신", map[string]string{"2": "RS-232", "5": "RS-485"}[c.UserSerial]}, {"VHL Maker", c.Maker}, {"필터", strings.TrimLeft(c.FilterCurrent, "0") + "A"}, {"BOARD 핀", map[string]string{"24v-gnd": "1=24V / 2=GND", "gnd-24v": "1=GND / 2=24V"}[c.BoardPinOrder]}}
	addConn := func(f, code, model, maker, kind, colorKey string, pins int) {
		status := "connector"
		if code == "0" {
			model = "개방선 · 커넥터 없음"
			maker = "—"
			pins = 0
			status = "open"
		}
		color := "natural"
		if c.ConnectorColors[colorKey] != "" {
			color = c.ConnectorColors[colorKey]
		}
		if color == "black" && maker == "JST" && code != "0" {
			if model == "SMR-12V-N" {
				model = "SMR-12V-B"
			}
			if model == "SMR-03V-N" {
				model = "SMR-3V-B"
			}
		}
		m.Connectors = append(m.Connectors, Connector{Function: f, Code: code, Model: model, Maker: maker, Pins: pins, Kind: kind, Status: status, Color: color})
	}
	if c.IOConnector == "1" || c.IOConnector == "F" {
		model, maker := "HIF6-26D-1.27R", "HIROSE"
		if c.IOConnector == "F" {
			model = "501646-2600"
			maker = "MOLEX"
		}
		addConn("I/O", c.IOConnector, model, maker, "io", "io_in", 26)
	} else {
		addConn("I/O IN", c.IOConnector, "SMR-12V-N", "JST", "io", "io_in", 12)
		addConn("I/O OUT", c.IOConnector, "SMR-12V-N", "JST", "io", "io_out", 12)
	}
	serialModel := func(code string, user bool) (string, string, int) {
		switch code {
		case "1":
			return "51103-0310", "MOLEX", 3
		case "2":
			return "SMR-03V-N", "JST", 3
		case "3", "4":
			return "D-SUB 9P Female", "—", 9
		case "7":
			return "51163-0300", "MOLEX", 3
		}
		return "", "", 0
	}
	a, b, n := serialModel(c.MaintConnector, false)
	addConn("MAINT", c.MaintConnector, a, b, "serial", "maint", n)
	if c.Lengths.User > 0 {
		a, b, n = serialModel(c.UserConnector, true)
		addConn("USER", c.UserConnector, a, b, "serial", "user", n)
	}
	powerModel := func(code string, board bool) (string, int) {
		if board {
			switch code {
			case "1":
				return "5557-02", 2
			case "4":
				return "51067-0200", 2
			case "5":
				return "5559-02", 2
			case "6":
				return "5557-04", 4
			}
		} else {
			switch code {
			case "1":
				return "5559-02", 2
			case "4":
				return "5557-02", 2
			case "5":
				return "5559-04", 4
			}
		}
		return "", 0
	}
	a, n = powerModel(c.BeltConnector, false)
	addConn("BELT", c.BeltConnector, a, "MOLEX", "power", "belt", n)
	a, n = powerModel(c.BoardConnector, true)
	addConn("BOARD", c.BoardConnector, a, "MOLEX", "power", "board", n)
	m.Pins = makePins(c, m.Connectors)
	// Some original SLAVE drawings state NC and peer pins simultaneously.
	// Keep both observations visible instead of resolving that source conflict.
	if contains([]string{"d02-s2", "d03-s2", "d04-s2", "d05-s1"}, c.PresetID) {
		for _, p := range presets {
			if p.ID == c.PresetID && p.Config.IOConnector == c.IOConnector && p.Config.UserConnector == c.UserConnector {
				for i := range m.Pins {
					if m.Pins[i].Connector == "⑤" && m.Pins[i].Pin <= 3 {
						m.Pins[i].Target = "NC 표기 / " + m.Pins[i].Target + " (원문)"
					}
				}
			}
		}
	}
	bom := func(cat, model, maker string, qty int, note string) {
		m.BOM = append(m.BOM, BOMItem{Ref: fmt.Sprintf("%02d", len(m.BOM)+1), Category: cat, Model: model, Maker: maker, Quantity: qty, Note: note})
	}
	bom("본체 커넥터", "51021-1500", "MOLEX", 1, "① IN / VIN / MAINT")
	bom("본체 커넥터", "51021-1200", "MOLEX", 1, "② OUT / 제어 신호")
	bom("본체 커넥터", "51021-0500", "MOLEX", 1, "⑤ USER / PLC")
	for _, x := range m.Connectors {
		if x.Status == "connector" {
			note := x.Function
			if x.Color == "black" {
				note += " · Black"
			}
			bom("외부 커넥터", x.Model, x.Maker, 1, note)
		}
	}
	bom("메인 선재", "2464 28AWG 40C", "Kwangil", 1, "I/O 구간 "+strconv.Itoa(c.Lengths.IO)+"mm")
	aux := 2
	if c.Lengths.User > 0 {
		aux = 3
	}
	bom("보조 선재", "26AWG 3C · 규격/제조사 별도 확인", "—", aux, "MAINT / USER(존재 시) / PLC · 도면별 2464/2095 상이")
	bom("전원 선재", "2464 18AWG 2C", "—", 2, fmt.Sprintf("BELT %dmm / BOARD %dmm · 제조사 도면별 상이", c.Lengths.Belt, c.Lengths.Board))
	bom("필터", strings.TrimLeft(c.FilterCurrent, "0")+"A FILTER · 상세 품번 별도 확인", "—", 1, "단자·튜브·스토퍼는 이 주요 자재표에 미포함")
	return m, nil
}

func makePins(c Config, conns []Connector) []Pin {
	rows := []Pin{}
	add := func(conn, model string, p int, sig, col, target string) {
		rows = append(rows, Pin{conn, model, p, sig, col, target})
	}
	ioTarget := func(sig string) string {
		if c.IOConnector == "0" {
			if sig == "GO2" {
				return "상대 핀 미기재"
			}
			return "개방선"
		}
		target := map[string]string{}
		if c.IOConnector == "1" || c.IOConnector == "F" {
			target = map[string]string{"VIN": "I/O:12", "GND": "I/O:13", "SELECT": "I/O:14", "MODE": "I/O:15", "GO1": "I/O:2", "ES": "I/O:11", "TRIGGER": "I/O:24"}
			for i := 1; i <= 8; i++ {
				target[fmt.Sprintf("IN%d", i)] = fmt.Sprintf("I/O:%d", 15+i)
				target[fmt.Sprintf("OUT%d", i)] = fmt.Sprintf("I/O:%d", 2+i)
			}
		} else {
			target = map[string]string{"VIN": "OUT:11", "GND": "OUT:12", "SELECT": "IN:11", "MODE": "IN:12", "GO1": "OUT:10", "ES": "OUT:9", "TRIGGER": "IN:9"}
			if c.IOConnector == "L" {
				target["GND"] = "OUT:12 + IN:9 (분기)"
				target["SELECT"] = "OUT:9"
				target["MODE"] = "IN:10"
				target["ES"] = "IN:11"
				target["TRIGGER"] = "IN:12"
			}
			for i := 1; i <= 8; i++ {
				target[fmt.Sprintf("IN%d", i)] = fmt.Sprintf("IN:%d", i)
				target[fmt.Sprintf("OUT%d", i)] = fmt.Sprintf("OUT:%d", i)
			}
		}
		if v, ok := target[sig]; ok {
			return v
		}
		return "상대 핀 미기재"
	}
	maint := func(sig string) string {
		if c.MaintConnector == "0" {
			return "MAINT 개방선"
		}
		p := map[string]int{"GND": 3, "TX0": 1, "RX0": 2}
		if c.MaintConnector == "3" {
			p = map[string]int{"GND": 5, "TX0": 2, "RX0": 3}
		}
		return fmt.Sprintf("MAINT:%d", p[sig])
	}
	user := func(sig string) string {
		if c.Lengths.User == 0 {
			return "USER 케이블 없음"
		}
		if c.UserConnector == "0" {
			return "USER 개방선"
		}
		p := map[string]int{"GND": 3, "RX2": 2, "TX2": 1}
		if c.UserConnector == "4" {
			p = map[string]int{"GND": 5, "RX2": 3, "TX2": 2}
		}
		return fmt.Sprintf("USER:%d", p[sig])
	}
	sigs := []string{"VIN", "GND", "GND", "TX0", "RX0", "SELECT", "MODE", "IN8", "IN7", "IN6", "IN5", "IN4", "IN3", "IN2", "IN1"}
	cols := []string{"흰", "검", "빨", "검", "갈", "분-빨2", "분-빨1", "흰-빨4", "흰-빨3", "흰-빨2", "흰-빨1", "흰-검4", "흰-검3", "흰-검2", "흰-검1"}
	for i, s := range sigs {
		t := ioTarget(s)
		if i == 2 || i == 3 || i == 4 {
			t = maint(s)
		}
		add("①", "51021-1500", i+1, s, cols[i], t)
	}
	sigs = []string{"OUT1", "OUT2", "OUT3", "OUT4", "OUT5", "OUT6", "OUT7", "OUT8", "GO1", "GO2", "ES", "TRIGGER"}
	cols = []string{"주-검1", "주-검2", "주-검3", "주-검4", "주-빨1", "주-빨2", "주-빨3", "주-빨4", "분-검1", "분-빨4", "분-검2", "분-빨3"}
	for i, s := range sigs {
		add("②", "51021-1200", i+1, s, cols[i], ioTarget(s))
	}
	for i, s := range []string{"GND", "RX2", "TX2"} {
		add("⑤", "51021-0500", i+1, s, []string{"빨", "갈", "검"}[i], user(s))
	}
	add("⑤", "51021-0500", 4, "PLC GND", "검", "FILTER GND 분기")
	add("⑤", "51021-0500", 5, "PLC 24V", "빨", "FILTER 24V 분기")
	for _, x := range conns {
		if x.Kind != "power" || x.Status != "connector" {
			continue
		}
		s1, s2, col1, col2 := "24V", "GND", "흰", "검"
		if x.Function == "BOARD" && c.BoardPinOrder == "gnd-24v" {
			s1, s2, col1, col2 = "GND", "24V", "검", "흰"
		}
		add(x.Function, x.Model, 1, s1, col1, "FILTER")
		add(x.Function, x.Model, 2, s2, col2, "FILTER")
	}
	return rows
}
