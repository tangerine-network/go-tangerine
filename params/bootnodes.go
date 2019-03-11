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

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	"enode://0478aa13c91aa0db8e93b668313b7eb0532fbdb24f64772375373b14dbe326c238ad09ab4469f6442c9a9753f1275aeec2e531912c14a958ed1feb4ae7e227ef@127.0.0.1:30301",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Taiwan test network.
var TestnetBootnodes = []string{
	"enode://0478aa13c91aa0db8e93b668313b7eb0532fbdb24f64772375373b14dbe326c238ad09ab4469f6442c9a9753f1275aeec2e531912c14a958ed1feb4ae7e227ef@35.234.27.122:30301",
}

// TaipeiBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Taipei test network.
var TaipeiBootnodes = []string{
	"enode://0478aa13c91aa0db8e93b668313b7eb0532fbdb24f64772375373b14dbe326c238ad09ab4469f6442c9a9753f1275aeec2e531912c14a958ed1feb4ae7e227ef@34.80.32.107:30301",
}

// YilanBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Yilan test network.
var YilanBootnodes = []string{
	"enode://0478aa13c91aa0db8e93b668313b7eb0532fbdb24f64772375373b14dbe326c238ad09ab4469f6442c9a9753f1275aeec2e531912c14a958ed1feb4ae7e227ef@34.80.4.56:30301",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}
