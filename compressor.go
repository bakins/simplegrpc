package simplegrpc

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"sync"
)

type Compressor interface {
	Name() string
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
}

var GzipCompressor Compressor = gzipCompressor{}

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(ioutil.Discard)
	},
}

var gzipReaderPool = sync.Pool{}

type gzipCompressor struct{}

func (g gzipCompressor) Name() string {
	return "gzip"
}

func (g gzipCompressor) Compress(in []byte) ([]byte, error) {
	var w bytes.Buffer

	z := gzipWriterPool.Get().(*gzip.Writer)
	z.Reset(&w)
	defer gzipWriterPool.Put(z)

	if _, err := z.Write(in); err != nil {
		return nil, err
	}

	if err := z.Close(); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func (g gzipCompressor) Decompress(in []byte) ([]byte, error) {
	r := bytes.NewReader(in)

	z, ok := gzipReaderPool.Get().(*gzip.Reader)
	if !ok {
		n, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}

		z = n
		defer gzipReaderPool.Put(z)
	} else {
		defer gzipReaderPool.Put(z)

		if err := z.Reset(r); err != nil {
			return nil, err
		}
	}

	return ioutil.ReadAll(io.LimitReader(z, int64(maxReceiveMessageSize)+1))
}

/*
var SnappyCompressor Compressor = snappyCompressor{}

type snappyCompressor struct{}

func (s snappyCompressor) Compress(in []byte) ([]byte, error) {
	if n := snappy.MaxEncodedLen(len(in)); n < 0 {
		return nil, snappy.ErrTooLarge
	}

	return snappy.Encode(nil, in), nil
}

func (s snappyCompressor) Decompress(in []byte) ([]byte, error) {
	return snappy.Decode(nil, in)
}
*/
