package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// homeCourse 是首页推荐卡片使用的课程摘要，字段与课程目录保持一致，
// 前端可以直接用它打开课程详情。
type homeCourse struct {
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

type homePost struct {
	ID          int64      `json:"id"`
	CourseID    int64      `json:"course_id"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Year        *int       `json:"attendance_year"`
	YearLabel   string     `json:"attendance_year_label"`
	Identity    string     `json:"display_identity"`
	DisplayName string     `json:"display_name"`
	LikeCount   int        `json:"like_count"`
	CommentCnt  int        `json:"comment_count"`
	ViewCount   int        `json:"view_count"`
	Created     any        `json:"created_at"`
	Course      homeCourse `json:"course"`
}

func registerHomeRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/home", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		department := strings.TrimSpace(r.URL.Query().Get("department"))
		if department == "" {
			if err := pool.QueryRow(r.Context(), `SELECT department FROM users WHERE id=$1`, u.ID).Scan(&department); err != nil {
				writeJSON(w, 500, map[string]string{"error": "读取专业失败"})
				return
			}
			department = strings.TrimSpace(department)
		}
		var valid bool
		if department != "" {
			if err := pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM courses WHERE department=$1)`, department).Scan(&valid); err != nil {
				writeJSON(w, 500, map[string]string{"error": "读取专业失败"})
				return
			}
		}
		if department == "" || !valid {
			// 没有专业，或之前选择的专业已不在课程目录中：让客户端重新选择。
			writeJSON(w, 200, map[string]any{"department": "", "needs_department": true, "top_courses": []homeCourse{}, "trending_posts": []homePost{}})
			return
		}
		courses, err := homeTopCourses(r, pool, department)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取推荐课程失败"})
			return
		}
		posts, err := homeTrendingPosts(r, pool, department)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取热门经验帖失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"department": department, "needs_department": false, "top_courses": courses, "trending_posts": posts})
	}))))
}

func homeTopCourses(r *http.Request, pool *pgxpool.Pool, department string) ([]homeCourse, error) {
	rows, err := pool.Query(r.Context(), `
SELECT c.id,c.course_code,c.course_name,c.professor,c.semester,c.department,c.credits,
       coalesce(agg.rating,0),coalesce(agg.review_count,0),coalesce(pc.post_count,0)
FROM courses c
LEFT JOIN (
    SELECT course_id,avg((grading+workload+assessment+usefulness)/4.0) AS rating,count(*) AS review_count
    FROM course_reviews WHERE status='published' GROUP BY course_id
) agg ON agg.course_id=c.id
LEFT JOIN (
    SELECT course_id,count(*) AS post_count FROM course_posts WHERE status='published' GROUP BY course_id
) pc ON pc.course_id=c.id
WHERE c.department=$1 AND coalesce(agg.review_count,0)>0
ORDER BY coalesce(agg.rating,0) DESC,coalesce(agg.review_count,0) DESC,c.course_code
LIMIT 5`, department)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]homeCourse, 0, 5)
	for rows.Next() {
		var x homeCourse
		if rows.Scan(&x.ID, &x.CourseCode, &x.CourseName, &x.Professor, &x.Semester, &x.Department, &x.Credits, &x.Rating, &x.ReviewCount, &x.PostCount) != nil {
			return nil, err
		}
		x.Rating = roundOne(x.Rating)
		items = append(items, x)
	}
	return items, rows.Err()
}

func homeTrendingPosts(r *http.Request, pool *pgxpool.Pool, department string) ([]homePost, error) {
	rows, err := pool.Query(r.Context(), `
SELECT p.id,p.course_id,p.title,p.body,p.attendance_year,p.display_identity,p.created_at,
       CASE WHEN p.display_identity='anonymous' THEN '同校已验证' ELSE au.nickname END,
       coalesce(l.like_count,0),coalesce(cm.comment_count,0),coalesce(v.view_count,0),
       c.id,c.course_code,c.course_name,c.professor,c.semester,c.department,c.credits,
       coalesce(agg.rating,0),coalesce(agg.review_count,0),coalesce(pc.post_count,0)
FROM course_posts p
JOIN courses c ON c.id=p.course_id
JOIN users au ON au.id=p.author_id
LEFT JOIN (SELECT post_id,count(*) AS like_count FROM post_likes GROUP BY post_id) l ON l.post_id=p.id
LEFT JOIN (SELECT post_id,count(*) AS comment_count FROM post_comments GROUP BY post_id) cm ON cm.post_id=p.id
LEFT JOIN (SELECT post_id,count(*) AS view_count FROM post_view_events WHERE viewed_on>=CURRENT_DATE-29 GROUP BY post_id) v ON v.post_id=p.id
LEFT JOIN (
    SELECT course_id,avg((grading+workload+assessment+usefulness)/4.0) AS rating,count(*) AS review_count
    FROM course_reviews WHERE status='published' GROUP BY course_id
) agg ON agg.course_id=c.id
LEFT JOIN (
    SELECT course_id,count(*) AS post_count FROM course_posts WHERE status='published' GROUP BY course_id
) pc ON pc.course_id=c.id
WHERE p.status='published' AND c.department=$1
ORDER BY coalesce(v.view_count,0) DESC,coalesce(l.like_count,0) DESC,p.created_at DESC,p.id DESC
LIMIT 5`, department)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]homePost, 0, 5)
	for rows.Next() {
		var x homePost
		if rows.Scan(&x.ID, &x.CourseID, &x.Title, &x.Body, &x.Year, &x.Identity, &x.Created, &x.DisplayName,
			&x.LikeCount, &x.CommentCnt, &x.ViewCount,
			&x.Course.ID, &x.Course.CourseCode, &x.Course.CourseName, &x.Course.Professor, &x.Course.Semester, &x.Course.Department, &x.Course.Credits,
			&x.Course.Rating, &x.Course.ReviewCount, &x.Course.PostCount) != nil {
			return nil, err
		}
		x.Course.Rating = roundOne(x.Course.Rating)
		x.YearLabel = "年份未注明"
		if x.Year != nil {
			x.YearLabel = strconv.Itoa(*x.Year)
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
