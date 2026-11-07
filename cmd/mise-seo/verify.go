package main

import (
	"fmt"
	"os"
	"os/exec"
)

// verify is mise-seo verify (seo:verify): what the repo's GitHub workflow runs once it has checked out and installed
// mise: the repo's own setup and test tasks when it has them (an app's tests audit its local build, as the template's
// do), then check. Run here, it does what GitHub does.
func (r repo) verify(args []string) {
	parse(flags("verify"), args, 0)
	if _, err := exec.LookPath("mise"); err == nil {
		r.runTask("setup")
		r.runTask("test")
	} else {
		fmt.Println("· no mise here: the repo's own tasks are left to it (go test ./...)")
	}
	r.check(nil)
}

// runTask runs the repo's own task when it has one.
func (r repo) runTask(task string) {
	has := exec.Command("mise", "tasks", "info", task)
	has.Dir = r.dir
	if has.Run() != nil {
		return
	}
	fmt.Println("$ mise run " + task)
	run := exec.Command("mise", "run", task)
	run.Dir, run.Stdin, run.Stdout, run.Stderr = r.dir, os.Stdin, os.Stdout, os.Stderr
	if err := run.Run(); err != nil {
		fail(fmt.Errorf("mise run %s: %w", task, err))
	}
}
