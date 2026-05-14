package nats

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	DurableCoordinatorBuilds = "coordinator-builds"
	DurableCoordinatorJobs   = "coordinator-jobs"
	DurableWorkers           = "workers"

	FetchTimeout       = 2 * time.Second
	MaxDeliverAttempts = 5
)

var ErrUnmarshaling = errors.New("unmarshaling failed")

// unmarshalMsgIntoEvent десериализует сообщение в тип события(T).
func unmarshalMsgIntoEvent[T any](rawMsg []byte) (T, error) {
	var event T
	if err := json.Unmarshal(rawMsg, &event); err != nil {
		return event, fmt.Errorf("%w: %w", ErrUnmarshaling, err)
	}
	return event, nil
}
