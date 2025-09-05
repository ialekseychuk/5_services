package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

type ServiceConfig struct {
	ID            string        `env:"SERVICE_ID"`
	Port          int           `env:"SERVICE_PORT"`
	PollingPeriod time.Duration `env:"POLLING_PERIOD_SECONDS"`
	MessagePeriod time.Duration `env:"MESSAGE_PERIOD_SECONDS"`
}

func LoadConfig() (*ServiceConfig, error) {
	cfg := &ServiceConfig{}
	cfgVal := reflect.ValueOf(cfg).Elem()
	cfgType := cfgVal.Type()

	for i := 0; i < cfgVal.NumField(); i++ {
		field := cfgVal.Field(i)
		tag := cfgType.Field(i).Tag.Get("env")

		if tag == "" {
			fmt.Println("no tag")
			continue
		}

		envVal := os.Getenv(tag)

		if envVal == "" {
			return cfg, fmt.Errorf("environment variable %s is not set", tag)
		}

		switch field.Kind() {
		case reflect.String:
			cfgVal.Field(i).SetString(envVal)
		case reflect.Int:
			intVal, err := strconv.Atoi(envVal)
			if err != nil {
				return cfg, fmt.Errorf("environment variable %s is not a valid integer", tag)
			}
			cfgVal.Field(i).SetInt(int64(intVal))
		case reflect.Struct:
			if field.Type() == reflect.TypeOf(time.Duration(0)) {
				durVal, err := time.ParseDuration(envVal)
				if err != nil {
					return cfg, fmt.Errorf("environment variable %s is not a valid duration", tag)
				}
				cfgVal.Field(i).Set(reflect.ValueOf(durVal))
			}
		
		default:
			return cfg, fmt.Errorf("environment variable %s is not a valid type %v", tag, field.Type())
		}
	}
	return cfg, nil
}
