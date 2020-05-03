package rollout

import (
	"hash/crc32"
	"math"
	"strconv"
	"sync"

	msgpack "github.com/vmihailenco/msgpack/v4"
)

const (
	randBase = uint32((math.MaxUint32 - 1) / 100)
)

// NewFeature constructs a new Feature with the given name
func NewFeature(name string) *Feature {
	return &Feature{name: name}
}

// Feature represents a development feature toggle for rollout
type Feature struct {
	sync.Mutex

	name       string // the name of the feature
	percentage uint8  // the rollout percentage
	teamIDs    intSet // explicit team ids with the feature enabled
}

// EncodeMsgpack implements msgpack.CustomEncoder
func (f *Feature) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.EncodeMulti(f.percentage, f.teamIDs)
}

// DecodeMsgpack implements msgpack.CustomDecoder
func (f *Feature) DecodeMsgpack(dec *msgpack.Decoder) error {
	return dec.DecodeMulti(&f.percentage, &f.teamIDs)
}

// Name returns the name of the feature
func (f *Feature) Name() string {
	return f.name
}

func (f *Feature) activate() {
	f.percentage = 100
}

func (f *Feature) deactivate() {
	f.percentage = 0
	f.teamIDs = nil
}

func (f *Feature) activatePercentage(percentage uint8) {
	f.percentage = percentage
}

func (f *Feature) isActive() bool {
	return f.percentage == 100
}

func (f *Feature) activateTeam(teamID int64) {
	if f.teamIDs == nil {
		f.teamIDs = make(intSet)
	}

	f.teamIDs[teamID] = struct{}{}
}

func (f *Feature) deactivateTeam(teamID int64) {
	delete(f.teamIDs, teamID)
}

func (f *Feature) isTeamActive(teamID int64) bool {
	if f.percentage == 100 {
		return true
	} else if crc32.ChecksumIEEE([]byte(f.name+strconv.FormatInt(teamID, 10))) < randBase*uint32(f.percentage) {
		return true
	} else if _, active := f.teamIDs[teamID]; active {
		return true
	}

	return false
}

// ref: https://github.com/vmihailenco/msgpack/blob/master/types_test.go#L52
type intSet map[int64]struct{}

func (s intSet) EncodeMsgpack(enc *msgpack.Encoder) error {
	slice := make([]int64, 0, len(s))
	for n := range s {
		slice = append(slice, n)
	}
	return enc.Encode(slice)
}

func (s *intSet) DecodeMsgpack(dec *msgpack.Decoder) error {
	n, err := dec.DecodeArrayLen()
	if err != nil {
		return err
	}

	set := make(intSet, n)
	for i := 0; i < n; i++ {
		n, err := dec.DecodeInt64()
		if err != nil {
			return err
		}
		set[n] = struct{}{}
	}
	*s = set

	return nil
}
