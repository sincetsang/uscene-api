package sdlog

import (
	"fmt"
	"io"
	"os"
	"strings"

	"TMA/pkg/sdparse"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	Init(Options{
		Level:    "debug",
		Stdout:   true,
		Filename: "",
	})
	//logrus.AddHook(&filelineHook{})
}

type GlobalFieldsHook struct {
	Fields logrus.Fields
}

func (hook *GlobalFieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *GlobalFieldsHook) Fire(entry *logrus.Entry) error {
	for key, value := range hook.Fields {
		// Add the global fields to each log entry
		entry.Data[key] = value
	}
	return nil
}

type Options struct {
	Level       string `json:"level" toml:"level"`
	Stdout      bool   `json:"stdout" toml:"stdout"`
	Filename    string `json:"file" toml:"file"`
	MaxSizeMB   int    `json:"max_size" toml:"max_size"`
	MaxAgeDays  int    `json:"max_age" toml:"max_age"`
	MaxBackups  int    `json:"max_backups" toml:"max_backups"`
	Formatter   string `json:"formatter" toml:"formatter"`
	ServiceName string `json:"service_name" toml:"service_name"`
}

func InitLogger(opts Options, logger *logrus.Logger) *logrus.Logger {
	if logger == nil {
		return logger
	}

	pretty := sdparse.BoolDef(os.Getenv("SDLOG_PRETTY"), false)
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 200
	}

	setPrettyFormat := func(pretty bool) {
		var tf logrus.Formatter

		switch opts.Formatter {
		case "json":
			tf = &logrus.JSONFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			}
		default:
			tf = &logrus.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: "2006-01-02 15:04:05",
				DisableColors:   !pretty,
			}
		}

		logger.SetFormatter(tf)
	}

	// level
	var logLevel = logrus.DebugLevel
	if opts.Level != "" {
		logLevel1, err := logrus.ParseLevel(strings.ToLower(opts.Level))
		if err != nil {
			logLevel = logrus.DebugLevel // default
		} else {
			logLevel = logLevel1
		}
	}
	logger.SetLevel(logLevel)

	// format
	if opts.Filename != "" {
		setPrettyFormat(false)
	} else {
		setPrettyFormat(pretty)
	}

	// output
	logger.SetOutput(GetOutPut(&opts))
	logger.AddHook(&GlobalFieldsHook{
		Fields: logrus.Fields{"module": opts.ServiceName},
	})
	return logger
}

func Init(opts Options) {
	InitLogger(opts, logrus.StandardLogger())
}

func GetOutPut(opts *Options) io.Writer {
	var output1, output2 io.Writer
	getOutput1 := func() io.Writer {
		if opts.Stdout {
			return os.Stdout
		} else {
			return nil
		}
	}
	switch strings.ToLower(opts.Filename) {
	case "", "stdout":
		output1, output2 = os.Stdout, nil
	case "stderr":
		output1, output2 = getOutput1(), os.Stderr
	case "discard", "disabled", "disable", "off":
		output1, output2 = getOutput1(), nil
	default:
		output1 = getOutput1()
		output2 = &lumberjack.Logger{
			Filename:   opts.Filename,
			MaxSize:    opts.MaxSizeMB,
			MaxAge:     opts.MaxAgeDays,
			MaxBackups: opts.MaxBackups,
		}
	}
	if output1 != nil && output2 != nil {
		return io.MultiWriter(output1, output2)
	} else if output1 != nil && output2 == nil {
		return output1
	} else if output1 == nil && output2 != nil {
		return output2
	} else {
		return io.Discard
	}
}

type Fields = logrus.Fields

var (
	// Context
	WithContext = logrus.WithContext

	// Level
	SetLevel = logrus.SetLevel
	GetLevel = logrus.GetLevel

	// With info
	//WithError  = logrus.WithError
	WithField  = logrus.WithField
	WithFields = logrus.WithFields

	// Log
	Debug   = logrus.Debug
	Print   = logrus.Print
	Info    = logrus.Info
	Warn    = logrus.Warn
	Warning = logrus.Warning
	Error   = logrus.Error
	Panic   = logrus.Panic
	Fatal   = logrus.Fatal

	// Logf
	Debugf   = logrus.Debugf
	Printf   = logrus.Printf
	Infof    = logrus.Infof
	Warnf    = logrus.Warnf
	Warningf = logrus.Warningf
	Errorf   = logrus.Errorf
	Panicf   = logrus.Panicf
	Fatalf   = logrus.Fatalf

	// Others
	ErrorKey = logrus.ErrorKey

	ModuleLog = func(module string) *logrus.Entry {
		return logrus.WithField("module", module)
	}
)

func WithError(err error) *logrus.Entry {
	return logrus.WithField(ErrorKey, fmt.Sprintf("%+v", err))
}
