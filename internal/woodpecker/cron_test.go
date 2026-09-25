package woodpecker

import (
	"encoding/json"
	"testing"
)

func TestCronEnabledJSON(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		data, err := json.Marshal(&Cron{Enabled: enabled})
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["enabled"] != enabled {
			t.Fatalf("enabled = %v, want %v", decoded["enabled"], enabled)
		}
	}
}
