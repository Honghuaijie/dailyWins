package service

import (
	"context"
	"net"
	nethttp "net/http"
	"strings"
	"time"
	"unicode/utf8"

	v1 "dailyWins-service/api/all-pb-go/v1"
	"dailyWins-service/internal/auth"
	"dailyWins-service/internal/biz"
	e "dailyWins-service/internal/errors"
	"github.com/go-kratos/kratos/v2/transport"
	kh "github.com/go-kratos/kratos/v2/transport/http"
)

const sessionCookie = "daily_wins_session"

type DailyWinsService struct {
	v1.UnimplementedDailyWinsServer
	accounts *biz.AccountUsecase
	tasks    *biz.TaskUsecase
	options  *auth.Options
}

func NewDailyWinsService(a *biz.AccountUsecase, t *biz.TaskUsecase, o *auth.Options) *DailyWinsService {
	return &DailyWinsService{accounts: a, tasks: t, options: o}
}

func httpTransport(ctx context.Context) (*kh.Transport, error) {
	t, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil, e.E(e.ErrUnauthorized)
	}
	h, ok := t.(*kh.Transport)
	if !ok {
		return nil, e.E(e.ErrUnauthorized)
	}
	return h, nil
}
func (s *DailyWinsService) requireUser(ctx context.Context) (*biz.Session, error) {
	h, err := httpTransport(ctx)
	if err != nil {
		return nil, err
	}
	token := ""
	header := h.Request().Header.Get("Authorization")
	if header != "" {
		if !strings.HasPrefix(header, "Bearer ") {
			return nil, e.E(e.ErrInvalidToken)
		}
		token = strings.TrimPrefix(header, "Bearer ")
	} else if c, err := h.Request().Cookie(sessionCookie); err == nil {
		token = c.Value
	}
	return s.accounts.Authenticate(ctx, token)
}
func (s *DailyWinsService) cookie(h *kh.Transport, token string, expires time.Time, maxAge int) {
	c := &nethttp.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: s.options.Secure, SameSite: nethttp.SameSiteLaxMode, Expires: expires, MaxAge: maxAge}
	h.ReplyHeader().Set("Set-Cookie", c.String())
}
func (s *DailyWinsService) Login(ctx context.Context, r *v1.LoginRequest) (*v1.LoginReply, error) {
	if r == nil || strings.TrimSpace(r.Username) == "" || utf8.RuneCountInString(r.Username) > 64 || len(r.Password) > 72 || r.Password == "" {
		return nil, e.EWithMessage(e.ErrInvalidParam, "请输入有效账号和密码")
	}
	h, err := httpTransport(ctx)
	if err != nil {
		return nil, err
	}
	source, _, err := net.SplitHostPort(h.Request().RemoteAddr)
	if err != nil {
		source = h.Request().RemoteAddr
	}
	d, err := s.accounts.Login(ctx, strings.TrimSpace(r.Username), r.Password, source)
	if err != nil {
		return nil, err
	}
	if !r.Bearer {
		expires, _ := time.Parse(time.RFC3339, d.ExpiresAt)
		s.cookie(h, d.Token, expires, int(s.options.TTL.Seconds()))
		d.Token = ""
	}
	return &v1.LoginReply{Message: successMessage("Login"), Data: d}, nil
}
func (s *DailyWinsService) Me(ctx context.Context, _ *v1.MeRequest) (*v1.MeReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.accounts.Me(ctx, a)
	if err != nil {
		return nil, err
	}
	return &v1.MeReply{Message: successMessage("Me"), Data: d}, nil
}
func (s *DailyWinsService) Logout(ctx context.Context, _ *v1.LogoutRequest) (*v1.LogoutReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.accounts.Logout(ctx, a)
	if err != nil {
		return nil, err
	}
	h, _ := httpTransport(ctx)
	s.cookie(h, "", time.Unix(1, 0), -1)
	return &v1.LogoutReply{Message: successMessage("Logout"), Data: d}, nil
}
func (s *DailyWinsService) ListTasks(ctx context.Context, r *v1.ListTasksRequest) (*v1.ListTasksReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.tasks.ListTasks(ctx, a.UserID, r.From, r.To)
	if err != nil {
		return nil, err
	}
	return &v1.ListTasksReply{Message: successMessage("ListTasks"), Data: d}, nil
}

func (s *DailyWinsService) CreateTask(ctx context.Context, r *v1.CreateTaskRequest) (*v1.CreateTaskReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.tasks.CreateTask(ctx, a.UserID, biz.TaskMutation{RequestID: r.RequestId, ID: r.Id, Date: r.Date, Name: r.Name, Note: r.Note, Done: r.Done})
	if err != nil {
		return nil, err
	}
	return &v1.CreateTaskReply{Message: successMessage("CreateTask"), Data: d}, nil
}

func (s *DailyWinsService) UpdateTask(ctx context.Context, r *v1.UpdateTaskRequest) (*v1.UpdateTaskReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.tasks.UpdateTask(ctx, a.UserID, biz.TaskMutation{RequestID: r.RequestId, ID: r.Id, Version: r.Version, Name: r.Name, Note: r.Note, Done: r.Done})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateTaskReply{Message: successMessage("UpdateTask"), Data: d}, nil
}

func (s *DailyWinsService) MoveTask(ctx context.Context, r *v1.MoveTaskRequest) (*v1.MoveTaskReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.tasks.MoveTask(ctx, a.UserID, biz.TaskMutation{RequestID: r.RequestId, ID: r.Id, Version: r.Version, Date: r.Date})
	if err != nil {
		return nil, err
	}
	return &v1.MoveTaskReply{Message: successMessage("MoveTask"), Data: d}, nil
}

func (s *DailyWinsService) DeleteTask(ctx context.Context, r *v1.DeleteTaskRequest) (*v1.DeleteTaskReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.tasks.DeleteTask(ctx, a.UserID, biz.TaskMutation{RequestID: r.RequestId, ID: r.Id, Version: r.Version})
	if err != nil {
		return nil, err
	}
	return &v1.DeleteTaskReply{Message: successMessage("DeleteTask"), Data: d}, nil
}

func (s *DailyWinsService) RestoreTask(ctx context.Context, r *v1.RestoreTaskRequest) (*v1.RestoreTaskReply, error) {
	a, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.tasks.RestoreTask(ctx, a.UserID, biz.TaskMutation{RequestID: r.RequestId, ID: r.Id, Version: r.Version})
	if err != nil {
		return nil, err
	}
	return &v1.RestoreTaskReply{Message: successMessage("RestoreTask"), Data: d}, nil
}
