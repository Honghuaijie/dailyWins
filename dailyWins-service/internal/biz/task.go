package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	v1 "dailyWins-service/api/all-pb-go/v1"
	e "dailyWins-service/internal/errors"
	"github.com/google/uuid"
)

type Task struct {
	ID                   string
	UserID               int32
	Date, Name, Note     string
	Done, Deleted        bool
	Version              int32
	CreatedAt, UpdatedAt time.Time
}

type TaskMutation struct {
	Action, RequestID, ID, Date, Name, Note string
	Version                                 int32
	Done                                    bool
}

type TaskOperation struct {
	Fingerprint string
	Task        *Task
}

type TaskRepo interface {
	LockOwner(context.Context, int32) error
	GetTask(context.Context, int32, string) (*Task, error)
	ListTasks(context.Context, int32, string, string) ([]*Task, error)
	SaveTask(context.Context, *Task) error
	GetOperation(context.Context, int32, string) (*TaskOperation, error)
	SaveOperation(context.Context, int32, string, *TaskOperation) error
}

type TaskUsecase struct {
	repo TaskRepo
	tx   Transaction
}

func NewTaskUsecase(repo TaskRepo, tx Transaction) *TaskUsecase { return &TaskUsecase{repo, tx} }

func ValidateDate(value string) error {
	d, err := time.Parse("2006-01-02", value)
	if err != nil || len(value) != 10 || d.Year() < 1 || d.Format("2006-01-02") != value {
		return e.EWithMessage(e.ErrInvalidParam, "日期须为有效的 YYYY-MM-DD")
	}
	return nil
}

func ValidateMutation(m TaskMutation) error {
	for _, id := range []string{m.ID, m.RequestID} {
		u, err := uuid.Parse(id)
		if err != nil || u.String() != id {
			return e.EWithMessage(e.ErrInvalidParam, "id 和 request_id 必须是标准小写 UUID")
		}
	}
	if m.Action == "create" || m.Action == "move" {
		if err := ValidateDate(m.Date); err != nil {
			return err
		}
	}
	if m.Action == "create" || m.Action == "update" {
		if strings.TrimSpace(m.Name) == "" || utf8.RuneCountInString(m.Name) > 200 || utf8.RuneCountInString(m.Note) > 4000 {
			return e.EWithMessage(e.ErrInvalidParam, "任务名称须为 1–200 字，备注最多 4000 字")
		}
	}
	if m.Action != "create" && m.Version < 1 {
		return e.EWithMessage(e.ErrInvalidParam, "缺少有效任务版本")
	}
	return nil
}

// ApplyMutation 是纯业务转换；版本检查与状态判断在同一个数据库事务内调用。
func ApplyMutation(old *Task, m TaskMutation) (*Task, error) {
	if old == nil {
		return nil, e.E(e.ErrNotFound)
	}
	if old.Version != m.Version {
		return nil, e.EWithMessage(e.ErrConflict, "记录已被其他操作修改；请保留输入并重新加载后处理")
	}
	t := *old
	if t.Deleted && m.Action != "restore" {
		return nil, e.E(e.ErrNotFound)
	}
	switch m.Action {
	case "update":
		t.Name = strings.TrimSpace(m.Name)
		t.Note = m.Note
		t.Done = m.Done
	case "move":
		if t.Done {
			return nil, e.EWithMessage(e.ErrConflict, "只能移动未完成任务")
		}
		if err := ValidateDate(m.Date); err != nil {
			return nil, err
		}
		t.Date = m.Date
	case "delete":
		t.Deleted = true
	case "restore":
		if !t.Deleted {
			return nil, e.EWithMessage(e.ErrConflict, "任务未删除")
		}
		t.Deleted = false
	default:
		return nil, e.E(e.ErrInvalidParam)
	}
	t.Version++
	t.UpdatedAt = time.Now().UTC()
	return &t, nil
}

