// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: GPL-3.0-or-later

/**
 * QuakeAlert ESP32 - Network & Location
 */

#ifndef NETWORK_H
#define NETWORK_H

#include <Arduino.h>

void initWifi();
bool maintainWifiConnection();
void checkNtpSync();
bool refreshLocation();
void networkMaintenanceTask(void* pvParameters);
void handleProvisioningLoop();


#endif  // NETWORK_H