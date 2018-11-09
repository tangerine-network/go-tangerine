// Copyright 2018 The dexon-consensus Authors
// This file is part of the dexon-consensus library.
//
// The dexon-consensus library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus library. If not, see
// <http://www.gnu.org/licenses/>.

package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "os/signal"
    "runtime"
    "syscall"
)

func init() {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGUSR1)
    fmt.Printf("Listening on SIGUSR1\n")
    go func() {
        for range sigChan {
            fmt.Printf("Receive SIGUSR1\n")
            // Dump stack.
            buf := make([]byte, 4*1024*1024)
            buf = buf[:runtime.Stack(buf, true)]
            if err := ioutil.WriteFile("stack.log", buf, 0644); err != nil {
                fmt.Printf("Unable to dump stack trace: %s\n", err)
            }
        }
    }()
}
