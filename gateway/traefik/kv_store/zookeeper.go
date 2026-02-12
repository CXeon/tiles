package kv_store

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
)

type zkStore struct {
	conn *zk.Conn
}

type ZookeeperConfig struct {
	Endpoints      []string
	ConnectTimeout time.Duration // 连接超时时间，单位：time.Duration（如 5*time.Second）
	SessionTimeout time.Duration // 会话超时时间，单位：time.Duration（如 10*time.Second）
}

func NewZookeeperStore(cfg ZookeeperConfig) (KvStore, error) {
	// Set default timeouts
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.SessionTimeout == 0 {
		cfg.SessionTimeout = 10 * time.Second
	}

	conn, _, err := zk.Connect(cfg.Endpoints, cfg.SessionTimeout)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Best-effort wait for connection
	if conn.State() != zk.StateHasSession && conn.State() != zk.StateConnected {
		time.Sleep(100 * time.Millisecond)
	}

	return &zkStore{
		conn: conn,
	}, nil
}

func (s *zkStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
	key = normalizePath(key)

	// Ensure parent nodes exist
	if err := s.ensureParentPath(ctx, key); err != nil {
		return err
	}

	// 判断是否需要 TTL（使用临时节点）
	var flags int32
	if len(expired) > 0 && expired[0] > 0 {
		// ZooKeeper 临时节点：客户端断开连接时自动删除
		// 注意：ZooKeeper 不支持精确 TTL，只能通过 Session 超时来控制
		flags = zk.FlagEphemeral
	}

	// Try to create first
	_, err := s.conn.Create(key, value, flags, zk.WorldACL(zk.PermAll))
	if err == zk.ErrNodeExists {
		// Node exists, update it
		_, err = s.conn.Set(key, value, -1) // -1 means any version
		if err != nil {
			return fmt.Errorf("zk set error: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("zk create error: %w", err)
	}
	return nil
}

func (s *zkStore) Get(ctx context.Context, key string) ([]byte, error) {
	key = normalizePath(key)
	data, _, err := s.conn.Get(key)
	if err == zk.ErrNoNode {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("zk get error: %w", err)
	}
	return data, nil
}

func (s *zkStore) GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	prefix = normalizePath(prefix)
	result := make(map[string][]byte)

	err := s.recursiveGet(ctx, prefix, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *zkStore) Delete(ctx context.Context, key string) error {
	key = normalizePath(key)
	err := s.conn.Delete(key, -1) // -1 means any version
	if err == zk.ErrNoNode {
		return nil // Already deleted
	}
	if err != nil {
		return fmt.Errorf("zk delete error: %w", err)
	}
	return nil
}

func (s *zkStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	prefix = normalizePath(prefix)

	// Recursively collect all paths
	paths, err := s.collectPaths(ctx, prefix)
	if err != nil {
		return err
	}

	// Delete from leaf to root (reverse order)
	for i := len(paths) - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return fmt.Errorf("delete cancelled: %w", ctx.Err())
		default:
		}

		err := s.conn.Delete(paths[i], -1)
		if err != nil && err != zk.ErrNoNode {
			return fmt.Errorf("zk delete %s: %w", paths[i], err)
		}
	}

	return nil
}

func (s *zkStore) Close() error {
	s.conn.Close()
	return nil
}

func (s *zkStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
	// ZooKeeper 临时节点自动维护，无需手动续约
	// 只要连接存活，所有临时节点就存活
	// 这里为 No-op
	return nil
}

func (s *zkStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
	// ZooKeeper 临时节点自动维护，无需手动续约
	// No-op
	return nil
}

// Helper functions

func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}

func (s *zkStore) ensureParentPath(ctx context.Context, zkPath string) error {
	parent := path.Dir(zkPath)
	if parent == "/" || parent == "." {
		return nil
	}

	// Check if parent exists
	exists, _, err := s.conn.Exists(parent)
	if err != nil {
		return fmt.Errorf("zk check parent error: %w", err)
	}
	if exists {
		return nil
	}

	// Create parent recursively
	if err := s.ensureParentPath(ctx, parent); err != nil {
		return err
	}

	_, err = s.conn.Create(parent, []byte{}, 0, zk.WorldACL(zk.PermAll))
	if err != nil && err != zk.ErrNodeExists {
		return fmt.Errorf("zk create parent error: %w", err)
	}

	return nil
}

func (s *zkStore) recursiveGet(ctx context.Context, zkPath string, result map[string][]byte) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("get cancelled: %w", ctx.Err())
	default:
	}

	// Get current node data
	data, _, err := s.conn.Get(zkPath)
	if err == zk.ErrNoNode {
		return nil // Node doesn't exist, skip
	}
	if err != nil {
		return fmt.Errorf("zk get %s: %w", zkPath, err)
	}

	result[zkPath] = data

	// Get children
	children, _, err := s.conn.Children(zkPath)
	if err == zk.ErrNoNode {
		return nil
	}
	if err != nil {
		return fmt.Errorf("zk children %s: %w", zkPath, err)
	}

	// Recursively get children
	for _, child := range children {
		childPath := path.Join(zkPath, child)
		if err := s.recursiveGet(ctx, childPath, result); err != nil {
			return err
		}
	}

	return nil
}

func (s *zkStore) collectPaths(ctx context.Context, zkPath string) ([]string, error) {
	var paths []string

	err := s.recursiveCollect(ctx, zkPath, &paths)
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (s *zkStore) recursiveCollect(ctx context.Context, zkPath string, paths *[]string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("collect cancelled: %w", ctx.Err())
	default:
	}

	// Check if node exists
	exists, _, err := s.conn.Exists(zkPath)
	if err != nil {
		return fmt.Errorf("zk exists %s: %w", zkPath, err)
	}
	if !exists {
		return nil
	}

	// Add current path
	*paths = append(*paths, zkPath)

	// Get children
	children, _, err := s.conn.Children(zkPath)
	if err == zk.ErrNoNode {
		return nil
	}
	if err != nil {
		return fmt.Errorf("zk children %s: %w", zkPath, err)
	}

	// Recursively collect children
	for _, child := range children {
		childPath := path.Join(zkPath, child)
		if err := s.recursiveCollect(ctx, childPath, paths); err != nil {
			return err
		}
	}

	return nil
}
