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

package api

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-ini/ini"
	"github.com/stretchr/testify/require"
)

func TestChannel(t *testing.T) {
	legitChannelNamesToChannel := map[string]Channel{
		"1": ChannelONE,
		"2": ChannelTWO,
		"c": ChannelC,
		"d": ChannelD,
	}
	for channelS, channel := range legitChannelNamesToChannel {
		c, err := ChannelFromString(channelS)
		require.NoError(t, err)
		require.Equal(t, channel, *c)

		c = new(Channel)
		err = c.UnmarshalText([]byte(channelS))
		require.NoError(t, err)
		require.Equal(t, channel, *c)

	}

	wrongChannelNames := []string{"", "?", "z", "foo"}
	for _, channelS := range wrongChannelNames {
		c, err := ChannelFromString(channelS)
		require.Nil(t, c)
		require.ErrorIs(t, errBadChannel, err)

		c = new(Channel)
		err = c.UnmarshalText([]byte(channelS))
		require.ErrorIs(t, errBadChannel, err)
	}
}

func TestProbe(t *testing.T) {
	legitProbeNamesToProbe := map[string]Probe{
		"ntp": ProbeNTP,
		"ptp": ProbePTP,
	}
	for probeS, probe := range legitProbeNamesToProbe {
		p, err := ProbeFromString(probeS)
		require.NoError(t, err)
		require.Equal(t, probe, *p)

		p = new(Probe)
		err = p.UnmarshalText([]byte(probeS))
		require.NoError(t, err)
		require.Equal(t, probe, *p)
	}
	wrongProbeNames := []string{"", "?", "z", "dns"}
	for _, probeS := range wrongProbeNames {
		p, err := ProbeFromString(probeS)
		require.Nil(t, p)
		require.ErrorIs(t, errBadProbe, err)

		p = new(Probe)
		err = p.UnmarshalText([]byte(probeS))
		require.ErrorIs(t, errBadProbe, err)
	}
}

func TestProbeFromCalnex(t *testing.T) {
	legitProbeNamesToProbe := map[string]Probe{
		"2": ProbeNTP,
		"0": ProbePTP,
	}
	for probeH, probe := range legitProbeNamesToProbe {
		p, err := ProbeFromCalnex(probeH)
		require.NoError(t, err)
		require.Equal(t, probe, *p)
	}
	wrongProbeNames := []string{"", "?", "z", "dns"}
	for _, probe := range wrongProbeNames {
		p, err := ProbeFromCalnex(probe)
		require.Nil(t, p)
		require.ErrorIs(t, errBadProbe, err)
	}
}

func TestCalnexName(t *testing.T) {
	require.Equal(t, "NTP client", ProbeNTP.CalnexName())
	require.Equal(t, "PTP slave", ProbePTP.CalnexName())
}

func TestTLSSetting(t *testing.T) {
	calnexAPI := NewAPI("localhost", false)
	// Never ever ever allow insucure over https
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	require.Equal(t, transport, calnexAPI.Client.Transport)

	calnexAPI = NewAPI("localhost", true)
	transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	require.Equal(t, transport, calnexAPI.Client.Transport)
}

func TestFetchCsv(t *testing.T) {
	sampleResp := "1607961193.773740,-000.000000250501"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	legitChannelNames := []Channel{ChannelONE, ChannelTWO, ChannelC, ChannelD}

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()
	for _, channel := range legitChannelNames {
		lines, err := calnexAPI.FetchCsv(channel)
		require.NoError(t, err)
		require.Equal(t, 1, len(lines))
		require.Equal(t, sampleResp, strings.Join(lines[0], ","))
	}
}

func TestFetchChannelProtocol_NTP(t *testing.T) {
	sampleResp := "measure/ch6/ptp_synce/mode/probe_type=2"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	probe, err := calnexAPI.FetchChannelProbe(ChannelONE)
	require.NoError(t, err)
	require.Equal(t, ProbeNTP, *probe)
}

func TestFetchChannelProtocol_PTP(t *testing.T) {
	sampleResp := "measure/ch7/ptp_synce/mode/probe_type=0"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	probe, err := calnexAPI.FetchChannelProbe(ChannelTWO)
	require.NoError(t, err)
	require.Equal(t, ProbePTP, *probe)
}

func TestFetchChannelTargetIP_NTP(t *testing.T) {
	sampleResp := "measure/ch6/ptp_synce/ntp/server_ip=fd00:3116:301a::3e"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	ip, err := calnexAPI.FetchChannelTargetIP(ChannelONE, ProbeNTP)
	require.NoError(t, err)
	require.Equal(t, "fd00:3116:301a::3e", ip)
}

func TestFetchChannelTargetIP_PTP(t *testing.T) {
	sampleResp := "measure/ch7/ptp_synce/ptp/master_ip=fd00:3116:301a::3e"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	ip, err := calnexAPI.FetchChannelTargetIP(ChannelTWO, ProbePTP)
	require.NoError(t, err)
	require.Equal(t, "fd00:3116:301a::3e", ip)
}

func TestFetchUsedChannels(t *testing.T) {
	sampleResp := "[measure]\nch0\\used=Yes\nch6\\used=No\nch7\\used=Yes\n"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	expected := []Channel{ChannelA, ChannelTWO}
	used, err := calnexAPI.FetchUsedChannels()
	require.NoError(t, err)
	require.ElementsMatch(t, expected, used)
}

