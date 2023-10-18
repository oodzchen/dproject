package model

import (
	"fmt"
	"testing"
)

type userData struct {
	Name     string
	Email    string
	Password string
}

func TestUserValid(t *testing.T) {
	tests := []struct {
		desc  string
		in    *userData
		valid bool
	}{
		{
			desc:  "All valid",
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "111abc@.,222"},
			valid: true,
		},
		{
			desc:  "Name is requried",
			in:    &userData{Name: "", Email: "aaa@test.com", Password: "111abc@222"},
			valid: false,
		},
		{
			desc:  "Email is required",
			in:    &userData{Name: "Mark", Email: "", Password: "111abc@222"},
			valid: false,
		},
		{
			desc:  "Password is required",
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: ""},
			valid: false,
		},
		{
			desc:  "Email format",
			in:    &userData{Name: "Mark", Email: "aaa#test.com", Password: "111abc@222"},
			valid: false,
		},
		{
			desc:  "Name format",
			in:    &userData{Name: "@", Email: "aaa@test.com", Password: "111abc@222"},
			valid: false,
		},
		{
			desc:  "Password length",
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "1a@"},
			valid: false,
		},
		{
			desc:  "Password format",
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "111122222"},
			valid: false,
		},
		{
			desc:  "Password format",
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "abcabcabc"},
			valid: false,
		},
		{
			desc:  "Password format",
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "@#$@#$@#$"},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			u := &User{
				Name:     tt.in.Name,
				Email:    tt.in.Email,
				Password: tt.in.Password,
			}
			err := u.Valid(true)
			// if err != nil {
			// 	fmt.Println("err: ", err)
			// 	fmt.Println("err is ErrValidUserFailed: ", errors.Is(err, ErrValidUserFailed))
			// }
			got := err == nil
			want := tt.valid

			if got != want {
				t.Errorf("user: %+v \nvalidate result should be %t, but got %t, error: %v", tt.in, want, got, err)
			}
		})
	}
}

func TestExtractNameFromEmail(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			"abc@test.com",
			"abc",
		},
		{
			"$abc#@test.com",
			"abc",
		},
		{
			"a@test.com",
			"a",
		},
		{
			"abc.efg@test.com",
			"abc.efg",
		},
		{
			".abc_@test.com",
			"abc",
		},
		{
			"a#b^c@test.com",
			"a.b.c",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("extract name from %s", tt.in), func(t *testing.T) {
			got := ExtractNameFromEmail(tt.in)
			if got != tt.want {
				t.Errorf("want %s, but got %s", tt.want, got)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{
			in:   "abc@test.com",
			want: true,
		},
		{
			in:   "abc.123.def@test.com",
			want: true,
		},
		{
			in:   "123@test.com.cn",
			want: true,
		},
		{
			in:   "abc@test.com'org",
			want: false,
		},
	}

	for _, tt := range tests {
		err := ValidateEmail(tt.in)
		var got = false
		if err == nil {
			got = true
		}

		if got != tt.want {
			t.Errorf("%s validate result should be %t, but got %t", tt.in, tt.want, got)
		}
	}
}
