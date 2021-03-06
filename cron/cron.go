/*
Package cron supports a distributed cluster for running scheduled jobs.
This package supports both the server and a CLI for maintaining the schedule.
*/
package cron

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	log "github.com/tooda02/castle-cron/logging"
)

// Zookeeper nodes used by this application
const (
	APP_NAME      = "castle-cron"
	NAMESPACE     = "/" + APP_NAME         // Root node; can be set to empty string if desired
	PATH_SERVERS  = NAMESPACE + "/servers" // Root of ephemeral nodes for each server
	PATH_JOBS     = NAMESPACE + "/jobs"    // Root of nodes for each job
	PATH_NEXT_JOB = NAMESPACE + "/nextjob" // Single node holding next job to run
	PATH_JOBLOCK  = NAMESPACE + "/joblock" // Single node holding lock
)

var (
	zkConn     *zk.Conn // Zookeeper connection for both server and CLI
	hostname   string   // hostname (set for server only)
	serverName string   // server name (set for server only; defaults to hostname)
	isRunning  bool     // Server is running
)

// Connect to Zookeeper
func Init(server string, timeout int) (e error) {
	if zkConn != nil {
		e = fmt.Errorf("cron Init() called more than once")
	} else {
		if hostname, e = os.Hostname(); e != nil {
			hostname = fmt.Sprintf("unknown-%d", os.Getpid())
			log.Warning.Printf("castle-cron running on unknown host (%s)", e.Error())
		}
		zks := strings.Split(server, ",")
		if zkConn, _, e = zk.Connect(zks, time.Duration(timeout)*time.Second); e == nil {
			log.Trace.Printf("Zookeeper connection %#v", zkConn)
			createIfNecessary(NAMESPACE)
			createIfNecessary(PATH_JOBS)
			createIfNecessary(PATH_NEXT_JOB)
			createIfNecessary(PATH_SERVERS)
			createIfNecessary(PATH_JOBLOCK)
			lock = zk.NewLock(zkConn, PATH_JOBLOCK, zk.WorldACL(zk.PermAll))
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
		log.Trace.Printf("Closed Zookeeper connection")
	}
}

// Create Zookeeper znode /servers/<serverName>
func setServerName(name string, force bool) error {
	if name == "" {
		serverName = hostname
	} else {
		serverName = strings.Replace(name, "%h", hostname, -1)
		serverName = strings.Replace(serverName, "%p", fmt.Sprintf("%d", os.Getpid()), -1)
	}
	serverPath := fmt.Sprintf("%s/%s", PATH_SERVERS, serverName)
	if exists, _, err := zkConn.Exists(serverPath); err != nil {
		return fmt.Errorf("Unable to check server existence: %s", err.Error())
	} else if exists {
		if !force {
			return fmt.Errorf("Server %s is already running.  Use -f argument to run anyway.", serverName)
		}
		log.Warning.Printf("Deleting previously-existing znode %s", serverPath)
		zkConn.Delete(serverPath, -1)
	}
	if _, err := zkConn.Create(serverPath, []byte(APP_NAME), zk.FlagEphemeral, zk.WorldACL(zk.PermAll)); err != nil {
		return fmt.Errorf("Unable to create znode %s: %s", serverPath, err.Error())
	} else {
		log.Trace.Printf("Created znode %s", serverPath)
	}
	return nil
}

// Tell user this server has started, log a list of all servers running,
// and report when any other server starts or stops
func reportServers() error {
	allServers, _, watch, err := zkConn.ChildrenW(PATH_SERVERS)
	if err != nil {
		return fmt.Errorf("Can't get list of %s servers: %s", APP_NAME, err.Error())
	}
	serverMap := map[string]int{}
	mapUpdCount := 1
	if len(allServers) == 0 {
		// Shouldn't ever happen as the list should include this server
		log.Warning.Printf("%s server %s started; server list missing", APP_NAME, serverName)
	} else {
		for _, server := range allServers {
			serverMap[server] = mapUpdCount
		}
		sort.Strings(allServers)
	}
	log.Info.Printf("%s server %s started; %d server(s) running %v", APP_NAME, serverName, len(allServers), allServers)
	go func() {
		for isRunning {
			evt := <-watch
			if evt.Err != nil {
				log.Error.Printf("Error watching for changes in server list: %s", evt.Err.Error())
				break
			}
			allServers, _, watch, err = zkConn.ChildrenW(PATH_SERVERS)
			if err != nil {
				log.Error.Printf("Can't get updated list of %s servers: %s", APP_NAME, err.Error())
				break
			}
			newServers := []string{}
			deletedServers := []string{}
			mapUpdCount++
			for _, server := range allServers {
				if serverMap[server] == 0 && server != serverName {
					newServers = append(newServers, server)
				}
				serverMap[server] = mapUpdCount
			}
			for server, serverUpdCount := range serverMap {
				if serverUpdCount != mapUpdCount {
					deletedServers = append(deletedServers, server)
					delete(serverMap, server)
				}
			}
			sort.Strings(allServers)
			if len(newServers) > 0 {
				sort.Strings(newServers)
				log.Info.Printf("New %s server(s) %v started; %d server(s) now running %v", APP_NAME, newServers, len(allServers), allServers)
			}
			if len(deletedServers) > 0 {
				sort.Strings(deletedServers)
				log.Info.Printf("%s server(s) %v stopped; %d server(s) now running %v", APP_NAME, deletedServers, len(allServers), allServers)
			}
		}
		if isRunning {
			log.Error.Printf("%s server change reporting terminated due to previous error", APP_NAME)
		}
	}()
	return nil
}

// Check whether a specified znode exists and create if it does not
func createIfNecessary(znode string) {
	if znode != "" {
		if exists, _, err := zkConn.Exists(znode); err != nil {
			log.Error.Fatalf("Unable to check for %s: %s", znode, err.Error())
		} else if !exists {
			if _, err = zkConn.Create(znode, []byte{}, 0x0, zk.WorldACL(zk.PermAll)); err != nil {
				log.Error.Fatalf("Unable to create %s: %s", znode, err.Error())
			}
		}
	}
}
