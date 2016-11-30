package cron

import (
	"fmt"
	"time"

	"github.com/samuel/go-zookeeper/zk"
	log "github.com/tooda02/castle-cron/logging"
)

var (
	lock *zk.Lock // Lock for /jobs
	hasLock bool  // true => We have acquired the lock
)

/*
Schedule and run jobs.  We do the following:
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
func Run(name string, force bool) (e error) {
	if e = setServerName(name, force); e != nil {
		return fmt.Errorf("Unable to set server name: %s", e.Error())
	}
	reportServers()
	
	isRunning = true
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
			log.Trace.Printf("Sleeping until job %s schedule start of %s", job.Name, job.FmtNextRuntime())
			if err = releaseJobsLock(); err != nil {
				return err
			}
			select {
			case evt := <-watch:
				if evt.Err != nil {
					return fmt.Errorf("Error from %s update event: %s", PATH_NEXT_JOB, evt.Err.Error())
				}
				log.Trace.Printf("Got notification of nextjob update event - checking schedule")

			case <-time.After(job.NextRuntime.Sub(now)):
			log.Trace.Printf("Wait time expired - checking schedule")
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

// Check whether a job just added, updated, or deleted affects nextjob
// The caller must take the lock before calling this function
func checkForNextjobUpdate(job *Job) (e error) {
	newScheduleNeeded := false // Set when user deletes currently scheduled job
	if b, _, err := zkConn.Get(PATH_NEXT_JOB); err != nil {
		return fmt.Errorf("Unable to check schedule after job update: %s", err.Error())
	} else if nextjob, err := Deserialize(b); err != nil {
		return err
	} else if nextjob.Name == NULL_JOBNAME {		
		// Schedule is currently empty - add the job we just created
		
		if job.HasError {
			// Uh-oh - nothing in the schedule and we just deleted a job
			// This shouldn't ever happen; log an error and treat as first-time schedule
			log.Error.Printf("Job delete succeeded, but schedule is currently empty")
			newScheduleNeeded = true
		} else if err := job.UpdateZkNextjob(); err != nil {
			return err
		} else {
			log.Trace.Printf("Scheduled first job %s to start at %s", job.Name, job.FmtNextRuntime())
		}
	} else if nextjob.Name != job.Name {
		log.Trace.Printf("Next scheduled job %s unchanged by update; will run at %s", nextjob.Name, nextjob.FmtNextRuntime())
	} else if job.HasError {
		log.Trace.Printf("Next scheduled job %s deleted by update")
		newScheduleNeeded = true
	} else if err := job.UpdateZkNextjob(); err != nil {
		return err
	} else {
		log.Trace.Printf("Updated currently scheduled job %s to start at %s", job.Name, job.FmtNextRuntime())
	}
	
	// If user deleted the currently scheduled job, refresh it
	
	if newScheduleNeeded {
		e = setNextjob()
	}
	return
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
	} else {
		log.Info.Printf("Job %s next run time %s", job.Name, job.FmtNextRuntime())
	}

	if err := setNextjob(); err != nil {
		return err
	}

	return releaseJobsLock() // Explicit unlock to ensure logging of any error
}

// Scan all jobs and save the next to run in /nextjobs.  
// The caller must acquire the lock prior to calling this function
func setNextjob() error {
	if jobs, _, err := zkConn.Children(PATH_JOBS); err != nil {
		return fmt.Errorf("Unable to get list of jobs to calculate schedule: %s", err.Error())
	} else {
		var job *Job
		for _, jobName := range jobs {
			if jobData, _, err := zkConn.Get(fmt.Sprintf("%s/%s", PATH_JOBS, jobName)); err != nil {
				return err
			} else if job2, err := Deserialize(jobData); err != nil {
				return err
			} else if job == nil || (job.NextRuntime.After(job2.NextRuntime) && !job2.HasError) {
				job = job2
			}
		}
		if job == nil {
			log.Warning.Printf("There are no jobs remaining to schedule")
		} else if jobData, err := job.Serialize(); err != nil {
			return err
		} else if _, err = zkConn.Set(PATH_NEXT_JOB, jobData, -1); err != nil {
			return fmt.Errorf("Unable to update schedule to run next job %s: %s", job.Name, err.Error())
		}
	}
	return nil
}

// Grab the lock if we don't already have it
func getJobsLock() error {
	if !hasLock {
		log.Trace.Printf("Requesting %s lock", PATH_JOBLOCK)
		if err := lock.Lock(); err != nil {
			return fmt.Errorf("Unable to get %s lock: %s", PATH_JOBLOCK, err.Error())
		}
		log.Trace.Printf("Taking %s lock", PATH_JOBLOCK)
		hasLock = true
	}
	return nil
}

// Release the lock if we have it
func releaseJobsLock() error {
	if hasLock {
		log.Trace.Printf("Releasing %s lock", PATH_JOBLOCK)
		if err := lock.Unlock(); err != nil {
			return fmt.Errorf("Unable to release %s lock: %s", PATH_JOBLOCK, err.Error())
		}
		hasLock = false
	}
	return nil
}
