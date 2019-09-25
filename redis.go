package main

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/pilu/go-base62"
	"io"
	"time"
)

const (
	URLIDKEY = "next:url:id"
	//s->l
	SHORTLINKKEY = "shortlink:%s:url"
	//l->s
	URLHASHKEY = "urlhash:%s:url"
	//s detail
	SHORTLINKDETAILKEY = "shortlink:%s:detail"
)

//
type RedisCli struct {
	Cli *redis.Client
}

type URLDetail struct {
	URL                 string        `json:"url"`
	CreatedAt           string        `json:"created_at"`
	ExpirationInMinutes time.Duration `json:"expiration_in_minutes"`
}

//init redis
func NewRedisCli(addr string, pwd string, db int) *RedisCli {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       db,
	})

	if _, err := client.Ping().Result(); err != nil {
		panic(err)
	}
	return &RedisCli{Cli: client}
}

func (r *RedisCli) Shorten(url string, exp int64) (string, error) {
	//sha1编码url
	urlHash := toSha1(url)
	//查找缓存中是否有对应的短链
	d, err := r.Cli.Get(fmt.Sprintf(URLHASHKEY, urlHash)).Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exist")
	} else if err != nil {
		return "", err
	} else {
		return d, nil
	}

	//首次转换
	id, err := r.Cli.Incr(URLIDKEY).Result()
	if err != nil {
		return "", err
	}
	//对计数id,base62后即为短链
	idint := int(id)
	eid := base62.Encode(idint)

	//保存短链到长链的映射
	err = r.Cli.Set(fmt.Sprintf(SHORTLINKKEY, eid), url, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	//长地址到都短地址
	err = r.Cli.Set(fmt.Sprintf(URLHASHKEY, urlHash), eid, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	//详情
	detail, err := json.Marshal(
		&URLDetail{
			URL:                 url,
			CreatedAt:           time.Now().String(),
			ExpirationInMinutes: time.Duration(exp)})
	if err != nil {
		return "", err
	}

	err = r.Cli.Set(fmt.Sprintf(SHORTLINKDETAILKEY, eid), detail, time.Duration(exp)*time.Minute).Err()
	if err != nil {
		return "", err
	}
	return eid, nil

}
func (r *RedisCli) Unshorten(eid string) (string, error) {
	d, err := r.Cli.Get(fmt.Sprintf(SHORTLINKKEY, eid)).Result()
	if err == redis.Nil {
		return "", StatusError{404, errors.New("unknow short url")}
	} else if err != nil {
		return "", err
	} else {
		return d, nil
	}
}
func (r *RedisCli) ShortlinkInfo(eid string) (interface{}, error) {
	d, err := r.Cli.Get(fmt.Sprintf(SHORTLINKDETAILKEY, eid)).Result()
	if err == redis.Nil {
		return "", StatusError{404, errors.New("unknow short url")}
	} else if err != nil {
		return "", err
	} else {
		return d, nil
	}
	return eid, nil
}

func toSha1(s string) string {
	t := sha1.New()
	io.WriteString(t, s)
	return fmt.Sprintf("%x", t.Sum(nil))
}
