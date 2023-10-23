package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oodzchen/dproject/model"
	"github.com/redis/go-redis/v9"
)

type SettingsManager struct {
	Rdb      *redis.Client
	LifeTime time.Duration
}

var DefaultSettingsLifeTime = 30 * 24 * time.Hour

func genSettingsKey(id string) string {
	return fmt.Sprintf("%s_%s", "ui_settings_", id)
}

func (sm *SettingsManager) SaveSettings(uuid string, val *model.UISettings) error {
	jsonStr, err := json.Marshal(val)
	if err != nil {
		return err
	}

	// fmt.Println("json str: ", jsonStr)

	return sm.Rdb.Set(context.Background(), genSettingsKey(uuid), jsonStr, sm.LifeTime).Err()
}

func (sm *SettingsManager) GetSettings(uuid string) (*model.UISettings, error) {
	str, err := sm.Rdb.Get(context.Background(), genSettingsKey(uuid)).Result()
	if err != nil {
		return nil, err
	}
	settings := &model.UISettings{}
	err = json.Unmarshal([]byte(str), &settings)
	if err != nil {
		return nil, err
	}

	return settings, nil
}
