/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2025. All rights reserved.
 * iSulad licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: liuxu
 * Create: 2025-03-17
 * Description: provide device plugin
 *********************************************************************************/

package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	deviceName                      = "qtbox_service"
	socketPath                      = pluginapi.DevicePluginPath + deviceName + ".sock"
	resourceName                    = "huawei.com/qt_enclaves"
	devicePluginServerReadyTimeout  = 10 * time.Second
	devicePluginHealthCheckInterval = 5 * time.Second
	enclavesPerInstance             = 1
)

var deviceIdCounter = 0

type IBasicDevicePlugin interface {
	Start() error
	Stop() error
}

// QtEnclavesDevicePlugin implements the Kubernetes device plugin API
type QtEnclavesDevicePlugin struct {
	devs   []*pluginapi.Device
	socket string

	stop   chan interface{}
	health chan *pluginapi.Device

	server *grpc.Server

	IBasicDevicePlugin
}

func devicePath(deviceId string) string {
	return "/dev/" + deviceId
}

func generateDeviceID(deviceName string) string {
	ctr := deviceIdCounter
	deviceIdCounter++
	return deviceName + strconv.Itoa(ctr)
}

func (qtedp *QtEnclavesDevicePlugin) cleanup() error {
	if err := os.Remove(qtedp.socket); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (qtedp *QtEnclavesDevicePlugin) register(kubeletEndpoint, resourceName string) error {
	glog.V(0).Info("Attempting to connect to kubelet...")

	conn, err := dial(kubeletEndpoint, devicePluginServerReadyTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	glog.V(0).Info("Connected to kubelet.")

	client := pluginapi.NewRegistrationClient(conn)
	_, err = client.Register(context.Background(), &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(qtedp.socket),
		ResourceName: resourceName,
	})

	return err
}

func (qtedp *QtEnclavesDevicePlugin) healthcheck() {
	for {
		select {
		case <-qtedp.stop:
			return
		default:
		}
		for _, dev := range qtedp.devs {
			tmpHealth := dev.Health
			devPath := devicePath(dev.ID)
			_, err := os.Stat(devPath)
			if err != nil {
				if os.IsNotExist(err) {
					glog.Error("Device is not exist:", devPath)
				} else {
					glog.Error("Device %s: %v", devPath, err)
				}
				tmpHealth = pluginapi.Unhealthy
			} else {
				tmpHealth = pluginapi.Healthy
			}

			if dev.Health != tmpHealth {
				dev.Health = tmpHealth
				qtedp.health <- dev
			}
		}
		time.Sleep(devicePluginHealthCheckInterval)
	}
}

func (qtedp *QtEnclavesDevicePlugin) deviceExists(id string) bool {
	for _, d := range qtedp.devs {
		if d.ID == id {
			return true
		}
	}
	return false
}

// Allocate is called during container creation so that the Device
// Plugin can run device specific operations and instruct Kubelet
// of the steps to make the Device available in the container
func (qtedp *QtEnclavesDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	responses := pluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		var devicesList []*pluginapi.DeviceSpec
		for _, id := range req.DevicesIDs {
			if !qtedp.deviceExists(id) {
				return nil, fmt.Errorf("invalid allocation request: unknown device: %s", id)
			}
			glog.V(1).Info("Allocation request for device ID: ", id)

			ds := &pluginapi.DeviceSpec{
				ContainerPath: devicePath(id),
				HostPath:      devicePath(id),
				Permissions:   "rw",
			}
			devicesList = append(devicesList, ds)
		}

		responses.ContainerResponses = append(responses.ContainerResponses, &pluginapi.ContainerAllocateResponse{
			Devices: devicesList,
		})
	}

	return &responses, nil
}

func (qtedp *QtEnclavesDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (qtedp *QtEnclavesDevicePlugin) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// ListAndWatch lists devices and update that list according to the health status
func (qtedp *QtEnclavesDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: qtedp.devs})

	for {
		select {
		case <-qtedp.stop:
			glog.V(0).Infof("Device stopped")
			return nil
		case d := <-qtedp.health:
			glog.V(1).Infof("Device %s health changed to: %s", d.ID, d.Health)
			s.Send(&pluginapi.ListAndWatchResponse{Devices: qtedp.devs})
		}
	}
}

func (qtedp *QtEnclavesDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

// Start device plugin server
func (qtedp *QtEnclavesDevicePlugin) Start() error {
	err := qtedp.cleanup()
	if err != nil {
		return err
	}
	glog.V(0).Info("Starting Qt Enclaves device plugin server...")

	sock, err := net.Listen("unix", qtedp.socket)
	if err != nil {
		glog.Error("Error while creating socket: ", qtedp.socket)
		return err
	}

	qtedp.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(qtedp.server, qtedp)
	qtedp.stop = make(chan interface{})

	go qtedp.server.Serve(sock)

	// Wait for server to start by launching a blocking connection
	conn, err := dial(qtedp.socket, devicePluginServerReadyTimeout)
	if err != nil {
		return err
	}
	conn.Close()

	if err := qtedp.register(pluginapi.KubeletSocket, resourceName); err != nil {
		glog.Errorf("Error while registering device plugin with kubelet! (Reason: %s)", err)
		qtedp.Stop()
		return err
	}

	glog.V(0).Info("Registered device plugin with Kubelet: ", resourceName)

	go qtedp.healthcheck()

	return nil
}

// Stop device plugin server
func (qtedp *QtEnclavesDevicePlugin) Stop() error {
	if qtedp.server != nil {
		qtedp.server.Stop()
		qtedp.server = nil
		close(qtedp.stop)
	}

	if err := qtedp.cleanup(); err != nil {
		return err
	}
	glog.V(0).Infof("Device plugin stopped. (Socket: %s)", qtedp.socket)

	return nil
}

// NewQtEnclavesDevicePlugin returns an initialized QtEnclavesDevicePlugin
func NewQtEnclavesDevicePlugin() *QtEnclavesDevicePlugin {
	devs := []*pluginapi.Device{}
	for i := 0; i < enclavesPerInstance; i++ {
		devs = append(devs, &pluginapi.Device{
			ID:     generateDeviceID(deviceName),
			Health: pluginapi.Healthy,
		})
	}
	return &QtEnclavesDevicePlugin{
		devs:   devs,
		socket: socketPath,
		health: make(chan *pluginapi.Device),
	}
}
