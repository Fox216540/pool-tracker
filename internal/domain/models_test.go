package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGameSessionJSONContract(t *testing.T) {
	row := GameSession{ID: "session-1", UserID: "private-user", GameDate: "2026-10-21", DurationMinutes: 135, Note: "break practice"}
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{`"id":"session-1"`, `"game_date":"2026-10-21"`, `"duration_minutes":135`, `"note":"break practice"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("response %s does not contain %s", got, want)
		}
	}
	if strings.Contains(got, "private-user") || strings.Contains(got, "UserID") {
		t.Fatalf("response leaks user id: %s", got)
	}
}
