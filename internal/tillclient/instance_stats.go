package tillclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/go-resty/resty/v2"
)

type InstanceStatsService service

type InstanceStat struct {
	Name     *string `json:"name,omitempty"`
	Requests *uint64 `json:"requests,omitempty"`
}

// InstanceStatMutex is used for struct to for atomic incr and decr of counters
type InstanceStatMutex struct {
	InstanceStat
	Mutex *sync.Mutex
}

func (s *InstanceStatsService) Update(ctx context.Context, is InstanceStat) (*Instance, *resty.Response, error) {
	if is.GetName() == "" {
		return nil, nil, errors.New("instance name is required")
	}

	u := fmt.Sprintf("instances/%v/stats", is.GetName())
	req, err := s.client.NewRequest(ctx, u, nil)

	req.SetBody(is)

	instance := new(Instance)

	resp, err := req.Put(u)
	if err != nil {
		return nil, resp, err
	}
	defer resp.RawResponse.Body.Close()
	b, err := ioutil.ReadAll(resp.RawResponse.Body)
	if err != nil {
		return nil, nil, err
	}

	json.Unmarshal(b, &instance)

	return instance, resp, nil
}

// GetRequests returns the Requests field if it's non-nil, zero value otherwise.
func (is *InstanceStat) GetRequests() uint64 {
	if is == nil || is.Requests == nil {
		return 0
	}
	return *is.Requests
}

// GetName returns the Name field if it's non-nil, zero value otherwise.
func (is *InstanceStat) GetName() string {
	if is == nil || is.Name == nil {
		return ""
	}
	return *is.Name
}

// IsZero checks if it is zero value
func (is *InstanceStat) IsZero() bool {
	if is.GetRequests() == 0 {
		return true
	}
	return false
}

// DeepCopy goes through the fields and copy them, so that all values are copied, an all pointer don't point to the same address
func (is *InstanceStat) DeepCopy() (nis InstanceStat) {
	if is.Name != nil {
		name := is.GetName()
		nis.Name = &name
	}

	if is.Requests != nil {
		requests := is.GetRequests()
		nis.Requests = &requests
	}

	return nis
}
