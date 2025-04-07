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
 * Description: provide device plugin monitor
 *********************************************************************************/

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	pluginStartRetryTimeout = 3 * time.Second
)

type QtEnclavesPluginMonitor struct {
	devicePlugin IBasicDevicePlugin
	fsWatcher    *fsnotify.Watcher
	sigWatcher   chan os.Signal
	restart      bool
}

func newFSWatcher(file string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.Add(file)
	if err != nil {
		watcher.Close()
		return nil, err
	}

	return watcher, nil
}

func newOSWatcher(sigs ...os.Signal) chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, sigs...)

	return sigChan
}

func (qtepm *QtEnclavesPluginMonitor) Init() error {
	glog.V(0).Info("Creating plugin monitor...")

	var err error

	glog.V(0).Info("Starting FS watcher.")
	qtepm.fsWatcher, err = newFSWatcher(pluginapi.DevicePluginPath)
	if err != nil {
		glog.Error("Failed to created FS watcher:", pluginapi.DevicePluginPath)
		return err
	}

	glog.V(0).Info("Starting OS watcher.")
	qtepm.sigWatcher = newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	glog.V(0).Info("Plugin monitor has been successfully created.")

	qtepm.restart = true

	return nil
}

func (qtepm *QtEnclavesPluginMonitor) Run() {
	defer qtepm.fsWatcher.Close()

L:
	for {
		if qtepm.restart {
			if err := qtepm.devicePlugin.Start(); err != nil {
				// Sleep and try again as long as the monitor is running.
				glog.V(0).Info("Could not contact Kubelet, retrying. Did you enable the device plugin feature gate?")
				time.Sleep(pluginStartRetryTimeout)
				continue
			} else {
				qtepm.restart = false
			}
		}

		select {
		case event := <-qtepm.fsWatcher.Events:
			if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				glog.V(0).Infof("Kubelet sock has been re/created. The plugin needs a restart.")
				qtepm.devicePlugin.Stop()
				qtepm.restart = true
			}

		case err := <-qtepm.fsWatcher.Errors:
			glog.V(0).Infof("Terminating plugin monitor... (Reason: inotify: %s)", err)
			qtepm.devicePlugin.Stop()
			break L

		case sig := <-qtepm.sigWatcher:
			switch sig {
			case syscall.SIGHUP:
				glog.V(0).Infof("Received SIGHUP, restarting.")
				qtepm.devicePlugin.Stop()
				qtepm.restart = true
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				glog.V(0).Infof("Terminating plugin monitor... (Reason: \"%v\")", sig)
				qtepm.devicePlugin.Stop()
				break L
			}
		}
	}
}

// Create a new plugin monitor.
func NewQtEnclavesPluginMonitor(qtedp *QtEnclavesDevicePlugin) *QtEnclavesPluginMonitor {
	qtepm := &QtEnclavesPluginMonitor{
		devicePlugin: qtedp,
	}

	if qtepm.Init() != nil {
		qtepm = nil
	}

	return qtepm
}
