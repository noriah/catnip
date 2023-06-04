package pipewire

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func pwLink(outPortID, inPortID pwObjectID) error {
	cmd := exec.Command("pw-link", "-L", fmt.Sprint(outPortID), fmt.Sprint(inPortID))
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.Stderr != nil {
			return errors.Wrapf(err, "failed to run pw-link: %s", exitErr.Stderr)
		}
		return err
	}
	return nil
}

type pwLinkObject struct {
	DeviceName string
	PortID     pwObjectID
	PortName   string // usually like {input,output}_{FL,FR}
}

func pwLinkObjectParse(line string) (pwLinkObject, error) {
	var obj pwLinkObject

	idStr, portStr, ok := strings.Cut(line, " ")
	if !ok {
		return obj, fmt.Errorf("failed to parse pw-link object %q", line)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return obj, errors.Wrapf(err, "failed to parse pw-link object id %q", idStr)
	}

	name, port, ok := strings.Cut(portStr, ":")
	if !ok {
		return obj, fmt.Errorf("failed to parse pw-link port string %q", portStr)
	}

	obj = pwLinkObject{
		PortID:     pwObjectID(id),
		DeviceName: name,
		PortName:   port,
	}

	return obj, nil
}

type pwLinkType string

const (
	pwLinkInputPorts  pwLinkType = "i"
	pwLinkOutputPorts pwLinkType = "o"
)

type pwLinkEvent interface {
	pwLinkEvent()
}

type pwLinkAdd pwLinkObject
type pwLinkRemove pwLinkObject

func (pwLinkAdd) pwLinkEvent()    {}
func (pwLinkRemove) pwLinkEvent() {}

func pwLinkMonitor(ctx context.Context, typ pwLinkType, ch chan<- pwLinkEvent) error {
	cmd := exec.CommandContext(ctx, "pw-link", "-mI"+string(typ))
	cmd.Stderr = os.Stderr

	o, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "failed to get stdout pipe")
	}
	defer o.Close()

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "pw-link -m")
	}

	scanner := bufio.NewScanner(o)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		mark := line[0]

		line = strings.TrimSpace(line[1:])

		obj, err := pwLinkObjectParse(line)
		if err != nil {
			continue
		}

		var ev pwLinkEvent
		switch mark {
		case '=':
			fallthrough
		case '+':
			ev = pwLinkAdd(obj)
		case '-':
			ev = pwLinkRemove(obj)
		default:
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- ev:
		}
	}

	return errors.Wrap(cmd.Wait(), "pw-link exited")
}
