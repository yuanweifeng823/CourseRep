package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type correctionFields struct {
	CourseName *string  `json:"course_name,omitempty"`
	Professor  *string  `json:"professor,omitempty"`
	Semester   *string  `json:"semester,omitempty"`
	Department *string  `json:"department,omitempty"`
	Credits    *float64 `json:"credits,omitempty"`
}

func registerCourseCorrectionRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("POST /api/v1/courses/{courseID}/corrections", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		courseID, ok := pathID(r)
		var fields correctionFields
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if !ok || decoder.Decode(&fields) != nil || !validCorrection(fields) {
			writeJSON(w, 400, map[string]string{"error": "纠错字段无效，请至少提供一个有效字段"})
			return
		}
		payload, _ := json.Marshal(fields)
		u := userFromContext(r.Context())
		var id int64
		e := pool.QueryRow(r.Context(), `INSERT INTO course_corrections(course_id,author_id,proposed_fields) SELECT id,$2,$3 FROM courses WHERE id=$1 RETURNING id`, courseID, u.ID, payload).Scan(&id)
		if e != nil {
			writeJSON(w, 404, map[string]string{"error": "课程不存在或无法提交纠错"})
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "status": "open", "message": "纠错建议已提交给运营人员"})
	}))))
	mux.Handle("GET /api/v1/admin/course-corrections", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, e := queryInt(r, "page", 1, 1, 1_000_000)
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "page 无效"})
			return
		}
		size, e := queryInt(r, "page_size", 20, 1, 100)
		if e != nil {
			writeJSON(w, 400, map[string]string{"error": "page_size 无效"})
			return
		}
		rows, e := pool.Query(r.Context(), `SELECT cc.id,cc.course_id,c.course_code,c.course_name,cc.proposed_fields,cc.created_at FROM course_corrections cc JOIN courses c ON c.id=cc.course_id WHERE cc.status='open' ORDER BY cc.created_at,cc.id LIMIT $1 OFFSET $2`, size, (page-1)*size)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取纠错队列失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID         int64           `json:"id"`
			CourseID   int64           `json:"course_id"`
			CourseCode string          `json:"course_code"`
			CourseName string          `json:"course_name"`
			Proposed   json.RawMessage `json:"proposed_fields"`
			Created    any             `json:"created_at"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.CourseID, &x.CourseCode, &x.CourseName, &x.Proposed, &x.Created) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取纠错队列失败"})
				return
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size})
	}))))
	mux.Handle("POST /api/v1/admin/course-corrections/{correctionID}/decision", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, e := strconv.ParseInt(r.PathValue("correctionID"), 10, 64)
		var in struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if e != nil || id < 1 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&in) != nil || (in.Decision != "accept" && in.Decision != "reject") || len([]rune(strings.TrimSpace(in.Reason))) < 2 || len([]rune(in.Reason)) > 500 {
			writeJSON(w, 400, map[string]string{"error": "处理决定或原因无效"})
			return
		}
		u := userFromContext(r.Context())
		tx, e := pool.Begin(r.Context())
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "无法处理纠错"})
			return
		}
		defer tx.Rollback(r.Context())
		var courseID, owner int64
		var raw []byte
		e = tx.QueryRow(r.Context(), `SELECT course_id,author_id,proposed_fields FROM course_corrections WHERE id=$1 AND status='open' FOR UPDATE`, id).Scan(&courseID, &owner, &raw)
		if e != nil {
			writeJSON(w, 404, map[string]string{"error": "纠错建议不存在或已处理"})
			return
		}
		var fields correctionFields
		if json.Unmarshal(raw, &fields) != nil {
			writeJSON(w, 500, map[string]string{"error": "纠错数据无效"})
			return
		}
		if in.Decision == "accept" {
			_, e = tx.Exec(r.Context(), `UPDATE courses SET course_name=coalesce($2,course_name),professor=coalesce($3,professor),semester=coalesce($4,semester),department=coalesce($5,department),credits=coalesce($6,credits),updated_at=now() WHERE id=$1`, courseID, fields.CourseName, fields.Professor, fields.Semester, fields.Department, fields.Credits)
		}
		status := "rejected"
		if in.Decision == "accept" {
			status = "accepted"
		}
		if e == nil {
			_, e = tx.Exec(r.Context(), `UPDATE course_corrections SET status=$1,reviewed_by=$2,reviewed_at=now(),resolution=$3 WHERE id=$4`, status, u.ID, strings.TrimSpace(in.Reason), id)
		}
		if e == nil {
			_, e = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'course_correction_result',jsonb_build_object('correction_id',$2,'status',$3))`, owner, id, status)
		}
		if e != nil || tx.Commit(r.Context()) != nil {
			writeJSON(w, 500, map[string]string{"error": "纠错审核事务失败"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": status})
	}))))
}

func validCorrection(c correctionFields) bool {
	count := 0
	for _, v := range []*string{c.CourseName, c.Professor, c.Semester, c.Department} {
		if v != nil {
			count++
			*v = strings.TrimSpace(*v)
			if len([]rune(*v)) > 200 {
				return false
			}
		}
	}
	if c.CourseName != nil && len([]rune(*c.CourseName)) == 0 {
		return false
	}
	if c.Semester != nil && (len([]rune(*c.Semester)) == 0 || len([]rune(*c.Semester)) > 40) {
		return false
	}
	if c.Department != nil && (len([]rune(*c.Department)) == 0 || len([]rune(*c.Department)) > 120) {
		return false
	}
	if c.Professor != nil && len([]rune(*c.Professor)) > 120 {
		return false
	}
	if c.CourseName != nil && len([]rune(*c.CourseName)) > 200 {
		return false
	}
	if c.Credits != nil {
		count++
		if *c.Credits < 0 || *c.Credits > 99 {
			return false
		}
	}
	return count > 0
}
