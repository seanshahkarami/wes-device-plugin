package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// The k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1 package provides all
// the required Kubernetes device plugin .proto files and dervied Go types.

type Device struct{}

type DevicePlugin struct {
	ticker *time.Ticker
	stop   chan interface{}
}

func NewDevicePlugin() *DevicePlugin {
	return &DevicePlugin{
		ticker: time.NewTicker(15 * time.Second),
		stop:   make(chan interface{}),
	}
}

func (m *DevicePlugin) ListenAndServe() error {
	os.Remove("device-server")

	ln, err := net.Listen("unix", "device-server")
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	pluginapi.RegisterDevicePluginServer(server, m)
	return server.Serve(ln)
}

func (m *DevicePlugin) Shutdown() {
	close(m.stop)
	m.ticker.Stop()
}

// ListAndWatch returns a stream of List of Devices
// Whenever a Device state change or a Device disappears, ListAndWatch
// returns the new list
//   rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}
func (m *DevicePlugin) ListAndWatch(_ *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	for {
		select {
		case <-m.stop:
			return nil
		// make this a timer which just scans periodically...
		case <-m.ticker.C:
			// TODO implement behind a watcher which manages changes
			log.Printf("updating device list\n")

			resp := &pluginapi.ListAndWatchResponse{
				Devices: []*pluginapi.Device{
					{
						ID:     "bme280-nxcore",
						Health: pluginapi.Healthy,
					},
				},
			}

			// send updated list
			if err := s.Send(resp); err != nil {
				return err
			}
		}
	}
}

// Allocate is called during container creation so that the Device
// Plugin can run device specific operations and instruct Kubelet
// of the steps to make the Device available in the container
// rpc Allocate(AllocateRequest) returns (AllocateResponse) {}
func (m *DevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	return &pluginapi.AllocateResponse{}, nil
}

// GetDevicePluginOptions returns options to be communicated with Device Manager.
// rpc GetDevicePluginOptions(Empty) returns (DevicePluginOptions) {}
func (m *DevicePlugin) GetDevicePluginOptions(ctx context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	// we will not provide a useful implementation for either of these functions.
	// see the documentation about these for more info.
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

// PreStartContainer is called, if indicated by Device Plugin during registeration phase,
// before each container start. Device plugin can run device specific operations
// such as resetting the device before making devices available to the container.
// rpc PreStartContainer(PreStartContainerRequest) returns (PreStartContainerResponse) {}
func (m *DevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

// GetPreferredAllocation returns a preferred set of devices to allocate
// from a list of available ones. The resulting preferred allocation is not
// guaranteed to be the allocation ultimately performed by the
// devicemanager. It is only designed to help the devicemanager make a more
// informed allocation decision when possible.
// rpc GetPreferredAllocation(PreferredAllocationRequest) returns (PreferredAllocationResponse) {}
func (m *DevicePlugin) GetPreferredAllocation(ctx context.Context, r *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func main() {
	m := NewDevicePlugin()
	if err := m.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
