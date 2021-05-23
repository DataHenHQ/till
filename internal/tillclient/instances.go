package tillclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/DataHenHQ/tillup/features"
	"github.com/go-resty/resty/v2"
)

type InstancesService service

type Instance struct {
	ID                  *int64              `json:"id,omitempty"`
	Name                *string             `json:"name,omitempty"`
	Description         *string             `json:"description,omitempty"`
	Requests            *int64              `json:"requests,omitempty"`
	InterceptedRequests *int64              `json:"intercepted_requests,omitempty"`
	FailedRequests      *int64              `json:"failed_requests,omitempty"`
	CacheHits           *int64              `json:"cache_hits,omitempty"`
	CacheSets           *int64              `json:"cache_sets,omitempty"`
	Features            *[]features.Feature `json:"features,omitempty"`

	CreatedAt *Timestamp `json:"created_at,omitempty"`
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

func (s *InstancesService) Get(ctx context.Context, name string) (*Instance, *resty.Response, error) {
	u := fmt.Sprintf("instances/%v", name)
	req, err := s.client.NewRequest(ctx, u, nil)

	instance := new(Instance)

	resp, err := req.Get(u)
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

// GetID returns the ID field if it's non-nil, zero value otherwise.
func (i *Instance) GetID() int64 {
	if i == nil || i.ID == nil {
		return 0
	}
	return *i.ID
}

// GetName returns the Name field if it's non-nil, zero value otherwise.
func (i *Instance) GetName() string {
	if i == nil || i.Name == nil {
		return ""
	}
	return *i.Name
}

// GetDescription returns the Description field if it's non-nil, zero value otherwise.
func (i *Instance) GetDescription() string {
	if i == nil || i.Description == nil {
		return ""
	}
	return *i.Description
}

// GetFeatures returns the Features field if it's non-nil, zero value otherwise.
func (i *Instance) GetFeatures() []features.Feature {
	if i == nil || i.Features == nil {
		return []features.Feature{}
	}
	return *i.Features
}

// GetCreatedAt returns the CreatedAt field if it's non-nil, zero value otherwise.
func (i *Instance) GetCreatedAt() Timestamp {
	if i == nil || i.CreatedAt == nil {
		return Timestamp{}
	}
	return *i.CreatedAt
}

// GetRequests returns the Requests field if it's non-nil, zero value otherwise.
func (i *Instance) GetRequests() int64 {
	if i == nil || i.Requests == nil {
		return 0
	}
	return *i.Requests
}

// GetUpdatedAt returns the UpdatedAt field if it's non-nil, zero value otherwise.
func (i *Instance) GetUpdatedAt() Timestamp {
	if i == nil || i.UpdatedAt == nil {
		return Timestamp{}
	}
	return *i.UpdatedAt
}
