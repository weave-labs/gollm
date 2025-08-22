package llm

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/weave-labs/gollm/providers"
	"io"
	"time"
)

// StreamToken represents a single token from the streaming response.
type StreamToken struct {
	Metadata     map[string]any
	Text         string
	Type         string
	Index        int
	InputTokens  int64
	OutputTokens int64
}

// TokenStream represents a stream of tokens from the LLM.
// It follows Go's io.ReadCloser pattern but with token-level granularity.
type TokenStream interface {
	// Next returns the next token in the stream.
	// When the stream is finished, it returns io.EOF.
	Next(ctx context.Context) (*StreamToken, error)

	// Closer Close releases any resources associated with the stream.
	io.Closer
}

// SSEDecoder handles Server-Sent Events (SSE) streaming
type SSEDecoder struct {
	err     error
	reader  *bufio.Scanner
	current Event
}

type Event struct {
	Type string
	Data []byte
}

func NewSSEDecoder(reader io.Reader) *SSEDecoder {
	return &SSEDecoder{
		reader: bufio.NewScanner(reader),
	}
}

func (d *SSEDecoder) Next() bool {
	if d.err != nil {
		return false
	}

	event := ""
	data := bytes.NewBuffer(nil)

	for d.reader.Scan() {
		line := d.reader.Bytes()

		// Dispatch event on empty line
		if len(line) == 0 {
			d.current = Event{
				Type: event,
				Data: data.Bytes(),
			}
			return true
		}

		// Split "event: value" into parts
		name, value, _ := bytes.Cut(line, []byte(":"))

		// Remove optional space after colon
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		switch string(name) {
		case "":
			continue // Skip comments
		case "event":
			event = string(value)
		case "data":
			data.Write(value)
			data.WriteByte('\n')
		}
	}

	return false
}

func (d *SSEDecoder) Event() Event {
	return d.current
}

func (d *SSEDecoder) Err() error {
	return d.err
}

// providerStream implements TokenStream for a specific provider
type providerStream struct {
	provider      providers.Provider
	retryStrategy RetryStrategy
	decoder       *SSEDecoder
	config        *GenerateConfig
	buffer        []byte
	currentIndex  int
}

func newProviderStream(reader io.ReadCloser, provider providers.Provider, cfg *GenerateConfig) *providerStream {
	return &providerStream{
		decoder:       NewSSEDecoder(reader),
		provider:      provider,
		config:        cfg,
		buffer:        make([]byte, 0, DefaultStreamBufferSize),
		currentIndex:  0,
		retryStrategy: cfg.RetryStrategy,
	}
}

func (s *providerStream) Next(ctx context.Context) (*StreamToken, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled: %w", ctx.Err())
		default:
			token, shouldContinue, err := s.processNextEvent()
			if err != nil {
				return nil, err
			}
			if shouldContinue {
				continue
			}
			return token, nil
		}
	}
}

// processNextEvent handles the next event from the decoder
func (s *providerStream) processNextEvent() (*StreamToken, bool, error) {
	if !s.decoder.Next() {
		return s.handleDecoderEnd()
	}

	event := s.decoder.Event()
	if len(event.Data) == 0 {
		return nil, true, nil // continue
	}

	return s.processEventData(event)
}

// handleDecoderEnd handles the case when decoder has no more events
func (s *providerStream) handleDecoderEnd() (*StreamToken, bool, error) {
	if err := s.decoder.Err(); err != nil {
		if s.retryStrategy.ShouldRetry(err) {
			time.Sleep(s.retryStrategy.NextDelay())
			return nil, true, nil // continue
		}
		return nil, false, err
	}
	return nil, false, io.EOF
}

// processEventData processes the event data and creates a stream token
func (s *providerStream) processEventData(event Event) (*StreamToken, bool, error) {
	resp, err := s.provider.ParseStreamResponse(event.Data)
	if err != nil {
		if err.Error() == "skip resp" {
			return nil, true, nil // continue
		}
		if errors.Is(err, io.EOF) {
			return nil, false, io.EOF
		}
		return nil, true, nil // continue - Not enough data or malformed
	}

	return s.createStreamToken(event, resp), false, nil
}

// createStreamToken creates a stream token from the response
func (s *providerStream) createStreamToken(event Event, resp *providers.Response) *StreamToken {
	streamToken := &StreamToken{
		Text:  "",
		Type:  event.Type,
		Index: s.currentIndex,
	}

	if resp == nil {
		return streamToken
	}

	if resp.Content != nil {
		streamToken.Text = resp.AsText()
	}

	if resp.Usage != nil {
		streamToken.InputTokens = resp.Usage.InputTokens
		streamToken.OutputTokens = resp.Usage.OutputTokens
	}

	return streamToken
}

func (s *providerStream) Close() error {
	return nil
}
