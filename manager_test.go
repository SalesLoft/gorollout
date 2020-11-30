package rollout

import (
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v4"
)

const (
	mockKeyPrefix = "dealsff"
)

// MockClient is a mock redis client that overrides the methods used in this package for testing
type MockClient struct {
	redis.Cmdable

	setKey                       string
	feature                      Feature
	features                     []*Feature
	shouldError                  bool
	getWasCalled, setWasCalled   bool
	mgetWasCalled, msetWasCalled bool
}

func (c *MockClient) Get(key string) *redis.StringCmd {
	c.getWasCalled = true

	if c.shouldError {
		return redis.NewStringResult("", errors.New("mock error"))
	}

	if c.feature.name != "" {
		// mock return the feature
		data, err := msgpack.Marshal(&c.feature)
		if err != nil {
			return redis.NewStringResult("", err)
		}

		return redis.NewStringResult(string(data), nil)
	}

	// mock not found
	return redis.NewStringResult("", redis.Nil)
}

func (c *MockClient) MGet(keys ...string) *redis.SliceCmd {
	c.mgetWasCalled = true

	if c.shouldError {
		return redis.NewSliceResult(nil, errors.New("mock error"))
	}

	if c.features != nil {
		// mock return the found features
		val := make([]interface{}, len(keys))

		for i, key := range keys {
			for _, feature := range c.features {
				if mockKeyPrefix+":"+feature.name == key {
					var err error
					data, err := msgpack.Marshal(feature)
					if err != nil {
						return redis.NewSliceResult(nil, err)
					}
					val[i] = string(data)
					break
				}
			}
		}

		return redis.NewSliceResult(val, nil)
	}

	// mock not found
	val := make([]interface{}, len(keys))
	for i := range keys {
		val[i] = nil
	}

	return redis.NewSliceResult(val, nil)
}

func (c *MockClient) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	c.setWasCalled = true
	c.setKey = key

	if c.shouldError {
		return redis.NewStatusResult("", errors.New("mock error"))
	}

	if err := msgpack.Unmarshal(value.([]byte), &c.feature); err != nil {
		return redis.NewStatusResult("", err)
	}

	return redis.NewStatusResult("", nil)
}

func TestNewManager(t *testing.T) {
	manager := NewManager(&MockClient{}, mockKeyPrefix, false)
	assert.NotNil(t, manager)
}

func TestGet(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	client.feature = Feature{
		name:       "example",
		percentage: 50,
		teamIDs:    intSet{1: struct{}{}, 2: struct{}{}},
	}
	f := NewFeature("example")

	err := manager.get(f)
	assert.NoError(t, err)
	assert.Equal(t, uint8(50), f.percentage)
	assert.Equal(t, struct{}{}, f.teamIDs[1])
	assert.Equal(t, struct{}{}, f.teamIDs[2])
}

func TestActivate(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	f := NewFeature("example")

	// activate a feature
	err := manager.Activate(f)
	assert.NoError(t, err)

	assert.True(t, f.isActive())
	assert.True(t, client.setWasCalled)
	assert.Equal(t, mockKeyPrefix+":example", client.setKey)

	// mock error
	client.setWasCalled = false
	client.shouldError = true
	err = manager.Activate(f)
	assert.EqualError(t, err, "mock error")
}

func TestDeactivate(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	f := NewFeature("example")
	f.activate()

	// deactivate a feature
	err := manager.Deactivate(f)
	assert.NoError(t, err)

	assert.False(t, f.isActive())
	assert.True(t, client.setWasCalled)
	assert.Equal(t, mockKeyPrefix+":example", client.setKey)

	// mock error
	client.setWasCalled = false
	client.shouldError = true
	err = manager.Deactivate(f)
	assert.EqualError(t, err, "mock error")
}

