package stream_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"net"
	"testing"
	"time"

	stream "github.com/stweiz/go-aead-iostream"
)

func TestAEADConnStream(t *testing.T) {
	seed := hash([]byte("Hello"))
	rawMessage := []byte("Package cipher implements standard block cipher modes that can be wrapped around low-level block cipher implementations. See https://csrc.nist.gov/groups/ST/toolkit/BCM/current_modes.html and NIST Special Publication 800-38A.")
	chunkSize := 64

	c1, err := aes.NewCipher(seed)
	if err != nil {
		panic(err)
	}

	c2, err := aes.NewCipher(seed)
	if err != nil {
		panic(err)
	}

	aead1, err := cipher.NewGCM(c1)
	if err != nil {
		panic(err)
	}
	aead2, err := cipher.NewGCM(c2)
	if err != nil {
		panic(err)
	}

	passCh := make(chan struct{})

	// WRITE

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	go func() {

		for {
			conn, err := l.Accept()
			if err != nil {
				panic(err)
			}

			w := stream.NewStreamWriteCloser(seed, chunkSize, conn, aead2)
			w.Write(rawMessage)
			w.Close()
		}

	}()

	// READ
	go func() {
		var conn net.Conn
		retry := 0
		for {
			conn, err = net.Dial("tcp", l.Addr().String())
			if err == nil {
				break
			}
			t.Log(err)
			time.Sleep(1000)
			if retry > 5 {
				panic(err)
			}
			retry++
		}

		r := stream.NewStreamReader(seed, chunkSize, conn, aead1)
		buf := make([]byte, 2048)
		dst := make([]byte, 0)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				dst = append(dst, buf[:n]...)
			}
			if err != nil && err != io.EOF {
				panic(err)
			}
			if err == io.EOF {
				break
			}
		}

		if !bytes.Equal(dst, rawMessage) {
			t.Errorf("dst is %s, but raw is %s", dst, rawMessage)
		} else {
			t.Log("pass")
		}

		passCh <- struct{}{}
	}()

	<-passCh
}
