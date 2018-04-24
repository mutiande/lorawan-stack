// Copyright © 2018 The Things Network Foundation, The Things Industries B.V.
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

package band

import (
	"github.com/TheThingsNetwork/ttn/pkg/errors"
	"github.com/TheThingsNetwork/ttn/pkg/types"
)

var eu_433 Band

const (
	// EU_433 is the ID of the European 433Mhz band
	EU_433 string = "EU_433"
)

func init() {
	defaultChannels := []Channel{
		{Frequency: 433175000, DataRateIndexes: []int{0, 1, 2, 3, 4, 5}},
		{Frequency: 433375000, DataRateIndexes: []int{0, 1, 2, 3, 4, 5}},
		{Frequency: 433575000, DataRateIndexes: []int{0, 1, 2, 3, 4, 5}},
	}
	eu433BeaconChannel := uint32(434655000)
	eu_433 = Band{
		ID: EU_433,

		UplinkChannels:   defaultChannels,
		DownlinkChannels: defaultChannels,

		BandDutyCycles: []DutyCycle{
			{
				MinFrequency: 433175000,
				MaxFrequency: 434665000,
				DutyCycle:    0.01,
			},
		},

		DataRates: [16]DataRate{
			{Rate: types.DataRate{LoRa: "SF12BW125"}, DefaultMaxSize: maxPayloadSize{59, 51}, NoRepeaterMaxSize: maxPayloadSize{59, 51}},
			{Rate: types.DataRate{LoRa: "SF11BW125"}, DefaultMaxSize: maxPayloadSize{59, 51}, NoRepeaterMaxSize: maxPayloadSize{59, 51}},
			{Rate: types.DataRate{LoRa: "SF10BW125"}, DefaultMaxSize: maxPayloadSize{59, 51}, NoRepeaterMaxSize: maxPayloadSize{59, 51}},
			{Rate: types.DataRate{LoRa: "SF9BW125"}, DefaultMaxSize: maxPayloadSize{123, 115}, NoRepeaterMaxSize: maxPayloadSize{123, 115}},
			{Rate: types.DataRate{LoRa: "SF8BW125"}, DefaultMaxSize: maxPayloadSize{230, 222}, NoRepeaterMaxSize: maxPayloadSize{250, 242}},
			{Rate: types.DataRate{LoRa: "SF7BW125"}, DefaultMaxSize: maxPayloadSize{230, 222}, NoRepeaterMaxSize: maxPayloadSize{250, 242}},
			{Rate: types.DataRate{LoRa: "SF7BW250"}, DefaultMaxSize: maxPayloadSize{230, 222}, NoRepeaterMaxSize: maxPayloadSize{250, 242}},
			{Rate: types.DataRate{FSK: 50000}, DefaultMaxSize: maxPayloadSize{230, 222}, NoRepeaterMaxSize: maxPayloadSize{250, 242}},
			{}, {}, {}, {}, {}, {}, {}, // RFU
			{}, // Used by LinkADRReq starting from LoRaWAN Regional Parameters 1.1, RFU before
		},

		ReceiveDelay1:    defaultReceiveDelay1,
		ReceiveDelay2:    defaultReceiveDelay2,
		JoinAcceptDelay1: defaultJoinAcceptDelay2,
		JoinAcceptDelay2: defaultJoinAcceptDelay2,
		MaxFCntGap:       defaultMaxFCntGap,
		ADRAckLimit:      defaultADRAckLimit,
		ADRAckDelay:      defaultADRAckDelay,
		MinAckTimeout:    defaultAckTimeout - defaultAckTimeoutMargin,
		MaxAckTimeout:    defaultAckTimeout + defaultAckTimeoutMargin,

		DefaultMaxEIRP: 12.15,
		TxOffset: [16]float32{0, -2, -4, -6, -8, -10,
			0, 0, 0, 0, 0, 0, 0, 0, 0, // RFU
			0, // Used by LinkADRReq starting from LoRaWAN Regional Parameters 1.1, RFU before
		},

		Rx1Channel: rx1ChannelIdentity,
		Rx1DataRate: func(idx, offset uint32, _ bool) (uint32, error) {
			if idx > 7 {
				return 0, ErrLoRaWANParametersInvalid.NewWithCause(nil, errors.New("Data rate index must be lower or equal to 7"))
			}
			if offset > 5 {
				return 0, ErrLoRaWANParametersInvalid.NewWithCause(nil, errors.New("Offset must be lower or equal to 5"))
			}

			si := int(idx - offset)
			switch {
			case si <= 0:
				return 0, nil
			case si >= 7:
				return 7, nil
			}
			return uint32(si), nil
		},

		ImplementsCFList: true,

		DefaultRx2Parameters: Rx2Parameters{0, 434665000},

		Beacon: Beacon{
			DataRateIndex:    3,
			CodingRate:       "4/5",
			BroadcastChannel: func(_ float64) uint32 { return eu433BeaconChannel },
			PingSlotChannels: []uint32{eu433BeaconChannel},
		},

		regionalParameters1_0:   bandIdentity,
		regionalParameters1_0_1: bandIdentity,
		regionalParameters1_0_2: bandIdentity,
		regionalParameters1_1A:  bandIdentity,
	}
	All = append(All, eu_433)
}