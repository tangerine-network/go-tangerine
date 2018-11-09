// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/dexon-foundation/dexon/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("dex/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("dex/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("dex/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("dex/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("dex/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("dex/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("dex/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("dex/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("dex/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("dex/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("dex/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("dex/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("dex/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("dex/downloader/states/drop", nil)
)
