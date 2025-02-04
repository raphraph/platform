package snap

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

type ClientStub struct {
	error bool
}

func (c *ClientStub) Get(url string) (resp *http.Response, err error) {
	if c.error {
		return nil, fmt.Errorf("error")
	}
	json := `
{ 
	"result": [ 
		{ 
			"name": "test",
			"summary": "test",
			"channel": "stable",
			"version": "1",
			"apps": [ 
				{
					"name": "test",
					"snap": "test"
				}
			]
		} 
	]
}
`
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	return &http.Response{
		StatusCode: 200,
		Body:       r,
	}, nil
}

func TestAppsOK(t *testing.T) {

	snapd := New(&ClientStub{error: false})
	apps, err := snapd.ListAllApps()

	assert.Nil(t, err)
	assert.Equal(t, len(apps), 1)
	assert.Equal(t, apps[0].Apps[0].Name, "test")
}

func TestAppsError(t *testing.T) {

	snapd := New(&ClientStub{error: true})
	apps, err := snapd.ListAllApps()

	assert.Nil(t, apps)
	assert.NotNil(t, err)
}
