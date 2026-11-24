package application

import (
	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

type Service struct{ Port access.Service }

func (s Service) Login(ctx context.Context, in access.LoginInput, requestID string) (access.LoginResult, error) {
	login, valid := domain.NormalizeLogin(in.Login)
	in.Login = login
	if !valid || len(in.Password) == 0 || len(in.Password) > 1024 {
		return access.LoginResult{}, foundation.Fail(401, "AUTH_REQUIRED", "登录名或密码不正确")
	}
	return s.Port.Login(ctx, in, requestID)
}
func (s Service) Session(ctx context.Context, c access.Credentials) (access.Principal, error) {
	return s.Port.Session(ctx, c)
}
func (s Service) Logout(ctx context.Context, c access.Credentials) error {
	return s.Port.Logout(ctx, c)
}
