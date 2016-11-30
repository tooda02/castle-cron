/*
castle-cron
*/
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tooda02/castle-cron/cli"
	"github.com/tooda02/castle-cron/cron"
	log "github.com/tooda02/castle-cron/logging"
)

const (
	DEFAULT_ZK_TIMEOUT = 10
)

var (
	debug     *bool                // true => TRACE logging on
	isServer  *bool                // true => start server daemon
	force     *bool                // true => force setup even if server already active
	help      *bool                // true => print usage and exit
	name      string               // name of server
	zkServer  string               // Zookeeper server
	zkTimeout = DEFAULT_ZK_TIMEOUT // Zookeeper session timeout
)

func init() {
	debug = flag.Bool("d", false, "Provide TRACE logging")
	help = flag.Bool("h", false, "Print help and exit")
	isServer = flag.Bool("s", false, "Run as a castle-cron server daemon")
	force = flag.Bool("f", false, "Force running server even if server of that name is already active")
	flag.StringVar(&name, "n", "", "Name of server when -s specified (default %h); %h->hostname; %p->pid")
	flag.StringVar(&zkServer, "zk", "ZOOKEEPER_SERVERS", "Comma-separated list of Zookeeper server(s) in form host:port")
	flag.IntVar(&zkTimeout, "zt", DEFAULT_ZK_TIMEOUT, "Zookeeper session timeout in seconds")
}

func usage(rc int) {
	fmt.Printf("Usage: castle-cron [-d] [-f] [-s] [-n name] [-zk server:port] [-zt timeout]\n")
	fmt.Printf("       castle-cron add|upd|del|list jobname \"schedule\" cmd args...\n\n")
	fmt.Printf("Run a castle-cron job scheduler server and/or maintain its job queue.\n")
	fmt.Printf("The second form of the command maintains the job queue.  Use castle-cron help <cmd> for help on its subcommands.\n\n")
	flag.PrintDefaults()
	os.Exit(rc)
}

func main() {
	flag.Parse()
	if *help || (flag.NArg() == 1 && flag.Arg(0) == "help") {
		usage(0)
	}
	log.SetDebug(*debug)
	overrideFromEnv(&zkServer, "ZOOKEEPER_SERVERS")
	log.Trace.Printf("s(%t) zk(%s) zt(%d)", isServer, zkServer, zkTimeout)
	if zkServer == "" {
		log.Error.Printf("Required Zookeeper server not provided")
		usage(2)
	}

	// Connect to Zookeeper and initialize for this run

	if err := cron.Init(zkServer, zkTimeout); err != nil {
		log.Error.Fatalf("Unable to connect to Zookeeper: %s", err.Error())
	} else {
		defer cron.Stop()
		log.Info.Printf("Connected to Zookeeper server %s with session timeout %d seconds", zkServer, zkTimeout)
	}

	// If non-flag arguments were specified, execute the CLI command

	if flag.NArg() > 0 {
		if err := cli.RunCommand(flag.Args()); err != nil {
			log.Error.Printf(err.Error())
			os.Exit(1)
		}
	}

	// If -s was specified, run a castle-cron server
	if *isServer {
		cron.Run(name, *force)
	}
}

func overrideFromEnv(value *string, envname string) {
	if value != nil && (*value == "" || *value == envname) {
		*value = os.Getenv(envname)
	}
}
