package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"
)

func exportTestModel() Model {
	m := Model{Code: "CTS-pCOM-A8-M-02-SE-03-222-11-040-010-030-030", Config: Config{Role: "M", IOConnector: "2", FilterCurrent: "03", UserSerial: "2", Revision: "V1.0", Date: "2026-10-28", Lengths: Lengths{IO: 400, Maint: 100, User: 300, PLC: 300, Belt: 550, Board: 630}}, RoleLabel: "MASTER", ProfileLabel: "SMR 12핀 ×2 · A Type"}
	for _, f := range []string{"I/O IN", "I/O OUT", "MAINT", "USER", "BELT", "BOARD"} {
		m.Connectors = append(m.Connectors, Connector{Function: f, Code: "2", Model: "SMR-12V-N", Pins: 12, Status: "connector", Color: "natural"})
	}
	for i := 1; i <= 43; i++ {
		m.Pins = append(m.Pins, Pin{Connector: "①", Pin: i, Signal: fmt.Sprintf("SIGNAL_%d", i), Color: "흰", Target: "IN:1"})
	}
	m.BOM = []BOMItem{{Ref: "C1", Category: "커넥터", Model: "51021-1500", Maker: "MOLEX", Quantity: 1, Note: "본체 IN"}}
	return m
}
func checkXML(t *testing.T, name string, data []byte) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		_, e := dec.Token()
		if e == io.EOF {
			return
		}
		if e != nil {
			t.Fatalf("%s is invalid XML: %v", name, e)
		}
	}
}
func TestSVGNoInjectedMarkupAndExplicitDimensions(t *testing.T) {
	m := exportTestModel()
	m.Code = `x<&"<script>alert(1)</script>`
	svg := renderSVG(m)
	checkXML(t, "SVG", []byte(svg))
	if strings.Contains(svg, "<script>") || !strings.Contains(svg, "&lt;script&gt;") {
		t.Fatal("input was not escaped")
	}
	for _, text := range []string{"kkk  400 mm", "lll  100 mm", "mmm  300 mm", "nnn  300 mm", "BELT 550 mm", "BOARD 630 mm"} {
		if !strings.Contains(svg, text) {
			t.Errorf("missing actual dimension %q", text)
		}
	}
	m.Config.UserConnector = "0"
	m.Config.Lengths.User = 0
	if !strings.Contains(renderSVG(m), "USER 케이블 없음") {
		t.Fatal("no-cable case should not draw a USER branch")
	}
}
func TestPPTXEditableAndContainsAllPaginatedContent(t *testing.T) {
	m := exportTestModel()
	m.Config.Notes = strings.Repeat("메모 <&>\n", 65)
	for i := 0; i < 40; i++ {
		m.Warnings = append(m.Warnings, Warning{Code: "review", Message: fmt.Sprintf("검토 내용 %d", i)})
	}
	data, e := renderPPTX(m)
	if e != nil {
		t.Fatal(e)
	}
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		t.Fatal(e)
	}
	all := ""
	slides, shapes := 0, 0
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			checkXML(t, f.Name, b)
		}
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slides++
			all += string(b)
			shapes += strings.Count(string(b), "<p:sp>")
		}
		if strings.HasPrefix(f.Name, "ppt/media/") {
			t.Fatal("editable drawing should not be flattened to an image")
		}
	}
	if slides < 8 {
		t.Fatalf("long notes/review and pin tables not paginated: %d pages", slides)
	}
	if shapes < 150 {
		t.Fatalf("expected editable shapes and text, got %d", shapes)
	}
	for _, v := range []string{"SIGNAL_43", "검토 내용 39", "51021-1500", "BELT 550 mm", "BOARD 630 mm", "&lt;&amp;&gt;"} {
		if !strings.Contains(all, v) {
			t.Errorf("export omitted %q", v)
		}
	}
}
