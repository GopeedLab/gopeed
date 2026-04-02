//go:build !darwin

package goprovider

func RunMainThreadLoop(run func() int) int {
	return run()
}

func postMainThreadTask(func()) bool {
	return false
}
