/*
Copyright (c) Facebook, Inc. and its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package oscillatord implements monitoring protocol used by Orolia's oscillatord,
daemon for disciplining an oscillator.

All references throughout the code relate to the https://github.com/Orolia2s/oscillatord code.
*/
package oscillatord

import (
	"encoding/json"
	"fmt"
	"io"
)

// AntennaStatus is an enum describing antenna status as reported by oscillatord
type AntennaStatus int

// from oscillatord src/gnss.c
const (
	AntStatusInit AntennaStatus = iota
	AntStatusDontKnow
	AntStatusOK
	AntStatusSHORT
	AntStatusOpen
	AntStatusUndefined
)

var antennaStatusToString = map[AntennaStatus]string{
	AntStatusInit:      "INIT",
	AntStatusDontKnow:  "DONTKNOW",
	AntStatusOK:        "OK",
	AntStatusSHORT:     "SHORT",
	AntStatusOpen:      "OPEN",
	AntStatusUndefined: "UNDEFINED",
}

func (a AntennaStatus) String() string {
	s, found := antennaStatusToString[a]
	if !found {
		return "UNSUPPORTED VALUE"
	}
	return s
}

// AntennaPower is an enum describing antenna power status as reported by oscillatord
type AntennaPower int

// from oscillatord src/gnss.c
const (
	AntPowerOff AntennaPower = iota
	AntPowerOn
	AntPowerDontKnow
	AntPowerIdle
	AntPowerUndefined
)

var antennaPowerToString = map[AntennaPower]string{
	AntPowerOff:       "OFF",
	AntPowerOn:        "ON",
	AntPowerDontKnow:  "DONTKNOW",
	AntPowerIdle:      "IDLE",
	AntPowerUndefined: "UNDEFINED",
}

func (p AntennaPower) String() string {
	s, found := antennaPowerToString[p]
	if !found {
		return "UNSUPPORTED VALUE"
	}
	return s
}

// GNSSFix is an enum describing GNSS fix status as reported by oscillatord
type GNSSFix int

// from oscillatord src/gnss.c
const (
	FixUnknown GNSSFix = iota
	FixNoFix
	FixDROnly
	FixTime
	Fix2D
	Fix3D
	Fix3DDr
	FixRTKFloat
	FixRTKFixed
	FixFloatDr
	FixFixedDr
)

var gnssFixToString = map[GNSSFix]string{
	FixUnknown:  "Unknown",
	FixNoFix:    "No fix",
	FixDROnly:   "DR only",
	FixTime:     "Time",
	Fix2D:       "2D",
	Fix3D:       "3D",
	Fix3DDr:     "3D_DR",
	FixRTKFloat: "RTK_FLOAT",
	FixRTKFixed: "RTK_FIXED",
	FixFloatDr:  "RTK_FLOAT_DR",
	FixFixedDr:  "RTK_FIXED_DR",
}

func (f GNSSFix) String() string {
	s, found := gnssFixToString[f]
	if !found {
		return "UNSUPPORTED VALUE"
	}
	return s
}

// LeapSecondChange is enum that oscillatord uses to indicate leap second change
type LeapSecondChange int

// from oscillatord src/gnss.c
const (
	LeapNoWarning LeapSecondChange = 0
	LeapAddSecond LeapSecondChange = 1
	LeapDelSecond LeapSecondChange = -1
)

var leapSecondChangeToString = map[LeapSecondChange]string{
	LeapNoWarning: "NO WARNING",
	LeapAddSecond: "ADD SECOND",
	LeapDelSecond: "DEL SECOND",
}

func (c LeapSecondChange) String() string {
	s, found := leapSecondChangeToString[c]
	if !found {
		return "UNSUPPORTED VALUE"
	}
	return s
}

// Oscillator describes structure that oscillatord returns for oscillator
type Oscillator struct {
	Model       string  `json:"model"`
	FineCtrl    int     `json:"fine_ctrl"`
	CoarseCtrl  int     `json:"coarse_ctrl"`
	Lock        bool    `json:"lock"`
	Temperature float64 `json:"temperature"`
}

// GNSS describes structure that oscillatord returns for gnss
type GNSS struct {
	Fix           GNSSFix          `json:"fix"`
	FixOK         bool             `json:"fixOk"`
	AntennaPower  AntennaPower     `json:"antenna_power"`
	AntennaStatus AntennaStatus    `json:"antenna_status"`
	LSChange      LeapSecondChange `json:"lsChange"`
	LeapSeconds   int              `json:"leap_seconds"`
}

// Status is whole structure that oscillatord returns for monitoring
type Status struct {
	Oscillator Oscillator `json:"oscillator"`
	GNSS       GNSS       `json:"gnss"`
}

// ReadStatus talks to oscillatord via monitoring port connection and reads reported Status
func ReadStatus(conn io.ReadWriter) (*Status, error) {
	// send newline to make oscillatord send us data
	_, err := conn.Write([]byte{'\n'})
	if err != nil {
		return nil, fmt.Errorf("writing to oscillatord conn: %w", err)
	}
	buf := make([]byte, 1000)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("reading from oscillatord conn: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("read 0 bytes from oscillatord")
	}
	var status Status
	if err := json.Unmarshal(buf[:n], &status); err != nil {
		return nil, fmt.Errorf("unmarshalling JSON: %w", err)
	}
	return &status, nil
}
