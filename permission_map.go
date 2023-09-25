package main

var permissionMap = map[string][]RouteRe{
	"article.create": {
		{
			`^GET$`,
			`^/articles/new($|/)`,
		},
		{
			`^POST$`,
			`^/articles($|/)`,
		},
	},
	"article.edit_mine": {
		{
			`^(GET|POST)$`,
			`^/articles/\d+/edit($|/)`,
		},
	},
	// "article.edit_others": {
	// 	{
	// 		`^(GET|POST)$`,
	// 		`^/articles/\d+/edit($|/)`,
	// 	},
	// },
	"article.delete_mine": {
		{
			`^(GET|POST)$`,
			`^/articles/\d+/delete($|/)`,
		},
	},
	// "article.delete_others": {
	// 	{
	// 		`^(GET|POST)$`,
	// 		`^/articles/\d+/delete($|/)`,
	// 	},
	// },
	"article.reply": {
		{
			`^GET$`,
			`^/articles/\d+/reply($|/)`,
		},
	},
	"article.save": {
		{
			`^POST$`,
			`^/articles/\d+/save($|/)`,
		},
	},
	"article.react": {
		{
			`^POST$`,
			`^/articles/\d+/react($|/)`,
		},
	},
	"article.vote_up": {
		{
			`^POST$`,
			`^/articles/\d+/vote($|/)`,
		},
	},
	"article.vote_down": {
		{
			`^POST$`,
			`^/articles/\d+/vote($|/)`,
		},
	},
	"user.update_intro_mine": {
		{
			`^POST$`,
			`^/settings/account($|/)`,
		},
	},
	// "user.update_intro_others": {
	// 	{
	// 		`^POST$`,
	// 		`^/settings/account($|/)`,
	// 	},
	// },
	"user.manage,user.ban": {
		{
			`^GET$`,
			`^/users/\d+/ban($|/)`,
		},
	},
	"user.manage,user.ban,user.set_moderator,user.set_admin,user.update_role": {
		{
			`^POST$`,
			`^/users/\d+/set_role($|/)`,
		},
	},
	"user.list_access": {
		{
			`^GET$`,
			`^/manage/users($|/)`,
		},
	},
	"manage.access": {
		{
			`^GET$`,
			`^/manage/?`,
		},
	},
	"permission.access": {
		{
			`^GET$`,
			`^/manage/permissions/?`,
		},
	},
	"role.access": {
		{
			`^GET$`,
			`^/manage/roles/?`,
		},
	},
	"role.add": {
		{
			`^GET$`,
			`^/manage/roles/new/?`,
		},
		{
			`^POST`,
			`^/manage/roles/?`,
		},
	},
	"role.edit": {
		{
			`^(GET|POST)$`,
			`^/manage/roles/\d+/edit/?`,
		},
	},
}
