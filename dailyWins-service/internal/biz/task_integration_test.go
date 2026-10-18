package biz_test

import (
	"context"
	"strings"
	"testing"

	"dailyWins-service/internal/biz"
	e "dailyWins-service/internal/errors"
	"dailyWins-service/internal/testutil"
	"github.com/google/uuid"
)

func TestTasksOwnershipIdempotencyAndVersions(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore()
	store.Accounts[1] = &biz.Account{ID: 1}
	store.Accounts[2] = &biz.Account{ID: 2}
	uc := biz.NewTaskUsecase(store, store)
	m := biz.TaskMutation{ID: uuid.NewString(), RequestID: uuid.NewString(), Date: "2024-02-29", Name: "过去的记录", Note: "保留备注", Done: true}
	created, err := uc.CreateTask(ctx, 1, m)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := uc.CreateTask(ctx, 1, m)
	if err != nil || repeated.Task.Id != created.Task.Id || len(store.Tasks) != 1 {
		t.Fatal("duplicate submission not deduplicated", err)
	}
	m.Name = "不同内容"
	if _, err := uc.CreateTask(ctx, 1, m); e.FromError(err).HttpCode != 409 {
		t.Fatal("request key reused")
	}
	list, err := uc.ListTasks(ctx, 2, "2024-02-01", "2024-02-29")
	if err != nil || len(list.Items) != 0 {
		t.Fatal("cross user read")
	}
	for _, action := range []string{"update", "move", "delete", "restore"} {
		m := biz.TaskMutation{ID: created.Task.Id, RequestID: uuid.NewString(), Version: 1, Name: "偷改", Date: "2024-03-01"}
		var err error
		switch action {
		case "update":
			_, err = uc.UpdateTask(ctx, 2, m)
		case "move":
			_, err = uc.MoveTask(ctx, 2, m)
		case "delete":
			_, err = uc.DeleteTask(ctx, 2, m)
		case "restore":
			_, err = uc.RestoreTask(ctx, 2, m)
		}
		if err == nil || e.FromError(err).HttpCode != 404 {
			t.Fatalf("cross owner %s: %v", action, err)
		}
	}
	m = biz.TaskMutation{ID: created.Task.Id, RequestID: uuid.NewString(), Version: 1, Name: "改名", Note: "保留备注", Done: false}
	updated, err := uc.UpdateTask(ctx, 1, m)
	if err != nil || updated.Task.Version != 2 {
		t.Fatal(err)
	}
	m.RequestID = uuid.NewString()
	if _, err := uc.UpdateTask(ctx, 1, m); err == nil || e.FromError(err).HttpCode != 409 {
		t.Fatal("stale write accepted")
	}
	moved, err := uc.MoveTask(ctx, 1, biz.TaskMutation{ID: m.ID, RequestID: uuid.NewString(), Version: 2, Date: "2028-02-29"})
	if err != nil || moved.Task.Id != m.ID || moved.Task.Note != "保留备注" {
		t.Fatal(err)
	}
	old, _ := uc.ListTasks(ctx, 1, "2024-02-01", "2024-02-29")
	if len(old.Items) != 0 {
		t.Fatal("move left a copy")
	}
	deleted, err := uc.DeleteTask(ctx, 1, biz.TaskMutation{ID: m.ID, RequestID: uuid.NewString(), Version: 3})
	if err != nil || !deleted.Task.Deleted {
		t.Fatal(err)
	}
	restored, err := uc.RestoreTask(ctx, 1, biz.TaskMutation{ID: m.ID, RequestID: uuid.NewString(), Version: 4})
	if err != nil || restored.Task.Deleted || restored.Task.Id != m.ID {
		t.Fatal(err)
	}
}
func TestTaskValidation(t *testing.T) {
	for _, m := range []biz.TaskMutation{
		{Action: "create", ID: uuid.NewString(), RequestID: uuid.NewString(), Date: "2026-10-17", Name: " "},
		{Action: "create", ID: uuid.NewString(), RequestID: uuid.NewString(), Date: "2026-10-17", Name: strings.Repeat("字", 201)},
		{Action: "create", ID: uuid.NewString(), RequestID: uuid.NewString(), Date: "2026-10-17", Name: "名称", Note: strings.Repeat("字", 4001)},
		{Action: "move", ID: uuid.NewString(), RequestID: uuid.NewString(), Date: "2026-02-29", Version: 1},
	} {
		if biz.ValidateMutation(m) == nil {
			t.Fatalf("invalid accepted: %+v", m)
		}
	}
	store := testutil.NewStore()
	uc := biz.NewTaskUsecase(store, store)
	for _, r := range [][2]string{{"2026-01-02", "2026-01-01"}, {"2024-01-01", "2026-01-01"}, {"2026-01-01T00:00:00Z", "2026-02-01"}} {
		if _, err := uc.ListTasks(context.Background(), 1, r[0], r[1]); err == nil {
			t.Fatal("invalid range accepted")
		}
	}
}
