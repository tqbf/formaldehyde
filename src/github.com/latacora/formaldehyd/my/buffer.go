package my

import (
	"bytes"
	"net/http"
)

type BufferedResponseWriter struct {
	buf    bytes.Buffer
	status int
	http.ResponseWriter
}

func (self *BufferedResponseWriter) WriteHeader(status int) {
	self.status = status
}

func (self *BufferedResponseWriter) Write(b []byte) (int, error) {
	return self.buf.Write(b)
}

func (self *BufferedResponseWriter) Flush() {
	if self.status != 0 {
		self.ResponseWriter.WriteHeader(self.status)
	}
	self.ResponseWriter.Write(self.buf.Bytes())
	self.ResponseWriter.(http.Flusher).Flush()
	self.buf.Reset()
}

func NewBufferedResponseWriter(w http.ResponseWriter) *BufferedResponseWriter {
	return &BufferedResponseWriter{
		buf:            bytes.Buffer{},
		status:         200,
		ResponseWriter: w,
	}
}

type LoggingResponseWriter struct {
	Count  int
	Status int
	http.ResponseWriter
}

func (self *LoggingResponseWriter) WriteHeader(status int) {
	self.Status = status
	self.ResponseWriter.WriteHeader(status)
}

func (self *LoggingResponseWriter) Write(b []byte) (int, error) {
	self.Count += len(b)
	return self.ResponseWriter.Write(b)
}

func (self *LoggingResponseWriter) Flush() {
	self.ResponseWriter.(http.Flusher).Flush()
}

func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		Status:         200,
		ResponseWriter: w,
	}
}
