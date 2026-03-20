package testutil

import (
	"fmt"
	"strings"
)

// MockExecutor records calls and returns pre-programmed outputs/errors.
type MockExecutor struct {
	Calls   [][]string
	Outputs map[string][]byte
	Errors  map[string]error
}

// NewMockExecutor creates a MockExecutor with empty maps.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Outputs: make(map[string][]byte),
		Errors:  make(map[string]error),
	}
}

func key(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

// Run records the call and returns any pre-programmed error.
func (m *MockExecutor) Run(name string, args ...string) error {
	m.Calls = append(m.Calls, append([]string{name}, args...))
	if err, ok := m.Errors[key(name, args)]; ok {
		return err
	}
	return nil
}

// Output records the call and returns any pre-programmed output or error.
func (m *MockExecutor) Output(name string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, append([]string{name}, args...))
	k := key(name, args)
	if err, ok := m.Errors[k]; ok {
		return nil, err
	}
	if out, ok := m.Outputs[k]; ok {
		return out, nil
	}
	return nil, nil
}

// SetOutput pre-programs an output for a command invocation.
func (m *MockExecutor) SetOutput(output []byte, name string, args ...string) {
	m.Outputs[key(name, args)] = output
}

// SetError pre-programs an error for a command invocation.
func (m *MockExecutor) SetError(err error, name string, args ...string) {
	m.Errors[key(name, args)] = err
}

// WasCalled returns true if the given command was called at least once.
func (m *MockExecutor) WasCalled(name string, args ...string) bool {
	target := append([]string{name}, args...)
	for _, call := range m.Calls {
		if len(call) == len(target) {
			match := true
			for i := range call {
				if call[i] != target[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

// CallCount returns how many times any command was called.
func (m *MockExecutor) CallCount() int {
	return len(m.Calls)
}

// AssertCalled fails the test if the command was not called.
func (m *MockExecutor) AssertCalled(t interface {
	Helper()
	Errorf(string, ...any)
}, name string, args ...string) {
	t.Helper()
	if !m.WasCalled(name, args...) {
		t.Errorf("expected call %v but it was not made; calls were: %v", append([]string{name}, args...), m.Calls)
	}
}

// AssertNotCalled fails the test if the command was called.
func (m *MockExecutor) AssertNotCalled(t interface {
	Helper()
	Errorf(string, ...any)
}, name string, args ...string) {
	t.Helper()
	if m.WasCalled(name, args...) {
		t.Errorf("expected %v NOT to be called, but it was", append([]string{name}, args...))
	}
}

// MockHTTPGet returns a function that serves pre-programmed byte responses.
// Keys are URLs; values are raw response bodies (nil means return ErrCommand).
func MockHTTPGet(responses map[string][]byte) func(string) (*MockHTTPResponse, error) {
	return func(url string) (*MockHTTPResponse, error) {
		body, ok := responses[url]
		if !ok {
			return nil, fmt.Errorf("no mock response for URL %q", url)
		}
		return &MockHTTPResponse{Body: body}, nil
	}
}

// MockHTTPResponse is a minimal stand-in for *http.Response used in tests.
type MockHTTPResponse struct {
	Body       []byte
	StatusCode int
}

// ErrCommand is a sentinel error for simulating command failures in tests.
var ErrCommand = fmt.Errorf("mock command error")
