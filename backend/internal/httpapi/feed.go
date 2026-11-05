package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// feedPost 是「分享」页信息流卡片，包含课程摘要，前端可直接打开课程详情。
type feedPost struct {
	ID           int64      `json:"id"`
	CourseID     int64      `json:"course_id"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	ImageURLs    []string   `json:"image_urls"`
	TopicTags    []string   `json:"topic_tags"`
	Year         *int       `json:"attendance_year"`
	YearLabel    string     `json:"attendance_year_label"`
	Identity     string     `json:"display_identity"`
	DisplayName  string     `json:"display_name"`
	LikeCount    int        `json:"like_count"`
	CommentCount int        `json:"comment_count"`
	ViewCount    int        `json:"view_count"`
	LikedByMe    bool       `json:"liked_by_me"`
	Bookmarked   bool       `json:"bookmarked_by_me"`
	Created      any        `json:"created_at"`
	Course       homeCourse `json:"course"`
}

// registerFeedRoutes 提供跨课程的经验帖信息流，供「分享」页按专业/话题/年份筛选。
func registerFeedRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.Handle("GET /api/v1/posts", authenticate(pool, canBrowse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, err := queryInt(r, "page", 1, 1, 1_000_000)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "page 无效"})
			return
		}
		size, err := queryInt(r, "page_size", 20, 1, 100)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "page_size 无效"})
			return
		}
		department := strings.TrimSpace(r.URL.Query().Get("department"))
		tag := strings.TrimSpace(r.URL.Query().Get("tag"))
		year := strings.TrimSpace(r.URL.Query().Get("year"))
		sortBy := r.URL.Query().Get("sort")
		if sortBy == "" {
			sortBy = "newest"
		}
		if len([]rune(department)) > 100 {
			writeJSON(w, 400, map[string]string{"error": "专业筛选无效"})
			return
		}
		if year != "" && year != "unspecified" {
			n, convErr := strconv.Atoi(year)
			if convErr != nil || n < 1990 || n > currentYear() {
				writeJSON(w, 400, map[string]string{"error": "年份筛选无效"})
				return
			}
		}
		if sortBy != "newest" && sortBy != "popular" {
			writeJSON(w, 400, map[string]string{"error": "排序方式无效"})
			return
		}
		order := "p.created_at DESC,p.id DESC"
		if sortBy == "popular" {
			order = "like_count DESC,comment_count DESC,p.created_at DESC,p.id DESC"
		}
		viewer := userFromContext(r.Context())
		query := `SELECT p.id,p.course_id,p.title,p.body,p.image_urls,p.topic_tags,p.attendance_year,p.display_identity,p.created_at,
		       CASE WHEN p.display_identity='anonymous' THEN '同校已验证' ELSE au.nickname END,
		       coalesce(l.like_count,0),coalesce(cm.comment_count,0),coalesce(v.view_count,0),
		       EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id=p.id AND pl.user_id=$4),
		       EXISTS(SELECT 1 FROM post_bookmarks pb WHERE pb.post_id=p.id AND pb.user_id=$4),
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
		WHERE p.status='published'
		  AND ($1='' OR c.department=$1)
		  AND ($2='' OR $2=ANY(p.topic_tags))
		  AND ($3='' OR ($3='unspecified' AND p.attendance_year IS NULL) OR p.attendance_year::text=$3)
		ORDER BY ` + order + ` LIMIT $5 OFFSET $6`
		rows, err := pool.Query(r.Context(), query, department, tag, year, viewer.ID, size, (page-1)*size)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取经验帖失败"})
			return
		}
		defer rows.Close()
		items := make([]feedPost, 0, size)
		for rows.Next() {
			var x feedPost
			if err := rows.Scan(&x.ID, &x.CourseID, &x.Title, &x.Body, &x.ImageURLs, &x.TopicTags, &x.Year, &x.Identity, &x.Created,
				&x.DisplayName, &x.LikeCount, &x.CommentCount, &x.ViewCount, &x.LikedByMe, &x.Bookmarked,
				&x.Course.ID, &x.Course.CourseCode, &x.Course.CourseName, &x.Course.Professor, &x.Course.Semester, &x.Course.Department, &x.Course.Credits,
				&x.Course.Rating, &x.Course.ReviewCount, &x.Course.PostCount); err != nil {
				writeJSON(w, 500, map[string]string{"error": "读取经验帖失败"})
				return
			}
			x.Course.Rating = roundOne(x.Course.Rating)
			x.YearLabel = "年份未注明"
			if x.Year != nil {
				x.YearLabel = strconv.Itoa(*x.Year)
			}
			items = append(items, x)
		}
		if rows.Err() != nil {
			writeJSON(w, 500, map[string]string{"error": "读取经验帖失败"})
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size, "has_more": len(items) == size})
	}))))
}
