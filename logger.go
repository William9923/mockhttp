package mockhttp

import (
	"fmt"
	"net/http"
)

// Logger interface allows to use other loggers than
// standard log.Logger.
type Logger interface {
	Printf(string, ...interface{})
}

// LeveledLogger is an interface that can be implemented by any logger or a
// logger wrapper to provide leveled logging. The methods accept a message
// string and a variadic number of key-value pairs. For log.Printf style
// formatting where message string contains a format specifier, use Logger
// interface.
type LeveledLogger interface {
	Error(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
}

// hookLogger adapts an LeveledLogger to Logger for use by the existing hook functions
// without changing the API.
type hookLogger struct {
	LeveledLogger
}

func (h hookLogger) Printf(s string, args ...interface{}) {
	h.Info(fmt.Sprintf(s, args...))
}

// RequestLogHook allows a function to run before http call executed.
// The HTTP request which will be made.
type RequestLogHook func(Logger, *http.Request)

// ResponseLogHook is like RequestLogHook, but allows running a function
// on each HTTP response. This function will be invoked at the end of
// every HTTP request executed, regardless of whether a subsequent retry
// needs to be performed or not. If the response body is read or closed
// from this method, this will affect the response returned from Do().
type ResponseLogHook func(Logger, *http.Response)
