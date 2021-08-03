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
	Name                *string `json:"name,omitempty"`
	Requests            *uint64 `json:"requests,omitempty"`
	InterceptedRequests *uint64 `json:"intercepted_requests,omitempty"`
	SuccessfulRequests  *uint64 `json:"successful_requests,omitempty"`
	FailedRequests      *uint64 `json:"failed_requests,omitempty"`
	CacheHits           *uint64 `json:"cache_hits,omitempty"`
	CacheSets           *uint64 `json:"cache_sets,omitempty"`
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

// GetInterceptedRequests returns the InterceptedRequests field if it's non-nil, zero value otherwise.
func (is *InstanceStat) GetInterceptedRequests() uint64 {
	if is == nil || is.InterceptedRequests == nil {
		return 0
	}
	return *is.InterceptedRequests
}

// GetFailedRequests returns the FailedRequests field if it's non-nil, zero value otherwise.
func (is *InstanceStat) GetFailedRequests() uint64 {
	if is == nil || is.FailedRequests == nil {
		return 0
	}
	return *is.FailedRequests
}

// GetSuccessfulRequests returns the SuccessfulRequests field if it's non-nil, zero value otherwise.
func (is *InstanceStat) GetSuccessfulRequests() uint64 {
	if is == nil || is.SuccessfulRequests == nil {
		return 0
	}
	return *is.SuccessfulRequests
}

// GetCacheHits returns the CacheHits field if it's non-nil, zero value otherwise.
func (is *InstanceStat) GetCacheHits() uint64 {
	if is == nil || is.CacheHits == nil {
		return 0
	}
	return *is.CacheHits
}

// GetCacheSets returns the CacheSets field if it's non-nil, zero value otherwise.
func (is *InstanceStat) GetCacheSets() uint64 {
	if is == nil || is.CacheSets == nil {
		return 0
	}
	return *is.CacheSets
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
	if is.GetRequests() == 0 &&
		is.GetInterceptedRequests() == 0 &&
		is.GetSuccessfulRequests() == 0 &&
		is.GetFailedRequests() == 0 &&
		is.GetCacheHits() == 0 &&
		is.GetCacheSets() == 0 {
		return true
	}
	return false
}

// DeepCopy goes through the fields and copy them, so that all values are copied, and all pointer don't point to the same address
func (is *InstanceStat) DeepCopy() (nis InstanceStat) {
	if is.Name != nil {
		name := is.GetName()
		nis.Name = &name
	}

	if is.Requests != nil {
		val := is.GetRequests()
		nis.Requests = &val
	}

	if is.SuccessfulRequests != nil {
		val := is.GetSuccessfulRequests()
		nis.SuccessfulRequests = &val
	}

	if is.FailedRequests != nil {
		val := is.GetFailedRequests()
		nis.FailedRequests = &val
	}

	if is.InterceptedRequests != nil {
		val := is.GetInterceptedRequests()
		nis.InterceptedRequests = &val
	}

	if is.CacheHits != nil {
		val := is.GetCacheHits()
		nis.CacheHits = &val
	}

	if is.CacheSets != nil {
		val := is.GetCacheSets()
		nis.CacheSets = &val
	}

	return nis
}
