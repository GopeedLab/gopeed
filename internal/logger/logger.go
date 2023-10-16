package logger

import (
	"github.com/GopeedLab/gopeed/pkg/util"
	"github.com/rs/zerolog"
	"io"
	"os"
	"path/filepath"
)

type Logger struct {
	zerolog.Logger
	logFile *os.File
}

func (l *Logger) CLose() {
	l.logFile.Close()
}

// NewLogger create a new logger
func NewLogger(logFile bool, logPath string) *Logger {
	var out io.Writer
	if logFile {
		// log to file
		logDir := filepath.Dir(logPath)
		if err := util.CreateDirIfNotExist(logDir); err != nil {
			panic(err)
		}
		var (
			logfile *os.File
			err     error
		)
		logfile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		out = logfile
	} else {
		out = os.Stdout
	}

	logger := &Logger{}
	if logFile {
		logger.logFile = out.(*os.File)
	}
	logger.Logger = zerolog.New(zerolog.ConsoleWriter{
		NoColor:    true,
		Out:        out,
		TimeFormat: "2006-01-02 15:04:05",
	}).With().Timestamp().Logger()
	return logger
}
