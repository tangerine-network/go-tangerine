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
	"enode://801efa60c24a34a5025bb4472f25e65de951657d2944afe12371c167ba14b9da706dc6e584a2cd22faf98d4975edfc624f63da7accc188ea946a0e2c3e0df132@35.201.155.158:30301",
	"enode://35d0154883fb8571a49951a24a589f16adbd693c06a36e472c4a8c29ed7400d87359e24e54140016ec88106ba12d5678fafc0c86400ed5cf78ba089717c4dc6c@34.73.40.90:30301",
}

// TaipeiBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Taipei test network.
var TaipeiBootnodes = []string{
	"enode://66f54114842accd1e09620d804e114f3f967193bf37a391fed44ea208a5280198d183f1e7297c59719c2b2426fb0e10a4da5f820a2262993f698e16666427224@104.199.141.226:30301",
	"enode://8aafeabce292097e68da9e84e57453af2229340257931e2fecfa527cb77189dd8a70144854e279918ad4c631674dfbaf0e3058dab451d5964d7c9fdc6c79cbcc@35.196.137.55:30301",
}

// YilanBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Yilan test network.
var YilanBootnodes = []string{
	"enode://242df07a1fa3c337b50c0d0f4d7d305c00f2610383a8e468567dba53d77c5e67213e85f00d884baef96dfc726da5f5a4de7c4ef3e295fceb543c8ed4337b599f@34.80.4.56:30301",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}
