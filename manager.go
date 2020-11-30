package rollout

import (
	"fmt"

	redis "github.com/go-redis/redis/v7"
	"github.com/vmihailenco/msgpack/v4"
)

// Manager persists and fetches feature toggles to/from redis
type Manager struct {
	client              redis.Cmdable
	keyPrefix           string
	randomizePercentage bool
}

// NewManager constructs a new Manager instance
func NewManager(client redis.Cmdable, keyPrefix string, randomizePercentage bool) *Manager {
	// nothing is retrieved from redis at this point
	// everything is fetched on demand
	return &Manager{
		client:              client,
		keyPrefix:           keyPrefix,
		randomizePercentage: randomizePercentage,
	}
}

func (m *Manager) keyName(feature *Feature) string {
	return m.keyPrefix + ":" + feature.Name()
}

// get updates the Feature to align with the current value in redis
func (m *Manager) get(feature *Feature) error {
	// retrieve feature from redis
	data, err := m.client.Get(m.keyName(feature)).Bytes()
	if err != nil {
		if err == redis.Nil {
			// feature isn't in redis, so should be inactive
			feature.Lock()
			feature.deactivate()
			feature.Unlock()
			return nil
		}
		return err
	}

	feature.Lock()
	defer feature.Unlock()
	if err := msgpack.Unmarshal(data, feature); err != nil {
		return err
	}

	return nil
}

// Activate globally activates the feature
func (m *Manager) Activate(feature *Feature) error {
	if err := m.get(feature); err != nil {
		return err
	}

	feature.Lock()
	feature.activate()

	data, err := msgpack.Marshal(feature)
	if err != nil {
		return err
	}
	feature.Unlock()

	if err := m.client.Set(m.keyName(feature), data, 0).Err(); err != nil {
		return err
	}

	return nil
}

// Deactivate globally deactivates the feature
func (m *Manager) Deactivate(feature *Feature) error {
	if err := m.get(feature); err != nil {
		return err
	}

	feature.Lock()
	feature.deactivate()

	data, err := msgpack.Marshal(feature)
	if err != nil {
		return err
	}
	feature.Unlock()

	if err := m.client.Set(m.keyName(feature), data, 0).Err(); err != nil {
		return err
	}

	return nil
}

// ActivatePercentage activates the feature for a percentage of teams
func (m *Manager) ActivatePercentage(feature *Feature, percentage uint8) error {
	if err := m.get(feature); err != nil {
		return err
	}

	feature.Lock()
	feature.activatePercentage(percentage)

	data, err := msgpack.Marshal(feature)
	if err != nil {
		return err
	}
	feature.Unlock()

	if err := m.client.Set(m.keyName(feature), data, 0).Err(); err != nil {
		return err
	}

	return nil
}

// IsActive returns whether the given feature is globally active
func (m *Manager) IsActive(feature *Feature) (bool, error) {
	// retrieve feature from redis
	data, err := m.client.Get(m.keyName(feature)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	feature.Lock()
	defer feature.Unlock()
	if err := msgpack.Unmarshal(data, feature); err != nil {
		return false, err
	}

	return feature.isActive(), nil
}

// IsActiveMulti returns whether the given features are globally active
func (m *Manager) IsActiveMulti(features ...*Feature) ([]bool, error) {
	if len(features) == 0 {
		return nil, nil
	}

	featureNames := make([]string, len(features))
	for i, feature := range features {
		featureNames[i] = m.keyName(feature)
	}

	// retrieve features from redis
	val, err := m.client.MGet(featureNames...).Result()
	if err != nil {
		return nil, err
	}

	for _, feature := range features {
		feature.Lock()
		defer feature.Unlock()
	}

	results := make([]bool, len(features))

	for i, v := range val {
		switch t := v.(type) {
		case nil:
			// feature wasn't found in redis, so considered inactive globally
			features[i].deactivate()

		case string:
			if err := msgpack.Unmarshal([]byte(t), features[i]); err != nil {
				return nil, err
			}
			results[i] = features[i].isActive()

		default:
			return nil, fmt.Errorf("unexpected type (%T) for msgpack value: %v", v, v)
		}
	}

	return results, nil
}

// ActivateTeam activates the feature for specific team
func (m *Manager) ActivateTeam(teamID int64, feature *Feature) error {
	if err := m.get(feature); err != nil {
		return err
	}

	feature.Lock()
	feature.activateTeam(teamID)

	data, err := msgpack.Marshal(feature)
	if err != nil {
		return err
	}
	feature.Unlock()

	if err := m.client.Set(m.keyName(feature), data, 0).Err(); err != nil {
		return err
	}

	return nil
}

// DeactivateTeam deactivates the feature for specific team
func (m *Manager) DeactivateTeam(teamID int64, feature *Feature) error {
	if err := m.get(feature); err != nil {
		return err
	}

	feature.Lock()
	feature.deactivateTeam(teamID)

	data, err := msgpack.Marshal(feature)
	if err != nil {
		return err
	}
	feature.Unlock()

	if err := m.client.Set(m.keyName(feature), data, 0).Err(); err != nil {
		return err
	}

	return nil
}

// IsTeamActive returns whether the given feature is active for a team
func (m *Manager) IsTeamActive(teamID int64, feature *Feature) (bool, error) {
	// retrieve feature from redis
	data, err := m.client.Get(m.keyName(feature)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	feature.Lock()
	defer feature.Unlock()
	if err := msgpack.Unmarshal(data, feature); err != nil {
		return false, err
	}

	return feature.isTeamActive(teamID, m.randomizePercentage), nil
}

// IsTeamActiveMulti returns whether the given features are globally active
func (m *Manager) IsTeamActiveMulti(teamID int64, features ...*Feature) ([]bool, error) {
	if len(features) == 0 {
		return nil, nil
	}

	featureNames := make([]string, len(features))
	for i, feature := range features {
		featureNames[i] = m.keyName(feature)
	}

	// retrieve features from redis
	val, err := m.client.MGet(featureNames...).Result()
	if err != nil {
		return nil, err
	}

	for _, feature := range features {
		feature.Lock()
		defer feature.Unlock()
	}

	results := make([]bool, len(features))

	for i, v := range val {
		switch t := v.(type) {
		case nil:
			// feature wasn't found in redis, so considered inactive globally
			features[i].deactivate()

		case string:
			if err := msgpack.Unmarshal([]byte(t), features[i]); err != nil {
				return nil, err
			}
			results[i] = features[i].isTeamActive(teamID, m.randomizePercentage)

		default:
			return nil, fmt.Errorf("unexpected type (%T) for msgpack value: %v", v, v)
		}
	}

	return results, nil
}
