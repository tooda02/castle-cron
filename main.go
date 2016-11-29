/*
castle-cron
*/
package main

import (
	"flag"
	"fmt"
	"os"

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
	name      string               // name of server or job
	zkServer  string               // Zookeeper server
	zkTimeout = DEFAULT_ZK_TIMEOUT // Zookeeper session timeout
)

func init() {
	debug = flag.Bool("d", false, "Specifies whether to provide TRACE logging")
	isServer = flag.Bool("s", false, "Specifies whether to run as a castle-cron server daemon")
	force = flag.Bool("f", false, "Force setup even if server is already active")
	flag.StringVar(&name, "n", "", "Name of server or job; %h->hostname; %p->pid")
	flag.StringVar(&zkServer, "zk", "ZOOKEEPER_SERVERS", "Comma-separated list of Zookeeper server(s) in form host:port")
	flag.IntVar(&zkTimeout, "zt", DEFAULT_ZK_TIMEOUT, "Zookeeper session timeout in seconds")
}

func usage() {
	fmt.Printf("Usage: castle-cron [-d] [-zk server:port] [-zt timeout]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Parse()
	log.SetDebug(*debug)
	overrideFromEnv(&zkServer, "ZOOKEEPER_SERVERS")
	log.Trace.Printf("f(%t) s(%t) zk(%s) zt(%d)", *force, *isServer, zkServer, zkTimeout)
	if zkServer == "" {
		log.Error.Printf("Required Zookeeper server not provided")
		usage()
	}

	// Initialize cron server

	if e := cron.Init(zkServer, zkTimeout); e != nil {
		log.Error.Fatalf("Unable to connect to Zookeeper: %s", e.Error())
	} else {
		defer cron.Stop()
		log.Info.Printf("Connected to Zookeeper server %s with session timeout %d seconds", zkServer, zkTimeout)
	}
	if *isServer {
		cron.Run(name, *force)
	}
}

func overrideFromEnv(value *string, envname string) {
	if value != nil && (*value == "" || *value == envname) {
		*value = os.Getenv(envname)
	}
}