func TestFetchChannelTargetName(t *testing.T) {
	sampleResp := "measure/ch7/ptp_synce/ptp/master_ip=127.0.0.1"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	ip, err := calnexAPI.FetchChannelTargetName(ChannelTWO, ProbePTP)
	require.NoError(t, err)
	require.Equal(t, "localhost", ip)
}

func TestFetchSettings(t *testing.T) {
	sampleResp := "[measure]\nch0\\synce_enabled=Off\n"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	f, err := calnexAPI.FetchSettings()
	require.NoError(t, err)
	require.Equal(t, f.Section("measure").Key("ch0\\synce_enabled").Value(), OFF)
}

func TestFetchStatus(t *testing.T) {
	sampleResp := "{\n\"referenceReady\": true,\n\"modulesReady\": true,\n\"measurementActive\": false\n}"
	expected := &Status{
		ModulesReady:      true,
		ReferenceReady:    true,
		MeasurementActive: false,
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	f, err := calnexAPI.FetchStatus()
	require.NoError(t, err)
	require.Equal(t, expected, f)
}

func TestPushSettings(t *testing.T) {
	sampleResp := "{\n\"result\": true\n}"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	sampleConfig := "[measure]\nch0\\synce_enabled=Off\n"
	f, err := ini.Load([]byte(sampleConfig))
	require.NoError(t, err)

	err = calnexAPI.PushSettings(f)
	require.NoError(t, err)
}

func TestFetchVersion(t *testing.T) {
	sampleResp := "{\"firmware\": \"2.13.1.0.5583D-20210924\"}"
	expected := &Version{
		Firmware: "2.13.1.0.5583D-20210924",
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	f, err := calnexAPI.FetchVersion()
	require.NoError(t, err)
	require.Equal(t, expected, f)
}

func TestPushVersion(t *testing.T) {
	sampleResp := "{\n\"result\" : true,\n\"message\" : \"Installing firmware Version: 2.13.1.0.5583D-20210924\"\n}"
	expected := &Result{
		Result:  true,
		Message: "Installing firmware Version: 2.13.1.0.5583D-20210924",
	}
	// Firmware file itself
	fw, err := ioutil.TempFile("/tmp", "calnex")
	require.NoError(t, err)
	defer fw.Close()
	defer os.Remove(fw.Name())
	_, err = fw.WriteString("Hello Calnex!")
	require.NoError(t, err)

	// Firmware file saved via http
	fwres, err := ioutil.TempFile("/tmp", "calnex")
	require.NoError(t, err)
	defer os.Remove(fwres.Name())

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		defer r.Body.Close()
		defer fwres.Close()
		_, err := io.Copy(fwres, r.Body)
		require.NoError(t, err)

		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	r, err := calnexAPI.PushVersion(fw.Name())
	require.NoError(t, err)
	require.Equal(t, expected, r)

	originalFW, err := ioutil.ReadFile(fw.Name())
	require.NoError(t, err)

	uploadedFW, err := ioutil.ReadFile(fwres.Name())
	require.NoError(t, err)

	require.Equal(t, originalFW, uploadedFW)
}

func TestPost(t *testing.T) {
	sampleResp := "{\n\"result\" : true,\n\"message\" : \"LGTM\"\n}"
	expected := &Result{
		Result:  true,
		Message: "LGTM",
	}
	postData := []byte("Whatever")
	serverReceived := &bytes.Buffer{}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		defer r.Body.Close()
		_, err := serverReceived.ReadFrom(r.Body)
		require.NoError(t, err)
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	buf := bytes.NewBuffer(postData)
	r, err := calnexAPI.post(parsed.String(), buf)
	require.NoError(t, err)
	require.Equal(t, expected, r)
	require.Equal(t, postData, serverReceived.Bytes())
}

func TestGet(t *testing.T) {
	sampleResp := "{\n\"result\": true\n}"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, sampleResp)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	err := calnexAPI.StartMeasure()
	require.NoError(t, err)

	err = calnexAPI.StopMeasure()
	require.NoError(t, err)

	err = calnexAPI.ClearDevice()
	require.NoError(t, err)

	err = calnexAPI.Reboot()
	require.NoError(t, err)
}

func TestHTTPError(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	f := ini.Empty()
	err := calnexAPI.PushSettings(f)
	require.Error(t, err)
}

func TestFetchProblemReport(t *testing.T) {
	expectedReportContent := "I am a problem report"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprint(w, expectedReportContent)
	}))
	defer ts.Close()

	parsed, _ := url.Parse(ts.URL)
	calnexAPI := NewAPI(parsed.Host, true)
	calnexAPI.Client = ts.Client()

	dir, err := ioutil.TempDir("/tmp", "calnex")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	reportFilePath, err := calnexAPI.FetchProblemReport(dir)
	require.NoError(t, err)
	require.FileExists(t, reportFilePath)
	defer os.Remove(reportFilePath)

	require.Contains(t, reportFilePath, "calnex_problem_report_")
	require.Contains(t, reportFilePath, ".tar")

	reportContent, err := os.ReadFile(reportFilePath)
	require.NoError(t, err)

	require.Equal(t, expectedReportContent, string(reportContent))
}
