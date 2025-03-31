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
	"flag"
	"os"

	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	glog.V(0).Info("Loading K8s Qt Enclaves device plugin...")

	devicePlugin := NewQtEnclavesDevicePlugin()

	monitor := NewQtEnclavesPluginMonitor(devicePlugin)
	if monitor == nil {
		glog.Error("Error while initializing Qt Enclaves device plugin monitor!")
		os.Exit(1)
	}

	monitor.Run()
}
