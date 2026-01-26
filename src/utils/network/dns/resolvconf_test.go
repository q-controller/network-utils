package dns

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createTempResolvConf(t *testing.T, content string) (string, func()) {
	tmp, err := os.CreateTemp(os.TempDir(), "resolvconf_test_*.conf")
	assert.NoError(t, err)
	if content != "" {
		_, err = tmp.WriteString(content)
		assert.NoError(t, err)
	}
	_ = tmp.Close()
	cleanup := func() { os.Remove(tmp.Name()) }
	return tmp.Name(), cleanup
}

func TestReadUpstreams_MissingFile(t *testing.T) {
	result := readUpstreams("/path/to/nonexistent/file.conf")
	assert.Error(t, result.Error)
	assert.Empty(t, result.Endpoints)
}

func TestReadUpstreams(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected []string
		wantErr  bool
	}{
		{"Valid", "nameserver 8.8.8.8\nnameserver 1.1.1.1\n", []string{"8.8.8.8:53", "1.1.1.1:53"}, false},
		{"Empty", "", []string{}, false},
		{"Malformed", "nameserver\n", []string{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, cleanup := createTempResolvConf(t, tc.content)
			defer cleanup()
			result := readUpstreams(file)
			if tc.wantErr {
				assert.Error(t, result.Error)
			} else {
				assert.NoError(t, result.Error)
				assert.Equal(t, tc.expected, result.Endpoints)
			}
		})
	}
}

func TestGetUpstreamDNS_NonExistentFile(t *testing.T) {
	ch, err := GetUpstreamDNSFromFile(context.Background(), "/dir/does/not/exist/nonexistent.conf")
	assert.Error(t, err)
	assert.Nil(t, ch)
}

func TestGetUpstreamDNS_ChannelReceives(t *testing.T) {
	file, createErr := os.CreateTemp(os.TempDir(), "resolvconf_test_*.conf")
	assert.NoError(t, createErr)
	defer os.Remove(file.Name())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	writes := []string{
		"nameserver 9.9.9.9\n",
		"nameserver 8.8.4.4\nnameserver 1.0.0.1\n",
		"nameserver 127.0.0.1\n",
	}
	expects := [][]string{
		{"9.9.9.9:53"},
		{"8.8.4.4:53", "1.0.0.1:53"},
		{"127.0.0.1:53"},
	}

	results := [][]string{}
	stop := make(chan struct{})
	go func() {
		defer close(stop)
		ch, _ := GetUpstreamDNSFromFile(ctx, file.Name())
		for ups := range ch {
			if ups.Error == nil && len(ups.Endpoints) > 0 {
				results = append(results, ups.Endpoints)
			}
		}
	}()

	go func() {
		for _, data := range writes {
			writeErr := os.WriteFile(file.Name(), []byte(data), 0644)
			assert.NoError(t, writeErr)
			time.Sleep(1 * time.Second)
		}
		cancel()
	}()
	<-stop

	assert.Len(t, results, len(expects))
	for i := range expects {
		assert.True(t, Same(results[i], expects[i]), "expected endpoints in result %d to match", i)
	}
}
