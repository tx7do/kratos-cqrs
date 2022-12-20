package server

import (
	"context"
	"fmt"

	"github.com/tx7do/kratos-transport/broker"
	svcV1 "kratos-cqrs/api/logger/service/v1"
)

func SensorCreator() broker.Any     { return &svcV1.Sensor{} }
func SensorDataCreator() broker.Any { return &[]svcV1.SensorData{} }

type SensorHandler func(_ context.Context, topic string, headers broker.Headers, msg *svcV1.Sensor) error
type SensorDataHandler func(_ context.Context, topic string, headers broker.Headers, msg *[]svcV1.SensorData) error

func registerSensorDataHandler(fnc SensorDataHandler) broker.Handler {
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

func registerSensorHandler(fnc SensorHandler) broker.Handler {
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
