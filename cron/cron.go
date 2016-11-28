package cron

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	log "github.com/tooda02/castle-cron/logging"
)

var (
	zkConn   *zk.Conn
	hostname string
)

type Job struct {
	Name        string // Name of this job
	Cmd         string // Command to run
	NextRuntime time   // Time of next execution
	Hour        []int  // Hour to run
	Minute      []int  // Minute to run
	Month       []int  // Month to run
	Weekday     []int  // Weekday to run
}

// Connect to Zookeeper
func Init(server string, timeout int) (e error) {
	if zkConn != nil {
		e = fmt.Errorf("cron Init called more than once")
	} else {
		if hostname, e = os.Hostname(); e != nil {
			hostname = fmt.Sprintf("unknown-%d", os.Getpid())
			log.Warning.Printf("castle-cron running on unknown host (%s)", e.Error())
		}
		zks := strings.Split(server, ",")
		if zkConn, _, e = zk.Connect(zks, time.Duration(timeout)*time.Second); e == nil {
			log.Trace.Printf("Zookeeper connection %#v", zkConn)
		}
	}
	return
}

// Shut down
func Stop() {
	if zkConn == nil {
		log.Warning.Printf("cron Close called when cron server not started")
	} else {
		zkConn.Close()
		zkConn = nil
	}
}
