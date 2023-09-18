package model

import (
	"testing"
)

type roleData struct {
	FrontId string
	Name    string
}

func TestRoleValid(t *testing.T) {
	tests := []struct {
		desc     string
		in       *roleData
		isUpdate bool
		valid    bool
	}{
		{
			desc:     "All valid",
			in:       &roleData{FrontId: "common_user", Name: "user"},
			isUpdate: false,
			valid:    true,
		},
		{
			desc:     "Name is requried",
			in:       &roleData{FrontId: "commen_user", Name: ""},
			isUpdate: false,
			valid:    false,
		},
		{
			desc:     "Front id is required",
			in:       &roleData{FrontId: "", Name: "Common User"},
			isUpdate: false,
			valid:    false,
		},
		{
			desc:     "Front id format",
			in:       &roleData{FrontId: "common-user", Name: "Common User"},
			isUpdate: false,
			valid:    false,
		},
		{
			desc:     "Name format",
			in:       &roleData{FrontId: "common_user", Name: "Read#Role"},
			isUpdate: false,
			valid:    false,
		},
		{
			desc:     "Front id length",
			in:       &roleData{FrontId: "commen_usercommen_usercommen_usercommen_usercommen_user ", Name: "Common User"},
			isUpdate: false,
			valid:    false,
		},
		{
			desc:     "Name length",
			in:       &roleData{FrontId: "common_user", Name: "Common User Common User Common User Common User Common User Common User "},
			isUpdate: false,
			valid:    false,
		},
		{
			desc:     "Update Name",
			in:       &roleData{FrontId: "", Name: "Common User"},
			isUpdate: true,
			valid:    true,
		},
		{
			desc:     "Update Name format error",
			in:       &roleData{FrontId: "", Name: "Common#User"},
			isUpdate: true,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			u := &Role{
				FrontId: tt.in.FrontId,
				Name:    tt.in.Name,
			}

			err := u.Valid(tt.isUpdate)
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
