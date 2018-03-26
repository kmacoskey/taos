package app

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type CustomFormatter struct {
}

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 36
	gray    = 37
)

func (f *CustomFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
}

func (f *CustomFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}

func (f *CustomFormatter) needsQuoting(text string) bool {
	return true
}

func (f *CustomFormatter) printColored(b *bytes.Buffer, entry *log.Entry, keys []string, timestampFormat string) {
	var levelColor int
	switch entry.Level {
	case log.DebugLevel:
		levelColor = gray
	case log.WarnLevel:
		levelColor = yellow
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}

	levelText := strings.ToUpper(entry.Level.String())[0:4]

	sort.Strings(keys)

	// fmt.Fprintf(b, "\x1b[%dm[%s]\x1b[0m[%s] %-100s ", levelColor, levelText, entry.Time.Format(timestampFormat), entry.Message)
	fmt.Fprintf(b, "\x1b[%dm[%s]\x1b[0m[%s]", levelColor, levelText, entry.Time.Format(timestampFormat))
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " \x1b[%dm%s\x1b[0m=", levelColor, k)
		f.appendValue(b, v)
	}
	fmt.Fprintf(b, "\n\n\x1b[%dm%-10s%s\x1b[0m\n", levelColor, " ", entry.Message)
}

func (f *CustomFormatter) Format(entry *log.Entry) ([]byte, error) {
	var b *bytes.Buffer
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	timestampFormat := time.RFC3339

	f.printColored(b, entry, keys, timestampFormat)
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func InitLogger(config LoggingConfig) error {

	switch config.Format {
	case "text":
		log.SetFormatter(&log.TextFormatter{})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "custom":
		log.SetFormatter(new(CustomFormatter))
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	switch config.Level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.ErrorLevel)
	}

	// Output to stdout instead of the default stderr
	// Can be any io.Writer
	log.SetOutput(os.Stdout)

	logger := log.WithFields(log.Fields{
		"topic":   "taos",
		"package": "app",
		"context": "logger",
		"event":   "startup",
	})

	logger.Info("logging begins")

	return nil
}
