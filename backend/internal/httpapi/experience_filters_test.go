package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestExperienceFilters(t *testing.T) {
	dsn := os.Getenv("COURSEREP_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set COURSEREP_TEST_DATABASE_URL to run against PostgreSQL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 1 // Temporary tables are scoped to the database connection.
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		CREATE TEMP TABLE users (id bigint, status text, role text, nickname text);
		CREATE TEMP TABLE courses (
			id bigint, course_code text, course_name text, professor text, semester text, department text, credits numeric
		);
		CREATE TEMP TABLE course_reviews (
			id bigint, author_id bigint, course_id bigint, semester text, grading int, workload int, assessment int,
			usefulness int, tags text[], body text, status text, created_at timestamp
		);
		CREATE TEMP TABLE course_posts (
			id bigint, course_id bigint, author_id bigint, title text, body text, image_urls text[], topic_tags text[],
			attendance_year int, display_identity text, status text, created_at timestamp
		);
		CREATE TEMP TABLE post_likes (post_id bigint, user_id bigint);
		CREATE TEMP TABLE post_bookmarks (post_id bigint, user_id bigint);
		CREATE TEMP TABLE post_comments (post_id bigint);
		CREATE TEMP TABLE post_view_events (post_id bigint, viewed_on date);

		INSERT INTO users VALUES (1,'verified','user','Viewer'), (2,'verified','user','Other');
		INSERT INTO courses VALUES
			(3000000001,'TEST101','First course','Teacher','2025 A','Test',3),
			(3000000002,'TEST102','Second course','Teacher','2025 A','Test',3);
		INSERT INTO course_posts VALUES
			(11,3000000001,1,'Match','Body','{}','{study}',NULL,'anonymous','published',now()),
			(12,3000000001,1,'Wrong year','Body','{}','{study}',2025,'named','published',now()),
			(13,3000000002,2,'Wrong course','Body','{}','{study}',NULL,'named','published',now()),
			(14,3000000001,1,'Pending','Body','{}','{study}',NULL,'named','submitted',now()),
			(15,3000000001,2,'Wrong tag','Body','{}','{exam}',NULL,'named','published',now());
		INSERT INTO course_reviews VALUES
			(21,1,3000000001,'2025 A',3,3,3,3,'{}','Body','rejected',now()),
			(22,2,3000000002,'2025 A',3,3,3,3,'{}','Body','published',now());
	`)
	if err != nil {
		t.Fatal(err)
	}

	const secret = "local-test-secret-at-least-32-bytes"
	t.Setenv("AUTH_SECRET", secret)
	token, err := signAccess(1, 0, []byte(secret), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	handler := http.NewServeMux()
	registerFeedRoutes(handler, pool)
	registerProfileRoutes(handler, pool)
	for _, tc := range []struct {
		path   string
		status int
		wantID int64
	}{
		{"/api/v1/posts?course_id=3000000001&year=unspecified&tag=study", http.StatusOK, 11},
		{"/api/v1/posts?course_id=-1", http.StatusBadRequest, 0},
		{"/api/v1/users/me/posts?id=14", http.StatusOK, 14},
		{"/api/v1/users/me/posts?id=13", http.StatusOK, 0},
		{"/api/v1/users/me/reviews?id=21", http.StatusOK, 21},
		{"/api/v1/users/me/reviews?id=22", http.StatusOK, 0},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil).WithContext(ctx)
			req.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, tc.status, response.Body)
			}
			if tc.status != http.StatusOK {
				return
			}
			var result struct {
				Items []struct {
					ID int64 `json:"id"`
				} `json:"items"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if tc.wantID == 0 && len(result.Items) != 0 ||
				tc.wantID != 0 && (len(result.Items) != 1 || result.Items[0].ID != tc.wantID) {
				t.Fatalf("want only ID %d (0 means empty), got %+v", tc.wantID, result.Items)
			}
		})
	}
}
