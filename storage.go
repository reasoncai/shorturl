package main

type Storage interface {
	Shorten(url string, exp int64) (string, error)
	Unshorten(eid string) (string, error)
	ShortlinkInfo(eid string) (interface{}, error)
}
