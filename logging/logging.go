/*
Package logging wraps the standard log package to provide additional features.

 - default flags
 - conditional trace logging
 - color coded logs
 - echo to file

It exports five services, Trace, Info, Warning, Error, and Plain.
Each service has Println() and Printf() methods for logging a line, but have
different default settings:

 Trace - prefix "TRACE"; color blue; output only when the debug flag is set
 Info  - prefix "INFO"; color black
 Warning - prefix "WARNING"; color magenta
 Error - prefix "ERROR"; color red
 Plain - no prefix, no date or time stamp, no file name

Settings can be modified with a service's SetConfig() method.
*/
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/daviddengcn/go-colortext"
)

/*
Use the Config struct with SetConfig() to specify settings for a logger. For example:
  Error.SetConfig(Config{Prefix: "OOPS"})
Settings omitted from a Config struct passed to SetConfig() are left unchanged.
See the Constants section for values.
*/
type Config struct {
	Prefix  string   // Prefix on output (e.g. TRACE, INFO). Set to NoPrefix to clear prefix
	Options int      // Logger options (e.g. logging.Ldate). Set to NoOptions to clear all options.
	Color   ct.Color // Output color (e.g. logging.Red). Set to NoColor to reset
	Debug   bool     // Output only when Debug option set
	NoDebug bool     // Output regardless of Debug option
}

type LoggerWrapper struct {
	logger *log.Logger
	color  ct.Color
	debug  bool
}

type globalFlags struct {
	debug bool
	color bool
}

const (
	// NoPrefix can be used as a setting for Config.Prefix to specify no logger prefix.
	NoPrefix = "-"

	// Options below can be ored together as a value for Config.Options to specify loggger options.
	// For example: {Config.Options: Ldate|Ltime|Lshortfile}
	Ldate         = log.Ldate         // The date: 2015/05/15
	Ltime         = log.Ltime         // The time: 01:23:23
	Lmicroseconds = log.Lmicroseconds // Microsecond resolution: 01:23:23.123123.  assumes Ltime
	Llongfile     = log.Llongfile     // Full file name and line number: /a/b/c/d.go:23
	Lshortfile    = log.Lshortfile    // Final file name element and line number: d.go:23. overrides Llongfile
	LstdFlags     = log.LstdFlags     // Initial values for the standard logger (Ldate | Ltime)

	// NoOptions can be used as a setting for Config.Options to specify no logger options.
	NoOptions = -1

	// Color values below can be used as a setting for Config.Color to set output log color.
	Black   ct.Color = ct.Black
	Red     ct.Color = ct.Red
	Green   ct.Color = ct.Green
	Yellow  ct.Color = ct.Yellow
	Blue    ct.Color = ct.Blue
	Magenta ct.Color = ct.Magenta
	Cyan    ct.Color = ct.Cyan

	// NoColor can be used as a setting for Config.Color to specify no log coloring.
	NoColor ct.Color = ct.Black
)

