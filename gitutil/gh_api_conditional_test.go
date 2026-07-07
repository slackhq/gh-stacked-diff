package gitutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGhApiIncludeResponse_200(t *testing.T) {
	assert := assert.New(t)
	raw := "HTTP/2.0 200 OK\r\nETag: W/\"abc123\"\r\nContent-Type: application/json\r\n\r\n{\"id\": 1}"

	statusCode, etag, err := parseGhApiIncludeResponse(raw)

	assert.NoError(err)
	assert.Equal(200, statusCode)
	assert.Equal("W/\"abc123\"", etag)
}

func TestParseGhApiIncludeResponse_304(t *testing.T) {
	assert := assert.New(t)
	raw := "HTTP/2.0 304 Not Modified\r\nETag: W/\"abc123\"\r\n\r\n"

	statusCode, etag, err := parseGhApiIncludeResponse(raw)

	assert.NoError(err)
	assert.Equal(304, statusCode)
	assert.Equal("W/\"abc123\"", etag)
}

func TestParseGhApiIncludeResponse_UnixLineEndings(t *testing.T) {
	assert := assert.New(t)
	raw := "HTTP/1.1 200 OK\nEtag: \"def456\"\nContent-Type: application/json\n\n{\"state\": \"open\"}"

	statusCode, etag, err := parseGhApiIncludeResponse(raw)

	assert.NoError(err)
	assert.Equal(200, statusCode)
	assert.Equal("\"def456\"", etag)
}

func TestParseGhApiIncludeResponse_NoETag(t *testing.T) {
	assert := assert.New(t)
	raw := "HTTP/2.0 200 OK\r\nContent-Type: application/json\r\n\r\n{}"

	statusCode, etag, err := parseGhApiIncludeResponse(raw)

	assert.NoError(err)
	assert.Equal(200, statusCode)
	assert.Equal("", etag)
}

func TestParseGhApiIncludeResponse_InvalidFormat(t *testing.T) {
	assert := assert.New(t)
	raw := "garbage data with no separator"

	_, _, err := parseGhApiIncludeResponse(raw)

	assert.Error(err)
}
