package log

import (
	"io"
	stdlog "log"
	"os"

	"github.com/sirupsen/logrus"
)

type Fields logrus.Fields

var log *logrus.Logger

func Init(logLevel string, logFilePath string) {
	log = logrus.New()
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		stdlog.Fatalf("set log level faild: %s", err)
	}

	log.SetLevel(level)

	f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		defer f.Close()
		log.Fatalf("error opening file: %v", err)
	}

	writer := io.MultiWriter(os.Stdout, f)
	log.SetOutput(writer)
}

func Trace(args ...interface{}) {
	log.Trace(args...)
}

func Tracef(format string, args ...interface{}) {
	log.Tracef(format, args...)
}

func Debug(args ...interface{}) {
	log.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Info(args ...interface{}) {
	log.Info(args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Warn(args ...interface{}) {
	log.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

func Error(args ...interface{}) {
	log.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func Panic(args ...interface{}) {
	log.Panic(args...)
}

func Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}
