// +build darwin linux

package cli

import (
	"errors"
	"io"
	"os/exec"
	"time"
)

//CommandName represent the os shell
const CommandName string = "sh"

//Shell type manages the commads execution on a system
type Shell struct {
	outputReader *io.ReadCloser
	busy         bool
	cmd          *exec.Cmd
}

//New creates a Shell object and returns a pointer to it
func New() *Shell {
	var sh Shell
	return &sh
}

//MustExec waits until the previous commad be executed then executes the passed command
func (sh *Shell) MustExec(c string) error {
	var err error
	for (*sh).busy {
		time.Sleep(10 * time.Millisecond)
	}
	(*sh).cmd = exec.Command(CommandName, c)
	(*(*sh).outputReader), err = (*sh).cmd.StdoutPipe()
	if err != nil {
		return err
	}
	(*sh).busy = true
	err = (*sh).cmd.Start()
	if err != nil {
		return err
	}
	go func() {
		(*sh).cmd.Wait()
		(*sh).busy = false
	}()
	return err
}

//Exec executes the passed command if it is freed by the time otherwise return apropirate error.
func (sh *Shell) Exec(c string) error {
	var err error
	if (*sh).busy {
		return errors.New("Shell is not freed yet")
	}
	(*sh).cmd = exec.Command(CommandName, c)
	(*(*sh).outputReader), err = (*sh).cmd.StdoutPipe()
	if err != nil {
		return err
	}
	(*sh).busy = true
	err = (*sh).cmd.Start()
	if err != nil {
		return err
	}
	go func() {
		(*sh).cmd.Wait()
		(*sh).busy = false
	}()
	return err
}

//Output returns a ReadCloser which reads the last command's output
func (sh *Shell) Output() *io.ReadCloser {
	return (*sh).outputReader
}

//IsBusy declares if shell is busy of the last command (returns true) or is freed by now (returns false)
func (sh *Shell) IsBusy() bool {
	return (*sh).busy
}
