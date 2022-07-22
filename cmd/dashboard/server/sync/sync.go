package sync

import (
	"context"

	"github.com/robfig/cron/v3"

	"github.com/sincaw/archivedb/cmd/dashboard/server/common"
	"github.com/sincaw/archivedb/cmd/dashboard/server/utils"
	"github.com/sincaw/archivedb/pkg"
)

var (
	logger = utils.Logger()
)

const (
	FavAPI = "https://weibo.com/ajax/favorites/all_fav?uid=%s&page=%d"
)

type Sync struct {
	ctx context.Context

	ns        pkg.Namespace
	favBucket pkg.Bucket

	config common.SyncerConfig

	httpCli  *common.HttpCli
	cron     *cron.Cron
	notifyCh chan struct{}
}

// New Sync instance with db ns and its configuration
func New(ctx context.Context, ns pkg.Namespace, config common.SyncerConfig) (*Sync, error) {
	err := common.ValidateSyncerConfig(config)
	if err != nil {
		return nil, err
	}

	cli, err := common.NewWithHeader(map[string]string{"cookie": config.Cookie})
	if err != nil {
		return nil, err
	}

	notifyCh := make(chan struct{}, 1)
	c := cron.New()
	_, err = c.AddFunc(config.Cron, func() {
		select {
		case notifyCh <- struct{}{}:
		default:
		}
	})
	if err != nil {
		return nil, err
	}

	fav, err := ns.CreateBucket([]byte(common.WeiboFavIndexBucket))
	if err != nil {
		return nil, err
	}

	return &Sync{
		ctx: ctx,

		ns:        ns,
		favBucket: fav,

		config: config,

		httpCli:  cli,
		cron:     c,
		notifyCh: notifyCh,
	}, nil
}

func (s *Sync) Start() {
	// sync right now
	s.notifyCh <- struct{}{}

	s.cron.Start()
	defer s.cron.Stop()

	for {
		select {
		case <-s.notifyCh:
			s.syncFavorite()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Sync) Accept(cookie string) {
	cli, err := common.NewWithHeader(map[string]string{"cookie": cookie})
	if err != nil {
		logger.Error("update cookie fail ", err)
		return
	}
	s.httpCli = cli
	logger.Info("update cookie success, trigger sync")
	s.notifyCh <- struct{}{}
}
