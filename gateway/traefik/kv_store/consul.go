package kv_store

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
)

type consulStore struct {
	client *api.Client
	ctx    context.Context
}

func NewConsulStore(ctx context.Context, endpoints []string, token string) (KvStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config := api.DefaultConfig()
	config.Address = endpoints[0]
	config.Token = token

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &consulStore{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *consulStore) Put(key string, value []byte) error {
	p := &api.KVPair{Key: key, Value: value}
	_, err := s.client.KV().Put(p, nil)
	return err
}

func (s *consulStore) Add(key string, value []byte) error {
	// Consul doesn't have a direct "add to list" in KV.
	// We just Put.
	return s.Put(key, value)
}

func (s *consulStore) Get(key string) ([]byte, error) {
	pair, _, err := s.client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, nil
	}
	return pair.Value, nil
}

func (s *consulStore) GetByPrefix(prefix string) (map[string][]byte, error) {
	pairs, _, err := s.client.KV().List(prefix, nil)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for _, pair := range pairs {
		result[pair.Key] = pair.Value
	}
	return result, nil
}

func (s *consulStore) Delete(key string) error {
	_, err := s.client.KV().Delete(key, nil)
	return err
}

func (s *consulStore) Close() error {
	// Consul client doesn't have a Close method in its API.
	return nil
}
