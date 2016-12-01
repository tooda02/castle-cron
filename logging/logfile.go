/*
Handles the echo to file feature
*/

package logging

import (
	"log"
	"os"
	"strings"
)

var (
	logfile         *os.File
	logfileLogger   *log.Logger
	filename        string
	censoredWordMap map[string]string = map[string]string{}
)

// Return name of active log file (empty string if none)
func GetFile() string {
	return filename
}

/*
SetFile specifies the name of a file to which all subsequent log lines
are echoed. Any previously active log file is closed. Calling SetFile()
with an empty string for file name turns off the echo to file feature and
closes the file.  It is idiomatic to do this whenever using SetFile - see the example.

The logfile has logger options Ldate|Ltime|Lmicroseconds|Lshortfile. All
log lines are echoed to the log, even those suppressed from the console
due to debug settings.  Arguments are:

  name - name of file to write.  An empty string closes the log file.
*/
func SetFile(name string) (e error) {
	if name != filename {
		if logfile != nil {
			Trace.Printf("Stopped archiving log output to %s", filename)
			logfile.Close()
			logfile = nil
			logfileLogger = nil
		}

		filename = name

		if len(name) > 0 {
			logfile, e = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if e == nil {
				logfileLogger = log.New(logfile, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
				Trace.Printf("Now archiving log output to %s", filename)
			}
		}
	}
	return
}

// Specify words that will be censored from the output logfile
// Censored words are replaced with a string of eight asterisks
func SetCensoredWord(word string) {
	censoredWordMap[word] = "********"
}

// Specify words that will be censored from the output logfile,
// including their replacement string
func SetCensoredWordReplacement(word, replacement string) {
	censoredWordMap[word] = replacement
}

func (lw *LoggerWrapper) writeToLogfile(line string) {
	if logfileLogger != nil {
		logfileLogger.SetPrefix(lw.logger.Prefix())
		for word, replacement := range censoredWordMap {
			line = strings.Replace(line, word, replacement, -1)
		}
		logfileLogger.Output(4, line)
	}
}
