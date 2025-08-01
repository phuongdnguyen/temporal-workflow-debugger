package daptest

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/go-dap"

	"custom-debugger/pkg/utils"
)

// GetThreadsResponse reads a protocol message from the connection
func (c *Client) GetThreadsResponse() (*dap.ThreadsResponse, error) {
	m, err := dap.ReadProtocolMessage(c.reader)
	if err != nil {
		return nil, err
	}
	r, ok := m.(*dap.ThreadsResponse)
	if !ok {
		return nil, fmt.Errorf("got %#v, want *dap.ThreadsResponse", m)
	}
	return r, nil
}

// GetStacktraceResponse reads a protocol message from the connection
func (c *Client) GetStacktraceResponse() (*dap.StackTraceResponse, error) {
	m, err := dap.ReadProtocolMessage(c.reader)
	if err != nil {
		return nil, err
	}
	r, ok := m.(*dap.StackTraceResponse)
	if !ok {
		return nil, fmt.Errorf("got %#v, want *dap.StackTraceResponse", m)
	}
	return r, nil
}

// GetNextResponse reads a protocol message from the connection
func (c *Client) GetNextResponse() (*dap.NextResponse, error) {
	m, err := dap.ReadProtocolMessage(c.reader)
	if err != nil {
		return nil, err
	}
	r, ok := m.(*dap.NextResponse)
	if !ok {
		return nil, fmt.Errorf("got %#v, want *dap.NextResponse", m)
	}
	return r, nil
}

func (c *Client) GetNextResponseWithFiltering() (resp *dap.NextResponse, remaining []byte, err error) {
	for {
		m, err := dap.ReadProtocolMessage(c.reader)
		if err != nil {
			return nil, nil, err
		}

		switch msg := m.(type) {
		case *dap.NextResponse:
			log.Printf("Client.GetNextResponseWithFiltering found next response: %+v\n", msg)
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, nil, err
			}
			remaining = append(remaining, utils.BuildDAPMessage(b)...)
			return msg, remaining, nil
		default:
			// Buffer other messages and continue waiting
			log.Printf("Client.GetNextResponseWithFiltering buffering message type %T while waiting for NextResponse"+
				"\n", msg)
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, nil, fmt.Errorf("could not marshal buffering message in Client.NextResponse: %w", err)
			}
			remaining = append(remaining, utils.BuildDAPMessage(b)...)
		}
	}
}

func (c *Client) GetThreadsResponseWithFiltering() (resp *dap.ThreadsResponse, remaining []byte, err error) {
	for {
		m, err := dap.ReadProtocolMessage(c.reader)
		if err != nil {
			return nil, nil, err
		}
		switch msg := m.(type) {
		case *dap.ThreadsResponse:
			log.Printf("Client.GetThreadsResponseWithFiltering found threads: %+v\n", msg)
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, nil, err
			}
			remaining = append(remaining, utils.BuildDAPMessage(b)...)
			return msg, remaining, nil
		default:
			log.Printf("Client.GetThreadsResponseWithFiltering buffering message type %T", msg)
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, nil, fmt.Errorf("could not marshal buffering message in Client.ThreadsResponse: %w", err)
			}
			remaining = append(remaining, utils.BuildDAPMessage(b)...)
		}
	}
}

func (c *Client) GetStacktraceResponseWithFiltering() (resp *dap.StackTraceResponse, remaining []byte, err error) {
	for {
		m, err := dap.ReadProtocolMessage(c.reader)
		if err != nil {
			return nil, nil, err
		}
		switch msg := m.(type) {
		case *dap.StackTraceResponse:
			log.Printf("Client.GetStacktraceResponseWithFiltering found stacktrace: %+v\n", msg)
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, nil, err
			}
			remaining = append(remaining, utils.BuildDAPMessage(b)...)
			return msg, remaining, nil
		default:
			log.Printf("Client.GetStacktraceResponseWithFiltering buffering message type %T\n", msg)
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, nil, fmt.Errorf("could not marshal buffering message in Client.StacktraceResponse: %w", err)
			}
			remaining = append(remaining, utils.BuildDAPMessage(b)...)
		}
	}
}
