package fastlimiter_test

import (
	"compress/gzip"
	"compress/zlib"
	"crypto/rand"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/mushroomsir/fastlimiter"
	"github.com/stretchr/testify/assert"
)

func TestFastlimiter(t *testing.T) {
	t.Run("Fastlimiter with default Options should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})

		id := genID()
		policy := []int32{10, 1000}

		res, err := limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)
		assert.Equal(1000, int(res.Duration/time.Millisecond))
		assert.True(res.Reset.After(time.Now()))
		res, err = limiter.Get(id, policy...)
		assert.Equal(10, res.Total)
		assert.Equal(8, res.Remaining)
	})
	t.Run("Fastlimiter with expire should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})

		id := genID()
		policy := []int32{10, 100}

		res, err := limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)
		res, err = limiter.Get(id, policy...)
		assert.Equal(8, res.Remaining)

		time.Sleep(100 * time.Millisecond)
		res, err = limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)
	})
	t.Run("Fastlimiter with goroutine should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})
		policy := []int32{10, 500}
		id := genID()
		res, err := limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)
		for i := 0; i < 100; i++ {
			go limiter.Get(id, policy...)
		}
		time.Sleep(200 * time.Millisecond)
		res, err = limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(10, res.Total)
		assert.Equal(-1, res.Remaining)
	})
	t.Run("Fastlimiter with multi-policy should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})

		id := genID()
		policy := []int32{3, 100, 2, 200}
		res, err := limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(2, res.Remaining)
		res, err = limiter.Get(id, policy...)
		assert.Equal(1, res.Remaining)
		res, err = limiter.Get(id, policy...)
		assert.Equal(0, res.Remaining)
		res, err = limiter.Get(id, policy...)
		assert.Equal(-1, res.Remaining)
		res, err = limiter.Get(id, policy...)
		assert.Equal(-1, res.Remaining)
		assert.True(res.Reset.After(time.Now()))

		time.Sleep(res.Duration + time.Millisecond)
		res, err = limiter.Get(id, policy...)
		assert.Equal(1, res.Remaining)

		res, err = limiter.Get(id, policy...)
		assert.Equal(0, res.Remaining)
		res, err = limiter.Get(id, policy...)
		assert.Equal(-1, res.Remaining)

		time.Sleep(res.Duration + time.Millisecond)
		res, err = limiter.Get(id, policy...)
		assert.Equal(2, res.Remaining)
	})

	t.Run("Fastlimiter with Remove id should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})

		id := genID()
		policy := []int32{10, 1000}

		res, err := limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)
		limiter.Remove(id)
		res, err = limiter.Get(id, policy...)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)
	})

	t.Run("Fastlimiter with wrong policy id should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})

		id := genID()
		policy := []int32{10, 1000, 1}

		res, err := limiter.Get(id, policy...)
		assert.Error(err)
		assert.Equal(0, res.Total)
		assert.Equal(0, res.Remaining)

	})
	t.Run("Fastlimiter with empty policy id should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})

		id := genID()
		policy := []int32{}

		res, _ := limiter.Get(id, policy...)
		assert.Equal(1000, res.Total)
		assert.Equal(999, res.Remaining)
		assert.Equal(time.Minute, res.Duration)

	})

	t.Run("Fastlimiter with Clean cache should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})

		id := genID()
		policy := []int32{10, 100}

		res, _ := limiter.Get(id, policy...)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)

		time.Sleep(res.Duration + time.Millisecond)
		limiter.Clean()
		res, _ = limiter.Get(id, policy...)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)

		time.Sleep(res.Duration + time.Millisecond)
		limiter.Clean()
		res, _ = limiter.Get(id, policy...)
		assert.Equal(10, res.Total)
		assert.Equal(9, res.Remaining)
	})
	t.Run("Fastlimiter with big goroutine should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{})
		policy := []int32{1000, 1000}
		id := genID()
		for i := 0; i < 1000; i++ {
			go limiter.Get(id, policy...)
		}
		time.Sleep(300 * time.Millisecond)

		res, err := limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(1000, res.Total)
		assert.Equal(-1, res.Remaining)
	})

	t.Run("Fastlimiter with CleanDuration should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{
			CleanDuration: 100 * time.Millisecond,
		})
		policy := []int32{100, 100}
		id := genID()

		res, err := limiter.Get(id, policy...)
		assert.Nil(err)
		assert.Equal(100, res.Total)
		assert.Equal(99, res.Remaining)
		time.Sleep(res.Duration + time.Millisecond)
		res, err = limiter.Get(id, policy...)
		assert.Equal(100, res.Total)
		assert.Equal(99, res.Remaining)
	})
	t.Run("Fastlimiter with CleanDuration should be", func(t *testing.T) {
		assert := assert.New(t)
		limiter := fastlimiter.New(&fastlimiter.Options{
			CleanDuration: 100 * time.Millisecond,
		})

		limiter.Get("1", []int32{100, 100}...)
		limiter.Get("2", []int32{100, 100}...)
		assert.Equal(2, limiter.Count())
	})
}

// ------Helpers for help test --------
var DefaultClient = &http.Client{}

type GearResponse struct {
	*http.Response
}

func RequestBy(method, url string) (*GearResponse, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func DefaultClientDo(req *http.Request) (*GearResponse, error) {
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func DefaultClientDoWithCookies(req *http.Request, cookies map[string]string) (*http.Response, error) {
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	return DefaultClient.Do(req)
}
func NewRequst(method, url string) (*http.Request, error) {
	return http.NewRequest(method, url, nil)
}

func (resp *GearResponse) OK() bool {
	return resp.StatusCode < 400
}
func (resp *GearResponse) Content() (val []byte, err error) {
	var b []byte
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		if reader, err = gzip.NewReader(resp.Body); err != nil {
			return nil, err
		}
	case "deflate":
		if reader, err = zlib.NewReader(resp.Body); err != nil {
			return nil, err
		}
	default:
		reader = resp.Body
	}

	defer reader.Close()
	if b, err = ioutil.ReadAll(reader); err != nil {
		return nil, err
	}
	return b, err
}

func (resp *GearResponse) Text() (val string, err error) {
	b, err := resp.Content()
	if err != nil {
		return "", err
	}
	return string(b), err
}
func genID() string {
	buf := make([]byte, 12)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

//--------- End ---------
