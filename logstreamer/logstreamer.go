package logstreamer

import (
	"bytes"
	"io"
	"sync"
)

type Logstreamer struct {
	writer io.Writer
	mu     *sync.Mutex
	buf    *bytes.Buffer
	prefix string
}

type NewLogstreamerInput struct {
	Writer        io.Writer
	Mutex         *sync.Mutex
	DisablePrefix bool
}

func NewLogstreamer(input NewLogstreamerInput) *Logstreamer {
	prefix := "       "
	if input.DisablePrefix {
		prefix = ""
	}
	streamer := &Logstreamer{
		writer: input.Writer,
		mu:     input.Mutex,
		buf:    bytes.NewBuffer([]byte("")),
		prefix: prefix,
	}

	return streamer
}

func (l *Logstreamer) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.OutputLines()
	return
}

func (l *Logstreamer) Close() error {
	l.Flush()
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

func (l *Logstreamer) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	return l.out(string(p))
}

func (l *Logstreamer) OutputLines() (err error) {
	for {
		line, err := l.buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := l.out(line); err != nil {
			return err
		}
	}

	return nil
}

func (l *Logstreamer) out(str string) (err error) {
	str = l.prefix + str

	_, err = l.writer.Write([]byte(str))

	return err
}
