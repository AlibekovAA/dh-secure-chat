package websocket

import (
	"context"
	"hash/fnv"
	"sync"

	"github.com/AlibekovAA/dh-secure-chat/backend/internal/common/logger"
	userrepo "github.com/AlibekovAA/dh-secure-chat/backend/internal/user/repository"
)

type ShardedHub struct {
	shards []*Hub
	count  int
	log    *logger.Logger
}

func NewShardedHub(log *logger.Logger, userRepo userrepo.Repository, config HubConfig, shardCount int) *ShardedHub {
	if shardCount <= 0 {
		shardCount = 4
	}

	shards := make([]*Hub, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = NewHub(log, userRepo, config)
	}

	return &ShardedHub{
		shards: shards,
		count:  shardCount,
		log:    log,
	}
}

func (sh *ShardedHub) getShard(userID string) *Hub {
	hash := fnv.New32a()
	hash.Write([]byte(userID))
	index := int(hash.Sum32()) % sh.count
	return sh.shards[index]
}

func (sh *ShardedHub) Register(client *Client) {
	shard := sh.getShard(client.userID)
	shard.Register(client)
}

func (sh *ShardedHub) Unregister(client *Client) {
	shard := sh.getShard(client.userID)
	shard.Unregister(client)
}

func (sh *ShardedHub) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, shard := range sh.shards {
		wg.Add(1)
		go func(s *Hub) {
			defer wg.Done()
			s.Run(ctx)
		}(shard)
	}
	wg.Wait()
}

func (sh *ShardedHub) SendToUser(userID string, message *WSMessage) bool {
	shard := sh.getShard(userID)
	return shard.SendToUser(userID, message)
}

func (sh *ShardedHub) SendToUserWithContext(ctx context.Context, userID string, message *WSMessage) error {
	shard := sh.getShard(userID)
	return shard.SendToUserWithContext(ctx, userID, message)
}

func (sh *ShardedHub) IsUserOnline(userID string) bool {
	shard := sh.getShard(userID)
	return shard.IsUserOnline(userID)
}

func (sh *ShardedHub) HandleMessage(client *Client, msg *WSMessage) {
	shard := sh.getShard(client.userID)
	shard.HandleMessage(client, msg)
}

func (sh *ShardedHub) Shutdown() {
	for _, shard := range sh.shards {
		shard.Shutdown()
	}
}

func (sh *ShardedHub) GetShardCount() int {
	return sh.count
}

func (sh *ShardedHub) CountClients() int {
	total := 0
	for _, shard := range sh.shards {
		total += shard.countClients()
	}
	return total
}
