package ffmpeg

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

func TestSpoolSparseCrossBlock(t *testing.T) {
	s, err := newBlockSpool(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	offset := 7*SpoolBlockSize + 123
	data := bytes.Repeat([]byte{91}, int(3*SpoolBlockSize+37))
	if n, err := s.WriteAt(data, offset); err != nil || n != len(data) {
		t.Fatal(n, err)
	}
	if n, err, wake := s.readAt(make([]byte, 16), 7*SpoolBlockSize); n != 0 || err != nil || wake == nil {
		t.Fatal("sparse hole was exposed", n, err)
	}
	got := make([]byte, len(data))
	pos := 0
	for pos < len(got) {
		n, err, _ := s.readAt(got[pos:], offset+int64(pos))
		if err != nil || n == 0 {
			t.Fatal(n, err)
		}
		pos += n
	}
	if !bytes.Equal(data, got) {
		t.Fatal("cross-block contents differ")
	}
	if err := s.discard(8); err != nil {
		t.Fatal(err)
	}
	if n, _, wake := s.readAt(got[:1], 8*SpoolBlockSize); n != 0 || wake == nil {
		t.Fatal("deleted block remained readable")
	}
	if _, err := s.WriteAt(data, offset); err != nil {
		t.Fatal(err)
	}
	unique, received := s.progress()
	if unique != int64(len(data)) || received != 2*unique {
		t.Fatal(unique, received)
	}
}
func TestDiskInputDrainAndCleanup(t *testing.T) {
	s, err := NewDiskInput(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := bytes.Repeat([]byte{33}, int(3*SpoolBlockSize+99))
	if _, err := s.Write(data); err != nil {
		t.Fatal(err)
	}
	s.End(nil)
	got, err := io.ReadAll(s)
	if err != nil || !bytes.Equal(data, got) {
		t.Fatal("drain", err)
	}
	files, err := os.ReadDir(s.spool.dir)
	if err != nil || len(files) != 0 {
		t.Fatal("consumed blocks retained", len(files), err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.spool.dir); !os.IsNotExist(err) {
		t.Fatal("spool not removed", err)
	}
}
func TestDiskInputCancelAndWriteFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s, err := NewDiskInput(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cancel()
	if _, err := s.Read(make([]byte, 1)); err != context.Canceled {
		t.Fatal(err)
	}
	broken, err := NewDiskInput(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer broken.Close()
	if err := os.RemoveAll(broken.spool.dir); err != nil {
		t.Fatal(err)
	}
	if _, err := broken.Write([]byte{1}); err == nil {
		t.Fatal("missing disk error")
	}
	if _, err := broken.Read(make([]byte, 1)); err == nil {
		t.Fatal("disk error not propagated")
	}
}
