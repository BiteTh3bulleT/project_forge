package ephemeral

import "fmt"

const (
	maxEphemeralValueBytes          = 1 << 20
	maxEphemeralProgressReadEntries = 4096
)

func validateEphemeralValueBytes(value []byte) error {
	if len(value) > maxEphemeralValueBytes {
		return fmt.Errorf("%w: %d > %d bytes", ErrValueTooLarge, len(value), maxEphemeralValueBytes)
	}
	return nil
}

func validateEphemeralValueString(value string) error {
	if len(value) > maxEphemeralValueBytes {
		return fmt.Errorf("%w: %d > %d bytes", ErrValueTooLarge, len(value), maxEphemeralValueBytes)
	}
	return nil
}

func validateProgressEntryValue(entry ProgressEntry) error {
	if err := validateEphemeralValueString(entry.ID); err != nil {
		return err
	}
	if err := validateEphemeralValueString(entry.Message); err != nil {
		return err
	}
	return nil
}

func normalizeProgressReadLimit(limit int) int {
	if limit <= 0 || limit > maxEphemeralProgressReadEntries {
		return maxEphemeralProgressReadEntries
	}
	return limit
}
