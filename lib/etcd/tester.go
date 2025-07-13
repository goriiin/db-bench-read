package etcd

import (
	"context"
	"db-bench/lib/conf"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdTester struct {
	client *clientv3.Client
	cfg    *conf.Config
}

func NewEtcdTester(ctx context.Context, cfg *conf.Config) (*EtcdTester, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.URI},
		DialTimeout: cfg.ConnectTimeout,
	})
	if err != nil {
		return nil, err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.Status(ctx, cfg.URI)
	if err != nil {
		client.Close()
		return nil, err
	}

	return &EtcdTester{
		client: client,
		cfg:    cfg,
	}, nil
}

func (t *EtcdTester) Close() {
	if t.client != nil {
		t.client.Close()
	}
}
