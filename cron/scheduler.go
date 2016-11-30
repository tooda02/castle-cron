package cron

import (
	"fmt"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	log "github.com/tooda02/castle-cron/logging"
)

const (
	PATH_JOBS     = "/jobs"
	PATH_NEXT_JOB = "/nextjob"
)

var (
	lock    = zk.NewLock(zkConn, PATH_JOBS, zk.WorldACL(zk.PermAll))
	hasLock bool // true => We have acquired the lock
)

/*
Schedule and run the next job.  We do the following:
1. Retrieve the next job scheduled from znode /nextjob and set a watch.
2. If the job's execution time is in the future, set a timer and wait
   for either timer expiration or the watch event, and return to step 1.
3. If the job is ready to run, request a lock on /jobs.
4. When the lock is granted, check if the job in /nextjob is still ready to run.
   If not, release the lock and return to step 2.
5. Run the job.
6. Determine the next job to schedule and update /nextjob
7. Release the lock and return to step 1.
*/
func runNextJob() error {
	createIfNecessary(PATH_JOBS)
	createIfNecessary(PATH_NEXT_JOB)
	for isRunning {

		// 1. Retrieve the next scheduled job.  This is always in /nextjob

		jobData, _, watch, err := zkConn.GetW(PATH_NEXT_JOB)
		if err != nil {
			releaseJobsLock()
			return fmt.Errorf("Unable to retrieve next job: %s", err.Error())
		}
		now := time.Now()
		job, err := Deserialize(jobData)
		if err != nil {
			releaseJobsLock()
			return fmt.Errorf("Unable to decode next job: %s", err.Error())
		}

		// 2. If the next job is in the future, wait until its scheduled
		//    execution time or an update to the schedule for the next job.

		if job.NextRuntime.After(now) {
			if err = releaseJobsLock(); err != nil {
				return err
			}
			select {
			case evt := <-watch:
				if evt.Err != nil {
					return fmt.Errorf("Error from %s update event: %s", PATH_NEXT_JOB, evt.Err.Error())
				}

			case <-time.After(job.NextRuntime.Sub(now)):
			}
			continue
		}

		// 3. If the job is ready to run and we don't have the lock, request it
		// 4. Once the lock is granted, continue to request the next job again.

		if !hasLock {
			if err := getJobsLock(); err != nil {
				return err
			}
			continue
		}

		// 5. Run the job.  We do this asynchronously so that we can release the lock
		//    while the job continues to run.  Note that this means there's no recovery
		//    if the job fails or the server crashes while it's running.

		go job.Run()

		// 6. Determine runtime of the next job in the schedule and update /jobsnext

		if err := updateSchedule(job); err != nil {
			return err
		}

	}
	return nil
}

// Reschedule current job and set the next scheduled job in /jobsnext
// This function releases the /jobs lock; the caller is responsible for obtaining it.
func updateSchedule(job *Job) error {
	defer releaseJobsLock()

	// Update the next run time of the job we just ran

	if changed, err := job.SetNextRuntime(); err != nil {
		log.Error.Printf("Can't reschedule job %s: %s", job.Name, err.Error())
		job.HasError = true
	} else if !changed {
		log.Error.Printf("Attempt to reschedule job %s failed as no new run time available", job.Name)
		job.HasError = true
	} else if err = job.UpdateZk(); err != nil {
		return err
	}

	// Determine runtime of the next job in the schedule and update /jobsnext

	if jobs, _, err := zkConn.Children(PATH_JOBS); err != nil {
		return fmt.Errorf("Unable to get list of jobs to calculate schedule: %s", err.Error())
	} else {
		nextRuntime := job.NextRuntime
		for _, jobName := range jobs {
			if jobData, _, err := zkConn.Get(fmt.Sprintf("%s/%s", PATH_JOBS, jobName)); err != nil {
				return err
			} else if job2, err := Deserialize(jobData); err != nil {
				return err
			} else if job2.NextRuntime.Before(nextRuntime) && !job.HasError {
				job = job2
			}
		}
		if jobData, err := job.Serialize(); err != nil {
			return err
		} else if _, err = zkConn.Set(PATH_NEXT_JOB, jobData, -1); err != nil {
			return fmt.Errorf("Unable to update schedule to run next job %s: %s", job.Name, err.Error())
		}
	}

	return releaseJobsLock() // Explicit unlock to ensure logging of any error
}

// Grab the lock if we don't already have it
func getJobsLock() error {
	if !hasLock {
		log.Trace.Printf("Requesting %s lock", PATH_NEXT_JOB)
		if err := lock.Lock(); err != nil {
			return fmt.Errorf("Unable to get %s lock: %s", PATH_NEXT_JOB, err.Error())
		}
		log.Trace.Printf("Taking %s lock", PATH_NEXT_JOB)
		hasLock = true
	}
	return nil
}

// Release the lock if we have it
func releaseJobsLock() error {
	if hasLock {
		log.Trace.Printf("Releasing %s lock", PATH_NEXT_JOB)
		if err := lock.Unlock(); err != nil {
			return fmt.Errorf("Unable to release %s lock: %s", PATH_NEXT_JOB, err.Error())
		}
		hasLock = false
	}
	return nil
}
