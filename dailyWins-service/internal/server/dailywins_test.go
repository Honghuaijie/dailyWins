package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dailyWins-service/internal/auth"
	"dailyWins-service/internal/biz"
	"dailyWins-service/internal/service"
	"dailyWins-service/internal/testutil"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

func newDailyServer(t *testing.T) (*httptest.Server, *testutil.Store) {
	t.Helper()
	store := testutil.NewStore()
	hash, err := auth.HashPassword("test-password-long")
	if err != nil {
		t.Fatal(err)
	}
	store.Accounts[1] = &biz.Account{ID: 1, Username: "owner", PasswordHash: hash}
	options := &auth.Options{Secret: strings.Repeat("s", 32), TTL: time.Hour, Origins: []string{"http://allowed.example"}}
	accounts, err := biz.NewAccountUsecase(store, options)
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewDailyWinsService(accounts, biz.NewTaskUsecase(store, store), options)
	server := httptest.NewServer(NewHTTPServer(nil, svc, options, log.NewStdLogger(io.Discard)))
	t.Cleanup(server.Close)
	return server, store
}
func call(t *testing.T, url, method string, body interface{}, cookie *http.Cookie, token, origin string, protect bool) (int, map[string]interface{}, []*http.Cookie) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, url, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if protect {
		req.Header.Set("X-Daily-Wins", "1")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	result := map[string]interface{}{}
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result, resp.Cookies()
}
func TestDailyHTTPAuthenticationAndSecurity(t *testing.T) {
	ts, _ := newDailyServer(t)
	status, _, _ := call(t, ts.URL+"/v1/user/tasks/list?from=2026-10-01&to=2026-10-31", "GET", nil, nil, "", "", false)
	if status != 401 {
		t.Fatal("anonymous tasks allowed", status)
	}
	login := map[string]interface{}{"username": "owner", "password": "test-password-long"}
	status, _, _ = call(t, ts.URL+"/v1/public/auth/login", "POST", login, nil, "", "", false)
	if status != 403 {
		t.Fatal("missing CSRF accepted")
	}
	status, _, _ = call(t, ts.URL+"/v1/public/auth/login", "POST", login, nil, "", "http://evil.example", true)
	if status != 403 {
		t.Fatal("untrusted origin accepted")
	}
	status, result, cookies := call(t, ts.URL+"/v1/public/auth/login", "POST", login, nil, "", "http://allowed.example", true)
	if status != 200 || len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || result["data"].(map[string]interface{})["token"] != "" {
		t.Fatalf("invalid browser login: %d %+v", status, result)
	}
	cookie := cookies[0]
	status, _, _ = call(t, ts.URL+"/v1/user/auth/me", "GET", nil, cookie, "", "", false)
	if status != 200 {
		t.Fatal("cookie rejected", status)
	}
	status, _, _ = call(t, ts.URL+"/v1/user/auth/logout", "POST", map[string]interface{}{}, cookie, "", "", true)
	if status != 200 {
		t.Fatal("logout failed")
	}
	status, _, _ = call(t, ts.URL+"/v1/user/auth/me", "GET", nil, cookie, "", "", false)
	if status != 401 {
		t.Fatal("logout did not revoke")
	}
	login["bearer"] = true
	status, result, _ = call(t, ts.URL+"/v1/public/auth/login", "POST", login, nil, "", "", true)
	token := result["data"].(map[string]interface{})["token"].(string)
	status, _, _ = call(t, ts.URL+"/v1/user/auth/me", "GET", nil, nil, token, "", false)
	if status != 200 {
		t.Fatal("bearer rejected")
	}
	for _, path := range []string{"/v1/users/create", "/v1/users/list", "/v1/users/avatarUpload"} {
		status, _, _ = call(t, ts.URL+path, "POST", map[string]interface{}{}, nil, "", "", true)
		if status != 404 {
			t.Fatal("template route exposed", path, status)
		}
	}
	status, _, _ = call(t, ts.URL+"/v1/user/tasks/create", "POST", map[string]interface{}{"unknown": true}, nil, token, "", true)
	if status != 400 {
		t.Fatal("unknown field accepted")
	}
}
func TestDailyHTTPTaskLifecycle(t *testing.T) {
	ts, _ := newDailyServer(t)
	_, _, cookies := call(t, ts.URL+"/v1/public/auth/login", "POST", map[string]interface{}{"username": "owner", "password": "test-password-long"}, nil, "", "", true)
	cookie := cookies[0]
	id := uuid.NewString()
	payload := map[string]interface{}{"id": id, "requestId": uuid.NewString(), "name": "补记昨天", "date": "2026-10-16", "done": true, "note": "备注"}
	status, result, _ := call(t, ts.URL+"/v1/user/tasks/create", "POST", payload, cookie, "", "", true)
	if status != 200 {
		t.Fatalf("create %d %+v", status, result)
	}
	status, _, _ = call(t, ts.URL+"/v1/user/tasks/create", "POST", payload, cookie, "", "", true)
	if status != 200 {
		t.Fatal("retry failed")
	}
	status, result, _ = call(t, ts.URL+"/v1/user/tasks/list?from=2026-10-01&to=2026-10-31", "GET", nil, cookie, "", "", false)
	if status != 200 || len(result["data"].(map[string]interface{})["items"].([]interface{})) != 1 {
		t.Fatal("duplicate created")
	}
	status, _, _ = call(t, ts.URL+"/v1/user/tasks/move", "POST", map[string]interface{}{"id": id, "requestId": uuid.NewString(), "version": 1, "date": "2027-01-01"}, cookie, "", "", true)
	if status != 409 {
		t.Fatal("completed moved")
	}
	status, _, _ = call(t, ts.URL+"/v1/user/tasks/delete", "POST", map[string]interface{}{"id": id, "requestId": uuid.NewString(), "version": 1}, cookie, "", "", true)
	if status != 200 {
		t.Fatal("delete failed")
	}
	status, _, _ = call(t, ts.URL+"/v1/user/tasks/restore", "POST", map[string]interface{}{"id": id, "requestId": uuid.NewString(), "version": 2}, cookie, "", "", true)
	if status != 200 {
		t.Fatal("restore failed")
	}
}
func TestLoginFailureLimit(t *testing.T) {
	ts, _ := newDailyServer(t)
	for i := 0; i < 6; i++ {
		status, _, _ := call(t, ts.URL+"/v1/public/auth/login", "POST", map[string]interface{}{"username": "missing", "password": "wrong-password"}, nil, "", "", true)
		want := 401
		if i == 5 {
			want = 429
		}
		if status != want {
			t.Fatalf("attempt %d: %d", i, status)
		}
	}
}
