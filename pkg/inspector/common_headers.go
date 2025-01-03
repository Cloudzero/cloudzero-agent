package inspector

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

func init() {
	for _, v := range listOfHeaderExactToLog {
		headerExactToLog[v] = true
	}
}

var listOfHeaderPrefixesToLog = []string{
	"x-amz-",
	"x-cloudzero-",
}

var listOfHeaderContainsToLog = []string{
	"request-id",
	"trace-id",
}

// This is a little ugly because Go doesn't support static maps. We have to
// build them up in the init function.
var headerExactToLog = map[string]bool{}
var listOfHeaderExactToLog = []string{
	"content-type",
}

// addCommonHeaders adds common headers, such as request-id, trace-id, etc., to the logger.
func addCommonHeaders(logger zerolog.Logger, headers http.Header) zerolog.Logger {
	for k, v := range headers {
		lowerK := strings.ToLower(k)

		for _, keyPrefix := range listOfHeaderPrefixesToLog {
			if strings.HasPrefix(lowerK, keyPrefix) {
				logger = logger.With().Str(k, strings.Join(v, ", ")).Logger()
			}
		}
		for _, keyContains := range listOfHeaderContainsToLog {
			if strings.Contains(lowerK, keyContains) {
				logger = logger.With().Str(k, strings.Join(v, ", ")).Logger()
			}
		}
		if _, keyMatches := headerExactToLog[lowerK]; keyMatches {
			logger = logger.With().Str(k, strings.Join(v, ", ")).Logger()
		}
	}

	return logger
}
