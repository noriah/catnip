package pipewire

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/noriah/catnip/input"
	"github.com/noriah/catnip/input/common/execread"
	"github.com/pkg/errors"
)

func init() {
	input.RegisterBackend("pipewire", Backend{})
}

type Backend struct{}

func (p Backend) Init() error {
	return nil
}

func (p Backend) Close() error {
	return nil
}

func (p Backend) Devices() ([]input.Device, error) {
	pwObjs, err := pwDump(context.Background())
	if err != nil {
		return nil, err
	}

	pwSinks := pwObjs.Filter(func(o pwObject) bool {
		return o.Type == pwInterfaceNode &&
			o.Info.Props.MediaClass == pwAudioSink ||
			o.Info.Props.MediaClass == pwStreamOutputAudio
	})

	devices := make([]input.Device, len(pwSinks))
	for i, device := range pwSinks {
		devices[i] = AudioDevice{device.Info.Props.NodeName}
	}

	return devices, nil
}

func (p Backend) DefaultDevice() (input.Device, error) {
	return AudioDevice{"auto"}, nil
}

func (p Backend) Start(cfg input.SessionConfig) (input.Session, error) {
	return NewSession(cfg)
}

type AudioDevice struct {
	name string
}

func (d AudioDevice) String() string {
	return d.name
}

type catnipProps struct {
	ApplicationName string `json:"application.name"`
	CatnipID        string `json:"catnip.id"`
}

// Session is a PipeWire session.
type Session struct {
	session    execread.Session
	props      catnipProps
	targetName string
}

// NewSession creates a new PipeWire session.
func NewSession(cfg input.SessionConfig) (*Session, error) {
	currentProps := catnipProps{
		ApplicationName: "catnip",
		CatnipID:        generateID(),
	}

	propsJSON, err := json.Marshal(currentProps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal props")
	}

	dv, ok := cfg.Device.(AudioDevice)
	if !ok {
		return nil, fmt.Errorf("invalid device type %T", cfg.Device)
	}

	target := "0"
	if dv.name == "auto" {
		target = dv.name
	}

	args := []string{
		"pw-cat",
		"--record",
		"--format", "f32",
		"--rate", fmt.Sprint(cfg.SampleRate),
		"--latency", fmt.Sprint(cfg.SampleSize),
		"--channels", fmt.Sprint(cfg.FrameSize),
		"--target", target, // see .relink comment below
		"--quality", "0",
		"--media-category", "Capture",
		"--media-role", "DSP",
		"--properties", string(propsJSON),
	}

	// pw-cat 1.4.0 introduces explicit stdout support, needs --raw arg
	// see https://gitlab.freedesktop.org/pipewire/pipewire/-/issues/4629#top
	useRawArg, err := checkNeedRawArg()
	if err != nil {
		return nil, errors.Wrap(err, "failed to check need of pipewire '--raw' arg")
	}

	if useRawArg {
		args = append(args, "--raw")
	}

	// output to STDOUT
	args = append(args, "-")

	return &Session{
		session:    *execread.NewSession(args, true, cfg),
		props:      currentProps,
		targetName: dv.name,
	}, nil
}

// Start starts the session. It implements input.Session.
func (s *Session) Start(ctx context.Context, dst [][]input.Sample, kickChan chan bool, mu *sync.Mutex) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	setErr := func(err error) {
		select {
		case errCh <- err:
		default:
		}
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		setErr(s.session.Start(ctx, dst, kickChan, mu))
	}()

	// No relinking needed if we're not connecting to a specific device.
	if s.targetName != "auto" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			setErr(s.startRelinker(ctx))
		}()
	}

	return <-errCh
}

