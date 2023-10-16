package logger

import (
	"os"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger(false, "")
	logger.Info().Msg("test")
}

func TestNewLoggerFile(t *testing.T) {
	logPath := "./testdata/test.log"
	logger := NewLogger(true, logPath)
	defer func() {
		logger.CLose()
		os.Remove(logPath)
	}()
	logger.Info().Msg("test")

	// read log file, if not exist or empty, test fail.
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() == 0 {
		t.Fatal("log file is empty")
	}
}
