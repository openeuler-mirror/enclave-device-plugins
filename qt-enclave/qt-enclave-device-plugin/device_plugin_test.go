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
 * Description: provide device plugin testcase
 *********************************************************************************/

package main

import (
	"testing"
)

func TestIncrementalDeviceIdGenerationSuccess(t *testing.T) {
	deviceName := "dummy_device"
	expected := "dummy_device0"

	id := generateDeviceID(deviceName)

	if expected != id {
		t.Fatalf("Expected %s but got invalid id: %s!", expected, id)
		return
	}

	deviceIdCounter = 49
	_ = generateDeviceID(deviceName)
	deviceName = "qtbox_service"
	expected = "qtbox_service50"
	id = generateDeviceID(deviceName)

	if expected != id {
		t.Fatalf("Expected %s but got invalid id: %s!", expected, id)
		return
	}
}

func TestValidateDeviceNameSuccess(t *testing.T) {
	deviceIdCounter = 20
	p := NewQtEnclavesDevicePlugin()

	expected := "qtbox_service20"

	if expected != p.devs[0].ID {
		t.Fatalf("Expected %s but got invalid id: %s!", expected, p.devs[0].ID)
		return
	}
}