func TestActivatePercentage(t *testing.T) {
	// 75% < Team 1 < 100%
	// 25% < Team 2 < 50%
	// 0 % < Team 3 < 25%

	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, true)

	f := NewFeature("example")

	// activate 25%
	err := manager.ActivatePercentage(f, 25)
	assert.NoError(t, err)

	assert.False(t, f.isTeamActive(1, manager.randomizePercentage))
	assert.False(t, f.isTeamActive(2, manager.randomizePercentage))
	assert.True(t, f.isTeamActive(3, manager.randomizePercentage))
	assert.True(t, client.setWasCalled)
	assert.Equal(t, mockKeyPrefix+":example", client.setKey)

	// activate 50%
	client.setWasCalled = false
	err = manager.ActivatePercentage(f, 50)
	assert.NoError(t, err)

	assert.False(t, f.isTeamActive(1, manager.randomizePercentage))
	assert.True(t, f.isTeamActive(2, manager.randomizePercentage))
	assert.True(t, f.isTeamActive(3, manager.randomizePercentage))
	assert.True(t, client.setWasCalled)
	assert.Equal(t, mockKeyPrefix+":example", client.setKey)

	// mock error
	client.setWasCalled = false
	client.shouldError = true
	err = manager.ActivatePercentage(f, 50)
	assert.EqualError(t, err, "mock error")
}

func TestIsActive(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	f := NewFeature("example")

	// feature not in redis
	active, err := manager.IsActive(f)
	assert.NoError(t, err)
	assert.False(t, active)
	assert.True(t, client.getWasCalled)

	// feature in redis and active
	client.getWasCalled = false
	client.feature = Feature{name: "example", percentage: 100}
	active, err = manager.IsActive(f)
	assert.NoError(t, err)
	assert.True(t, active)
	assert.True(t, client.getWasCalled)

	// feature in redis and inactive
	client.getWasCalled = false
	client.feature = Feature{name: "example", percentage: 50}
	active, err = manager.IsActive(f)
	assert.NoError(t, err)
	assert.False(t, active)
	assert.True(t, client.getWasCalled)

	// mock error
	client.getWasCalled = false
	client.shouldError = true
	active, err = manager.IsActive(f)
	assert.EqualError(t, err, "mock error")
	assert.True(t, client.getWasCalled)
}

func TestIsActiveMulti(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	features := []*Feature{
		NewFeature("example1"),
		NewFeature("example2"),
		NewFeature("example3"),
	}

	// empty features arg
	active, err := manager.IsActiveMulti()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(active))

	// features not in redis
	active, err = manager.IsActiveMulti(features...)
	assert.NoError(t, err)
	assert.Equal(t, len(features), len(active))
	assert.True(t, !active[0] && !active[1] && !active[2])
	assert.True(t, client.mgetWasCalled)

	// some features in redis
	client.mgetWasCalled = false
	features[0].activate()
	client.features = features[:2]
	active, err = manager.IsActiveMulti(features...)
	assert.NoError(t, err)
	assert.Equal(t, len(features), len(active))
	assert.True(t, active[0] && !active[1] && !active[2])
	assert.True(t, client.mgetWasCalled)

	// all features in redis and active
	client.mgetWasCalled = false
	features[0].activate()
	features[1].activate()
	features[2].activate()
	client.features = features
	active, err = manager.IsActiveMulti(features...)
	assert.NoError(t, err)
	assert.Equal(t, len(features), len(active))
	assert.True(t, active[0] && active[1] && active[2])
	assert.True(t, client.mgetWasCalled)

	// mock error
	client.mgetWasCalled = false
	client.shouldError = true
	active, err = manager.IsActiveMulti(features...)
	assert.EqualError(t, err, "mock error")
	assert.True(t, client.mgetWasCalled)
}

func TestActivateTeam(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	f := NewFeature("example")

	// activate a team
	err := manager.ActivateTeam(1, f)
	assert.NoError(t, err)

	assert.True(t, f.isTeamActive(1, manager.randomizePercentage))
	assert.True(t, client.setWasCalled)
	assert.Equal(t, mockKeyPrefix+":example", client.setKey)

	// mock error
	client.setWasCalled = false
	client.shouldError = true
	err = manager.ActivateTeam(1, f)
	assert.EqualError(t, err, "mock error")
}

func TestDeactivateTeam(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	f := NewFeature("example")
	f.activateTeam(1)

	// deactivate a team
	err := manager.DeactivateTeam(1, f)
	assert.NoError(t, err)

	assert.False(t, f.isTeamActive(1, manager.randomizePercentage))
	assert.True(t, client.setWasCalled)
	assert.Equal(t, mockKeyPrefix+":example", client.setKey)

	// mock error
	client.setWasCalled = false
	client.shouldError = true
	err = manager.DeactivateTeam(1, f)
	assert.EqualError(t, err, "mock error")
}

