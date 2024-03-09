package cluster

import (
	"regexp"

	"github.com/rs/zerolog/log"
)

type logParser struct {
	regexp *regexp.Regexp
}

func newLogParser() *logParser {
	return &logParser{
		regexp: regexp.MustCompile(`(.*)\[(DEBUG|ERR|ERROR|INFO|WARNING|WARN)](.*)`),
	}
}

func (l *logParser) Write(in []byte) (int, error) {
	res := l.regexp.FindSubmatch(in)
	if len(res) != 4 {
		log.Warn().Msgf("unable to parse memberlist log message: %s", in)
	}

	switch string(res[2]) {
	case "ERR", "ERROR":
		log.Error().Msg(string(res[3]))
	case "WARN", "WARNING":
		log.Warn().Msg(string(res[3]))
	case "DEBUG":
		log.Debug().Msg(string(res[3]))
	case "INFO":
		log.Info().Msg(string(res[3]))
	default:
		log.Warn().Msgf("unable to parse memberlist log level from message: %s", in)
	}

	return len(in), nil
}
