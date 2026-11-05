package httpapi

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func TestValidateCourseRows(t *testing.T) {
	rows := [][]string{
		{"course_code", "course_name", "professor", "semester", "department", "credits"},
		{"CS1001", "Intro", "Prof A", "2026A", "CS", "3"},
		{"CS1001", "Duplicate", "Prof B", "2026A", "CS", "3"},
		{"CS2001", "Bad credit", "", "2026B", "CS", "-1"},
	}
	items, issues := validateCourseRows(rows)
	if len(items) != 1 || items[0].CourseCode != "CS1001" {
		t.Fatalf("expected only first row valid, got %#v", items)
	}
	if len(issues) != 2 || issues[0].Row != 3 || issues[1].Row != 4 {
		t.Fatalf("expected duplicate and bad credit row errors, got %#v", issues)
	}
}

func TestParseXLSXInlineStrings(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	entries := map[string]string{
		"xl/workbook.xml":            `<?xml version="1.0"?><workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Courses" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   `<?xml version="1.0"?><worksheet><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>course_code</t></is></c><c r="B1" t="inlineStr"><is><t>credits</t></is></c></row><row r="2"><c r="A2" t="inlineStr"><is><t>CS1001</t></is></c><c r="B2"><v>3</v></c></row></sheetData></worksheet>`,
	}
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = io.WriteString(f, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	rows, err := parseXLSX(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1][0] != "CS1001" || rows[1][1] != "3" {
		t.Fatalf("unexpected parsed workbook: %#v", rows)
	}
}
