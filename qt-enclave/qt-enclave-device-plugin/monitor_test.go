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
 * Create: 2025-04-07
 * Description: provide device plugin monitor testcase
 *********************************************************************************/
package main

import (
	"errors"
	"os"
	"testing"
	"time"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type DummyDevicePlugin struct {
	IBasicDevicePlugin
	startError error
}

func (d *DummyDevicePlugin) Start() error {
	return d.startError
}

func (*DummyDevicePlugin) Stop() error {
	return nil
}

// Whenever the Kubelet socket is recreated, the plugin
// needs a restart.
func TestIntegrationValidatePluginNeedsARestart(t *testing.T) {
	dp := pluginapi.DevicePluginPath
	ksn := pluginapi.KubeletSocket

	qtepm := &QtEnclavesPluginMonitor{
		devicePlugin: &DummyDevicePlugin{startError: errors.New("Some failure")},
	}

	// Check k8s socket file and create a tmp one
	_, err := os.Stat(ksn)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("Error while k8s socket %s already exists.", ksn)
		t.FailNow()
	}
	err = os.MkdirAll(dp, 0755)
	if err != nil {
		t.Fatalf("Failed to create %s", dp)
		t.FailNow()
	}

	result := qtepm.Init()
	if result != nil {
		t.Fatal("Error while initializing plugin monitor.")
		t.FailNow()
	}

	qtepm.restart = false
	go qtepm.Run()
	// Reschedule
	time.Sleep(100 * time.Millisecond)

	// Create a dummy socket file
	fdesc, err := os.Create(ksn)
	if err != nil {
		t.Fatalf("Failed to create %s", ksn)
		t.FailNow()
	}
	fdesc.Close()
	defer os.Remove(ksn)

	// Wait for the monitor state to change.
	for i := 0; i < 10; i++ {
		if qtepm.restart {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !qtepm.restart {
		t.Fatal("Socket file is generated, but the plugin didn't restart!")
		t.FailNow()
	}
}
