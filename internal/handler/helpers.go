package handler

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

func uuidToString(b [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseUUID(s string) (pgtype.UUID, error) {
	var uuid pgtype.UUID

	// Remove hyphens and parse
	clean := ""
	for _, c := range s {
		if c != '-' {
			clean += string(c)
		}
	}

	if len(clean) != 32 {
		return uuid, fmt.Errorf("invalid uuid length")
	}

	var bytes [16]byte
	for i := 0; i < 16; i++ {
		var b byte
		_, err := fmt.Sscanf(clean[i*2:i*2+2], "%02x", &b)
		if err != nil {
			return uuid, err
		}
		bytes[i] = b
	}

	uuid.Bytes = bytes
	uuid.Valid = true
	return uuid, nil
}
