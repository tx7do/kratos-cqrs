package server

import (
	"context"
	"fmt"

	"github.com/tx7do/kratos-transport/broker"
	svcV1 "kratos-cqrs/api/logger/service/v1"
)

func sensorCreator() broker.Any     { return &svcV1.Sensor{} }
func sensorDataCreator() broker.Any { return &[]svcV1.SensorData{} }

type sensorHandler func(_ context.Context, topic string, headers broker.Headers, msg *svcV1.Sensor) error
type sensorDataHandler func(_ context.Context, topic string, headers broker.Headers, msg *[]svcV1.SensorData) error

func registerSensorDataHandler(fnc sensorDataHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		switch t := event.Message().Body.(type) {
		case *[]svcV1.SensorData:
			if err := fnc(ctx, event.Topic(), event.Message().Headers, t); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}
		return nil
	}
}

func registerSensorHandler(fnc sensorHandler) broker.Handler {
	return func(ctx context.Context, event broker.Event) error {
		switch t := event.Message().Body.(type) {
		case *svcV1.Sensor:
			if err := fnc(ctx, event.Topic(), event.Message().Headers, t); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type: %T", t)
		}
		return nil
	}
}
