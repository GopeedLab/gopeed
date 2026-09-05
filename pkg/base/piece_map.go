package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
)

const (
	// PieceMapEncoding identifies the one-bit wire format used by PieceMap.
	// One byte stores eight ordered pieces, least-significant bit first: piece
	// 0 uses bit 0, piece 1 uses bit 1, and so on. A set bit means the piece is
	// locally complete and verified; a clear bit means it is not complete.
	PieceMapEncoding = "bitset-v1"

	piecesPerByte = 8
)

var ErrPieceIndexOutOfRange = errors.New("piece index out of range")

// PieceMap stores the verified completion state of ordered pieces as a bitset.
// It deliberately excludes scheduler states such as downloading, hashing and
// failed: those states are transient and are not stable across pause/restart.
//
// Fields are private so PieceCount, CompletedPieces and packed bytes remain
// consistent. Use Add, Delete, Update and Get for mutation, and Bytes when a
// binary copy is required. JSON serializes Data as Base64.
type PieceMap struct {
	pieceCount      int
	pieceSize       int64
	completedPieces int
	data            []byte
}

// NewPieceMap creates a map containing pieceCount incomplete pieces.
func NewPieceMap(pieceCount int, pieceSize int64) *PieceMap {
	if pieceCount < 0 {
		pieceCount = 0
	}
	return &PieceMap{
		pieceCount: pieceCount,
		pieceSize:  pieceSize,
		data:       make([]byte, packedPieceByteCount(pieceCount)),
	}
}

// NewPieceMapFromBytes restores a bitset-v1 map. Input bytes are copied and
// unused high bits in the last byte are cleared.
func NewPieceMapFromBytes(pieceCount int, pieceSize int64, data []byte) (*PieceMap, error) {
	if pieceCount < 0 {
		return nil, fmt.Errorf("%w: negative piece count", ErrPieceIndexOutOfRange)
	}
	expectedBytes := packedPieceByteCount(pieceCount)
	if len(data) != expectedBytes {
		return nil, fmt.Errorf("invalid piece data length: got %d, want %d", len(data), expectedBytes)
	}

	pieceMap := &PieceMap{
		pieceCount: pieceCount,
		pieceSize:  pieceSize,
		data:       append([]byte(nil), data...),
	}
	pieceMap.clearUnusedBits()
	for _, value := range pieceMap.data {
		pieceMap.completedPieces += bits.OnesCount8(value)
	}
	return pieceMap, nil
}

// Clone returns an independent ordered snapshot.
func (m *PieceMap) Clone() *PieceMap {
	if m == nil {
		return nil
	}
	return &PieceMap{
		pieceCount:      m.pieceCount,
		pieceSize:       m.pieceSize,
		completedPieces: m.completedPieces,
		data:            append([]byte(nil), m.data...),
	}
}

// Add appends a piece while preserving all existing indexes.
func (m *PieceMap) Add(completed bool) {
	if m.pieceCount%piecesPerByte == 0 {
		m.data = append(m.data, 0)
	}
	m.pieceCount++
	if completed {
		m.setCompleted(m.pieceCount-1, true)
		m.completedPieces++
	}
}

// Delete removes a piece and shifts every following piece one index left.
func (m *PieceMap) Delete(index int) error {
	if !m.validIndex(index) {
		return ErrPieceIndexOutOfRange
	}
	if m.completedAt(index) {
		m.completedPieces--
	}
	for current := index; current < m.pieceCount-1; current++ {
		m.setCompleted(current, m.completedAt(current+1))
	}
	m.pieceCount--
	m.data = m.data[:packedPieceByteCount(m.pieceCount)]
	m.clearUnusedBits()
	return nil
}

// Update changes whether the piece at index is complete and verified.
func (m *PieceMap) Update(index int, completed bool) error {
	if !m.validIndex(index) {
		return ErrPieceIndexOutOfRange
	}
	previous := m.completedAt(index)
	if previous == completed {
		return nil
	}
	if completed {
		m.completedPieces++
	} else {
		m.completedPieces--
	}
	m.setCompleted(index, completed)
	return nil
}

// Get reports whether the exact piece index is complete and verified.
func (m *PieceMap) Get(index int) (bool, error) {
	if !m.validIndex(index) {
		return false, ErrPieceIndexOutOfRange
	}
	return m.completedAt(index), nil
}

// Bytes returns an independent copy of the packed bitset-v1 bytes.
func (m *PieceMap) Bytes() []byte {
	return append([]byte(nil), m.data...)
}

func (m *PieceMap) PieceCount() int      { return m.pieceCount }
func (m *PieceMap) PieceSize() int64     { return m.pieceSize }
func (m *PieceMap) CompletedPieces() int { return m.completedPieces }

// MarshalJSON publishes the compact bytes without exposing mutable storage.
func (m PieceMap) MarshalJSON() ([]byte, error) {
	type pieceMapJSON struct {
		Encoding        string `json:"encoding"`
		PieceCount      int    `json:"pieceCount"`
		PieceSize       int64  `json:"pieceSize"`
		CompletedPieces int    `json:"completedPieces"`
		Data            []byte `json:"data"`
	}
	return json.Marshal(pieceMapJSON{
		Encoding:        PieceMapEncoding,
		PieceCount:      m.pieceCount,
		PieceSize:       m.pieceSize,
		CompletedPieces: m.completedPieces,
		Data:            m.data,
	})
}

func (m *PieceMap) validIndex(index int) bool {
	return m != nil && index >= 0 && index < m.pieceCount
}

func (m *PieceMap) completedAt(index int) bool {
	byteIndex, mask := piecePosition(index)
	return m.data[byteIndex]&mask != 0
}

func (m *PieceMap) setCompleted(index int, completed bool) {
	byteIndex, mask := piecePosition(index)
	if completed {
		m.data[byteIndex] |= mask
	} else {
		m.data[byteIndex] &^= mask
	}
}

func (m *PieceMap) clearUnusedBits() {
	if m.pieceCount == 0 || m.pieceCount%piecesPerByte == 0 {
		return
	}
	usedBits := uint(m.pieceCount % piecesPerByte)
	m.data[len(m.data)-1] &= byte(1<<usedBits) - 1
}

func piecePosition(index int) (byteIndex int, mask byte) {
	return index / piecesPerByte, byte(1 << uint(index%piecesPerByte))
}

func packedPieceByteCount(pieceCount int) int {
	return (pieceCount + piecesPerByte - 1) / piecesPerByte
}
