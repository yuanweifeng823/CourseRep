package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New creates the HTTP handler for the API. Keep health checks independent
// from product data so the service can start before the empty catalog is loaded.
func New(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	legacyMux := http.NewServeMux()
	if pool != nil {
		if err := syncAdminEmails(context.Background(), pool); err != nil {
			log.Printf("sync admin emails: %v", err)
		}
		registerRoutes(legacyMux, pool)
		registerReviewRoutes(legacyMux, pool)
		registerPostRoutes(legacyMux, pool)
		registerImageRoutes(legacyMux, pool)
		registerCourseImportRoutes(legacyMux, pool)
		registerModerationRoutes(legacyMux, pool)
		registerProfileRoutes(legacyMux, pool)
		registerHomeRoutes(legacyMux, pool)
		registerCourseCorrectionRoutes(legacyMux, pool)
		registerCommunitySettingsRoutes(legacyMux, pool)
	} else {
		unavailable := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "服务暂不可用"})
		})
		legacyMux.Handle("/api/v1/", unavailable)
		legacyMux.Handle("/uploads/", unavailable)
	}
	courseList := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "课程目录服务暂不可用"})
			return
		}
		page, err := queryInt(r, "page", 1, 1, 1_000_000)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page 必须是正整数"})
			return
		}
		pageSize, err := queryInt(r, "page_size", 20, 1, 100)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page_size 必须在 1 到 100 之间"})
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len([]rune(query)) > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "搜索内容最多 100 个字符"})
			return
		}
		department, semester := strings.TrimSpace(r.URL.Query().Get("department")), strings.TrimSpace(r.URL.Query().Get("semester"))
		if len([]rune(department)) > 100 || len([]rune(semester)) > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "筛选内容最多 100 个字符"})
			return
		}
		sortBy := r.URL.Query().Get("sort")
		if sortBy == "" {
			sortBy = "course"
		}
		if sortBy != "course" && sortBy != "rating" && sortBy != "reviews" && sortBy != "posts" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "排序方式无效"})
			return
		}
		offset := (page - 1) * pageSize
		rows, err := pool.Query(r.Context(), `
			SELECT c.id,c.course_code,c.course_name,c.professor,c.semester,c.department,c.credits,
			       COALESCE(rv.rating,0)::float8,COALESCE(rv.review_count,0)::int,COALESCE(pp.post_count,0)::int
			FROM courses c
			LEFT JOIN (SELECT course_id,AVG((grading+workload+assessment+usefulness)/4.0) rating,COUNT(*) review_count FROM course_reviews WHERE status='published' GROUP BY course_id) rv ON rv.course_id=c.id
			LEFT JOIN (SELECT course_id,COUNT(*) post_count FROM course_posts WHERE status='published' GROUP BY course_id) pp ON pp.course_id=c.id
			WHERE ($1='' OR c.course_code ILIKE '%' || $1 || '%' OR c.course_name ILIKE '%' || $1 || '%'
				OR c.professor ILIKE '%' || $1 || '%' OR c.department ILIKE '%' || $1 || '%')
			  AND ($2='' OR c.department=$2) AND ($3='' OR c.semester=$3)
			ORDER BY CASE WHEN $4='rating' THEN rv.rating END DESC NULLS LAST,
			         CASE WHEN $4='reviews' THEN rv.review_count END DESC NULLS LAST,
			         CASE WHEN $4='posts' THEN pp.post_count END DESC NULLS LAST,
			         c.course_code,c.semester,c.id
			LIMIT $5 OFFSET $6`, query, department, semester, sortBy, pageSize, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取课程目录失败"})
			return
		}
		defer rows.Close()
		type course struct {
			ID          int64   `json:"id"`
			CourseCode  string  `json:"course_code"`
			CourseName  string  `json:"course_name"`
			Professor   string  `json:"professor"`
			Semester    string  `json:"semester"`
			Department  string  `json:"department"`
			Credits     float64 `json:"credits"`
			Rating      float64 `json:"rating"`
			ReviewCount int     `json:"review_count"`
			PostCount   int     `json:"post_count"`
		}
		items := make([]course, 0)
		for rows.Next() {
			var item course
			if err := rows.Scan(&item.ID, &item.CourseCode, &item.CourseName, &item.Professor, &item.Semester, &item.Department, &item.Credits, &item.Rating, &item.ReviewCount, &item.PostCount); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取课程目录失败"})
				return
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取课程目录失败"})
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Items    []course `json:"items"`
			Page     int      `json:"page"`
			PageSize int      `json:"page_size"`
		}{Items: items, Page: page, PageSize: pageSize})
	})
	if pool != nil {
		legacyMux.Handle("GET /api/v1/courses", authenticate(pool, canBrowse(courseList)))
		legacyMux.Handle("GET /api/v1/course-facets", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rows, err := pool.Query(r.Context(), `SELECT department,semester FROM courses ORDER BY department,semester`)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": "读取筛选项失败"})
				return
			}
			defer rows.Close()
			departments, semesters := []string{}, []string{}
			departmentSet, semesterSet := map[string]bool{}, map[string]bool{}
			for rows.Next() {
				var d, s string
				if rows.Scan(&d, &s) != nil {
					writeJSON(w, 500, map[string]string{"error": "读取筛选项失败"})
					return
				}
				if !departmentSet[d] {
					departments = append(departments, d)
					departmentSet[d] = true
				}
				if !semesterSet[s] {
					semesters = append(semesters, s)
					semesterSet[s] = true
				}
			}
			if rows.Err() != nil {
				writeJSON(w, 500, map[string]string{"error": "读取筛选项失败"})
				return
			}
			writeJSON(w, 200, map[string]any{"departments": departments, "semesters": semesters})
		}))))
	} else {
		legacyMux.Handle("GET /api/v1/courses", courseList)
	}
	contractMux, _ := newContractAPI(legacyMux)
	mux.Handle("/", contractMux)
	return mux
}

func queryInt(r *http.Request, key string, fallback, min, max int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return n, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// nonNilStrings 把 JSON 中省略的可选字符串数组规范化为空切片，
// 避免 pgx 把 nil 编码成 NULL 而违反数据库的 NOT NULL 约束。
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
