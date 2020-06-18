/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package utils

// Limiter struct for handling maximum concurent request
type Limiter struct {
	limit              int
	concurrentRoutines chan struct{}
}

// NewLimiter creates new object and initialized with value
func NewLimiter(limit int) *Limiter {
	if limit <= 0 {
		limit = 10
	}
	// Initialize object
	limiter := &Limiter{
		limit:              limit,
		concurrentRoutines: make(chan struct{}, limit),
	}
	//Filling up buffered channel with empty struct, otherwise it
	//will get blocked immediately
	for i := 0; i < limiter.limit; i++ {
		limiter.concurrentRoutines <- struct{}{}
	}
	return limiter
}

// Execute Takes a function and execute a job in
// go routine with handling provided concurrency
func (limiter *Limiter) Execute(job func()) {
	//Receive from a channel and if there is a value so that we can start new job
	<-limiter.concurrentRoutines
	go func() {
		//once job is finished fill up buffered channel
		defer func() {
			limiter.concurrentRoutines <- struct{}{}
		}()
		job()
	}()
}

//Wait Allows a caller to wait for all the jobs to finish before exiting
func (limiter *Limiter) Wait() {
	//This will be blocked until all the jobs are finished doing their job
	for i := 0; i < limiter.limit; i++ {
		<-limiter.concurrentRoutines
	}
}
