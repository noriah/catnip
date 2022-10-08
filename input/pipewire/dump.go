package pipewire

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

type pwObjects []pwObject

func pwDump(ctx context.Context) (pwObjects, error) {
	cmd := exec.CommandContext(ctx, "pw-dump")
	cmd.Stderr = os.Stderr

	dumpOutput, err := cmd.Output()
	if err != nil {
		var execErr *exec.ExitError
		if errors.As(err, &execErr) {
			return nil, errors.Wrapf(err, "failed to run pw-dump: %s", execErr.Stderr)
		}
		return nil, errors.Wrap(err, "failed to run pw-dump")
	}

	var dump pwObjects
	if err := json.Unmarshal(dumpOutput, &dump); err != nil {
		return nil, errors.Wrap(err, "failed to parse pw-dump output")
	}

	return dump, nil
}

// Filter filters for the devices that satisfies f.
func (d pwObjects) Filter(fns ...func(pwObject) bool) pwObjects {
	filtered := make(pwObjects, 0, len(d))
loop:
	for _, device := range d {
		for _, f := range fns {
			if !f(device) {
				continue loop
			}
		}
		filtered = append(filtered, device)
	}
	return filtered
}

// Find returns the first object that satisfies f.
func (d pwObjects) Find(f func(pwObject) bool) *pwObject {
	for i, device := range d {
		if f(device) {
			return &d[i]
		}
	}
	return nil
}

// ResolvePorts returns all PipeWire port objects that belong to the given
// object.
func (d pwObjects) ResolvePorts(object *pwObject, dir pwPortDirection) pwObjects {
	return d.Filter(
		func(o pwObject) bool { return o.Type == pwInterfacePort },
		func(o pwObject) bool {
			return o.Info.Props.NodeID == object.ID && o.Info.Props.PortDirection == dir
		},
	)
}

type pwObjectID int64

type pwObjectType string

const (
	pwInterfaceDevice pwObjectType = "PipeWire:Interface:Device"
	pwInterfaceNode   pwObjectType = "PipeWire:Interface:Node"
	pwInterfacePort   pwObjectType = "PipeWire:Interface:Port"
	pwInterfaceLink   pwObjectType = "PipeWire:Interface:Link"
)

type pwObject struct {
	ID   pwObjectID   `json:"id"`
	Type pwObjectType `json:"type"`
	Info struct {
		Props pwInfoProps `json:"props"`
	} `json:"info"`
}

type pwInfoProps struct {
	pwDeviceProps
	pwNodeProps
	pwPortProps
	MediaClass string `json:"media.class"`

	JSON json.RawMessage `json:"-"`
}

func (p *pwInfoProps) UnmarshalJSON(data []byte) error {
	type Alias pwInfoProps
	if err := json.Unmarshal(data, (*Alias)(p)); err != nil {
		return err
	}
	p.JSON = append([]byte(nil), data...)
	return nil
}

type pwDeviceProps struct {
	DeviceName string `json:"device.name"`
}

// pwNodeProps is for Audio/Sink only.
type pwNodeProps struct {
	NodeName        string `json:"node.name"`
	NodeNick        string `json:"node.nick"`
	NodeDescription string `json:"node.description"`
}

// Constants for MediaClass.
const (
	pwAudioDevice       string = "Audio/Device"
	pwAudioSink         string = "Audio/Sink"
	pwStreamOutputAudio string = "Stream/Output/Audio"
)

type pwPortDirection string

const (
	pwPortIn  = "in"
	pwPortOut = "out"
)

type pwPortProps struct {
	PortID        pwObjectID      `json:"port.id"`
	PortName      string          `json:"port.name"`
	PortAlias     string          `json:"port.alias"`
	PortDirection pwPortDirection `json:"port.direction"`
	NodeID        pwObjectID      `json:"node.id"`
	ObjectPath    string          `json:"object.path"`
}
