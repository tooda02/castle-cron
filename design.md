# castle-cron

## Overview
castle-cron is a  distributed time-based job scheduler similar to cron.  It supports a CLI for maintaining a list of jobs and runs them at the appropriate time on one of its servers (chosen randomly).  It is highly available and supports any number of servers.  Servers can enter or leave the cluster at any time.  The system can survive process, machine and data center failures and will continue to function as long as at least one server is running.

## Design
There is one executable that supports both the CLI and the server, depending on invocation arguments.  The system requires and uses Zookeeper, which it uses to store and manage its job list and to report on server availability.

### Znodes
castle-cron uses four root znodes, all under the namespace `/castle-cron`:

znode | Usage
----- | -----
/servers | Root znode of any number of emphereral nodes, one for each active server.  The presence of znode `/servers/servername` signifies that server *servername* is active.
/jobs | Root znode of any number of permanent nodes, one for each job.  Znode `/jobs/jobname ` contains data holding a serialized Job struct (see below).
/nextjob | A znode with no children that holds the serialize Job structure of the next scheduled job.
/joblock | A znode with no children used to synchronize updates to `/nextjob`.  For example, a server runs the job in `/nextjob` only after it successfully obtains the lock at the job's scheduled start time.

### Server Operation
When a server starts, it does the following:

1. Connects to Zookeeper and creates a `/servers/servername` znode.
2. Starts a goroutine that reads the children of `/servers` and reports on all running servers.  In addition, it sets a watch and reports when a server enters or leaves the cluster.
3. Retrieves the Job stored in `/nextjobs` and sets a watch.
4. If the job's scheduled time is in the future, it sets a timer expiring at that time.  It then wait for either timer expiration or a watch event on the job in `/nextjobs`, returning to step 3 when either event occurs.
5. If the job is ready to run, but the server does not hold the lock, it requests the lock.
6. When the lock is granted, retrieves `/nextjob` again, as it may have changed during the wait.  If it is no longer ready to run, releases the lock and returns to  step 3.
7. If the job is ready to run and the server holds the lock, it starts the job in a goroutine, so it executes asynchronously.
8. Determines the next job to schedule and updates `/nextjob`
9. Releases the lock and return to step 3.

When there are multiple servers, they will all retrieve the same `/nextjob` and request the lock at the same time.  However, only one will successfully obtain the lock.  That server starts the job, updates `/nextjob`, and releases the lock.  The other servers will fetch the new `/nextjob` and set a fresh timer.  Meanwhile, the job executes in a goroutine on the original server.

### CLI Operation
The CLI allows a user to add, update, or delete a job.  Any of these operations could affect the schedule, so the CLI retrieves the current `/nextjob` and does the following:

* If there is no /nextjob (data at the znone is empty), this must be a new system, so the newly added job becomes `/nextjob`.
* If this is a delete operation to the current `/nextjob`, replace it with the next job to schedule.
* If this is an update operation to the current `/nextjob`, or the updated job has an earlier start time than the current `/nextjob`, replace `/nextjob` with the newly added job.

All servers have an active watch on `/nextjob`, so any change to it causes them to wake up and reset their schedule.

### The Job Struct
**Job** is the struct that castle-cron uses to maintain job information.  All jobs must have a unique name. castle-cron stores job information in znode /jobs/jobname and in addition stores a copy of the job next on the schedule in znode /nextjob.  The Job struct contains the following:

Field | Type | Significance
----- | ---- | ------------ 
Name | string | Unique name of this job.
Cmd  |  string | Command to run
Args | []string | Command arguments
HasError | bool | Job has an error - do not run.  This flag is set for a schedule error or for a deleted job.
NextRuntime | time.Time | Time of next execution.  This is calculated when the job is created and recalculated when it is updated or run.
Schedule | string | A cron-type schedule string consisting of 5 - 7 blank-separated values (seconds, minutes, hours, day of month, month, weekday, and year).  See [https://github.com/gorhill/cronexpr](https://github.com/gorhill/cronexpr) for documentation.
