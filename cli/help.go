package cli

import (
	"flag"
	"fmt"

	//"github.com/gorhill/cronexpr"

	//log "github.com/tooda02/castle-cron/logging"
)

func HelpCommand() error {
	switch flag.Arg(1) {
	case "add":
		fmt.Printf("castle-cron [-d] [-zk server:port] [-zt timeout] add name \"sched\" cmd [args...]\n\n" +
			"Add a new job to the schedule\n" +
			"  -d\tProvide TRACE logging\n" +
			"  -zk\tComma-separated list of Zookeeper server(s) in form host:port (defaults to ZOOKEEPER_SERVERS)\n" +
			"  -zt\tZookeeper session timeout\n" +
			"  name\tName of job; must be unique\n" +
			"  sched\tcron-like blank-separated schedule string; see help sched for details\n" +
			"  cmd\tCommand to run\n" +
			"  args\tCommand arguments\n")

	case "del":
		fmt.Printf("castle-cron [-d] [-zk server:port] [-zt timeout] del name\n\n" +
			"Delete a job from the schedule\n" +
			"  -d\tProvide TRACE logging\n" +
			"  -zk\tComma-separated list of Zookeeper server(s) in form host:port (defaults to ZOOKEEPER_SERVERS)\n" +
			"  -zt\tZookeeper session timeout\n" +
			"  name\tName of job; must already exist\n")

	case "list":
		fmt.Printf("castle-cron [-d] [-zk server:port] [-zt timeout] list [name]\n\n" +
			"Delete a job from the schedule\n" +
			"  -d\tProvide TRACE logging\n" +
			"  -zk\tComma-separated list of Zookeeper server(s) in form host:port (defaults to ZOOKEEPER_SERVERS)\n" +
			"  -zt\tZookeeper session timeout\n" +
			"  name\tName of job to list; can be omitted to list all jobs\n")

	case "sched":
		fmt.Printf("Job schedule; must be a quoted string containing 5 -7 blank-separated values.\n\n" +
			"  Field name\tMandatory?\tAllowed values\tAllowed special characters\n" +
			"  ----------\t----------\t--------------\t--------------------------\n" +
			"  Seconds\tNo\t\t0-59\t\t* / , -\n" +
			"  Minutes\tYes\t\t0-59\t\t* / , -\n" +
			"  Hours\t\tYes\t\t0-23\t\t* / , -\n" +
			"  Day of month\tYes\t\t1-31\t\t* / , - L W\n" +
			"  Month\t\tYes\t\t1-12 or JAN-DEC\t* / , -\n" +
			"  Day of week\tYes\t\t0-6 or SUN-SAT\t* / , - L #\n" +
			"  Year\t\tNo\t\t1970–2099\t* / , -\n")

	case "upd":
		fmt.Printf("castle-cron [-d] [-zk server:port] [-zt timeout] upd name \"sched\" cmd [args...]\n\n" +
			"Update a job in the schedule\n" +
			"  -d\tProvide TRACE logging\n" +
			"  -zk\tComma-separated list of Zookeeper server(s) in form host:port (defaults to ZOOKEEPER_SERVERS)\n" +
			"  -zt\tZookeeper session timeout\n" +
			"  name\tName of job; must already exist\n" +
			"  sched\tcron-like blank-separated schedule string; see help sched for details\n" +
			"  cmd\tCommand to run\n" +
			"  args\tCommand arguments\n")
	}
	return fmt.Errorf("Unknown command \"%s\"; must be add, del, list, sched, or upd", flag.Arg(1))
}
