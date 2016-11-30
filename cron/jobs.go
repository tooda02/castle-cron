package cron

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os/exec"
	"sort"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/samuel/go-zookeeper/zk"
	log "github.com/tooda02/castle-cron/logging"
)

type Job struct {
	Name        string    // Name of this job
	Cmd         string    // Command to run
	Args        []string  // Command arguments
	HasError    bool      // Job has an error - do not run
	NextRuntime time.Time // Time of next execution
	Schedule    string    // cron-type schedule string - see below
	/*
		Field name     Mandatory?   Allowed values    Allowed special characters
		----------     ----------   --------------    --------------------------
		Seconds        No           0-59              * / , -
		Minutes        Yes          0-59              * / , -
		Hours          Yes          0-23              * / , -
		Day of month   Yes          1-31              * / , - L W
		Month          Yes          1-12 or JAN-DEC   * / , -
		Day of week    Yes          0-6 or SUN-SAT    * / , - L #
		Year           No           1970–2099         * / , -

		From https://github.com/gorhill/cronexpr
	*/
}

// Deserialize a byte array into a Job struct
func Deserialize(b []byte) (job *Job, e error) {
	job = &Job{}
	if b == nil || len(b) == 0 {
		// Ensure null job isn't scheduled
		job.NextRuntime = time.Now().Add(time.Duration(24) * time.Hour)
	} else {
		buffer := bytes.NewBuffer(b)
		decoder := gob.NewDecoder(buffer)
		if err := decoder.Decode(&job); err != nil {
			e = fmt.Errorf("Unable to decode job: %s", err.Error())
		}
	}
	return
}

// Get a job or a list of jobs from Zookeeper
func ListJobs(name string) (jobs []*Job, e error) {
	jobs = []*Job{}
	jobnames := []string{name}
	if name == "" {
		// Empty name means list all jobs
		if jobnames, _, e = zkConn.Children(PATH_JOBS); e != nil {
			return nil, fmt.Errorf("Unable to retrieve job list: %s", e.Error())
		}
		sort.Strings(jobnames)
	}
	for _, jobname := range jobnames {
		if b, _, err := zkConn.Get(fmt.Sprintf("%s/%s", PATH_JOBS, jobname)); err != nil {
			return nil, fmt.Errorf("Can't fetch job %s: %s", jobname, err.Error())
		} else if job, err := Deserialize(b); err != nil {
			return nil, err
		} else {
			jobs = append(jobs, job)
		}
	}
	return
}

// Run a job
func (job *Job) Run() {
	log.Info.Printf("Running job %s", job.Name)
	start := time.Now()
	cmd := exec.Command(job.Cmd, job.Args...)
	if err := cmd.Run(); err != nil {
		log.Error.Printf("Job %s failed after %v seconds: %s", job.Name, time.Now().Sub(start).Seconds(), err.Error())
	} else {
		log.Info.Printf("Job %s complete after %v seconds", job.Name, time.Now().Sub(start).Seconds())
	}
}

// Calculate the next runtime of a job using its cron-style schedule
func (job *Job) SetNextRuntime() (changed bool, e error) {
	currNextRuntime := job.NextRuntime
	if cronSchedule, err := cronexpr.Parse(job.Schedule); err != nil {
		return false, fmt.Errorf("Invalid schedule string \"%s\" for job %s: %s", job.Schedule, job.Name, err.Error())
	} else {
		job.NextRuntime = cronSchedule.Next(time.Now())
		log.Info.Printf("Job %s next run time %s", job.Name, job.NextRuntime.Format("2006-01-02 15:04:05.99999999"))
	}
	return currNextRuntime != job.NextRuntime, nil
}

// Serialize a job into a byte array
func (job *Job) Serialize() (b []byte, e error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if e = encoder.Encode(job); e != nil {
		e = fmt.Errorf("Unable to serialize job %s: %s", job.Name, e.Error())
	} else {
		b = buffer.Bytes()
	}
	return
}

// Update job in znode /jobs/<jobname>
func (job *Job) UpdateZk() (e error) {
	if !hasLock {
		if e = getJobsLock(); e != nil {
			return
		}
		defer releaseJobsLock()
	}
	if b, err := job.Serialize(); err != nil {
		e = err
	} else if _, err = zkConn.Set(fmt.Sprintf("%s/%s", PATH_JOBS, job.Name), b, -1); err != nil {
		e = fmt.Errorf("Unable to update job %s: %s", job.Name, err.Error())
	}
	return
}

// Write new job to znode /jobs/<jobname>
func (job *Job) WriteToZk() (e error) {
	if !hasLock {
		if e = getJobsLock(); e != nil {
			return
		}
		defer releaseJobsLock()
	}
	if b, err := job.Serialize(); err != nil {
		e = err
	} else if _, err = zkConn.Create(fmt.Sprintf("%s/%s", PATH_JOBS, job.Name), b, 0x0, zk.WorldACL(zk.PermAll)); err != nil {
		e = fmt.Errorf("Unable to create job %s: %s", job.Name, err.Error())
	}
	return
}

// Delete job from znode /jobs/<jobname>
func (job *Job) DeleteFromZk() (e error) {
	if !hasLock {
		if e = getJobsLock(); e != nil {
			return
		}
		defer releaseJobsLock()
	}
	if e = zkConn.Delete(fmt.Sprintf("%s/%s", PATH_JOBS, job.Name), -1); e != nil {
		e = fmt.Errorf("Unable to delete job %s: %s", job.Name, e.Error())
	}
	return
}
