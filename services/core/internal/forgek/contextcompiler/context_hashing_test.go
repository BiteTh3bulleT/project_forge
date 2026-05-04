package contextcompiler

import "testing"

func TestTokenInputHashIsDeterministicForExactText(t *testing.T) {
	text := "[BLOCK: USER_MESSAGE]\nworkspace_id: workspace-a\nsummary:\nhello"
	if TokenInputHash(text) != TokenInputHash(text) {
		t.Fatal("token input hash is not deterministic")
	}
	if TokenInputHash(text) == TokenInputHash(text+"!") {
		t.Fatal("token input hash did not change for changed token input text")
	}
	if EstimateTokens(text) != EstimateTokens(text) || EstimateTokens(text) == 0 {
		t.Fatal("token estimate is not deterministic or is empty")
	}
}
