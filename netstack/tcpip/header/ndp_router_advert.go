// Copyright 2019 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package header

// NDPRouterAdvert is an NDP Router Advertisement message. It will only contain
// the body of an ICMPv6 packet.
//
// See RFC 4861 section 4.2 for more details.
type NDPRouterAdvert []byte

const (
	// NDPRAMinimumSize is the minimum size of a valid NDP Router
	// Advertisement message (body of an ICMPv6 packet).
	NDPRAMinimumSize = 12

	// ndpRACurrHopLimitOffset is the byte of the Curr Hop Limit field
	// within an NDPRouterAdvert.
	ndpRACurrHopLimitOffset = 0

	// ndpRAFlagsOffset is the byte with the NDP RA bit-fields/flags
	// within an NDPRouterAdvert.
	ndpRAFlagsOffset = 1

	// ndpRAManagedAddrConfFlagMask is the mask of the Managed Address
	// Configuration flag within the bit-field/flags byte of an
	// NDPRouterAdvert.
	ndpRAManagedAddrConfFlagMask = (1 << 7)

	// ndpRAOtherConfFlagMask is the mask of the Other Configuration flag
	// within the bit-field/flags byte of an NDPRouterAdvert.
	ndpRAOtherConfFlagMask = (1 << 6)

	// ndpRARouterLifetimeOffset is the start of the 2-byte Router Lifetime
	// field within an NDPRouterAdvert.
	ndpRARouterLifetimeOffset = 2

	// ndpRAReachableTimeOffset is the start of the 4-byte Reachable Time
	// field within an NDPRouterAdvert.
	ndpRAReachableTimeOffset = 4

	// ndpRARetransTimerOffset is the start of the 4-byte Retrans Timer
	// field within an NDPRouterAdvert.
	ndpRARetransTimerOffset = 8

	// ndpRAOptionsOffset is the start of the NDP options in an
	// NDPRouterAdvert.
	ndpRAOptionsOffset = 12
)
