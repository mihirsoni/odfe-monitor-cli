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
