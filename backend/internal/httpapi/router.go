package httpapi

import (
	"encoding/json"
	"fmt"
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
	mux.HandleFunc("GET /api/v1/courses", func(w http.ResponseWriter, r *http.Request) {
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
		offset := (page - 1) * pageSize
		rows, err := pool.Query(r.Context(), `
			SELECT id, course_code, course_name, professor, semester, department, credits
			FROM courses
			WHERE $1 = '' OR course_code ILIKE '%' || $1 || '%' OR course_name ILIKE '%' || $1 || '%'
				OR professor ILIKE '%' || $1 || '%' OR department ILIKE '%' || $1 || '%'
			ORDER BY course_code, semester, id
			LIMIT $2 OFFSET $3`, query, pageSize, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取课程目录失败"})
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
			var item course
			if err := rows.Scan(&item.ID, &item.CourseCode, &item.CourseName, &item.Professor, &item.Semester, &item.Department, &item.Credits); err != nil {
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
