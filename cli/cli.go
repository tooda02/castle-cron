package cli

import (
	"flag"
	"fmt"

	"github.com/tooda02/castle-cron/cron"
)

/*
Run a job maintenance command of the form castle-cron add|upd|del|list jobname \"schedule\" cmd args...
*/
func RunCommand() error {
	switch flag.Arg(0) {
	case "add":
		return AddCommand()

	case "del":
		return DelCommand()

	case "help":
		return HelpCommand()

	case "list":
		return ListCommand()

	case "upd":
		return UpdCommand()
	}
	return fmt.Errorf("Unknown command \"%s\"; must be add, del, help, list, or upd", flag.Arg(0))
}

func AddCommand() error {
	return doAddOrUpdate(true)
}

func UpdCommand() error {
	return doAddOrUpdate(false)
}

func doAddOrUpdate(isAdd bool) (e error) {
	job := &cron.Job{}
	if flag.NArg() < 4 {
		e = fmt.Errorf("Not enough arguments for %s subcommand", flag.Arg(0))
	} else {
		job.Name = flag.Arg(1)
		job.Schedule = flag.Arg(2)
		job.Cmd = flag.Arg(3)
		if flag.NArg() > 4 {
			job.Args = flag.Args()[4:]
		}
		if _, e = job.SetNextRuntime(); e == nil {
			if isAdd {
				e = job.WriteToZk()
			} else {
				e = job.UpdateZk()
			}
		}
	}
	return
}

func DelCommand() error {
	return nil
}

func ListCommand() error {
	return nil
}
