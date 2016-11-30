/*
Package cli implements the job maintenance CLI for castle-cron
*/
package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/tooda02/castle-cron/cron"
)

/*
Run a job maintenance command of the form castle-cron add|upd|del|list jobname \"schedule\" cmd args...
*/
func RunCommand(args []string) error {
	switch args[0] {
	case "add":
		return AddCommand(args)

	case "del":
		return DelCommand(args)

	case "help":
		return HelpCommand(args)

	case "list":
		return ListCommand(args)

	case "upd":
		return UpdCommand(args)
	}
	return fmt.Errorf("Unknown command \"%s\"; must be add, del, help, list, or upd", flag.Arg(0))
}

// Add a new job and store in Zookeeper
func AddCommand(args []string) (e error) {
	var job *cron.Job
	if job, e = buildJobFromArgs(args); e == nil {
		e = job.WriteToZk()
	}
	return
}

// Update an existing job in Zookeeper
func UpdCommand(args []string) (e error) {
	var job *cron.Job
	if job, e = buildJobFromArgs(args); e == nil {
		e = job.UpdateZk()
	}
	return
}

func buildJobFromArgs(args []string) (job *cron.Job, e error) {
	job = &cron.Job{}
	if len(args) < 4 {
		e = fmt.Errorf("Not enough arguments for %s subcommand", args[0])
	} else {
		job.Name = args[1]
		job.Schedule = args[2]
		job.Cmd = args[3]
		if len(args) > 4 {
			job.Args = args[4:]
		}
		_, e = job.SetNextRuntime()
	}
	return
}

// Delete a job from Zookeeper
func DelCommand(args []string) (e error) {
	if len(args) < 2 {
		e = fmt.Errorf("Job name not supplied for %s subcommand", args[0])
	} else {
		job := cron.Job{Name: args[1]}
		e = job.DeleteFromZk()
	}
	return nil
}

// List a job or all jobs
func ListCommand(args []string) error {
	var name string
	if len(args) > 1 {
		name = args[1]
	}
	if jobs, err := cron.ListJobs(name); err != nil {
		return err
	} else {
		output := []string{
			"Name | Next Runtime | Error | Command",
		}
		for _, job := range jobs {
			errFlag := ""
			if job.HasError {
				errFlag = "Err"
			}
			output = append(output,
				job.Name+" | "+
					job.NextRuntime.Format("2006-01-02 15:04:05.99999999")+" | "+
					errFlag+" | "+
					job.Cmd+" "+strings.Join(job.Args, " "))
		}
		fmt.Println(output)
	}
	return nil
}
