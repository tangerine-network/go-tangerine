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
	"enode://5a3cb179e9f1be6ef9178875f52737ef97d9f899fea11499c00b08518171c5132c6f29cc009730beb83d00dcd5be5ecfe85af770989d0f501bba2e4504df4011@35.229.197.69:30301",
	"enode://a3e837bc756817cd5b7c6c1551971786f4fb5d6c649b3d79e293e1ba1a2959257067618715a9b09c26ed949d8d100c7c2818aea0c0beb55317bc727b8e6e721c@35.229.124.196:30301",
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
