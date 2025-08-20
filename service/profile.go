package service

import (
	"errors"
	"tg-vpn-bot/client"
)

type ProfileService struct {
	HostUSA  string
	HostFIN  string
	Password string
}

func (s *ProfileService) Create(region, platform, name string) (string, []byte, error) {
	var host string
	switch region {
	case "USA":
		host = s.HostUSA
	case "FINLAND":
		host = s.HostFIN
	}
	if host == "" || s.Password == "" {
		return "", nil, errors.New("bad config (HOST/PASSWORD missing)")
	}

	reg := map[string]string{"USA": "u", "FINLAND": "f"}[region]
	plat := map[string]string{"phone": "i", "pc": "p"}[platform]

	return client.DoCreateConfig(host, s.Password, reg, name, plat)
}
