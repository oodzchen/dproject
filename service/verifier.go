package service

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Verifier struct {
	CodeLifeTime time.Duration
	Rdb          *redis.Client
}

const DefaultCodeLifeTime = 5 * time.Minute
const CodeLength = 6
const keyPrefix = "verif_code_"

func getKey(email string) string {
	return fmt.Sprintf("%s%s", keyPrefix, email)
}

func (v *Verifier) GenCode() string {
	var code string
	for i := 0; i < CodeLength; i++ {
		code += strconv.Itoa(rand.Intn(10))
	}

	return code
}

func (v *Verifier) EncryptCode(code string) (string, error) {
	encrypt, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(encrypt), nil
}

func (v *Verifier) VerifyCode(code string, encryptCode string) error {
	return bcrypt.CompareHashAndPassword([]byte(encryptCode), []byte(code))
}

func (v *Verifier) SaveCode(email string, code string) error {
	return v.Rdb.Set(context.Background(), getKey(email), code, v.CodeLifeTime).Err()
}

func (v *Verifier) GetCode(email string) (string, error) {
	return v.Rdb.Get(context.Background(), getKey(email)).Result()
}

func (v *Verifier) DeleteCode(email string) error {
	return v.Rdb.Del(context.Background(), getKey(email)).Err()
}
