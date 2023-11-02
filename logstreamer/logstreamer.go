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

func NewLogstreamer(out io.Writer, mu *sync.Mutex) *Logstreamer {
	streamer := &Logstreamer{
		writer: out,
		mu:     mu,
		buf:    bytes.NewBuffer([]byte("")),
		prefix: "       ",
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

	l.out(string(p))
	return nil
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

		l.out(line)
	}

	return nil
}

func (l *Logstreamer) out(str string) (err error) {
	str = l.prefix + str

	l.writer.Write([]byte(str))

	return nil
}
