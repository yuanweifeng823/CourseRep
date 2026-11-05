package httpapi

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"strconv"
)

func registerProfileRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/users/me/courses", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		rows, err := pool.Query(r.Context(), `SELECT c.id,c.course_code,c.course_name,c.professor,c.semester,c.department,c.credits FROM user_completed_courses uc JOIN courses c ON c.id=uc.course_id WHERE uc.user_id=$1 ORDER BY uc.completed_at DESC,c.course_code`, u.ID)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取已修读课程失败"})
			return
		}
		defer rows.Close()
		type course struct {
			ID         int64   `json:"id"`
			CourseCode string  `json:"course_code"`
			CourseName string  `json:"course_name"`
			Professor  string  `json:"professor"`
			Semester   string  `json:"semester"`
			Department string  `json:"department"`
			Credits    float64 `json:"credits"`
		}
		items := make([]course, 0)
		for rows.Next() {
			var x course
			if rows.Scan(&x.ID, &x.CourseCode, &x.CourseName, &x.Professor, &x.Semester, &x.Department, &x.Credits) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取已修读课程失败"})
				return
			}
			items = append(items, x)
		}
		if rows.Err() != nil {
			writeJSON(w, 500, map[string]string{"error": "读取已修读课程失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	})))
	for _, verb := range []string{"POST", "DELETE"} {
		method := verb
		mux.Handle(method+" /api/v1/courses/{courseID}/completion", authenticate(pool, requireVerified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			courseID, ok := pathID(r)
			if !ok {
				writeJSON(w, 400, map[string]string{"error": "课程编号无效"})
				return
			}
			u := userFromContext(r.Context())
			if method == "POST" {
				result, err := pool.Exec(r.Context(), `INSERT INTO user_completed_courses(user_id,course_id) SELECT $1,id FROM courses WHERE id=$2 ON CONFLICT DO NOTHING`, u.ID, courseID)
				if err != nil {
					writeJSON(w, 500, map[string]string{"error": "无法保存已修读课程"})
					return
				}
				if result.RowsAffected() == 0 {
					var exists bool
					if pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM courses WHERE id=$1)`, courseID).Scan(&exists) != nil || !exists {
						writeJSON(w, 404, map[string]string{"error": "课程不存在"})
						return
					}
				}
				writeJSON(w, 200, map[string]string{"status": "saved"})
				return
			}
			if _, err := pool.Exec(r.Context(), `DELETE FROM user_completed_courses WHERE user_id=$1 AND course_id=$2`, u.ID, courseID); err != nil {
				writeJSON(w, 500, map[string]string{"error": "无法移除已修读课程"})
				return
			}
			w.WriteHeader(204)
		}))))
	}
	mux.Handle("GET /api/v1/users/me", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		var nickname string
		if e := pool.QueryRow(r.Context(), `SELECT nickname FROM users WHERE id=$1`, u.ID).Scan(&nickname); e != nil {
			writeJSON(w, 404, map[string]string{"error": "用户不存在"})
			return
		}
		writeJSON(w, 200, map[string]any{"id": u.ID, "nickname": nickname, "status": u.Status, "role": u.Role})
	})))
	mux.Handle("GET /api/v1/users/me/reviews", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		rows, e := pool.Query(r.Context(), `SELECT r.id,r.status,r.semester,r.grading,r.workload,r.assessment,r.usefulness,r.tags,r.body,r.created_at,c.id,c.course_code,c.course_name FROM course_reviews r JOIN courses c ON c.id=r.course_id WHERE r.author_id=$1 ORDER BY r.created_at DESC,r.id DESC LIMIT 100`, u.ID)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取我的评价失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID         int64    `json:"id"`
			Status     string   `json:"status"`
			Semester   string   `json:"semester"`
			Grading    int      `json:"grading"`
			Workload   int      `json:"workload"`
			Assessment int      `json:"assessment"`
			Usefulness int      `json:"usefulness"`
			Tags       []string `json:"tags"`
			Body       string   `json:"body"`
			Created    any      `json:"created_at"`
			CourseID   int64    `json:"course_id"`
			CourseCode string   `json:"course_code"`
			CourseName string   `json:"course_name"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.Status, &x.Semester, &x.Grading, &x.Workload, &x.Assessment, &x.Usefulness, &x.Tags, &x.Body, &x.Created, &x.CourseID, &x.CourseCode, &x.CourseName) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取我的评价失败"})
				return
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items})
	})))
	mux.Handle("GET /api/v1/users/me/posts", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		rows, e := pool.Query(r.Context(), `SELECT p.id,p.course_id,c.course_code,c.course_name,p.title,p.body,p.image_urls,p.topic_tags,p.attendance_year,p.status,p.display_identity,p.created_at FROM course_posts p JOIN courses c ON c.id=p.course_id WHERE p.author_id=$1 ORDER BY p.created_at DESC,p.id DESC LIMIT 100`, u.ID)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取我的帖子失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID         int64    `json:"id"`
			CourseID   int64    `json:"course_id"`
			CourseCode string   `json:"course_code"`
			CourseName string   `json:"course_name"`
			Title      string   `json:"title"`
			Body       string   `json:"body"`
			ImageURLs  []string `json:"image_urls"`
			TopicTags  []string `json:"topic_tags"`
			Year       *int     `json:"attendance_year"`
			YearLabel  string   `json:"attendance_year_label"`
			Status     string   `json:"status"`
			Identity   string   `json:"display_identity"`
			Created    any      `json:"created_at"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.CourseID, &x.CourseCode, &x.CourseName, &x.Title, &x.Body, &x.ImageURLs, &x.TopicTags, &x.Year, &x.Status, &x.Identity, &x.Created) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取我的帖子失败"})
				return
			}
			x.YearLabel = "年份未注明"
			if x.Year != nil {
				x.YearLabel = strconv.Itoa(*x.Year)
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items})
	})))
	mux.Handle("GET /api/v1/users/me/bookmarks", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		rows, e := pool.Query(r.Context(), `SELECT p.id,p.course_id,p.title,p.body,p.attendance_year,p.display_identity,p.created_at,CASE WHEN p.display_identity='anonymous' THEN '同校已验证' ELSE author.nickname END,c.course_code,c.course_name FROM post_bookmarks b JOIN course_posts p ON p.id=b.post_id JOIN courses c ON c.id=p.course_id JOIN users author ON author.id=p.author_id WHERE b.user_id=$1 AND p.status='published' ORDER BY b.created_at DESC,b.post_id DESC LIMIT 100`, u.ID)
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "读取收藏失败"})
			return
		}
		defer rows.Close()
		type item struct {
			ID          int64  `json:"id"`
			CourseID    int64  `json:"course_id"`
			Title       string `json:"title"`
			Body        string `json:"body"`
			Year        *int   `json:"attendance_year"`
			YearLabel   string `json:"attendance_year_label"`
			Identity    string `json:"display_identity"`
			DisplayName string `json:"display_name"`
			Created     any    `json:"created_at"`
			CourseCode  string `json:"course_code"`
			CourseName  string `json:"course_name"`
		}
		items := make([]item, 0)
		for rows.Next() {
			var x item
			if rows.Scan(&x.ID, &x.CourseID, &x.Title, &x.Body, &x.Year, &x.Identity, &x.Created, &x.DisplayName, &x.CourseCode, &x.CourseName) != nil {
				writeJSON(w, 500, map[string]string{"error": "读取收藏失败"})
				return
			}
			x.YearLabel = "年份未注明"
			if x.Year != nil {
				x.YearLabel = strconv.Itoa(*x.Year)
			}
			items = append(items, x)
		}
		writeJSON(w, 200, map[string]any{"items": items})
	})))
}
