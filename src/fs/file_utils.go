package main

import (
	"errors"
	"fmt"
	"os/exec"
)

func Sed(offset, lines int) ([]byte, error) {
	out, err := exec.Command("sed", "-n", fmt.Sprintf("%d,%dp", offset, offset+lines), LOGFILE).Output()
	if err != nil {
		logError(err.Error())
		return nil, errors.New("cannot read from logs")
	}
	return out, nil
}

func Tail(lines int) ([]byte, error) {
	out, err := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), LOGFILE).Output()
	if err != nil {
		logError(err.Error())
		return nil, errors.New("cannot read from logs")
	}
	return out, nil
}
