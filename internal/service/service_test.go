package service

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/andrey/pool-tracker/internal/database"
)

func testService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close(db) })
	if err := database.Migrate(db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	return New(db, time.Hour)
}

func TestSessionCRUDAndStatistics(t *testing.T) {
	svc := testService(t)
	u, err := svc.FindOrCreateUser(1)
	if err != nil {
		t.Fatal(err)
	}
	row, err := svc.Create(u.ID, "2026-10-01", 90, "club")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(u.ID, "2026-10-01", 30, ""); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := svc.Create(u.ID, "2026-10-02", 0, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duration validation: %v", err)
	}
	if _, err := svc.Update(u.ID, row.ID, "2026-10-02", 120, "updated"); err != nil {
		t.Fatal(err)
	}
	stat, err := svc.Statistics(u.ID, "2026-10-01", "2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	if stat.PlayingDays != 1 || stat.TotalMinutes != 120 || stat.AverageMinutes != 120 || stat.LongestMinutes != 120 {
		t.Fatalf("stat = %+v", stat)
	}
	if err := svc.Delete(u.ID, row.ID); err != nil {
		t.Fatal(err)
	}
	empty, err := svc.Statistics(u.ID, "2026-10-01", "2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	if empty.PlayingDays != 0 || empty.TotalMinutes != 0 || empty.AverageMinutes != 0 || empty.LongestMinutes != 0 {
		t.Fatalf("empty = %+v", empty)
	}
}

func TestSessionOwnershipAndBoundaries(t *testing.T) {
	svc := testService(t)
	first, _ := svc.FindOrCreateUser(1)
	second, _ := svc.FindOrCreateUser(2)
	row, err := svc.Create(first.ID, "2024-02-29", 60, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(second.ID, row.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign get: %v", err)
	}
	if err := svc.Delete(second.ID, row.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign delete: %v", err)
	}
	if _, err := svc.Create(first.ID, "2025-02-29", 60, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid calendar date: %v", err)
	}
}