func TestIsTeamActive(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	f := NewFeature("example")

	// feature not in redis
	active, err := manager.IsTeamActive(1, f)
	assert.NoError(t, err)
	assert.False(t, active)
	assert.True(t, client.getWasCalled)

	// feature in redis and globally active
	client.getWasCalled = false
	client.feature = Feature{name: "example", percentage: 100}
	active, err = manager.IsTeamActive(1, f)
	assert.NoError(t, err)
	assert.True(t, active)
	assert.True(t, client.getWasCalled)

	// feature in redis and inactive for team
	client.getWasCalled = false
	client.feature = Feature{name: "example", percentage: 50}
	active, err = manager.IsTeamActive(1, f)
	assert.NoError(t, err)
	assert.False(t, active)
	assert.True(t, client.getWasCalled)

	// feature in redis and active for team by percentage
	client.getWasCalled = false
	client.feature = Feature{name: "example", percentage: 50}
	active, err = manager.IsTeamActive(2, f)
	assert.NoError(t, err)
	assert.True(t, active)
	assert.True(t, client.getWasCalled)

	// feature in redis and active for team explicitly
	client.getWasCalled = false
	client.feature = Feature{name: "example", percentage: 0}
	client.feature.activateTeam(1)
	active, err = manager.IsTeamActive(1, f)
	assert.NoError(t, err)
	assert.True(t, active)
	assert.True(t, client.getWasCalled)

	// mock error
	client.getWasCalled = false
	client.shouldError = true
	active, err = manager.IsTeamActive(1, f)
	assert.EqualError(t, err, "mock error")
	assert.True(t, client.getWasCalled)
}

func TestIsTeamActiveMulti(t *testing.T) {
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	features := []*Feature{
		NewFeature("example1"),
		NewFeature("example2"),
		NewFeature("example3"),
	}

	// empty features arg
	active, err := manager.IsTeamActiveMulti(1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(active))

	// features not in redis
	active, err = manager.IsTeamActiveMulti(1, features...)
	assert.NoError(t, err)
	assert.Equal(t, len(features), len(active))
	assert.True(t, !active[0] && !active[1] && !active[2])
	assert.True(t, client.mgetWasCalled)

	// some features in redis
	client.mgetWasCalled = false
	client.features = features[:2]
	client.features[0].activateTeam(1)
	active, err = manager.IsTeamActiveMulti(1, features...)
	assert.NoError(t, err)
	assert.Equal(t, len(features), len(active))
	assert.True(t, active[0] && !active[1] && !active[2])
	assert.True(t, client.mgetWasCalled)

	// all features in redis and active
	client.mgetWasCalled = false
	features[0].activateTeam(1)
	features[1].activateTeam(1)
	features[2].activateTeam(1)
	client.features = features
	active, err = manager.IsTeamActiveMulti(1, features...)
	assert.NoError(t, err)
	assert.Equal(t, len(features), len(active))
	assert.True(t, active[0] && active[1] && active[2])
	assert.True(t, client.mgetWasCalled)

	// mock error
	client.mgetWasCalled = false
	client.shouldError = true
	active, err = manager.IsTeamActiveMulti(1, features...)
	assert.EqualError(t, err, "mock error")
	assert.True(t, client.mgetWasCalled)
}

func TestFeatureDifferentThanServer(t *testing.T) {
	// mock a scenario where the state of the database is different than the feature variable
	client := &MockClient{}
	manager := NewManager(client, mockKeyPrefix, false)

	client.feature = Feature{
		name:       "example",
		percentage: 50,
		teamIDs:    intSet{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
	}

	active, err := manager.IsTeamActive(4, &Feature{
		name:       "example",
		percentage: 75,
		teamIDs:    intSet{4: struct{}{}},
	})
	assert.NoError(t, err)
	assert.False(t, active)

	active, err = manager.IsTeamActive(2, &Feature{
		name:       "example",
		percentage: 75,
		teamIDs:    intSet{4: struct{}{}},
	})
	assert.NoError(t, err)
	assert.True(t, active)
}
