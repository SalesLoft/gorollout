package rollout

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v4"
)

func TestNewFeature(t *testing.T) {
	f := NewFeature("example")

	assert.EqualValues(t, "example", f.name)
	assert.Zero(t, f.percentage)
	assert.Empty(t, f.teamIDs)
}

func TestEncodeDecode(t *testing.T) {
	// encode a mix of percentage and teams
	in := NewFeature("example")
	in.percentage = 50
	in.activateTeam(1)
	in.activateTeam(2)
	in.activateTeam(3)

	data, err := msgpack.Marshal(in)
	assert.NoError(t, err)

	out := NewFeature("example")
	err = msgpack.Unmarshal(data, out)
	assert.NoError(t, err)

	assert.EqualValues(t, in.Name(), out.Name())
	assert.EqualValues(t, in.percentage, out.percentage)
	assert.EqualValues(t, in.teamIDs, out.teamIDs)
	assert.True(t, out.isTeamActive(1, false))
	assert.True(t, out.isTeamActive(2, false))
	assert.True(t, out.isTeamActive(3, false))

	// encode just percentage
	in = NewFeature("example")
	in.percentage = 50

	data, err = msgpack.Marshal(in)
	assert.NoError(t, err)

	out = NewFeature("example")
	err = msgpack.Unmarshal(data, out)
	assert.NoError(t, err)

	assert.EqualValues(t, in.Name(), out.Name())
	assert.EqualValues(t, in.percentage, out.percentage)
	assert.EqualValues(t, in.teamIDs, out.teamIDs)

	// encode just teams
	in = NewFeature("example")
	in.activateTeam(1)
	in.activateTeam(2)
	in.activateTeam(3)

	data, err = msgpack.Marshal(in)
	assert.NoError(t, err)

	out = NewFeature("example")
	err = msgpack.Unmarshal(data, out)
	assert.NoError(t, err)

	assert.EqualValues(t, in.Name(), out.Name())
	assert.EqualValues(t, in.percentage, out.percentage)
	assert.EqualValues(t, in.teamIDs, out.teamIDs)

	assert.True(t, out.isTeamActive(1, false))
	assert.True(t, out.isTeamActive(2, false))
	assert.True(t, out.isTeamActive(3, false))
}

func TestEnableDisableTeam(t *testing.T) {
	f := NewFeature("example")
	assert.False(t, f.isTeamActive(1, false))

	f.activateTeam(1)
	f.activateTeam(2)

	assert.True(t, f.isTeamActive(1, false))
	assert.True(t, f.isTeamActive(2, false))
	assert.False(t, f.isTeamActive(3, false))

	f.deactivateTeam(1)

	assert.False(t, f.isTeamActive(1, false))
	assert.True(t, f.isTeamActive(2, false))
}

func TestEnableDisableFeature(t *testing.T) {
	f := NewFeature("example")
	assert.False(t, f.isTeamActive(1, false))

	f.activate()
	assert.True(t, f.isTeamActive(1, false))
	assert.True(t, f.isTeamActive(999999999999, false))

	f.deactivate()
	assert.False(t, f.isTeamActive(1, false))
	assert.False(t, f.isTeamActive(99999999999, false))
}

func TestRollout(t *testing.T) {
	f := NewFeature("example")

	// 75% < Team 1 < 100%
	// 25% < Team 2 < 50%
	// 0 % < Team 3 < 25%

	assert.False(t, f.isTeamActive(1, true))
	assert.False(t, f.isTeamActive(2, true))
	assert.False(t, f.isTeamActive(3, true))

	f.activatePercentage(25)

	assert.False(t, f.isTeamActive(1, true))
	assert.False(t, f.isTeamActive(2, true))
	assert.True(t, f.isTeamActive(3, true))

	f.activatePercentage(50)

	assert.False(t, f.isTeamActive(1, true))
	assert.True(t, f.isTeamActive(2, true))
	assert.True(t, f.isTeamActive(3, true))

	f.activatePercentage(75)

	assert.False(t, f.isTeamActive(1, true))
	assert.True(t, f.isTeamActive(2, true))
	assert.True(t, f.isTeamActive(3, true))

	f.activatePercentage(100)

	assert.True(t, f.isTeamActive(1, true))
	assert.True(t, f.isTeamActive(2, true))
	assert.True(t, f.isTeamActive(3, true))
}

func TestRolloutactivateTeamMix(t *testing.T) {
	f := NewFeature("example")

	// 75% < Team 1 < 100%
	// 25% < Team 2 < 50%
	// 0 % < Team 3 < 25%

	assert.False(t, f.isTeamActive(1, true))
	assert.False(t, f.isTeamActive(2, true))
	assert.False(t, f.isTeamActive(3, true))

	f.activateTeam(1)
	f.activatePercentage(25)

	assert.True(t, f.isTeamActive(1, true))
	assert.False(t, f.isTeamActive(2, true))
	assert.True(t, f.isTeamActive(3, true))
}

func TestRandomizePercentage(t *testing.T) {
	f := NewFeature("example")
	f.activatePercentage(50)

	// teamID == 10 will be active at 50% when randomized, but not when static
	teamID := int64(10)

	assert.True(t, f.isTeamActive(teamID, true))
	assert.False(t, f.isTeamActive(teamID, false))
}
