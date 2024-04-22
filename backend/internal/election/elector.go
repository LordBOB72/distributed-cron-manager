package election

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

const electionKey = "/cron-manager/leader"

type Elector struct {
	nodeID  string
	client  *clientv3.Client
	session *concurrency.Session
	log     *zap.Logger

	isLeader chan bool
}

func New(nodeID string, etcdEndpoints []string, log *zap.Logger) (*Elector, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd connect: %w", err)
	}

	sess, err := concurrency.NewSession(cli, concurrency.WithTTL(15))
	if err != nil {
		return nil, fmt.Errorf("etcd session: %w", err)
	}

	return &Elector{
		nodeID:   nodeID,
		client:   cli,
		session:  sess,
		log:      log,
		isLeader: make(chan bool, 1),
	}, nil
}

// Campaign blocks until this node becomes leader or ctx is cancelled.
// Sends true to IsLeader() when elected, false when leadership is lost.
func (e *Elector) Campaign(ctx context.Context) {
	election := concurrency.NewElection(e.session, electionKey)

	for {
		if ctx.Err() != nil {
			return
		}

		e.log.Info("campaigning for leader", zap.String("node", e.nodeID))
		if err := election.Campaign(ctx, e.nodeID); err != nil {
			if ctx.Err() != nil {
				return
			}
			e.log.Error("campaign error, retrying", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}

		e.log.Info("became leader", zap.String("node", e.nodeID))
		select {
		case e.isLeader <- true:
		default:
		}

		// hold leadership until session expires or we resign
		select {
		case <-e.session.Done():
			e.log.Warn("session expired, lost leadership")
			select {
			case e.isLeader <- false:
			default:
			}
			// create a new session and try again
			sess, err := concurrency.NewSession(e.client, concurrency.WithTTL(15))
			if err != nil {
				e.log.Error("session renewal failed", zap.Error(err))
				time.Sleep(5 * time.Second)
			} else {
				e.session = sess
				election = concurrency.NewElection(e.session, electionKey)
			}
		case <-ctx.Done():
			_ = election.Resign(context.Background())
			return
		}
	}
}

func (e *Elector) IsLeaderCh() <-chan bool {
	return e.isLeader
}

func (e *Elector) Close() {
	e.session.Close()
	e.client.Close()
}
