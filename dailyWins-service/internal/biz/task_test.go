package biz

import (
	"testing"
	"time"
)

func TestBusinessDates(t *testing.T) {
	for _, s := range []string{"2024-02-29", "2026-10-17", "0001-01-01", "9999-12-31"} {
		if err := ValidateDate(s); err != nil {
			t.Errorf("%s: %v", s, err)
		}
	}
	for _, s := range []string{"2026-02-29", "2026-2-01", "2026-10-17T00:00:00Z", "2026-04-31", "0000-01-01", ""} {
		if ValidateDate(s) == nil {
			t.Errorf("invalid date accepted %s", s)
		}
	}
}

func TestTaskRules(t *testing.T) {
	old := &Task{ID: "id", UserID: 1, Date: "2026-10-16", Name: "原名称", Note: "备注", Version: 2, CreatedAt: time.Now()}
	moved, err := ApplyMutation(old, TaskMutation{Action: "move", Version: 2, Date: "2027-01-01"})
	if err != nil || moved.ID != "id" || moved.Name != "原名称" || moved.Note != "备注" || moved.Date != "2027-01-01" || moved.Done || moved.Version != 3 {
		t.Fatalf("move: %+v %v", moved, err)
	}
	old.Done = true
	if _, err := ApplyMutation(old, TaskMutation{Action: "move", Version: 2, Date: "2027-01-01"}); err == nil {
		t.Fatal("completed task moved")
	}
	if _, err := ApplyMutation(old, TaskMutation{Action: "delete", Version: 1}); err == nil {
		t.Fatal("stale write accepted")
	}
	deleted, err := ApplyMutation(old, TaskMutation{Action: "delete", Version: 2})
	if err != nil || !deleted.Deleted {
		t.Fatal("delete failed")
	}
	restored, err := ApplyMutation(deleted, TaskMutation{Action: "restore", Version: 3})
	if err != nil || restored.Deleted || restored.ID != old.ID {
		t.Fatal("restore failed")
	}
}

func TestAchievementIsDerived(t *testing.T) {
	tasks := []*Task{{Date: "2026-10-16", Done: true}, {Date: "2026-10-16", Done: true}, {Date: "2026-10-16", Done: true}, {Date: "2026-10-17", Done: false}}
	if s := Summaries(tasks); len(s) != 2 || !s[0].Achieved || s[0].Completed != 3 || s[1].Achieved {
		t.Fatalf("bad summaries %+v", s)
	}
	tasks[0].Done = false
	if Summaries(tasks)[0].Achieved {
		t.Fatal("badge survived undo completion")
	}
	tasks[0].Done = true
	tasks[0].Deleted = true
	if Summaries(tasks)[0].Achieved {
		t.Fatal("badge survived deletion")
	}
}
