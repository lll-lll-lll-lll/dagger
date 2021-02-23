package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"io"
	"strings"
	"time"

	"github.com/mitchellh/colorstring"
	"github.com/rs/zerolog"
)

var colorize = colorstring.Colorize{
	Colors: colorstring.DefaultColors,
	Reset:  true,
}

type Console struct {
	Out       io.Writer
	maxLength int
}

func (c *Console) Write(p []byte) (n int, err error) {
	event := map[string]interface{}{}
	d := json.NewDecoder(bytes.NewReader(p))
	if err := d.Decode(&event); err != nil {
		return n, fmt.Errorf("cannot decode event: %s", err)
	}

	source := c.parseSource(event)
	if len(source) > c.maxLength {
		c.maxLength = len(source)
	}

	return fmt.Fprintln(c.Out,
		colorize.Color(fmt.Sprintf("%s %s %s%s%s",
			c.formatTimestamp(event),
			c.formatLevel(event),
			c.formatSource(source),
			c.formatMessage(event),
			c.formatFields(event),
		)))
}

func (c *Console) formatLevel(event map[string]interface{}) string {
	level := zerolog.DebugLevel
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		level, _ = zerolog.ParseLevel(l)
	}

	switch level {
	case zerolog.TraceLevel:
		return "[magenta]TRC[reset]"
	case zerolog.DebugLevel:
		return "[yellow]DBG[reset]"
	case zerolog.InfoLevel:
		return "[green]INF[reset]"
	case zerolog.WarnLevel:
		return "[red]WRN[reset]"
	case zerolog.ErrorLevel:
		return "[red]ERR[reset]"
	case zerolog.FatalLevel:
		return "[red]FTL[reset]"
	case zerolog.PanicLevel:
		return "[red]PNC[reset]"
	default:
		return "[bold]???[reset]"
	}
}

func (c *Console) formatTimestamp(event map[string]interface{}) string {
	ts, ok := event[zerolog.TimestampFieldName].(string)
	if !ok {
		return "???"
	}

	t, err := time.Parse(zerolog.TimeFieldFormat, ts)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("[dark_gray]%s[reset]", t.Format(time.Kitchen))
}

func (c *Console) formatMessage(event map[string]interface{}) string {
	message, ok := event[zerolog.MessageFieldName].(string)
	if !ok {
		return ""
	}
	message = strings.TrimSpace(message)

	if err, ok := event[zerolog.ErrorFieldName].(string); ok && err != "" {
		message = message + ": " + err
	}

	level := zerolog.DebugLevel
	if l, ok := event[zerolog.LevelFieldName].(string); ok {
		level, _ = zerolog.ParseLevel(l)
	}

	switch level {
	case zerolog.TraceLevel:
		return fmt.Sprintf("[dim]%s[reset]", message)
	case zerolog.DebugLevel:
		return fmt.Sprintf("[dim]%s[reset]", message)
	case zerolog.InfoLevel:
		return message
	case zerolog.WarnLevel:
		return fmt.Sprintf("[yellow]%s[reset]", message)
	case zerolog.ErrorLevel:
		return fmt.Sprintf("[red]%s[reset]", message)
	case zerolog.FatalLevel:
		return fmt.Sprintf("[red]%s[reset]", message)
	case zerolog.PanicLevel:
		return fmt.Sprintf("[red]%s[reset]", message)
	default:
		return message
	}
}

func (c *Console) parseSource(event map[string]interface{}) string {
	source := "system"
	if task, ok := event["component"].(string); ok && task != "" {
		source = task
	}
	return source
}

func (c *Console) formatSource(source string) string {
	return fmt.Sprintf("[%s]%s | [reset]",
		hashColor(source),
		source,
	)
}

func (c *Console) formatFields(entry map[string]interface{}) string {
	// these are the fields we don't want to expose, either because they're
	// already part of the Log structure or because they're internal
	fieldSkipList := map[string]struct{}{
		zerolog.MessageFieldName:   {},
		zerolog.LevelFieldName:     {},
		zerolog.TimestampFieldName: {},
		zerolog.ErrorFieldName:     {},
		zerolog.CallerFieldName:    {},
		"component":                {},
	}

	fields := []string{}
	for key, value := range entry {
		if _, ok := fieldSkipList[key]; ok {
			continue
		}
		switch v := value.(type) {
		case string:
			fields = append(fields, fmt.Sprintf("%s=%s", key, v))
		case int:
			fields = append(fields, fmt.Sprintf("%s=%v", key, v))
		case float64:
			dur := time.Duration(v) * time.Millisecond
			s := dur.Round(100 * time.Millisecond).String()
			fields = append(fields, fmt.Sprintf("%s=%s", key, s))
		case nil:
			fields = append(fields, fmt.Sprintf("%s=null", key))
		}
	}

	if len(fields) == 0 {
		return ""
	}
	return fmt.Sprintf("    [dim]%s[reset]", strings.Join(fields, " "))
}

// hashColor returns a consistent color for a given string
func hashColor(text string) string {
	colors := []string{
		"green",
		"light_green",
		"light_blue",
		"blue",
		"magenta",
		"light_magenta",
		"light_yellow",
		"cyan",
		"light_cyan",
		"red",
		"light_red",
	}
	h := adler32.Checksum([]byte(text))
	return colors[int(h)%len(colors)]
}
