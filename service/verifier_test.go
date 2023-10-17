package service

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/mocktool"
	"github.com/redis/go-redis/v9"
)

func TestVerifier(t *testing.T) {
	cfg, err := config.NewTest()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println("testing config: ", cfg)
	// fmt.Println("testing redis config: ", cfg.Redis)

	redisAddr := net.JoinHostPort(cfg.Redis.Host, cfg.Redis.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: cfg.Redis.User,
		Password: cfg.Redis.Password,
		DB:       0,
	})

	verifier := &Verifier{
		CodeLifeTime: 2 * time.Second,
		Rdb:          rdb,
	}

	code := verifier.GenCode()
	if len(code) != codeLength {
		t.Fatalf("gen code error, want code length %d, but got %d", codeLength, len(code))
	}
	fmt.Println("gen code success: ", code)

	encryptCode, err := verifier.EncryptCode(code)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("encrypt code success: ", encryptCode)

	email := mocktool.GenUser().Email
	err = verifier.SaveCode(email, encryptCode)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("save code success")

	savedCode, err := verifier.GetCode(email)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("get code success: ", savedCode)

	if err := verifier.VerifyCode(code, savedCode); err != nil {
		t.Fatalf("verify code %s failed, %v", code, err)
	}
	fmt.Println("verify code success")

	fmt.Println("sleep for code life time: ", verifier.CodeLifeTime)
	time.Sleep(verifier.CodeLifeTime)
	fmt.Println("sleep end")

	_, err = verifier.GetCode(email)
	if err != nil && err != redis.Nil {
		t.Fatal(err)
	}
	fmt.Println("code expired")
}
