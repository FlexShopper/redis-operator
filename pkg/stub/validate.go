package stub

import (
	"fmt"
	"github.com/flexshopper/redis-operator/pkg/apis/cache/v1alpha1"
	"regexp"
	"strconv"
	"strings"
	"errors"
)

// Effective Go is your friend
// https://golang.org/doc/effective_go.html#constants
const (
	maxMemory = "5gb"
	BYTE = 1 << (10 * iota)
	KILOBYTE
	MEGABYTE
	GIGABYTE
	TERABYTE
)

func convertMemoryToBytes(memory string) (int64, error) {
	r, _ := regexp.Compile("(\\d+)(\\w+)")

	matches := r.FindStringSubmatch(memory)
	amount, _ := strconv.ParseInt(matches[1], 10, 64)

	switch unit := strings.ToLower(matches[2]); unit {
	case "b":
		return int64(amount * BYTE), nil
	case "kb":
		return int64(amount * KILOBYTE), nil
	case "mb":
		return int64(amount * MEGABYTE), nil
	case "gb":
		return int64(amount * GIGABYTE), nil
	case "tb":
		return int64(amount * TERABYTE), nil
	default:
		return 0, errors.New("unsupported type")
	}
}

func validate(redis *v1alpha1.Redis) []string {

	var validationErrors []string

	maxMemoryBytes, _ := convertMemoryToBytes(maxMemory)
	requestedMemoryBytes, _ := convertMemoryToBytes(redis.Spec.MaxMemory)

	if requestedMemoryBytes > maxMemoryBytes {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf("maxMemory setting ( %s ) greater than allowed maxMemory ( %s )",
				maxMemory,
				redis.Spec.MaxMemory))
	}

	return validationErrors
}
