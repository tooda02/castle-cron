# castle-cron

## Overview
castle-cron is a  distributed time-based job scheduler similar to cron.  It supports a CLI for maintaining a list of jobs and runs them at the appropriate time on one of its servers (chosen randomly).  It is highly available and supports any number of servers.  Servers can enter or leave the cluster at any time.  The system can survive process, machine and data center failures and will continue to function as long as at least one server is running.

## Usage
There is one executable that supports both the CLI and the server, depending on invocation arguments.  The system requires and uses Zookeeper, which it uses to store and manage its job list, and to report on server availability.

#### Server

    castle-cron -s [-zk Zookeeper server(s)] [-zt timeout] [-n name] [-f] [-v]

Invokes castle-cron as a server daemon logging to the console.  It connects to the designated Zookeeper server and waits for the scheduled start time of the next job (or for a schedule change).  Once the scheduled time arrives, it competes with other servers for the right to run the job, and if successful, runs the job.  It then returns to the wait.

You can start any number of castle-cron servers.  Each server's console log reports when other servers arrive into or depart from the cluster.  Scheduled jobs are assign to a server at random from the servers available at the time the job runs.

Argument | Default | Significance
-------- | ------- | ------------
-s | | Required.  Indicates castle-cron should run as a server
-zk | ZOOKEEPER_SERVERS | Optional; if omitted, the value must be supplied in the ZOOKEEPER_SERVERS environment variable.  Specifies a comma-separated list of servers in the form *hostname:port[,hostname:port...]*
-zt | 10 | Zookeeper timeout.  Specifies the number of seconds of non-contact before a session times out.
-n | *hostname* | Server name.  Can include %h (hostname) and %p (pid).
-f | | Force start.  Starts the server even if its name duplicates another server.
-v | | Verbose.  Include TRACE logging.

#### CLI
    castle-cron [-zk Zookeeper server(s)] [-zt timeout] [-v] add jobname schedule cmd args
    castle-cron [-zk Zookeeper server(s)] [-zt timeout] [-v] upd jobname schedule cmd args
    castle-cron [-zk Zookeeper server(s)] [-zt timeout] [-v] del jobname
    castle-cron [-zk Zookeeper server(s)] [-zt timeout] [-v] list [jobname]
    castle-cron [-zk Zookeeper server(s)] [-zt timeout] [-v] help add|del|upd|list|sched

Maintains the job list.  All jobs must have a unique name, but are otherwise specified in a similar format to jobs in crontab.  CLI commands available are:

* **add** Adds a new job.  The schedule is a has a similar format to cron; see below.
* **upd** Updates an existing job.  All arguments must be provided.
* **del** Deletes a job.
* **list** Lists all or a subset of jobs. The optional *jobname* argument can asterisk as a wildcard character (matching one or more characters).  If *jobname* is omitted, list shows all jobs.
* **help** Shows help for CLI commands.  **help sched** describes the format of the schedule argument of add and upd

        Job schedule; must be a quoted string containing 5 - 7 blank-separated values.
        Field name    Mandatory?      Allowed values  Allowed special characters
        ----------    ----------      --------------  --------------------------
        Seconds       No              0-59            * / , -
        Minutes       Yes             0-59            * / , -
        Hours         Yes             0-23            * / , -
        Day of month  Yes             1-31            * / , - L W
        Month         Yes             1-12 or JAN-DEC * / , -
        Day of week   Yes             0-6 or SUN-SAT  * / , - L #
        Year          No              1970–2099       * / , -
