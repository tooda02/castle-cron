package cron

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/curator-go/curator"
	"github.com/curator-go/curator/recipes"
	log "github.com/tooda02/castle-cron/logging"
)

const (
	PATH_SERVERS = "/servers"
	APP_NAME     = "castle-cron"
)

var (
	client       curator.CuratorFramework // Curator client framework
	hostname     string                   // Name of server host
	serverName   string                   // Name of this server
	otherServers []string                 // Names of all other servers
	isRunning    bool                     // Server is running
)

type Job struct {
	Name        string    // Name of this job
	Cmd         string    // Command to run
	NextRuntime time.Time // Time of next execution
	Hour        []int     // Hour to run
	Minute      []int     // Minute to run
	Month       []int     // Month to run
	Weekday     []int     // Weekday to run
}

// Initialize Zookeeper connection with Curator
func Init(connString string, timeout int) (e error) {
	if client != nil && client.Started() {
		e = fmt.Errorf("cron Init called more than once")
	} else {
		if hostname, e = os.Hostname(); e != nil {
			hostname = fmt.Sprintf("unknown-%d", os.Getpid())
			log.Warning.Printf("castle-cron running on unknown host (%s)", e.Error())
		}
		retryPolicy := curator.NewExponentialBackoffRetry(time.Second, 3, time.Duration(timeout)*time.Second)
		client = curator.NewClient(connString, retryPolicy)
		client.Start()
		fmt.Printf("\n") // Circumvent minor Curator bug
	}
	return
}

// Run castle-cron server daemon
func Run(name string, force bool) {

	if err := setServerName(name, force); err != nil {
		log.Error.Fatalf("Unable to set server name: %s", err.Error())
	}

	// Report names of other servers and set watch for changes

	if allServers, err := client.GetChildren().Watched().ForPath(PATH_SERVERS); err != nil {
		log.Error.Printf("Can't get server list: %s", err.Error())
	} else {
		log.Warning.Printf("allServers(%#v)", allServers)
		otherServers = []string{}
		for _, server := range allServers {
			if server != serverName {
				otherServers = append(otherServers, server)
			}
		}
	}
	startWatcher()

	// Start the server

	log.Info.Printf("castle-cron server %s starting on %s; %d other server(s) running %v", serverName, hostname, len(otherServers), otherServers)
	isRunning = true
	for isRunning {
		log.Info.Printf("castle-cron server %s still running", serverName)
		time.Sleep(time.Duration(10) * time.Second)
	}
}

// Shut down
func Stop() {
	if client == nil || !client.Started() {
		log.Warning.Printf("cron Close called when cron server not started")
	} else {
		client.Close()
	}
}

// Watch for events
func startWatcher() {
	listener := curator.NewCuratorListener(eventOccured)
	client.CuratorListenable().AddListener(listener)
}

func eventOccured(client curator.CuratorFramework, event curator.CuratorEvent) error {
	log.Warning.Printf("Got event %#v", event)
	log.Trace.LogStackTrace("Event", true, 20)
	listener := curator.NewCuratorListener(eventOccured2)
	client.CuratorListenable().AddListener(listener)
	return nil
}

func eventOccured2(client curator.CuratorFramework, event curator.CuratorEvent) error {
	log.Warning.Printf("Got event %#v", event)
	log.Trace.LogStackTrace("Event", true, 20)
	listener := curator.NewCuratorListener(eventOccured)
	client.CuratorListenable().AddListener(listener)
	return nil
}

// Create Zookeeper znode /servers/<serverName>
// To ensure no duplicates, we do this after taking a lock on /servers
func setServerName(name string, force bool) (e error) {
	if name == "" {
		serverName = hostname
	} else {
		serverName = strings.Replace(name, "%h", hostname, -1)
		serverName = strings.Replace(serverName, "%p", fmt.Sprintf("%d", os.Getpid()), -1)
	}
	if lock, err := recipes.NewInterProcessMutex(client, PATH_SERVERS); err != nil {
		return fmt.Errorf("Can't set up servers lock: %s", err.Error())
	} else if acquired, err := lock.Acquire(); err != nil {
		return fmt.Errorf("Error attempting to acquire servers lock: %s", err.Error())
	} else if !acquired {
		return fmt.Errorf("Can't acquire servers lock")
	} else {
		defer lock.Release()

		// Ensure root path /servers exists

		if data, err := client.GetData().ForPath(PATH_SERVERS); err != nil {
			return fmt.Errorf("Error attempting to check existence of %s: %s", PATH_SERVERS, err.Error())
		} else if data == nil || len(data) == 0 || string(data) != APP_NAME {
			if _, err = client.Create().ForPathWithData(PATH_SERVERS, []byte(APP_NAME)); err != nil {
				return fmt.Errorf("Unable to create root znode %s: %s", PATH_SERVERS, err.Error())
			}
			log.Info.Printf("Created Zookeeper root znode %s", PATH_SERVERS)
		}

		// Verify /servers/serverName does not currently exist and then create it

		serverPath := fmt.Sprintf("%s/%s", PATH_SERVERS, serverName)
		if stat, err := client.CheckExists().ForPath(serverPath); err != nil {
			log.Warning.Printf("stat(%#v) err(%#v)", stat, err)
			return fmt.Errorf("Error attempting to check existence of %s: %s", serverPath, err.Error())
		} else if stat == nil {
			if _, err = client.Create().ForPathWithData(serverPath, []byte(hostname)); err != nil {
				return fmt.Errorf("Unable to create znode %s: %s", serverPath, err.Error())
			}
			log.Info.Printf("Created Zookeeper znode %s", serverPath)
		} else if !force {
			return fmt.Errorf("Server %s is already running.  Use -f argument to run anyway.", serverName)
		} else {
			if _, err = client.SetData().ForPathWithData(serverPath, []byte(APP_NAME)); err != nil {
				return fmt.Errorf("Unable to update znode %s: %s", serverPath, err.Error())
			}
			log.Info.Printf("Replaced Zookeeper znode %s", serverPath)
		}
	}
	return

}
