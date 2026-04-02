//go:build darwin

package goprovider

var mainThreadTasks chan func()

func RunMainThreadLoop(run func() int) int {
	if mainThreadTasks != nil {
		return run()
	}

	mainThreadTasks = make(chan func())
	resultCh := make(chan int, 1)

	go func() {
		resultCh <- run()
	}()

	for {
		select {
		case task := <-mainThreadTasks:
			task()
		case code := <-resultCh:
			mainThreadTasks = nil
			return code
		}
	}
}

func postMainThreadTask(task func()) bool {
	if mainThreadTasks == nil {
		return false
	}
	mainThreadTasks <- task
	return true
}
