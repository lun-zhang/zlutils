package zlutils

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/redis.v5"
	"time"
)

//NOTE: 只能用于初始化时候，失败则fatal
//redis基本不会是性能瓶颈，所以不放xray
func InitRedis(redisUrl string) (client *redis.Client) {
	redisOpt, err := redis.ParseURL(redisUrl)
	if err != nil {
		logrus.WithError(err).Fatalf("redis connect failed")
	}
	client = redis.NewClient(redisOpt)
	//NOTE: pipeline没法用这个打日志
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			begin := time.Now()
			err := oldProcess(cmd)
			end := time.Now()
			logrus.WithFields(logrus.Fields{
				"redis-cmd": cmd.String(),
				"duration":  end.Sub(begin).String(),
				"source":    GetSource(2),
				"stack":     nil,
			}).Debug()
			return err
		}
	})
	return
}
