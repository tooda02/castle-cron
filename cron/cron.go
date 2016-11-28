package cron

import (
	"fmt"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	log "github.com/tooda02/castle-cron/logging"
)

var (
	zkConn *zk.Conn
)

// Setup server and connect to Zookeeper
func Init(server string, timeout int) (e error) {
	if zkConn != nil {
		e = fmt.Errorf("cron Init called more than once")
	} else {
		zks := strings.Split(server, ",")
		if zkConn, _, e = zk.Connect(zks, time.Duration(timeout)*time.Second); e == nil {
			log.Trace.Printf("Zookeeper connection %#v", zkConn)
		}
	}
	return e
}

// Shut down server
func Stop() {
	if zkConn == nil {
		log.Warning.Printf("cron Close called when cron server not started")
	} else {
		zkConn.Close()
		zkConn = nil
	}
}
