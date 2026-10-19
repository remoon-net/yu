package httpapi

import "testing"

func TestValidateGoalRequest(t *testing.T) {
	valid, message, ok := validateGoalRequest(goalRequest{
		Name:       " 注册 ",
		Type:       "EVENT",
		MatchValue: " signup ",
	})
	if !ok {
		t.Fatalf("合法目标被拒绝: %s", message)
	}
	if valid.Name != "注册" || valid.Type != "event" || valid.MatchValue != "signup" {
		t.Fatalf("规整结果 = %+v", valid)
	}

	cases := []goalRequest{
		{Name: "", Type: "event", MatchValue: "x"},
		{Name: "x", Type: "other", MatchValue: "x"},
		{Name: "x", Type: "page", MatchValue: " "},
	}

	for _, req := range cases {
		if _, _, ok := validateGoalRequest(req); ok {
			t.Fatalf("非法目标未被拒绝: %+v", req)
		}
	}
}
