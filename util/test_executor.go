package util

import (
	"fmt"
	"log/slog"
	"slices"
)

type fakeResponse struct {
	out     string // Out to return.
	err     error  // error to return.
	isMatch func(programName string, args ...string) bool
}

// Response that was executed or will be faked.
type ExecutedResponse struct {
	Out         string   // Out to return.
	Err         error    // error to return.
	ProgramName string   // Program name to match or that was used.
	Args        []string // Arguments that were used.
	Faked       bool     // Whether this response was faked.
}

// Fake [Executor] for testing.
type TestExecutor struct {
	fakeResponses []fakeResponse
	Responses     []ExecutedResponse
}

// Can be used use as last value of [TestExecutor.fakeResponses] [ExecuteResponse.Args]
const MatchAnyRemainingArgs = "MatchCommandWithAnyRemainingArgs"

// Ensure that [TestExecutor] implements [Executor].
var _ Executor = &TestExecutor{}

// Checks [TestExecutor.fakeResponses] for any match before calling [DefaultExecutor.Execute].
func (t *TestExecutor) Execute(options ExecuteOptions, programName string, args ...string) (string, error) {
	for _, response := range slices.Backward(t.fakeResponses) {
		if response.isMatch(programName, args...) {
			executedResponse := ExecutedResponse{
				Out:         response.out,
				Err:         response.err,
				ProgramName: programName,
				Args:        args,
				Faked:       true,
			}
			t.Responses = append(t.Responses, executedResponse)
			slog.Debug(fmt.Sprint("Faked ", executedResponse))
			if options.Io.Out != nil {
				if _, err := options.Io.Out.Write([]byte(response.out)); err != nil {
					panic(err)
				}
			}
			return response.out, response.err
		}
	}
	out, err := (&DefaultExecutor{}).Execute(options, programName, args...)
	t.Responses = append(t.Responses, ExecutedResponse{Out: out, Err: err, ProgramName: programName, Args: args})
	return out, err
}

// Adds a response to [TestExecutor.fakeResponses].
// If [fakeArgs] ends with [MatchAnyRemainingArgs], then the last argument is treated as a wildcard
// for any remaining args.
// Response are checked in the reverse order that they are added (in other words, the most recent
// response that was added is checked first)
func (t *TestExecutor) SetResponse(out string, err error, fakeProgramName string, fakeArgs ...string) {
	isMatch := func(actualProgramName string, actualArgs ...string) bool {
		if fakeProgramName != actualProgramName {
			return false
		}
		var matchFakeArgs []string
		var matchActualArgs []string
		if len(fakeArgs) > 0 &&
			fakeArgs[len(fakeArgs)-1] == MatchAnyRemainingArgs &&
			len(actualArgs) >= len(fakeArgs)-1 {
			matchFakeArgs = fakeArgs[0 : len(fakeArgs)-1]
			matchActualArgs = actualArgs[0 : len(fakeArgs)-1]
		} else {
			matchFakeArgs = fakeArgs
			matchActualArgs = actualArgs
		}
		return slices.Compare(matchFakeArgs, matchActualArgs) == 0
	}
	t.fakeResponses = append(t.fakeResponses, fakeResponse{out: out, err: err, isMatch: isMatch})
}

// Adds a response to [TestExecutor.fakeResponses].
func (t *TestExecutor) SetResponseFunc(out string, err error, isMatch func(programName string, args ...string) bool) {
	t.fakeResponses = append(t.fakeResponses, fakeResponse{out: out, err: err, isMatch: isMatch})
}