func (uc *TaskUsecase) mutate(ctx context.Context, userID int32, m TaskMutation) (*v1.TaskInfo, error) {
	if userID <= 0 {
		return nil, e.E(e.ErrUnauthorized)
	}
	if err := ValidateMutation(m); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(m)
	sum := sha256.Sum256(raw)
	fingerprint := hex.EncodeToString(sum[:])
	var result *Task
	err := uc.tx.ExecTx(ctx, func(ctx context.Context) error {
		// 固定个人账号的写入量很小；按用户行串行化，确保幂等记录与任务原子提交。
		if err := uc.repo.LockOwner(ctx, userID); err != nil {
			return err
		}
		previous, err := uc.repo.GetOperation(ctx, userID, m.RequestID)
		if err != nil {
			return err
		}
		if previous != nil {
			if previous.Fingerprint != fingerprint {
				return e.EWithMessage(e.ErrConflict, "同一请求标识不能用于不同内容")
			}
			result = previous.Task
			return nil
		}
		old, err := uc.repo.GetTask(ctx, userID, m.ID)
		if err != nil {
			return err
		}
		if m.Action == "create" {
			if old != nil {
				return e.E(e.ErrDuplicate)
			}
			now := time.Now().UTC()
			result = &Task{ID: m.ID, UserID: userID, Date: m.Date, Name: strings.TrimSpace(m.Name), Note: m.Note, Done: m.Done, Version: 1, CreatedAt: now, UpdatedAt: now}
		} else {
			result, err = ApplyMutation(old, m)
			if err != nil {
				return err
			}
		}
		if err := uc.repo.SaveTask(ctx, result); err != nil {
			return err
		}
		return uc.repo.SaveOperation(ctx, userID, m.RequestID, &TaskOperation{Fingerprint: fingerprint, Task: result})
	})
	if err != nil {
		return nil, err
	}
	return taskInfo(result), nil
}

func (uc *TaskUsecase) ListTasks(ctx context.Context, userID int32, from, to string) (*v1.ListTasksReply_Data, error) {
	if userID <= 0 {
		return nil, e.E(e.ErrUnauthorized)
	}
	if err := ValidateDate(from); err != nil {
		return nil, err
	}
	if err := ValidateDate(to); err != nil {
		return nil, err
	}
	a, _ := time.Parse("2006-01-02", from)
	b, _ := time.Parse("2006-01-02", to)
	if b.Before(a) || b.Sub(a) > 366*24*time.Hour {
		return nil, e.EWithMessage(e.ErrInvalidParam, "查询日期范围须为 0–366 天")
	}
	tasks, err := uc.repo.ListTasks(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.TaskInfo, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, taskInfo(t))
	}
	return &v1.ListTasksReply_Data{Items: items, Days: Summaries(tasks)}, nil
}

func Summaries(tasks []*Task) []*v1.DaySummary {
	days := map[string]*v1.DaySummary{}
	for _, t := range tasks {
		if t.Deleted {
			continue
		}
		d := days[t.Date]
		if d == nil {
			d = &v1.DaySummary{Date: t.Date}
			days[t.Date] = d
		}
		if t.Done {
			d.Completed++
		}
		d.Achieved = d.Completed >= 3
	}
	result := make([]*v1.DaySummary, 0, len(days))
	for _, d := range days {
		result = append(result, d)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result
}
func taskInfo(t *Task) *v1.TaskInfo {
	return &v1.TaskInfo{Id: t.ID, UserId: t.UserID, Date: t.Date, Name: t.Name, Note: t.Note, Done: t.Done, Deleted: t.Deleted, Version: t.Version, CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: t.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}
func (uc *TaskUsecase) CreateTask(ctx context.Context, u int32, m TaskMutation) (*v1.CreateTaskReply_Data, error) {
	m.Action = "create"
	t, err := uc.mutate(ctx, u, m)
	return &v1.CreateTaskReply_Data{Task: t}, err
}
func (uc *TaskUsecase) UpdateTask(ctx context.Context, u int32, m TaskMutation) (*v1.UpdateTaskReply_Data, error) {
	m.Action = "update"
	t, err := uc.mutate(ctx, u, m)
	return &v1.UpdateTaskReply_Data{Task: t}, err
}
func (uc *TaskUsecase) MoveTask(ctx context.Context, u int32, m TaskMutation) (*v1.MoveTaskReply_Data, error) {
	m.Action = "move"
	t, err := uc.mutate(ctx, u, m)
	return &v1.MoveTaskReply_Data{Task: t}, err
}
func (uc *TaskUsecase) DeleteTask(ctx context.Context, u int32, m TaskMutation) (*v1.DeleteTaskReply_Data, error) {
	m.Action = "delete"
	t, err := uc.mutate(ctx, u, m)
	return &v1.DeleteTaskReply_Data{Task: t}, err
}
func (uc *TaskUsecase) RestoreTask(ctx context.Context, u int32, m TaskMutation) (*v1.RestoreTaskReply_Data, error) {
	m.Action = "restore"
	t, err := uc.mutate(ctx, u, m)
	return &v1.RestoreTaskReply_Data{Task: t}, err
}
