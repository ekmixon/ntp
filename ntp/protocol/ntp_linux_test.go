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

package protocol

import (
	"github.com/stretchr/testify/require"
	syscall "golang.org/x/sys/unix"
	"net"
	"testing"
)

func TestEnableKernelTimestampsSocket(t *testing.T) {
	// listen to incoming udp packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer conn.Close()

	connfd, err := connFd(conn)
	require.NoError(t, err)

	// Allow reading of kernel timestamps via socket
	err = EnableKernelTimestampsSocket(conn)
	require.NoError(t, err)

	// Check that socket option is set
	preciseKernelTimestampsEnabled, err := syscall.GetsockoptInt(connfd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMPNS)
	require.NoError(t, err)
	kernelTimestampsEnabled, err := syscall.GetsockoptInt(connfd, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP)
	require.NoError(t, err)

	// At least one of them should be set, which it > 0
	require.Greater(t, preciseKernelTimestampsEnabled+kernelTimestampsEnabled, 0, "None of the socket options is set")
}
