package service

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pcbook/pb"
)

type AuthServer struct {
	UserStore  UserStore   //用户信息存储器，用于保存用户账号信息
	JWTManager *JWTManager //JWT管理器，用于生成及验证json web token
}

func NewAuthServer(userStore UserStore, jwtManager *JWTManager) *AuthServer {
	return &AuthServer{
		UserStore:  userStore,
		JWTManager: jwtManager,
	}
}

func (server *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	//判断用户在存储器中是否存在
	user, err := server.UserStore.Find(req.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot find user: %v", err)
	}

	//判断用户密码是否正确
	if user == nil || !user.IsCorrectPassword(req.GetPassword()) {
		return nil, status.Errorf(codes.NotFound, "incorrect username/password")
	}

	//生成JWT
	token, err := server.JWTManager.Generate(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate access token: %v", err)
	}
	res := &pb.LoginResponse{
		AccessToken: token,
	}
	return res, nil
}
