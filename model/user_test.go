package model

import "testing"

type userData struct {
	Name     string
	Email    string
	Password string
}

func TestUserValid(t *testing.T) {
	tests := []struct {
		in    *userData
		valid bool
	}{
		{
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "111abc@.,222"},
			valid: true,
		},
		{
			in:    &userData{Name: "", Email: "aaa@test.com", Password: "111abc@222"},
			valid: false,
		},
		{
			in:    &userData{Name: "Mark", Email: "", Password: "111abc@222"},
			valid: false,
		},
		{
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: ""},
			valid: false,
		},
		{
			in:    &userData{Name: "Mark", Email: "aaa#test.com", Password: "111abc@222"},
			valid: false,
		},
		{
			in:    &userData{Name: "@", Email: "aaa@test.com", Password: "111abc@222"},
			valid: false,
		},
		{
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "1a@"},
			valid: false,
		},
		{
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "111122222"},
			valid: false,
		},
		{
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "abcabcabc"},
			valid: false,
		},
		{
			in:    &userData{Name: "Mark", Email: "aaa@test.com", Password: "@#$@#$@#$"},
			valid: false,
		},
	}

	for _, tt := range tests {
		u := &User{
			Name:     tt.in.Name,
			Email:    tt.in.Email,
			Password: tt.in.Password,
		}
		got := u.Valid() == nil
		want := tt.valid

		if got != want {
			t.Errorf("user %v validate result should be %t, but got %t", tt.in, want, got)
		}
	}
}
