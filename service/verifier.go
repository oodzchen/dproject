package service

import (
	"math/rand"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

type Verifier struct{}

const codeLength = 6

func (v *Verifier) GenCode() string {
	var code string
	for i := 0; i < codeLength; i++ {
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

// func (v *Verifier) SaveCode(code string){
// 	v.rdb
// }
