package utils

import (
    "compress/gzip"
    "io"
    "net/http"
    "strings"
    "sync"

)

var gzipWriterPool = sync.Pool{
    New: func() interface{} {
        return gzip.NewWriter(nil)
    },
}

func DecompressResponse(resp *http.Response) error {
    if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
        reader, err := gzip.NewReader(resp.Body)
        if err != nil {
            return err
        }
        resp.Body = reader
    }
    return nil
}

func CompressResponse(w http.ResponseWriter, r *http.Request) io.Writer {
    if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
        w.Header().Set("Content-Encoding", "gzip")
        gz := gzipWriterPool.Get().(*gzip.Writer)
        gz.Reset(w)
        return gz
    }
    return w
}

func CloseCompressedWriter(w io.Writer) {
    if gz, ok := w.(*gzip.Writer); ok {
        gz.Close()
        gzipWriterPool.Put(gz)
    }
}