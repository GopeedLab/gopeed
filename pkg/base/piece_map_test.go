package base

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPieceMapCRUDAndBytes(t *testing.T) {
	pieceMap := NewPieceMap(4, 16*1024)
	_ = pieceMap.Update(0, true)
	_ = pieceMap.Update(2, true)

	if got, want := pieceMap.Bytes(), []byte{0b00000101}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("Bytes() = %08b, want %08b", got, want)
	}
	if got := pieceMap.CompletedPieces(); got != 2 {
		t.Fatalf("CompletedPieces() = %d, want 2", got)
	}

	pieceMap.Add(true)
	if got := pieceMap.PieceCount(); got != 5 {
		t.Fatalf("PieceCount() = %d, want 5", got)
	}
	if err := pieceMap.Delete(1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	wantAfterDelete := []bool{true, true, false, true}
	for index, want := range wantAfterDelete {
		got, err := pieceMap.Get(index)
		if err != nil || got != want {
			t.Fatalf("Get(%d) = %v, %v; want %v", index, got, err, want)
		}
	}
	if got := pieceMap.CompletedPieces(); got != 3 {
		t.Fatalf("CompletedPieces() after delete = %d, want 3", got)
	}

	bytesCopy := pieceMap.Bytes()
	bytesCopy[0] = 0
	got, _ := pieceMap.Get(0)
	if !got {
		t.Fatal("Bytes returned mutable internal storage")
	}
}

func TestPieceMapValidationAndRestore(t *testing.T) {
	pieceMap, err := NewPieceMapFromBytes(10, 1024, []byte{0b10010001, 0xff})
	if err != nil {
		t.Fatalf("NewPieceMapFromBytes: %v", err)
	}
	// Only the low positions for pieces 8 and 9 belong to the map.
	if got, want := pieceMap.Bytes()[1], byte(0b00000011); got != want {
		t.Fatalf("last byte = %08b, want %08b", got, want)
	}
	if pieceMap.CompletedPieces() != 5 {
		t.Fatalf("CompletedPieces() = %d, want 5", pieceMap.CompletedPieces())
	}

	if _, err := NewPieceMapFromBytes(10, 1024, []byte{0}); err == nil {
		t.Fatal("expected invalid byte length error")
	}
	if err := pieceMap.Update(10, true); !errors.Is(err, ErrPieceIndexOutOfRange) {
		t.Fatalf("Update out of range error = %v", err)
	}
}

func TestPieceMapJSON(t *testing.T) {
	pieceMap := NewPieceMap(4, 4096)
	_ = pieceMap.Update(0, true)
	_ = pieceMap.Update(2, true)

	data, err := json.Marshal(pieceMap)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal JSON: %v", err)
	}
	if got["encoding"] != PieceMapEncoding || got["data"] != "BQ==" {
		t.Fatalf("unexpected JSON: %s", data)
	}
	if got["pieceCount"] != float64(4) || got["pieceSize"] != float64(4096) || got["completedPieces"] != float64(2) {
		t.Fatalf("unexpected metadata: %s", data)
	}
}

func TestPieceMapCloneIsIndependent(t *testing.T) {
	pieceMap := NewPieceMap(4, 4096)
	_ = pieceMap.Update(0, true)
	snapshot := pieceMap.Clone()
	_ = snapshot.Update(1, true)

	if got, _ := pieceMap.Get(1); got {
		t.Fatal("Clone mutated the source map")
	}
	if snapshot.CompletedPieces() != 2 {
		t.Fatalf("clone completed pieces = %d, want 2", snapshot.CompletedPieces())
	}
	if (*PieceMap)(nil).Clone() != nil {
		t.Fatal("nil map clone must stay nil")
	}
}
