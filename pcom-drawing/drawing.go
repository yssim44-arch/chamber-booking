package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"math"
	"sort"
	"strconv"
	"strings"
)

// All export shapes use this drawing coordinate system. PPTX text, lines and tables
// remain editable rather than being flattened into a picture.
const drawingWidth, drawingHeight = 1400.0, 900.0
const drawingEMU = 9144.0
const ink = "203247"
const muted = "617184"
const accent = "087F8C"
const lineColor = "485D70"

type drawElement struct {
	kind               string
	x, y, w, h         float64
	x2, y2             float64
	text, fill, stroke string
	size               float64
	bold               bool
	align              string
	dash, arrow        bool
}
type drawPage struct{ elements []drawElement }

func (p *drawPage) rect(x, y, w, h float64, fill, stroke string) {
	p.elements = append(p.elements, drawElement{kind: "rect", x: x, y: y, w: w, h: h, fill: fill, stroke: stroke})
}
func (p *drawPage) text(x, y, w, h float64, text string, size float64, bold bool, color, align string) {
	p.elements = append(p.elements, drawElement{kind: "text", x: x, y: y, w: w, h: h, text: text, size: size, bold: bold, fill: color, align: align})
}
func (p *drawPage) line(x, y, x2, y2 float64, color string, dash, arrow bool) {
	p.elements = append(p.elements, drawElement{kind: "line", x: x, y: y, x2: x2, y2: y2, stroke: color, dash: dash, arrow: arrow})
}
func (p *drawPage) ellipse(x, y, w, h float64, fill, stroke string) {
	p.elements = append(p.elements, drawElement{kind: "ellipse", x: x, y: y, w: w, h: h, fill: fill, stroke: stroke})
}
func (p *drawPage) dot(x, y float64) { p.ellipse(x-3, y-3, 6, 6, accent, accent) }
func (p *drawPage) dim(x1, x2, y float64, label string) {
	p.line(x1, y-7, x1, y+8, "8799AA", false, false)
	p.line(x2, y-7, x2, y+8, "8799AA", false, false)
	p.line(x1, y, x2, y, accent, false, true)
	w := float64(len([]rune(label)))*9 + 24
	p.rect((x1+x2-w)/2, y-14, w, 27, "FFFFFF", "")
	p.text((x1+x2-w)/2, y-14, w, 27, label, 14, true, accent, "center")
}
func drawingRole(m Model) string {
	if m.RoleLabel != "" {
		return m.RoleLabel
	}
	if m.Config.Role == "S" {
		return "SLAVE"
	}
	return "MASTER"
}
func drawingCurrent(m Model) string {
	v := strings.TrimLeft(m.Config.FilterCurrent, "0")
	if v == "" {
		v = "3"
	}
	return v + "A"
}
func drawingConnector(m Model, function string) Connector {
	for _, c := range m.Connectors {
		if strings.EqualFold(c.Function, function) {
			return c
		}
	}
	for _, c := range m.Connectors {
		if strings.Contains(strings.ToUpper(c.Function), function) {
			return c
		}
	}
	return Connector{Function: function, Model: "커넥터 없음", Status: "open"}
}
func drawingHasConnector(c Connector) bool {
	return c.Code != "0" && c.Status != "open" && c.Status != "none" && c.Model != "" && c.Model != "커넥터 없음"
}
func drawingModel(c Connector) string {
	if !drawingHasConnector(c) {
		return "개방단 · 커넥터 없음"
	}
	return c.Model
}
func connectorBlock(p *drawPage, x, y, w, h float64, c Connector, label string) {
	if drawingHasConnector(c) {
		p.rect(x, y, w, h, "F2F7FA", lineColor)
		p.rect(x+w-15, y+10, 8, h-20, "D0DCE4", lineColor)
		n := c.Pins
		if n > 12 {
			n = 12
		}
		if n < 2 {
			n = 3
		}
		for i := 0; i < n; i++ {
			dy := y + 9 + float64(i)*(h-18)/float64(n-1)
			p.line(x+7, dy, x+20, dy, lineColor, false, false)
		}
		p.text(x+24, y+7, w-45, 24, label, 15, true, ink, "center")
		if c.Pins > 0 {
			label := fmt.Sprintf("%d PIN", c.Pins)
			if strings.EqualFold(c.Color, "black") {
				label += " · BLACK"
			}
			p.text(x+24, y+30, w-45, 19, label, 10, false, muted, "center")
		}
	} else {
		for i := 0; i < 3; i++ {
			p.line(x+w-28, y+h/2-8+float64(i)*8, x+w+2, y+h/2-8+float64(i)*8, lineColor, false, false)
		}
		p.text(x, y, w-35, h, label+"\n개방단", 13, true, muted, "right")
	}
}
func pageFrame(m Model, title string, page, total int) *drawPage {
	p := &drawPage{}
	p.rect(0, 0, drawingWidth, drawingHeight, "FFFFFF", "")
	p.rect(26, 26, drawingWidth-52, drawingHeight-52, "FFFFFF", ink)
	p.rect(27, 27, drawingWidth-54, 72, "F3F7FA", "")
	p.text(48, 36, 900, 30, title, 25, true, ink, "left")
	p.text(49, 68, 1100, 19, m.Code, 14, true, accent, "left")
	p.text(1110, 37, 235, 28, "pCOM  /  8 BIT", 18, true, ink, "right")
	p.text(1110, 69, 235, 18, fmt.Sprintf("%02d / %02d", page, total), 12, false, muted, "right")
	p.line(27, 99, drawingWidth-27, 99, ink, false, false)
	p.line(27, 813, drawingWidth-27, 813, ink, false, false)
	p.text(48, 826, 865, 19, "DWG.  "+m.Code, 12, true, ink, "left")
	p.text(48, 849, 885, 17, "개략 연결도 · 치수 비례 아님  |  배선 정의와 검토 항목은 상세 시트 참조", 11, false, muted, "left")
	p.line(957, 813, 957, 873, ink, false, false)
	p.text(973, 823, 180, 18, "REV.  "+m.Config.Revision, 12, true, ink, "left")
	p.text(973, 846, 180, 17, "DATE  "+m.Config.Date, 11, false, muted, "left")
	p.line(1162, 813, 1162, 873, ink, false, false)
	p.text(1172, 823, 169, 18, drawingRole(m), 16, true, accent, "center")
	p.text(1172, 847, 169, 16, "CABLE ASSY", 10, false, muted, "center")
	return p
}
func diagramPage(m Model, page, total int) *drawPage {
	p := pageFrame(m, "pCOM 8BIT 케이블 · 연결 개략도", page, total)
	role := drawingRole(m)
	serial := "RS-232"
	if m.Config.UserSerial == "5" {
		serial = "RS-485"
	}
	p.text(50, 112, 1110, 24, role+"  /  "+m.ProfileLabel+"  /  USER "+serial+"  /  FILTER "+drawingCurrent(m), 16, true, ink, "left")
	p.text(50, 141, 1200, 19, "도면번호의 0은 커넥터 없음 · 길이가 지정된 개방 선재는 표시", 11, false, muted, "left")
	// pCOM-side interfaces, shown as logical housings rather than mechanical artwork.
	p.rect(1085, 196, 235, 416, "F6F9FB", lineColor)
	p.text(1105, 210, 194, 25, "pCOM · "+role, 19, true, ink, "center")
	p.text(1105, 239, 194, 18, "본체 측 커넥터", 12, false, muted, "center")
	for i, s := range []struct {
		title, sub string
		y          float64
	}{{"① 15 PIN", "51021-1500", 277}, {"② 12 PIN", "51021-1200", 367}, {"⑤ 5 PIN", "51021-0500", 520}} {
		p.rect(1110, s.y, 185, 59, "FFFFFF", lineColor)
		p.text(1120, s.y+5, 165, 23, s.title, 17, true, ink, "center")
		p.text(1120, s.y+31, 165, 18, s.sub, 12, false, muted, "center")
		if i == 0 {
			p.text(1096, 341, 211, 17, "IN1–8 · VIN · SELECT · MODE", 10, false, muted, "center")
		}
	}
	p.text(1096, 436, 211, 17, "OUT1–8 · GO1 · ES · TRIGGER", 10, false, muted, "center")
	p.text(1096, 585, 211, 17, "USER / PLC", 10, false, muted, "center")
	// I/O bundle.
	p.dim(268, 1039, 192, fmt.Sprintf("kkk  %d mm", m.Config.Lengths.IO))
	io := m.Config.IOConnector
	if io == "1" || io == "F" {
		c := drawingConnector(m, "I/O")
		p.rect(78, 222, 164, 111, "F2F7FA", lineColor)
		p.text(93, 231, 135, 23, "I/O 26 PIN", 17, true, ink, "center")
		p.text(85, 263, 150, 34, drawingModel(c), 12, false, ink, "center")
		p.text(85, 303, 150, 18, "IN + OUT", 11, true, accent, "center")
		p.line(242, 277, 1018, 277, lineColor, false, false)
	} else {
		ci := drawingConnector(m, "I/O IN")
		co := drawingConnector(m, "I/O OUT")
		if io == "0" {
			ci = Connector{Code: "0"}
			co = Connector{Code: "0"}
		}
		connectorBlock(p, 78, 220, 164, 60, ci, "IN")
		connectorBlock(p, 78, 311, 164, 60, co, "OUT")
		p.text(62, 283, 204, 17, drawingModel(ci), 11, false, muted, "center")
		p.text(62, 375, 204, 17, drawingModel(co), 11, false, muted, "center")
		p.line(242, 250, 290, 250, lineColor, false, false)
		p.line(290, 250, 290, 277, lineColor, false, false)
		p.line(242, 341, 290, 341, lineColor, false, false)
		p.line(290, 341, 290, 277, lineColor, false, false)
		p.dot(290, 277)
		p.line(290, 277, 1018, 277, lineColor, false, false)
	}
	p.line(1018, 277, 1018, 305, lineColor, false, false)
	p.line(1018, 305, 1110, 305, lineColor, false, false)
	p.line(1018, 277, 1044, 277, lineColor, false, false)
	p.line(1044, 277, 1044, 396, lineColor, false, false)
	p.line(1044, 396, 1110, 396, lineColor, false, false)
	p.rect(560, 263, 116, 28, "FFFFFF", lineColor)
	p.text(566, 264, 104, 25, role, 13, true, ink, "center")
	p.text(663, 285, 310, 18, "2464 · 28AWG · 40C", 11, false, muted, "right")
	// MAINT Serial branches into source housing 1.
	maint := drawingConnector(m, "MAINT")
	connectorBlock(p, 332, 413, 178, 63, maint, "MAINT")
	p.text(306, 483, 233, 20, drawingModel(maint), 11, false, muted, "center")
	p.line(510, 444, 994, 444, lineColor, false, false)
	p.line(994, 444, 994, 319, lineColor, false, false)
	p.line(994, 319, 1110, 319, lineColor, false, false)
	p.dim(530, 974, 416, fmt.Sprintf("lll  %d mm", m.Config.Lengths.Maint))
	p.text(681, 452, 295, 18, "① GND / TX0 / RX0", 11, false, muted, "right")
	// USER and PLC are separate legs from source 5-pin housing.
	if m.Config.Lengths.User > 0 {
		user := drawingConnector(m, "USER")
		connectorBlock(p, 332, 530, 178, 63, user, "USER")
		p.text(306, 599, 233, 19, drawingModel(user), 11, false, muted, "center")
		p.line(510, 561, 1110, 561, lineColor, false, false)
		p.dim(530, 1056, 532, fmt.Sprintf("mmm  %d mm", m.Config.Lengths.User))
		p.text(691, 570, 371, 18, "⑤ GND / RX2 / TX2  ·  "+serial, 11, false, muted, "right")
	} else {
		p.text(340, 540, 625, 35, "USER 케이블 없음  ·  mmm 000", 15, false, muted, "center")
	}
	p.line(884, 561, 884, 650, lineColor, false, false)
	p.dot(884, 561)
	p.line(510, 650, 884, 650, lineColor, false, false)
	p.rect(394, 636, 116, 28, "FFFFFF", lineColor)
	p.text(399, 637, 106, 26, "PLC", 13, true, ink, "center")
	p.dim(530, 864, 622, fmt.Sprintf("nnn  %d mm", m.Config.Lengths.PLC))
	p.text(530, 657, 320, 18, "⑤ PLC 24V / PLC GND", 11, false, muted, "center")
	// Power cable, distinct ends and filter. PLC pair branches into filter.
	p.line(394, 650, 365, 650, lineColor, false, false)
	p.line(365, 650, 365, 735, lineColor, false, false)
	p.line(365, 735, 642, 735, lineColor, false, false)
	p.dot(365, 735)
	beltLength, boardLength := m.Config.Lengths.Belt, m.Config.Lengths.Board
	if beltLength == 0 {
		beltLength = 110
	}
	if boardLength == 0 {
		boardLength = 110
	}
	p.dim(270, 620, 691, fmt.Sprintf("BELT %d mm", beltLength))
	p.dim(810, 1086, 691, fmt.Sprintf("BOARD %d mm", boardLength))
	belt := drawingConnector(m, "BELT")
	board := drawingConnector(m, "BOARD")
	connectorBlock(p, 78, 704, 164, 60, belt, "BELT")
	p.line(242, 735, 642, 735, lineColor, false, false)
	p.rect(642, 699, 146, 74, "EAF5F5", accent)
	p.text(655, 710, 120, 26, "FILTER", 19, true, accent, "center")
	p.text(655, 739, 120, 24, drawingCurrent(m), 19, true, accent, "center")
	p.line(788, 735, 1114, 735, lineColor, false, false)
	connectorBlock(p, 1114, 704, 190, 60, board, "BOARD")
	p.text(64, 774, 220, 18, drawingModel(belt), 11, false, muted, "center")
	p.text(1081, 774, 245, 18, drawingModel(board), 11, false, muted, "center")
	p.text(420, 748, 198, 17, "18AWG · 2C", 11, false, muted, "center")
	p.text(806, 748, 249, 17, "18AWG · 2C", 11, false, muted, "center")
	for _, function := range []string{"BELT", "BOARD"} {
		descriptions := []string{}
		for _, v := range m.Pins {
			if v.Connector == function {
				descriptions = append(descriptions, fmt.Sprintf("%d=%s", v.Pin, v.Signal))
			}
		}
		if len(descriptions) > 0 {
			x, w := 64.0, 220.0
			if function == "BOARD" {
				x = 1081
				w = 245
			}
			p.text(x, 794, w, 15, strings.Join(descriptions, " / "), 10, true, accent, "center")
		}
	}
	return p
}
func tablePage(m Model, title string, headers []string, rows [][]string, widths []float64, page, total int) *drawPage {
	p := pageFrame(m, title, page, total)
	p.text(50, 112, 1240, 23, drawingRole(m)+"  /  "+m.ProfileLabel, 15, true, ink, "left")
	y := 155.0
	rowH := 29.0
	x0 := 50.0
	for i, row := range append([][]string{headers}, rows...) {
		x := x0
		fill := "FFFFFF"
		color := ink
		bold := false
		if i == 0 {
			fill = ink
			color = "FFFFFF"
			bold = true
		} else if i%2 == 0 {
			fill = "F1F6F8"
		}
		for j, w := range widths {
			p.rect(x, y, w, rowH, fill, "D6E0E6")
			value := ""
			if j < len(row) {
				value = row[j]
			}
			p.text(x+7, y+3, w-14, rowH-6, value, 11.5, bold, color, "left")
			x += w
		}
		y += rowH
	}
	return p
}
func paginateRows(rows [][]string, n int) [][][]string {
	if len(rows) == 0 {
		return [][][]string{{}}
	}
	pages := [][][]string{}
	for start := 0; start < len(rows); start += n {
		end := start + n
		if end > len(rows) {
			end = len(rows)
		}
		pages = append(pages, rows[start:end])
	}
	return pages
}
func buildDrawingPages(m Model) []*drawPage {
	pinRows := [][]string{}
	for _, v := range m.Pins {
		pinRows = append(pinRows, []string{v.Connector, strconv.Itoa(v.Pin), v.Signal, v.Color, v.Target})
	}
	bomRows := [][]string{}
	for _, v := range m.BOM {
		bomRows = append(bomRows, []string{v.Ref, v.Category, v.Model, v.Maker, strconv.Itoa(v.Quantity), v.Note})
	}
	pinPages, bomPages := paginateRows(pinRows, 21), paginateRows(bomRows, 21)
	reviewLines := []string{}
	if m.Source != nil {
		reviewLines = append(reviewLines, "기준 자료")
		reviewLines = append(reviewLines, wrapDrawingText(m.Source.Filename, 90)...)
		reviewLines = append(reviewLines, fmt.Sprintf("슬라이드 %d  /  %s", m.Source.Slide, m.Source.DrawingNumber), "현재 출력 코드  "+m.Code, "")
	}
	if m.Config.Notes != "" {
		reviewLines = append(reviewLines, "작성 메모")
		reviewLines = append(reviewLines, wrapDrawingText(m.Config.Notes, 90)...)
		reviewLines = append(reviewLines, "")
	}
	if len(m.Warnings) > 0 {
		reviewLines = append(reviewLines, "검토 항목")
		for _, w := range m.Warnings {
			reviewLines = append(reviewLines, wrapDrawingText("• "+w.Message, 90)...)
			reviewLines = append(reviewLines, "")
		}
	}
	reviewPages := [][]string{}
	for start := 0; start < len(reviewLines); start += 28 {
		end := start + 28
		if end > len(reviewLines) {
			end = len(reviewLines)
		}
		reviewPages = append(reviewPages, reviewLines[start:end])
	}
	total := 1 + len(pinPages) + len(bomPages) + len(reviewPages)
	pages := []*drawPage{diagramPage(m, 1, total)}
	for i, rows := range pinPages {
		pages = append(pages, tablePage(m, fmt.Sprintf("핀 배선표  %d / %d", i+1, len(pinPages)), []string{"본체/기능", "PIN", "신호명", "선색", "상대 핀 / 처리"}, rows, []float64{145, 70, 235, 220, 630}, len(pages)+1, total))
	}
	for i, rows := range bomPages {
		pages = append(pages, tablePage(m, fmt.Sprintf("부품표 (BOM)  %d / %d", i+1, len(bomPages)), []string{"번호", "구분", "품명 / 사양", "제조사", "수량", "비고"}, rows, []float64{65, 105, 380, 125, 65, 560}, len(pages)+1, total))
	}
	for i, lines := range reviewPages {
		p := pageFrame(m, fmt.Sprintf("사양 근거 및 검토 항목  %d / %d", i+1, len(reviewPages)), len(pages)+1, total)
		p.text(51, 116, 1240, 23, "선택한 사양과 원본의 차이를 보존합니다.", 16, true, ink, "left")
		for j, line := range lines {
			p.text(54, 156+float64(j)*22, 1280, 21, line, 12, false, ink, "left")
		}
		pages = append(pages, p)
	}
	return pages
}
func wrapDrawingText(s string, width int) []string {
	lines := []string{}
	for _, line := range strings.Split(s, "\n") {
		r := []rune(line)
		if len(r) == 0 {
			lines = append(lines, "")
		}
		for len(r) > 0 {
			n := width
			if len(r) < n {
				n = len(r)
			}
			lines = append(lines, string(r[:n]))
			r = r[n:]
		}
	}
	return lines
}
func renderSVG(m Model) string {
	// SVG is the overview sheet; editable PPTX carries all detail tables.
	p := buildDrawingPages(m)[0]
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1400 900" role="img" aria-label="pCOM 8비트 케이블 연결도"><title>` + html.EscapeString(m.Code) + `</title><defs><marker id="dim" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="#087F8C"/></marker></defs>`)
	for _, e := range p.elements {
		fill := "none"
		if e.fill != "" {
			fill = "#" + e.fill
		}
		stroke := "none"
		if e.stroke != "" {
			stroke = "#" + e.stroke
		}
		switch e.kind {
		case "rect", "ellipse":
			if e.kind == "rect" {
				fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="%s" stroke-width="1.3"/>`, e.x, e.y, e.w, e.h, fill, stroke)
			} else {
				fmt.Fprintf(&b, `<ellipse cx="%.1f" cy="%.1f" rx="%.1f" ry="%.1f" fill="%s" stroke="%s"/>`, e.x+e.w/2, e.y+e.h/2, e.w/2, e.h/2, fill, stroke)
			}
		case "line":
			opts := ""
			if e.dash {
				opts += ` stroke-dasharray="7 5"`
			}
			if e.arrow {
				opts += ` marker-start="url(#dim)" marker-end="url(#dim)"`
			}
			fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1.7"%s/>`, e.x, e.y, e.x2, e.y2, stroke, opts)
		case "text":
			anchor := "start"
			x := e.x
			if e.align == "center" {
				anchor = "middle"
				x += e.w / 2
			} else if e.align == "right" {
				anchor = "end"
				x += e.w
			}
			weight := "400"
			if e.bold {
				weight = "700"
			}
			lines := strings.Split(e.text, "\n")
			lh := e.size * 1.3
			y := e.y + e.h/2 - float64(len(lines)-1)*lh/2
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-family="Malgun Gothic, Apple SD Gothic Neo, Noto Sans KR, sans-serif" font-size="%.1f" font-weight="%s" fill="%s" text-anchor="%s" dominant-baseline="central">`, x, y, e.size, weight, fill, anchor)
			for i, line := range lines {
				if i == 0 {
					b.WriteString(html.EscapeString(line))
				} else {
					fmt.Fprintf(&b, `<tspan x="%.1f" dy="%.1f">%s</tspan>`, x, lh, html.EscapeString(line))
				}
			}
			b.WriteString(`</text>`)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}
func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
func emu(v float64) int64 { return int64(math.Round(v * drawingEMU)) }
func pptxText(e drawElement, id int) string {
	align := "l"
	if e.align == "center" {
		align = "ctr"
	} else if e.align == "right" {
		align = "r"
	}
	sz := int(math.Round(e.size * 72))
	bold := "0"
	if e.bold {
		bold = "1"
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<p:sp><p:nvSpPr><p:cNvPr id="%d" name="Text %d"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr><p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/><a:ln><a:noFill/></a:ln></p:spPr><p:txBody><a:bodyPr wrap="none" lIns="0" rIns="0" tIns="0" bIns="0" anchor="ctr"><a:noAutofit/></a:bodyPr><a:lstStyle/>`, id, id, emu(e.x), emu(e.y), emu(e.w), emu(e.h))
	for _, s := range strings.Split(e.text, "\n") {
		fmt.Fprintf(&b, `<a:p><a:pPr algn="%s"><a:lnSpc><a:spcPct val="115000"/></a:lnSpc><a:spcAft><a:spcPts val="0"/></a:spcAft></a:pPr><a:r><a:rPr lang="ko-KR" sz="%d" b="%s" dirty="0"><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:latin typeface="Malgun Gothic"/><a:ea typeface="Malgun Gothic"/><a:cs typeface="Arial"/></a:rPr><a:t>%s</a:t></a:r><a:endParaRPr lang="ko-KR" sz="%d"/></a:p>`, align, sz, bold, e.fill, xmlEscape(s), sz)
	}
	b.WriteString(`</p:txBody></p:sp>`)
	return b.String()
}
func pptxShape(e drawElement, id int) string {
	if e.kind == "text" {
		return pptxText(e, id)
	}
	var b strings.Builder
	if e.kind == "line" {
		x, y := math.Min(e.x, e.x2), math.Min(e.y, e.y2)
		w, h := math.Abs(e.x2-e.x), math.Abs(e.y2-e.y)
		flip := ""
		if e.x2 < e.x {
			flip += ` flipH="1"`
		}
		if e.y2 < e.y {
			flip += ` flipV="1"`
		}
		fmt.Fprintf(&b, `<p:cxnSp><p:nvCxnSpPr><p:cNvPr id="%d" name="Line %d"/><p:cNvCxnSpPr/><p:nvPr/></p:nvCxnSpPr><p:spPr><a:xfrm%s><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="line"><a:avLst/></a:prstGeom><a:ln w="15240"><a:solidFill><a:srgbClr val="%s"/></a:solidFill>`, id, id, flip, emu(x), emu(y), emu(w), emu(h), e.stroke)
		if e.dash {
			b.WriteString(`<a:prstDash val="dash"/>`)
		}
		if e.arrow {
			b.WriteString(`<a:headEnd type="triangle" w="sm" len="sm"/><a:tailEnd type="triangle" w="sm" len="sm"/>`)
		}
		b.WriteString(`</a:ln></p:spPr><p:style><a:lnRef idx="0"><a:schemeClr val="accent1"/></a:lnRef><a:fillRef idx="0"><a:schemeClr val="accent1"/></a:fillRef><a:effectRef idx="0"><a:schemeClr val="accent1"/></a:effectRef><a:fontRef idx="minor"><a:schemeClr val="lt1"/></a:fontRef></p:style></p:cxnSp>`)
	} else {
		geom := "rect"
		if e.kind == "ellipse" {
			geom = "ellipse"
		}
		fmt.Fprintf(&b, `<p:sp><p:nvSpPr><p:cNvPr id="%d" name="Shape %d"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr><p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="%s"><a:avLst/></a:prstGeom>`, id, id, emu(e.x), emu(e.y), emu(e.w), emu(e.h), geom)
		if e.fill == "" {
			b.WriteString(`<a:noFill/>`)
		} else {
			b.WriteString(`<a:solidFill><a:srgbClr val="` + e.fill + `"/></a:solidFill>`)
		}
		if e.stroke == "" {
			b.WriteString(`<a:ln><a:noFill/></a:ln>`)
		} else {
			b.WriteString(`<a:ln w="12700"><a:solidFill><a:srgbClr val="` + e.stroke + `"/></a:solidFill></a:ln>`)
		}
		b.WriteString(`</p:spPr></p:sp>`)
	}
	return b.String()
}

const pptxNS = `xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"`
const xmlStart = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`
const groupTree = `<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>`

func renderPPTX(m Model) ([]byte, error) {
	pages := buildDrawingPages(m)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	add := func(name, content string) error {
		w, e := zw.Create(name)
		if e != nil {
			return e
		}
		_, e = w.Write([]byte(content))
		return e
	}
	files := map[string]string{}
	content := xmlStart + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/><Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/><Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/><Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>`
	pres := xmlStart + `<p:presentation ` + pptxNS + `><p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst><p:sldIdLst>`
	rels := xmlStart + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>`
	for i, p := range pages {
		n := i + 1
		content += fmt.Sprintf(`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, n)
		pres += fmt.Sprintf(`<p:sldId id="%d" r:id="rId%d"/>`, 256+i, n+1)
		rels += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, n+1, n)
		var sl strings.Builder
		sl.WriteString(xmlStart + `<p:sld ` + pptxNS + `><p:cSld><p:spTree>` + groupTree)
		for j, e := range p.elements {
			sl.WriteString(pptxShape(e, j+2))
		}
		sl.WriteString(`</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sld>`)
		files[fmt.Sprintf("ppt/slides/slide%d.xml", n)] = sl.String()
		files[fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", n)] = xmlStart + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/></Relationships>`
	}
	files["[Content_Types].xml"] = content + `</Types>`
	files["ppt/presentation.xml"] = pres + fmt.Sprintf(`</p:sldIdLst><p:sldSz cx="%d" cy="%d" type="custom"/><p:notesSz cx="6858000" cy="9144000"/><p:defaultTextStyle><a:defPPr><a:defRPr lang="ko-KR"/></a:defPPr></p:defaultTextStyle></p:presentation>`, emu(drawingWidth), emu(drawingHeight))
	files["ppt/_rels/presentation.xml.rels"] = rels + `</Relationships>`
	files["_rels/.rels"] = xmlStart + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
	files["docProps/core.xml"] = xmlStart + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>` + xmlEscape(m.Code) + `</dc:title><dc:creator>pCOM Drawing Studio</dc:creator><dc:description>Editable pCOM 8-bit cable schematic, wiring and BOM</dc:description></cp:coreProperties>`
	files["docProps/app.xml"] = xmlStart + fmt.Sprintf(`<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>pCOM Drawing Studio</Application><Slides>%d</Slides><PresentationFormat>Custom</PresentationFormat></Properties>`, len(pages))
	files["ppt/slideMasters/slideMaster1.xml"] = xmlStart + `<p:sldMaster ` + pptxNS + `><p:cSld name="pCOM"><p:spTree>` + groupTree + `</p:spTree></p:cSld><p:clrMap accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" bg1="lt1" bg2="lt2" folHlink="folHlink" hlink="hlink" tx1="dk1" tx2="dk2"/><p:sldLayoutIdLst><p:sldLayoutId id="2147483649" r:id="rId1"/></p:sldLayoutIdLst><p:txStyles><p:titleStyle/><p:bodyStyle/><p:otherStyle/></p:txStyles></p:sldMaster>`
	files["ppt/slideMasters/_rels/slideMaster1.xml.rels"] = xmlStart + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/></Relationships>`
	files["ppt/slideLayouts/slideLayout1.xml"] = xmlStart + `<p:sldLayout ` + pptxNS + ` type="blank" preserve="1"><p:cSld name="Blank"><p:spTree>` + groupTree + `</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sldLayout>`
	files["ppt/slideLayouts/_rels/slideLayout1.xml.rels"] = xmlStart + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/></Relationships>`
	files["ppt/theme/theme1.xml"] = pptxTheme()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := add(name, files[name]); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func pptxTheme() string {
	var b strings.Builder
	b.WriteString(xmlStart + `<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="pCOM"><a:themeElements><a:clrScheme name="pCOM">`)
	for _, c := range []struct{ k, v string }{{"dk1", "203247"}, {"lt1", "FFFFFF"}, {"dk2", "485D70"}, {"lt2", "F3F7FA"}, {"accent1", "087F8C"}, {"accent2", "4177B6"}, {"accent3", "56A3A6"}, {"accent4", "D9A441"}, {"accent5", "7583A0"}, {"accent6", "B9795D"}, {"hlink", "0563C1"}, {"folHlink", "954F72"}} {
		b.WriteString(`<a:` + c.k + `><a:srgbClr val="` + c.v + `"/></a:` + c.k + `>`)
	}
	b.WriteString(`</a:clrScheme><a:fontScheme name="pCOM"><a:majorFont><a:latin typeface="Malgun Gothic"/><a:ea typeface="Malgun Gothic"/><a:cs typeface="Arial"/></a:majorFont><a:minorFont><a:latin typeface="Malgun Gothic"/><a:ea typeface="Malgun Gothic"/><a:cs typeface="Arial"/></a:minorFont></a:fontScheme><a:fmtScheme name="pCOM"><a:fillStyleLst>`)
	for i := 0; i < 3; i++ {
		b.WriteString(`<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>`)
	}
	b.WriteString(`</a:fillStyleLst><a:lnStyleLst>`)
	for _, w := range []string{"6350", "12700", "19050"} {
		b.WriteString(`<a:ln w="` + w + `" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/><a:miter lim="800000"/><a:headEnd type="none" w="med" len="med"/><a:tailEnd type="none" w="med" len="med"/></a:ln>`)
	}
	b.WriteString(`</a:lnStyleLst><a:effectStyleLst><a:effectStyle><a:effectLst/></a:effectStyle><a:effectStyle><a:effectLst/></a:effectStyle><a:effectStyle><a:effectLst/></a:effectStyle></a:effectStyleLst><a:bgFillStyleLst>`)
	for i := 0; i < 3; i++ {
		b.WriteString(`<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>`)
	}
	b.WriteString(`</a:bgFillStyleLst></a:fmtScheme></a:themeElements><a:objectDefaults/><a:extraClrSchemeLst/></a:theme>`)
	return b.String()
}
