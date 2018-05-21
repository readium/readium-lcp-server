/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package cron

import "time"

const (
	// Max number of jobs, hack it if you need.
	MAXJOBNUM = 10000
)

var (
	// Time location, default set by the time.Local (*time.Location)
	loc = time.Local
	// Map for the function task store
	funcs = map[string]interface{}{}
	// Map for function and  params of function
	fparams = map[string]([]interface{}){}

	// The following methods are shortcuts for not having to
	// create a Schduler instance

	defaultScheduler = NewScheduler()
	jobs             = defaultScheduler.jobs
)

type (
	Job struct {

		// pause interval * unit bettween runs
		interval uint64

		// the job jobFunc to run, func[jobFunc]
		jobFunc string
		// time units, ,e.g. 'minutes', 'hours'...
		unit string
		// optional time at which this job runs
		atTime string

		// datetime of last run
		lastRun time.Time
		// datetime of next run
		nextRun time.Time
		// cache the period between last an next run
		period time.Duration

		// Specific day of the week to start on
		startDay time.Weekday
	}
	// Class Scheduler, the only data member is the list of jobs.
	Scheduler struct {
		// Array store jobs
		jobs [MAXJOBNUM]*Job
		// Size of jobs which jobs holding.
		size int
	}
)
