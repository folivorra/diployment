package decode

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrUnmarshaling = errors.New("unmarshaling failed")

// UnmarshalMsgIntoEvent десериализует сообщение в тип события(T).
func UnmarshalMsgIntoEvent[T any](rawMsg []byte) (T, error) {
	var event T
	if err := json.Unmarshal(rawMsg, &event); err != nil {
		return event, fmt.Errorf("%w: %w", ErrUnmarshaling, err)
	}
	return event, nil
}
