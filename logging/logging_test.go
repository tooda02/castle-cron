package logging

/*
Not a conventional unit test, as there are no assertions.
It needs to be monitored by a human checking that the messages
printed by the various log statements are true about content and color
*/

import (
	"testing"
)

/*
func TestLog(t *testing.T) {
	SetFile("logging_test.log")
	defer SetFile("")
	Error.Printf("This is an error log that should be in red")
	Warning.Printf("This is an warning log that should be in magenta")
	Info.Printf("This is an info log in black that should NOT be followed by a trace log")
	Trace.Printf("This is an trace log that you should not see")
	Plain.Printf("This is an plain log with nothing but the text (no date, time, or file). It's not a trace log.")
	SetDebug(true)
	Trace.Printf("This is an trace log that should get printed in blue")
	Error.SetConfig(Config{Prefix: "OOPS"})
	Error.Printf("This is an oops log")
	Info.SetConfig(Config{Options: NoOptions, Color: Green})
	Info.Printf("This is an info log that should be in green without a date or source line")
	Plain.SetConfig(Config{Color: Magenta})
	Plain.Printf("This is an plain log that should be in magenta")
}
*/

func TestStackTrace(t *testing.T) {
	Warning.Printf("true, 0")
	Info.LogStackTrace("Hello", true, 0)
	Warning.Printf("true, 2")
	Info.LogStackTrace("Hello", true, 2)
	Warning.Printf("false, 0")
	Info.LogStackTrace("Hello", false, 0)
}

// Function examples

func ExampleSetFile() {
	SetFile("logfile.log") // Echo all subsequent log lines to file logfile.log
	defer SetFile("")      // Close log file at termination
}

func ExampleLoggerWrapper_SetConfig() {
	Plain.SetConfig(Config{Color: Magenta})
	Info.SetConfig(Config{Options: NoOptions, Prefix: NoPrefix})
}
