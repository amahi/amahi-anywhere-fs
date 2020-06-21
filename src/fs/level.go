package main


// log level
type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelError
	LevelWarn
	LevelFatal
	Access
)

func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelFatal:
		return "FATAL"
	case Access:
		return "ACCESS"
	default:
		return "DEBUG"
	}

}


