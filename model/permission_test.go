package model

import (
	"testing"
)

type permissionData struct {
	Module  string
	FrontId string
	Name    string
}

func TestPermissionValid(t *testing.T) {
	tests := []struct {
		desc  string
		in    *permissionData
		valid bool
	}{
		{
			desc:  "All valid",
			in:    &permissionData{Module: "user", FrontId: "role_read", Name: "Read Role"},
			valid: true,
		},
		{
			desc:  "Module is requried",
			in:    &permissionData{Module: "", FrontId: "role_read", Name: "Read Role"},
			valid: false,
		},
		{
			desc:  "Name is requried",
			in:    &permissionData{Module: "user", FrontId: "role_read", Name: ""},
			valid: false,
		},
		{
			desc:  "Front id is required",
			in:    &permissionData{Module: "user", FrontId: "", Name: "Read Role"},
			valid: false,
		},
		{
			desc:  "Front id format",
			in:    &permissionData{Module: "user", FrontId: "read-role", Name: "Read Role"},
			valid: false,
		},
		{
			desc:  "Name format",
			in:    &permissionData{Module: "user", FrontId: "read_role", Name: "Read#Role"},
			valid: false,
		},
		{
			desc:  "Front id length",
			in:    &permissionData{Module: "user", FrontId: "role_readaarole_readaarole_readaarole_readaarole_readaad", Name: "Read Role"},
			valid: false,
		},
		{
			desc:  "Name length",
			in:    &permissionData{Module: "user", FrontId: "read_role", Name: "Read Role Read Role Read Role Read Role Read Role Read Role "},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			u := &Permission{
				FrontId: tt.in.FrontId,
				Name:    tt.in.Name,
				Module:  PermissionModule(tt.in.Module),
			}
			err := u.Valid()
			// if err != nil {
			// 	fmt.Println("err: ", err)
			// 	fmt.Println("err is ErrValidUserFailed: ", errors.Is(err, ErrValidUserFailed))
			// }
			got := err == nil
			want := tt.valid

			if got != want {
				t.Errorf("permission: %+v \nvalidate result should be %t, but got %t, error: %v", tt.in, want, got, err)
			}
		})
	}
}

func TestValidPermissionModule(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		want bool
	}{
		{
			"User module",
			"user",
			true,
		},
		{
			"All module",
			"all",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := ValidPermissionModule(tt.in)
			if result != tt.want {
				t.Errorf("valid result should be %t, but got %t", tt.want, result)
			}
		})
	}
}