var (
	global globalFlags = globalFlags{debug: false, color: os.Getenv("SHIPPED_LOG_COLOR") != "0"}

	// Logger output file that's independent of os.Stdout
	logStdout *os.File = os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")

	// The Error logging service and its default settings.
	Error = &LoggerWrapper{
		logger: log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile),
		color:  ct.Red,
	}

	// The Info logging service and its default settings.
	Info = &LoggerWrapper{
		logger: log.New(logStdout, "INFO ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	// The Plain logging service and its default settings.
	Plain = &LoggerWrapper{
		logger: log.New(logStdout, "", 0),
	}

	// The Trace logging service and its default settings.
	Trace = &LoggerWrapper{
		logger: log.New(logStdout, "TRACE ", log.Ldate|log.Ltime|log.Lshortfile),
		debug:  true,
		color:  ct.Blue,
	}

	// The Warning logging service and its default settings.
	Warning = &LoggerWrapper{
		logger: log.New(logStdout, "WARNING ", log.Ldate|log.Ltime|log.Lshortfile),
		color:  ct.Magenta,
	}
)

// SetDebug sets the global debug flag controlling whether
// or not to output logs from loggers with their debug flag set
func SetDebug(debugArg bool) {
	global.debug = debugArg
}

// SetColor sets the global color flag controlling whether
// or not to change log color by service
func SetColor(colorArg bool) {
	global.color = colorArg
}

// Init is provided for backward compatibility; please use SetDebug() instead.
// Init sets the global debug flag; other config settings are ignored.
func Init(config Config) {
	global.debug = config.Debug
}

// Println writes a line to the log, honoring color and debug flags
func (lw *LoggerWrapper) Println(v ...interface{}) {
	lw.write(3, fmt.Sprintf("%v", v...))
}

// Print writes a formatted line to the log, honoring color and debug flags
func (lw *LoggerWrapper) Printf(format string, v ...interface{}) {
	lw.write(3, fmt.Sprintf(format, v...))
}

// Fatal writes a line to the log and then exits
func (lw *LoggerWrapper) Fatal(v ...interface{}) {
	lw.write(3, fmt.Sprintf("%v", v...))
	os.Exit(1)
}

// Fatalf writes a line to the log and then exits
func (lw *LoggerWrapper) Fatalf(format string, v ...interface{}) {
	lw.write(3, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Panic writes a line to the log and then panics
func (lw *LoggerWrapper) Panic(v ...interface{}) {
	s := fmt.Sprintf("%v", v...)
	lw.write(3, s)
	panic(s)
}

// Panicf writes a line to the log and then panics
func (lw *LoggerWrapper) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	lw.write(3, s)
	panic(s)
}

// Output writes a line to the log at a specified depth
func (lw *LoggerWrapper) Output(calldepth int, s string) error {
	return lw.write(calldepth+1, s)
}

/*
LogStackTrace writes a stack trace to the log
Arguments:

id - a human-readable id for the trace
topOnly - log only the top (first) goroutine
maxlines - maximum number of output lines to include in trace
*/
func (lw *LoggerWrapper) LogStackTrace(id string, topOnly bool, maxlines int) {
	buf := make([]byte, 1<<16)
	stackSize := runtime.Stack(buf, true)
	lines := strings.Split(string(buf[0:stackSize]), "\n")
	if len(lines) > 3 {
		if !topOnly {
			lines = append(lines[:1], lines[3:]...)
		} else {
			lines = lines[3:]
			for i, line := range lines {
				if len(line) == 0 {
					lines = lines[0:i]
					break
				}
			}
		}
	}
	if maxlines > 0 && len(lines) > maxlines*2 {
		lines = lines[0 : maxlines*2]
	}
	lw.Printf("Stack trace %s:\n%s\n", id, strings.Join(lines, "\n"))
}

// write writes a line to the log
func (lw *LoggerWrapper) write(calldepth int, line string) (e error) {
	if global.debug || !lw.debug {
		if lw.color == ct.None || !global.color {
			e = lw.logger.Output(calldepth, line)
		} else {
			ct.ChangeColor(lw.color, true, ct.None, false)
			e = lw.logger.Output(calldepth, line)
			ct.ResetColor()
		}
	}
	lw.writeToLogfile(line)
	return
}

// SetConfig changes the configuration of a logger
func (lw *LoggerWrapper) SetConfig(cfg Config) {
	if len(cfg.Prefix) > 0 {
		if cfg.Prefix == "-" {
			lw.logger.SetPrefix("")
		} else {
			lw.logger.SetPrefix(cfg.Prefix + " ")
		}
	}

	if cfg.Options != 0 {
		if cfg.Options == NoOptions {
			lw.logger.SetFlags(0)
		} else {
			lw.logger.SetFlags(cfg.Options)
		}
	}

	if cfg.Color != ct.None {
		if cfg.Color == NoColor {
			lw.color = ct.None
		} else {
			lw.color = cfg.Color
		}
	}

	if cfg.Debug {
		lw.debug = true
	}
	if cfg.NoDebug {
		lw.debug = false
	}
}

// Replace the logger with a new one with a different output writer
func (lw *LoggerWrapper) SetOutput(w io.Writer) {
	lw.logger = log.New(w, lw.logger.Prefix(), lw.logger.Flags())
}
