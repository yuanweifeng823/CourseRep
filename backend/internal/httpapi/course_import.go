package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxImportBytes = 10 << 20
const maxImportRows = 5000

type importedCourse struct {
	CourseCode string  `json:"course_code"`
	CourseName string  `json:"course_name"`
	Professor  string  `json:"professor"`
	Semester   string  `json:"semester"`
	Department string  `json:"department"`
	Credits    float64 `json:"credits"`
}
type importIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

var cellRefPattern = regexp.MustCompile(`^([A-Z]+)`)

func registerCourseImportRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("POST /api/v1/admin/course-imports/preview", authenticate(pool, requireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes+(1<<20))
		if r.ParseMultipartForm(maxImportBytes) != nil {
			writeJSON(w, 400, map[string]string{"error": "导入文件超过 10 MB 或表单无效"})
			return
		}
		defer r.MultipartForm.RemoveAll()
		f, h, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "请选择 CSV 或 XLSX 文件"})
			return
		}
		defer f.Close()
		if h.Size > maxImportBytes {
			writeJSON(w, 400, map[string]string{"error": "导入文件不能超过 10 MB"})
			return
		}
		data, err := io.ReadAll(io.LimitReader(f, maxImportBytes+1))
		if err != nil || len(data) > maxImportBytes {
			writeJSON(w, 400, map[string]string{"error": "导入文件无法读取"})
			return
		}
		var records [][]string
		ext := strings.ToLower(filepath.Ext(h.Filename))
		if ext == ".csv" {
			records, err = parseCSV(data)
		} else if ext == ".xlsx" {
			records, err = parseXLSX(data)
		} else {
			err = fmt.Errorf("unsupported file type")
		}
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "无法解析文件；请使用 UTF-8 CSV 或标准 XLSX"})
			return
		}
		courses, issues := validateCourseRows(records)
		rowsJSON, _ := json.Marshal(courses)
		issuesJSON, _ := json.Marshal(issues)
		u := userFromContext(r.Context())
		var batchID string
		if err = pool.QueryRow(r.Context(), `INSERT INTO course_import_batches(admin_id,rows,errors) VALUES($1,$2,$3) RETURNING id`, u.ID, rowsJSON, issuesJSON).Scan(&batchID); err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法保存导入预览"})
			return
		}
		writeJSON(w, 201, map[string]any{"batch_id": batchID, "rows": len(records) - 1, "valid_rows": len(courses), "errors": issues, "preview": courses})
	}))))
	mux.Handle("POST /api/v1/admin/course-imports/{batchID}/confirm", authenticate(pool, requireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		batchID := r.PathValue("batchID")
		u := userFromContext(r.Context())
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法确认导入"})
			return
		}
		defer tx.Rollback(r.Context())
		var rowsJSON, errorsJSON []byte
		var status string
		err = tx.QueryRow(r.Context(), `SELECT rows,errors,status FROM course_import_batches WHERE id=$1 AND admin_id=$2 AND expires_at>now() FOR UPDATE`, batchID, u.ID).Scan(&rowsJSON, &errorsJSON, &status)
		if err != nil || status != "pending" {
			writeJSON(w, 404, map[string]string{"error": "导入预览不存在或已处理"})
			return
		}
		var issues []importIssue
		if json.Unmarshal(errorsJSON, &issues) != nil || len(issues) > 0 {
			writeJSON(w, 400, map[string]string{"error": "预览存在错误行，请修正文件后重新预览"})
			return
		}
		var courses []importedCourse
		if json.Unmarshal(rowsJSON, &courses) != nil || len(courses) == 0 {
			writeJSON(w, 400, map[string]string{"error": "预览没有有效课程"})
			return
		}
		for _, c := range courses {
			_, err = tx.Exec(r.Context(), `INSERT INTO courses(course_code,course_name,professor,semester,department,credits) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(course_code,semester) DO UPDATE SET course_name=EXCLUDED.course_name,professor=EXCLUDED.professor,department=EXCLUDED.department,credits=EXCLUDED.credits,updated_at=now()`, c.CourseCode, c.CourseName, c.Professor, c.Semester, c.Department, c.Credits)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": "课程数据未通过数据库校验，未写入任何行"})
				return
			}
		}
		_, err = tx.Exec(r.Context(), `UPDATE course_import_batches SET status='confirmed',rows='[]'::jsonb,errors='[]'::jsonb WHERE id=$1`, batchID)
		if err != nil || tx.Commit(r.Context()) != nil {
			writeJSON(w, 500, map[string]string{"error": "导入事务失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"imported": len(courses), "status": "confirmed"})
	}))))
}

func parseCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = false
	return r.ReadAll()
}
func validateCourseRows(records [][]string) ([]importedCourse, []importIssue) {
	items := make([]importedCourse, 0)
	issues := make([]importIssue, 0)
	if len(records) < 2 {
		return items, []importIssue{{Row: 1, Message: "必须包含字段表头和至少一条数据"}}
	}
	if len(records) > maxImportRows+1 {
		return items, []importIssue{{Row: 0, Message: "最多导入 5000 行"}}
	}
	headers := map[string]int{}
	for i, h := range records[0] {
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))
		headers[name] = i
	}
	required := []string{"course_code", "course_name", "professor", "semester", "department", "credits"}
	for _, h := range required {
		if _, ok := headers[h]; !ok {
			issues = append(issues, importIssue{1, "缺少字段：" + h})
		}
	}
	if len(issues) > 0 {
		return items, issues
	}
	seen := map[string]bool{}
	for ri, row := range records[1:] {
		line := ri + 2
		get := func(k string) string {
			i := headers[k]
			if i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		c := importedCourse{CourseCode: get("course_code"), CourseName: get("course_name"), Professor: get("professor"), Semester: get("semester"), Department: get("department")}
		rawCredits := get("credits")
		bad := false
		for key, value := range map[string]string{"course_code": c.CourseCode, "course_name": c.CourseName, "semester": c.Semester, "department": c.Department, "credits": rawCredits} {
			if value == "" {
				issues = append(issues, importIssue{line, "必填字段为空：" + key})
				bad = true
			}
		}
		if len(c.CourseCode) > 40 || len(c.CourseName) > 200 || len(c.Professor) > 120 || len(c.Semester) > 40 || len(c.Department) > 120 {
			issues = append(issues, importIssue{line, "字段长度超限"})
			bad = true
		}
		if rawCredits != "" {
			n, e := strconv.ParseFloat(rawCredits, 64)
			if e != nil || n < 0 || n > 99 {
				issues = append(issues, importIssue{line, "学分必须是 0 到 99 的数字"})
				bad = true
			} else {
				c.Credits = n
			}
		}
		key := strings.ToUpper(c.CourseCode) + "\x00" + c.Semester
		if seen[key] {
			issues = append(issues, importIssue{line, "文件中课程代码与学期重复"})
			bad = true
		}
		seen[key] = true
		if !bad {
			items = append(items, c)
		}
	}
	return items, issues
}

type xlsxCell struct {
	Ref    string `xml:"r,attr"`
	Type   string `xml:"t,attr"`
	Value  string `xml:"v"`
	Inline struct {
		Text string `xml:"t"`
	} `xml:"is"`
}
type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}
type xlsxSheet struct {
	Rows []xlsxRow `xml:"sheetData>row"`
}

func parseXLSX(data []byte) ([][]string, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if len(z.File) > 256 {
		return nil, fmt.Errorf("too many archive entries")
	}
	var expanded uint64
	for _, f := range z.File {
		expanded += f.UncompressedSize64
		if expanded > 50<<20 {
			return nil, fmt.Errorf("expanded archive too large")
		}
	}
	files := map[string]*zip.File{}
	for _, f := range z.File {
		files[path.Clean(f.Name)] = f
	}
	read := func(name string) ([]byte, error) {
		f := files[name]
		if f == nil {
			return nil, fmt.Errorf("missing %s", name)
		}
		rc, e := f.Open()
		if e != nil {
			return nil, e
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, maxImportBytes))
	}
	var shared []string
	if b, e := read("xl/sharedStrings.xml"); e == nil {
		var ss struct {
			Items []struct {
				Text string `xml:"t"`
				Runs []struct {
					Text string `xml:"t"`
				} `xml:"r"`
			} `xml:"si"`
		}
		if xml.Unmarshal(b, &ss) != nil {
			return nil, fmt.Errorf("invalid shared strings")
		}
		for _, item := range ss.Items {
			v := item.Text
			if len(item.Runs) > 0 {
				v = ""
				for _, run := range item.Runs {
					v += run.Text
				}
			}
			shared = append(shared, v)
		}
	}
	workbook, err := read("xl/workbook.xml")
	if err != nil {
		return nil, err
	}
	var wb struct {
		Sheets []struct {
			Rel string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if xml.Unmarshal(workbook, &wb) != nil || len(wb.Sheets) == 0 {
		return nil, fmt.Errorf("no sheet")
	}
	relID := wb.Sheets[0].Rel
	rels, err := read("xl/_rels/workbook.xml.rels")
	if err != nil {
		return nil, err
	}
	var rel struct {
		Items []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if xml.Unmarshal(rels, &rel) != nil {
		return nil, fmt.Errorf("invalid relationships")
	}
	target := ""
	for _, x := range rel.Items {
		if x.ID == relID {
			target = x.Target
		}
	}
	if target == "" {
		return nil, fmt.Errorf("missing worksheet relationship")
	}
	sheetPath := path.Clean(path.Join("xl", target))
	if strings.HasPrefix(sheetPath, "../") {
		return nil, fmt.Errorf("invalid worksheet path")
	}
	sheetData, err := read(sheetPath)
	if err != nil {
		return nil, err
	}
	var sheet xlsxSheet
	if xml.Unmarshal(sheetData, &sheet) != nil {
		return nil, fmt.Errorf("invalid worksheet")
	}
	rows := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		values := make([]string, 0, len(row.Cells))
		for _, c := range row.Cells {
			m := cellRefPattern.FindStringSubmatch(c.Ref)
			if len(m) < 2 {
				return nil, fmt.Errorf("invalid cell reference")
			}
			col := 0
			for _, ch := range m[1] {
				col = col*26 + int(ch-'A'+1)
			}
			for len(values) < col {
				values = append(values, "")
			}
			value := c.Value
			if c.Type == "s" {
				index, e := strconv.Atoi(value)
				if e != nil || index < 0 || index >= len(shared) {
					return nil, fmt.Errorf("bad shared string")
				}
				value = shared[index]
			} else if c.Type == "inlineStr" {
				value = c.Inline.Text
			}
			values[col-1] = value
		}
		rows = append(rows, values)
		if len(rows) > maxImportRows+1 {
			return nil, fmt.Errorf("too many rows")
		}
	}
	return rows, nil
}
