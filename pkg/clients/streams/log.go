package streams

import (
	"bufio"
	"context"
	"fmt"
	"io"
)

// Istream interface can read from multiple io.readCloser's
// and write to a common output channel
type Istream interface {
	// AddStream adds a new io.ReadCloser. When writing to the common channel,
	// each line read from this readCloser will be prefixed with the prefix
	AddStream(prefix string, reader io.ReadCloser)
	
	// StartStream concurrently reads all the configured io.ReadClosers. It returns
	// a pointer to a Stream object
	StartStream() *Stream
}

// Stream consists of the common output log channel, its corresponding error channel
// and a cancel function.
type Stream struct {
	// Input readClosers and their prefix
	inStreams    map[*string]io.ReadCloser // prefix <-> reader(log)
	// Combined logs sent to this channel
	OutputStream chan []byte
	Err          chan error
	// To cancel a stream that has started
	Cancel       context.CancelFunc
}

// NewLogStreamI returns an Istream interface can read from multiple io.readCloser's
// and write to a common output channel
func NewLogStreamI() Istream {
	return &Stream{
		inStreams:    make(map[*string]io.ReadCloser),
		OutputStream: make(chan []byte),
		Err:          make(chan error),
		Cancel: nil,
	}
}

// AddStream adds a new io.ReadCloser to the current list. When writing to
// to the common channel, each line read from this readCloser will be
// prefixed with the prefix
func (s *Stream) AddStream(prefix string, reader io.ReadCloser) {
	if prefix != "" && reader != nil{
		s.inStreams[&prefix] = reader
	}
}
// StartStream concurrently reads all the configured io.ReadClosers. It returns
// a pointer to Stream.
func (s *Stream) StartStream() *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	defer ctx.Done()
	s.Cancel = cancel
	for prefix, logReader := range s.inStreams {
		go s.readPrefixWrite(ctx, *prefix, logReader)
	}
	return s
}
// readPrefixWrite reads from a single io.ReadCloser. For each line read,
// it adds the prefix and writes to the common log channel.
func (s *Stream) readPrefixWrite(ctx context.Context, prefix string, logReader io.ReadCloser) {
	defer func(logReader io.ReadCloser) {
		_ = logReader.Close()
	}(logReader)
	scanner := bufio.NewScanner(logReader)
	for {
		select {
		case <- ctx.Done():
			fmt.Println("stopped readPrefixWrite for", prefix)
			return
		default :
			for scanner.Scan(){
				log := "[" + prefix + "] " + scanner.Text()
				buffer := make([]byte, len(log))
				copy(buffer, log)
				s.OutputStream <- buffer
			}
			s.Err <- scanner.Err()
		}
	}
}

