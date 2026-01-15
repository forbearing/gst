package redis_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/forbearing/gst/bootstrap"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/util"
)

func BenchmarkRedis(b *testing.B) {
	os.Setenv(config.REDIS_ADDR, "127.0.0.1:6378")
	os.Setenv(config.REDIS_PASSWORD, "password123")
	os.Setenv(config.REDIS_ENABLE, "true")
	os.Setenv(config.LOGGER_FILE, "/tmp/test.log")
	os.Setenv(config.REDIS_EXPIRATION, "8h")
	util.RunOrDie(bootstrap.Bootstrap)

	groups := make([]*Group, 0, 1000)
	for i := range 1000 {
		groups = append(groups, &Group{
			Name:        fmt.Sprintf("group-%d", i),
			Desc:        fmt.Sprintf("desc-%d", i),
			MemberCount: i,
		})
	}

	for b.Loop() {
		if err := redis.SetML("groups", groups); err != nil {
			b.Fatalf("%+v\n", err)
		}
	}
}

type Group struct {
	Name        string `json:"name,omitempty" schema:"name" gorm:"unique" binding:"required"`
	Desc        string `json:"desc,omitempty" schema:"desc"`
	MemberCount int    `json:"member_count" gorm:"default:0"`

	model.Base
}