// We do a bit of tomfoolery here. Wireplumber actually is pretty incompetent at
// handling target.device, so our --target flag is pretty much useless. We have
// to do the node links ourselves.
//
// Relevant issues:
//
//   - https://gitlab.freedesktop.org/pipewire/pipewire/-/issues/2731
//   - https://gitlab.freedesktop.org/pipewire/wireplumber/-/issues/358
func (s *Session) startRelinker(ctx context.Context) error {
	var catnipPorts map[string]pwObjectID
	var err error
	// Employ this awful hack to get the needed port IDs for our session. We
	// won't rely on the pwLinkMonitor below, since it may appear out of order.
	for i := 0; i < 20; i++ {
		catnipPorts, err = findCatnipPorts(ctx, s.props)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return errors.Wrap(err, "failed to find catnip's input ports")
	}

	linkEvents := make(chan pwLinkEvent)
	linkError := make(chan error, 1)
	go func() { linkError <- pwLinkMonitor(ctx, pwLinkOutputPorts, linkEvents) }()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-linkError:
			return err
		case event := <-linkEvents:
			switch event := event.(type) {
			case pwLinkAdd:
				if event.DeviceName != s.targetName {
					break
				}

				catnipPort, ok := matchPort(event, catnipPorts)
				if !ok {
					log.Printf(
						"device port %s (%d) cannot be matched to catnip (ports: %v)",
						event.PortName, event.PortID, catnipPorts)
					break
				}

				if err := pwLink(event.PortID, catnipPort.PortID); err != nil {
					log.Printf(
						"failed to link catnip port %s (%d) to device port %s (%d): %v",
						catnipPort.PortName, catnipPort.PortID,
						event.PortName, event.PortID, err)
				}
			}
		}
	}
}

// matchPort tries to match the given link event to the catnip input ports. It
// returns the catnip port to link to, and whether the event was matched.
func matchPort(event pwLinkAdd, ourPorts map[string]pwObjectID) (pwLinkObject, bool) {
	if len(ourPorts) == 1 {
		// We only have one port, so we're probably in mono mode.
		for name, id := range ourPorts {
			return pwLinkObject{PortID: id, PortName: name}, true
		}
	}

	// Try directly matching the port channel with the event's. This usually
	// works if the number of ports is the same.
	_, channel, ok := strings.Cut(event.PortName, "_")
	if ok {
		port := "input_" + channel
		portID, ok := ourPorts[port]
		if ok {
			return pwLinkObject{PortID: portID, PortName: port}, true
		}
	}

	// TODO: use more sophisticated matching here.
	return pwLinkObject{}, false
}

func findCatnipPorts(ctx context.Context, ourProps catnipProps) (map[string]pwObjectID, error) {
	objs, err := pwDump(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pw-dump")
	}

	// Find the catnip node.
	nodeObj := objs.Find(func(obj pwObject) bool {
		if obj.Type != pwInterfaceNode {
			return false
		}
		var props catnipProps
		err := json.Unmarshal(obj.Info.Props.JSON, &props)
		return err == nil && props == ourProps
	})
	if nodeObj == nil {
		return nil, errors.New("failed to find catnip node in PipeWire")
	}

	// Find all of catnip's ports. We want catnip's input ports.
	portObjs := objs.ResolvePorts(nodeObj, pwPortIn)
	if len(portObjs) == 0 {
		return nil, errors.New("failed to find any catnip port in PipeWire")
	}

	portMap := make(map[string]pwObjectID)
	for _, obj := range portObjs {
		portMap[obj.Info.Props.PortName] = obj.ID
	}

	return portMap, nil
}

var sessionCounter uint64

// generateID generates a unique ID for this session.
func generateID() string {
	return fmt.Sprintf(
		"%d@%s#%d",
		os.Getpid(),
		shortEpoch(),
		atomic.AddUint64(&sessionCounter, 1),
	)
}

// shortEpoch generates a small string that is unique to the current epoch.
func shortEpoch() string {
	now := time.Now().Unix()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(now))
	return base64.RawURLEncoding.EncodeToString(buf[:])
}

func checkNeedRawArg() (bool, error) {
	cmd := exec.Command("pw-cat", "--help")

	out, err := cmd.Output()

	if err != nil {
		return false, err
	}

	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		if strings.Contains(line, "--raw") {
			return true, nil
		}
	}

	return false, nil
}
