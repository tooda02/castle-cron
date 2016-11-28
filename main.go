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

var (
	debug     *bool       // true => TRACE logging on
	zkServer  string      // Zookeeper server
	zkTimeout int    = 30 // Zookeeper session timeout

	hostname string
	e        error
)

func init() {
	debug = flag.Bool("d", false, "Specifies whether to provide TRACE logging")
	flag.StringVar(&zkServer, "zk", "ZOOKEEPER_SERVERS", "Zookeeper server in form host:port[,host:port...]")
	flag.IntVar(&zkTimeout, "t", 30, "Zookeeper session timeout in seconds")
}

func usage() {
	fmt.Printf("Usage: castle-cron [-d] [-zk server:port] [-t timeout]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Parse()
	log.SetDebug(*debug)
	overrideFromEnv(&zkServer, "ZOOKEEPER_SERVERS")
	log.Trace.Printf("zk(%s) t(%d)", zkServer, zkTimeout)
	if zkServer == "" {
		log.Error.Printf("Required Zookeeper server not provided")
		usage()
	}
	if hostname, e = os.Hostname(); e != nil {
		log.Warning.Printf("castle-cron starting on unknown host (%s)", e.Error())
	} else {
		log.Info.Printf("castle-cron starting on %s", hostname)
	}

	// Initialize cron server

	if e = cron.Init(zkServer, zkTimeout); e != nil {
		log.Error.Fatalf("Unable to connect to Zookeeper: %s", e.Error())
	} else {
		defer cron.Stop()
		log.Info.Printf("Connected to Zookeeper server %s with session timeout %d seconds", zkServer, zkTimeout)
	}

}

func overrideFromEnv(value *string, envname string) {
	if value != nil && len(*value) == 0 {
		*value = os.Getenv(envname)
	}
}
