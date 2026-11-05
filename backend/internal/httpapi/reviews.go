package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type reviewInput struct {
	Grading    int      `json:"grading"`
	Workload   int      `json:"workload"`
	Assessment int      `json:"assessment"`
	Usefulness int      `json:"usefulness"`
	Tags       []string `json:"tags"`
	Body       string   `json:"body"`
}

func registerReviewRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/admin/reviews", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		rows, e := pool.Query(r.Context(), `SELECT r.id,r.course_id,c.course_code,c.course_name,r.semester,r.grading,r.workload,r.assessment,r.usefulness,r.tags,r.body,r.status,r.created_at FROM course_reviews r JOIN courses c ON c.id=r.course_id WHERE r.status IN ('submitted','under_review') ORDER BY r.created_at,r.id LIMIT $1 OFFSET $2`, size, (page-1)*size)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取待审评价失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID         int64     `json:"id"`
			CourseID   int64     `json:"course_id"`
			CourseCode string    `json:"course_code"`
			CourseName string    `json:"course_name"`
			Semester   string    `json:"semester"`
			Grading    int       `json:"grading"`
			Workload   int       `json:"workload"`
			Assessment int       `json:"assessment"`
			Usefulness int       `json:"usefulness"`
			Tags       []string  `json:"tags"`
			Body       string    `json:"body"`
			Status     string    `json:"status"`
			CreatedAt  time.Time `json:"created_at"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.CourseID, &x.CourseCode, &x.CourseName, &x.Semester, &x.Grading, &x.Workload, &x.Assessment, &x.Usefulness, &x.Tags, &x.Body, &x.Status, &x.CreatedAt) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取待审评价失败"})
				return
			}
			items = append(items, x)
		}
		if rows.Err() != nil {
			writeJSON(w, 500, map[string]string{"error": "读取待审评价失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size})
	}))))
	mux.Handle("GET /api/v1/courses/{courseID}", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "课程编号无效"})
			return
		}
		var c struct {
			ID         int64   `json:"id"`
			Code       string  `json:"course_code"`
			Name       string  `json:"course_name"`
			Professor  string  `json:"professor"`
			Semester   string  `json:"semester"`
			Department string  `json:"department"`
			Credits    float64 `json:"credits"`
		}
		err := pool.QueryRow(r.Context(), `SELECT id,course_code,course_name,professor,semester,department,credits FROM courses WHERE id=$1`, id).Scan(&c.ID, &c.Code, &c.Name, &c.Professor, &c.Semester, &c.Department, &c.Credits)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "课程不存在"})
			return
		}
		viewer := userFromContext(r.Context())
		if _, err = pool.Exec(r.Context(), `INSERT INTO course_view_events(user_id,course_id,viewed_on) VALUES($1,$2,CURRENT_DATE) ON CONFLICT DO NOTHING`, viewer.ID, id); err != nil {
			writeJSON(w, 500, map[string]string{"error": "记录课程浏览失败"})
			return
		}
		var count int
		var grading, workload, assessment, usefulness float64
		err = pool.QueryRow(r.Context(), `SELECT count(*),coalesce(avg(grading),0),coalesce(avg(workload),0),coalesce(avg(assessment),0),coalesce(avg(usefulness),0) FROM course_reviews WHERE course_id=$1 AND status='published'`, id).Scan(&count, &grading, &workload, &assessment, &usefulness)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取课程评分失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"course": c, "review_count": count, "ratings": map[string]float64{"grading": roundOne(grading), "workload": roundOne(workload), "assessment": roundOne(assessment), "usefulness": roundOne(usefulness)}})
	}))))
	mux.Handle("GET /api/v1/courses/{courseID}/reviews", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "课程编号无效"})
			return
		}
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
		semester := strings.TrimSpace(r.URL.Query().Get("semester"))
		tag := strings.TrimSpace(r.URL.Query().Get("tag"))
		if len(semester) > 40 || len(tag) > 60 {
			writeJSON(w, 400, map[string]string{"error": "筛选条件过长"})
			return
		}
		rows, e := pool.Query(r.Context(), `SELECT id,semester,grading,workload,assessment,usefulness,tags,body,created_at FROM course_reviews WHERE course_id=$1 AND status='published' AND ($2='' OR semester=$2) AND ($3='' OR $3=ANY(tags)) ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`, id, semester, tag, size, (page-1)*size)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取评价失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID          int64    `json:"id"`
			Semester    string   `json:"semester"`
			Grading     int      `json:"grading"`
			Workload    int      `json:"workload"`
			Assessment  int      `json:"assessment"`
			Usefulness  int      `json:"usefulness"`
			Tags        []string `json:"tags"`
			Body        string   `json:"body"`
			CreatedAt   any      `json:"created_at"`
			DisplayName string   `json:"display_name"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.Semester, &x.Grading, &x.Workload, &x.Assessment, &x.Usefulness, &x.Tags, &x.Body, &x.CreatedAt) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取评价失败"})
				return
			}
			x.DisplayName = "同校已验证"
			items = append(items, x)
		}
		if rows.Err() != nil {
			writeJSON(w, 500, map[string]string{"error": "读取评价失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size})
	}))))
	mux.Handle("POST /api/v1/courses/{courseID}/reviews", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r)
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "课程编号无效"})
			return
		}
		var in reviewInput
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&in) != nil || !validReview(in) {
			writeJSON(w, 400, map[string]string{"error": "评价内容或评分无效"})
			return
		}
		if len(in.Tags) > 10 {
			writeJSON(w, 400, map[string]string{"error": "最多选择 10 个标签"})
			return
		}
		seen := map[string]bool{}
		for _, t := range in.Tags {
			t = strings.TrimSpace(t)
			if t == "" || seen[t] {
				writeJSON(w, 400, map[string]string{"error": "标签无效或重复"})
				return
			}
			seen[t] = true
		}
		var semester string
		if pool.QueryRow(r.Context(), `SELECT semester FROM courses WHERE id=$1`, id).Scan(&semester) != nil {
			writeJSON(w, 404, map[string]string{"error": "课程不存在"})
			return
		}
		var tagCount int
		if len(in.Tags) > 0 {
			if pool.QueryRow(r.Context(), `SELECT count(*) FROM topic_tags WHERE active AND name=ANY($1)`, in.Tags).Scan(&tagCount) != nil || tagCount != len(in.Tags) {
				writeJSON(w, 400, map[string]string{"error": "包含未启用的话题标签"})
				return
			}
		}
		u := userFromContext(r.Context())
		var reviewID int64
		err := pool.QueryRow(r.Context(), `INSERT INTO course_reviews(course_id,author_id,semester,grading,workload,assessment,usefulness,tags,body,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'submitted') RETURNING id`, id, u.ID, semester, in.Grading, in.Workload, in.Assessment, in.Usefulness, in.Tags, strings.TrimSpace(in.Body)).Scan(&reviewID)
		if err != nil {
			writeJSON(w, 409, map[string]string{"error": "该课程学期已有有效评价，或评价无法提交"})
			return
		}
		writeJSON(w, 201, map[string]any{"id": reviewID, "status": "submitted", "message": "评价已提交，审核通过后公开"})
	}))))
	mux.Handle("PUT /api/v1/courses/{courseID}/reviews/{reviewID}", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		courseID, ok := pathID(r)
		reviewID, e := strconv.ParseInt(r.PathValue("reviewID"), 10, 64)
		if !ok || e != nil || reviewID < 1 {
			writeJSON(w, 400, map[string]string{"error": "课程或评价编号无效"})
			return
		}
		var in reviewInput
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&in) != nil || !validReview(in) || len(in.Tags) > 10 {
			writeJSON(w, 400, map[string]string{"error": "评价内容或评分无效"})
			return
		}
		if !validTags(in.Tags) {
			writeJSON(w, 400, map[string]string{"error": "评价标签无效或重复"})
			return
		}
		if len(in.Tags) > 0 {
			var count int
			if pool.QueryRow(r.Context(), `SELECT count(*) FROM topic_tags WHERE active AND name=ANY($1)`, in.Tags).Scan(&count) != nil || count != len(in.Tags) {
				writeJSON(w, 400, map[string]string{"error": "包含未启用的话题标签"})
				return
			}
		}
		u := userFromContext(r.Context())
		tag, e := pool.Exec(r.Context(), `UPDATE course_reviews SET grading=$1,workload=$2,assessment=$3,usefulness=$4,tags=$5,body=$6,status='submitted',updated_at=now() WHERE id=$7 AND course_id=$8 AND author_id=$9 AND status IN ('published','rejected')`, in.Grading, in.Workload, in.Assessment, in.Usefulness, in.Tags, strings.TrimSpace(in.Body), reviewID, courseID, u.ID)
		if e != nil {
			writeJSON(w, 409, map[string]string{"error": "评价无法编辑"})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, 404, map[string]string{"error": "评价不存在或当前不可编辑"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "submitted", "message": "修改已提交审核"})
	}))))
	mux.Handle("POST /api/v1/admin/reviews/{reviewID}/decision", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reviewID, err := strconv.ParseInt(r.PathValue("reviewID"), 10, 64)
		if err != nil || reviewID < 1 {
			writeJSON(w, 400, map[string]string{"error": "评价编号无效"})
			return
		}
		var req struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || (req.Decision != "publish" && req.Decision != "reject" && req.Decision != "remove") || len([]rune(strings.TrimSpace(req.Reason))) < 2 || len([]rune(req.Reason)) > 500 {
			writeJSON(w, 400, map[string]string{"error": "审核决定或处理原因无效"})
			return
		}
		status := "published"
		action := req.Decision
		if req.Decision == "reject" {
			status = "rejected"
		}
		if req.Decision == "remove" {
			status = "removed"
		}
		u := userFromContext(r.Context())
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法处理审核"})
			return
		}
		defer tx.Rollback(r.Context())
		var owner int64
		var oldStatus string
		err = tx.QueryRow(r.Context(), `UPDATE course_reviews SET status=$1,updated_at=now() WHERE id=$2 AND (($3='remove' AND status='published') OR ($3<>'remove' AND status IN ('submitted','under_review'))) RETURNING author_id,status`, status, reviewID, req.Decision).Scan(&owner, &oldStatus)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "待处理评价不存在"})
			return
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO moderation_actions(moderator_id,review_id,action,reason) VALUES($1,$2,$3,$4)`, u.ID, reviewID, action, strings.TrimSpace(req.Reason))
		if err == nil {
			_, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,payload) VALUES($1,'review_moderation',jsonb_build_object('review_id',$2,'status',$3))`, owner, reviewID, status)
		}
		if err != nil || tx.Commit(r.Context()) != nil {
			writeJSON(w, 500, map[string]string{"error": "无法完成审核"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": status})
	}))))
	mux.Handle("GET /api/v1/admin/metrics", authenticate(pool, requireModerator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reviews, posts, activeUsers, reports, teacherComplaints, published int64
		queries := []struct {
			dst *int64
			sql string
		}{{&reviews, `SELECT count(*) FROM course_reviews WHERE status='published'`}, {&posts, `SELECT count(*) FROM course_posts WHERE status='published'`}, {&activeUsers, `SELECT count(DISTINCT user_id) FROM course_view_events WHERE viewed_on>=CURRENT_DATE-29`}, {&reports, `SELECT count(*) FROM reports WHERE created_at>=now()-interval '30 days'`}, {&teacherComplaints, `SELECT count(*) FROM reports WHERE created_at>=now()-interval '30 days' AND reason LIKE 'teacher_claim:%' AND status IN ('open','reviewing')`}, {&published, `SELECT (SELECT count(*) FROM course_reviews WHERE status='published')+(SELECT count(*) FROM course_posts WHERE status='published')`}}
		for _, q := range queries {
			if err := pool.QueryRow(r.Context(), q.sql).Scan(q.dst); err != nil {
				writeJSON(w, 500, map[string]string{"error": "读取运营指标失败"})
				return
			}
		}
		rate := float64(0)
		if published > 0 {
			rate = roundOne(float64(reports) * 100 / float64(published))
		}
		writeJSON(w, 200, map[string]any{"valid_reviews": reviews, "valid_posts": posts, "active_course_viewers_30d": activeUsers, "reports_30d": reports, "report_rate_percent": rate, "open_teacher_complaints_30d": teacherComplaints, "report_rate_definition": "近 30 天举报数 / 当前已发布评价与经验帖数 × 100", "active_user_definition": "近 30 天至少浏览过一门课程详情的去重账号数"})
	}))))
}

func validReview(v reviewInput) bool {
	return v.Grading >= 1 && v.Grading <= 5 && v.Workload >= 1 && v.Workload <= 5 && v.Assessment >= 1 && v.Assessment <= 5 && v.Usefulness >= 1 && v.Usefulness <= 5 && len([]rune(strings.TrimSpace(v.Body))) <= 2000
}
func validTags(tags []string) bool {
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			return false
		}
		seen[tag] = true
	}
	return true
}
func pathID(r *http.Request) (int64, bool) {
	n, e := strconv.ParseInt(r.PathValue("courseID"), 10, 64)
	return n, e == nil && n > 0
}
func canBrowse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u.Status != "verified" && u.Status != "alumni_readonly" {
			writeJSON(w, 403, map[string]string{"error": "需完成学生邮箱验证后浏览课程内容"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func roundOne(v float64) float64 { return float64(int(v*10+0.5)) / 10 }
