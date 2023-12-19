package pgstore

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type User struct {
	dbPool *pgxpool.Pool
}

func (u *User) List(page, pageSize int, oldest bool, username, roleFrontId string, authType model.AuthType) ([]*model.User, int, error) {
	if page < 1 {
		page = DefaultPage
	}

	if pageSize < 1 {
		pageSize = DefaultPage
	}

	sqlStr := `SELECT 
    u.id, u.username, u.email, u.created_at, u.banned_start_at, COALESCE(u.banned_day_num, 0), u.banned_count, u.auth_from,
    COALESCE(u.introduction, ''),
    COALESCE(r.name, '') as role_name, 
    COALESCE(r.front_id, '') AS role_front_id,
    COUNT(*) OVER() AS total
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON ur.role_id = r.id
`
	var conditions []string
	var args []any
	if strings.TrimSpace(username) != "" {
		args = append(args, "%"+username+"%")
		conditions = append(conditions, fmt.Sprintf(" username ILIKE $%d ", len(args)))
	}

	if strings.TrimSpace(roleFrontId) != "" {
		args = append(args, roleFrontId)
		conditions = append(conditions, fmt.Sprintf(" r.front_id = $%d ", len(args)))
	}

	if strings.TrimSpace(string(authType)) != "" {
		args = append(args, authType)
		conditions = append(conditions, fmt.Sprintf(" auth_from = $%d ", len(args)))
	}

	if len(conditions) > 0 {
		sqlStr += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	if oldest {
		sqlStr += ` ORDER BY u.created_at`
	} else {
		sqlStr += ` ORDER BY u.created_at DESC`
	}

	args = append(args, pageSize*(page-1), pageSize)
	sqlStr += fmt.Sprintf(` OFFSET $%d LIMIT $%d`, len(args)-1, len(args))

	// fmt.Println("user list sqlStr", sqlStr)

	rows, err := u.dbPool.Query(
		context.Background(),
		sqlStr,
		args...,
	)

	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	var list []*model.User
	// listMap := make(map[int]*model.User)
	var total int

	for rows.Next() {
		var item model.User
		// var pItem model.Permission
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.Email,
			&item.RegisteredAt,
			&item.NullBannedStartAt,
			&item.BannedDayNum,
			&item.BannedCount,
			&item.AuthFrom,
			&item.Introduction,
			&item.RoleName,
			&item.RoleFrontId,
			&total,
			// &pItem.Id,
			// &pItem.Name,
			// &pItem.FrontId,
			// &pItem.Module,
			// &pItem.CreatedAt,
		)

		if err != nil {
			return nil, 0, err
		}

		item.FormatNullVals()
		item.UpdateBannedState()
		list = append(list, &item)
	}

	return list, total, nil
}

func (u *User) Count() (int, error) {
	var count int
	err := u.dbPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM users;`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (u *User) Create(email, password, name string, roleFrontId string) (int, error) {
	// fmt.Printf("user.create item: %+v\n", item)
	var id int
	err := u.dbPool.QueryRow(context.Background(), "INSERT INTO users (email, password, username) VALUES ($1, $2, $3) RETURNING (id)",
		email,
		password,
		name).Scan(&id)
	if err != nil {
		return 0, err
	}

	_, err = u.dbPool.Exec(context.Background(), "INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE front_id = $2", id, roleFrontId)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (u *User) CreateWithOAuth(email, username, roleFrontId, authType string) (int, error) {
	// fmt.Printf("user.create item: %+v\n", item)
	var id int
	err := u.dbPool.QueryRow(context.Background(), "INSERT INTO users (email, username, auth_from) VALUES ($1, $2, $3) RETURNING (id)",
		email,
		username,
		authType).Scan(&id)
	if err != nil {
		return 0, err
	}

	_, err = u.dbPool.Exec(context.Background(), "INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE front_id = $2", id, roleFrontId)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func validUserUpdateField(key string) bool {
	allowedFields := []string{"Introduction", "Banned"}
	for _, field := range allowedFields {
		if key == field {
			return true
		}
	}
	return false
}

func (u *User) UpdateIntroduction(username, introduction string) error {
	_, err := u.dbPool.Exec(context.Background(), `UPDATE users SET introduction = $1 WHERE username = $2`, introduction, username)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) UpdatePassword(email, password string) (int, error) {
	// fmt.Println("email: ", email)
	// fmt.Println("password: ", password)
	var id int
	err := u.dbPool.QueryRow(context.Background(), "UPDATE users SET password = $1, auth_from = 'self' WHERE email = $2 RETURNING (id)", password, email).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (u *User) queryItem(fieldName string, val any) (*model.User, error) {
	var conditionStr string
	switch fieldName {
	case "id":
		conditionStr = "u.id = $1"
	case "email":
		conditionStr = "u.email = $1"
	case "username":
		conditionStr = "u.username = $1"
	}

	if conditionStr == "" {
		return nil, errors.New("wrong field name")
	}

	sqlStr := `SELECT u.id, u.username, u.email, u.created_at, u.super_admin, COALESCE(u.introduction, '') as introduction, u.auth_from, u.reputation, u.banned_start_at, COALESCE(u.banned_day_num, 0), u.banned_count,
COALESCE(r.name, '') as role_name, COALESCE(r.front_id, '') AS role_front_id,
COALESCE(p.id, 0) AS p_id, COALESCE(p.name, '') AS p_name, COALESCE(p.front_id, '') AS p_front_id, COALESCE(p.module, 'user') AS p_module, COALESCE(p.created_at, NOW()) AS p_created_at
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON ur.role_id = r.id
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id WHERE ` + conditionStr

	rows, err := u.dbPool.Query(context.Background(), sqlStr, val)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var item model.User
	for rows.Next() {
		var uItem model.User
		var pItem model.Permission

		err = rows.Scan(
			&uItem.Id,
			&uItem.Name,
			&uItem.Email,
			&uItem.RegisteredAt,
			&uItem.Super,
			&uItem.Introduction,
			&uItem.AuthFrom,
			&uItem.Reputation,
			&uItem.NullBannedStartAt,
			&uItem.BannedDayNum,
			&uItem.BannedCount,
			&uItem.RoleName,
			&uItem.RoleFrontId,
			&pItem.Id,
			&pItem.Name,
			&pItem.FrontId,
			&pItem.Module,
			&pItem.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		if item.Id < 1 {
			item = uItem
		}

		// fmt.Println("inscan:", item.Id)

		if pItem.Id != 0 {
			if item.Permissions != nil {
				item.Permissions = append(item.Permissions, &pItem)
			} else {
				item.Permissions = []*model.Permission{&pItem}
			}
		}
	}

	if item.Id < 1 {
		return nil, model.AppErrUserNotExist
	}

	item.FormatNullVals()
	item.UpdateBannedState()

	return &item, nil
}

func (u *User) Item(id int) (*model.User, error) {
	// fmt.Println("userId: ", id)
	return u.queryItem("id", id)
}

func (u *User) ItemWithEmail(email string) (*model.User, error) {
	// fmt.Println("userId: ", id)
	return u.queryItem("email", email)
}

func (u *User) ItemWithUsername(username string) (*model.User, error) {
	// fmt.Println("userId: ", id)
	return u.queryItem("username", username)
}

func (u *User) ItemWithUsernameEmail(usernameEmail string) (*model.User, error) {
	var isEmail = false

	if regexp.MustCompile(`@`).Match([]byte(usernameEmail)) {
		isEmail = true
	}
	if isEmail {
		return u.queryItem("email", usernameEmail)
	} else {
		return u.queryItem("username", usernameEmail)
	}
}

func (u *User) Exists(email, username string) (int, error) {
	var id int
	err := u.dbPool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1 OR username = $2", email, username).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (u *User) Delete(id int) error {
	err := u.dbPool.QueryRow(context.Background(), "UPDATE users SET deleted = true WHERE id = $1", id).Scan(nil)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) DeleteHard(id int) error {
	_, err := u.dbPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Ban(username string, bannedDays int) (int, error) {
	var userId int
	sqlStr := `UPDATE users SET banned_count = banned_count + 1, banned_start_at = NOW(), banned_day_num = $2 WHERE username = $1 RETURNING (id)`

	err := u.dbPool.QueryRow(context.Background(), sqlStr, username, bannedDays).Scan(&userId)
	if err != nil {
		return 0, err
	}

	_, err = u.SetRole(userId, "banned_user")
	if err != nil {
		return 0, err
	}

	return userId, nil
}

func (u *User) Unban(username string) (int, error) {
	var userId int
	sqlStr := `UPDATE users SET banned_start_at = null, banned_day_num = 0 WHERE username = $1 RETURNING (id)`

	err := u.dbPool.QueryRow(context.Background(), sqlStr, username).Scan(&userId)
	if err != nil {
		return 0, err
	}

	_, err = u.SetRole(userId, "common_user")
	if err != nil {
		return 0, err
	}

	return userId, nil
}

func (u *User) GetPassword(username string) (string, error) {
	var hasedPwd string
	var isEmail = false

	if regexp.MustCompile(`@`).Match([]byte(username)) {
		isEmail = true
	}

	sqlStr := `SELECT password FROM users `
	if isEmail {
		sqlStr += "WHERE email = $1"
	} else {
		sqlStr += "WHERE username ILIKE $1"
	}

	sqlStr += " AND auth_from = 'self'"

	err := u.dbPool.QueryRow(context.Background(), sqlStr, username).Scan(&hasedPwd)
	if err != nil {
		return "", err
	}

	return hasedPwd, nil
}

func (u *User) GetPosts(username string, listType string) ([]*model.Article, error) {
	sqlStrHead := `
SELECT
p.id,
p.title,
p.content,
p.created_at,
p.updated_at,
p.reply_to,
p.author_id,
u.username AS author_name,
p.depth,
p3.title AS root_article_title
FROM posts p
JOIN users u ON p.author_id = u.id
LEFT JOIN posts p3 ON p.root_article_id = p3.id
WHERE u.username = $1 AND p.deleted = false`
	sqlStrTail := ` ORDER BY p.created_at DESC`

	switch listType {
	case "article":
		sqlStrHead += ` AND p.reply_to = 0`
	case "reply":
		sqlStrHead += ` AND p.reply_to != 0`
	default:
	}

	sqlStr := sqlStrHead + sqlStrTail

	rows, err := u.dbPool.Query(context.Background(), sqlStr, username)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []*model.Article
	for rows.Next() {
		var item model.Article
		err = rows.Scan(
			&item.Id,
			&item.NullTitle,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyToId,
			&item.AuthorId,
			&item.AuthorName,
			&item.ReplyDepth,
			&item.NullReplyRootArticleTitle,
		)

		if err != nil {
			fmt.Printf("query user's posts error: %v", err)
			return nil, err
		}

		item.FormatNullValues()
		// item.FormatTimeStr()

		posts = append(posts, &item)
	}

	return posts, nil
}

func (u *User) GetSavedPosts(username string) ([]*model.Article, error) {
	sqlStr := `
SELECT
p.id,
p.title,
p.content,
p.created_at,
p.updated_at,
p.reply_to,
p.author_id,
u2.username AS author_name,
p.depth,
p3.title AS root_article_title
FROM post_saves ps
JOIN users u ON u.id = ps.user_id AND u.username = $1
LEFT JOIN posts p ON p.id = ps.post_id
LEFT JOIN posts p3 ON p.root_article_id = p3.id
LEFT JOIN users u2 ON u2.id = p.author_id
WHERE p.deleted = false
ORDER BY ps.created_at DESC`
	rows, err := u.dbPool.Query(context.Background(), sqlStr, username)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []*model.Article
	for rows.Next() {
		var item model.Article
		err = rows.Scan(
			&item.Id,
			&item.NullTitle,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyToId,
			&item.AuthorId,
			&item.AuthorName,
			&item.ReplyDepth,
			&item.NullReplyRootArticleTitle,
		)

		if err != nil {
			fmt.Printf("query user's saved posts error: %v", err)
			return nil, err
		}

		item.FormatNullValues()
		// item.FormatTimeStr()

		posts = append(posts, &item)
	}

	return posts, nil
}

func (u *User) SetRole(userId int, roleFrontId string) (int, error) {
	var roleId int
	err := u.dbPool.QueryRow(context.Background(), `SELECT id FROM roles WHERE front_id = $1`, roleFrontId).Scan(&roleId)
	if err != nil {
		return 0, err
	}

	_, err = u.dbPool.Exec(context.Background(), `DELETE FROM user_roles WHERE user_id = $1`, userId)
	if err != nil {
		return 0, err
	}

	_, err = u.dbPool.Exec(context.Background(), `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userId, roleId)
	if err != nil {
		return 0, err
	}

	return userId, nil
}

func (u *User) SetRoleManyWithFrontId(list []*model.User) error {
	sqlStr := `INSERT INTO user_roles (user_id, role_id) `
	var args []any
	var argCount = 1

	for idx, item := range list {
		if item.Id < 1 || item.RoleFrontId == "" {
			continue
		}

		if idx == 0 {
			sqlStr += fmt.Sprintf("\nSELECT $%d::int, r.id FROM roles r WHERE r.front_id = $%d \n", argCount, argCount+1)
		} else {
			sqlStr += fmt.Sprintf("UNION ALL SELECT $%d::int, r.id FROM roles r WHERE r.front_id = $%d \n", argCount, argCount+1)
		}
		args = append(args, item.Id, item.RoleFrontId)
		argCount += 2
	}

	// fmt.Println("set many roles sql string: \n", sqlStr)
	// fmt.Println("set many roles args: ", args)
	// fmt.Println("set many roles args length: ", len(args))

	_, err := u.dbPool.Exec(context.Background(), sqlStr, args...)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) GetSubscribedPosts(username string) ([]*model.Article, error) {
	sqlStr := `
SELECT
p.id,
p.title,
p.content,
p.created_at,
p.updated_at,
p.reply_to,
p.author_id,
u2.username AS author_name,
p.depth,
p3.title AS root_article_title,
true
FROM post_subs ps
JOIN users u ON u.id = ps.user_id AND u.username = $1
LEFT JOIN posts p ON p.id = ps.post_id
LEFT JOIN posts p3 ON p.root_article_id = p3.id
LEFT JOIN users u2 ON u2.id = p.author_id
WHERE p.deleted = false
ORDER BY ps.created_at DESC`
	rows, err := u.dbPool.Query(context.Background(), sqlStr, username)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []*model.Article
	for rows.Next() {
		var item model.Article
		var userState model.CurrUserState

		err = rows.Scan(
			&item.Id,
			&item.NullTitle,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyToId,
			&item.AuthorId,
			&item.AuthorName,
			&item.ReplyDepth,
			&item.NullReplyRootArticleTitle,
			&userState.Subscribed,
		)

		if err != nil {
			fmt.Printf("query user's subscribed posts error: %v", err)
			return nil, err
		}

		item.CurrUserState = &userState

		item.FormatNullValues()
		// item.FormatTimeStr()

		posts = append(posts, &item)
	}

	return posts, nil
}

func (u *User) doAddReputation(username string, value int, comment string, changeType model.ReputationChangeType, isRevert bool) error {
	var args = []any{username, value}
	sqlStr := `UPDATE users SET reputation = reputation + $2 WHERE username = $1 RETURNING (id)`

	var userId int
	err := u.dbPool.QueryRow(context.Background(), sqlStr, args...).Scan(&userId)
	if err != nil {
		return err
	}

	err = u.logReputation(userId, value, changeType, comment, isRevert)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) AddReputation(username string, changeType model.ReputationChangeType, isRevert bool) error {
	preReputation, err := u.getReputation(username)
	if err != nil {
		return err
	}

	changeVal := model.ReputationChangeValues[changeType]
	if changeType == model.RPCTypeBanned {
		changeVal = -int(math.Round(float64(preReputation) / 2))
	}

	if isRevert {
		changeVal = -changeVal
	}

	err = u.doAddReputation(username, changeVal, "", changeType, isRevert)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) AddReputationVal(username string, value int, comment string, isRevert bool) error {
	err := u.doAddReputation(username, value, comment, model.RPCTypeOther, isRevert)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) getReputation(username string) (int, error) {
	var reputation int
	err := u.dbPool.QueryRow(context.Background(), `SELECT reputation FROM users WHERE username = $1`, username).Scan(&reputation)
	if err != nil {
		return 0, err
	}
	return reputation, nil
}

func (u *User) logReputation(userId, value int, changeType model.ReputationChangeType, comment string, isRevert bool) error {
	sqlStr := `INSERT INTO reputation_log (user_id, value_diff, type, comment, is_revert) VALUES ($1, $2, $3, $4, $5)`

	_, err := u.dbPool.Exec(context.Background(), sqlStr, userId, value, changeType, comment, isRevert)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) GetVotedPosts(username string, voteType model.VoteType) ([]*model.Article, error) {
	sqlStr := `
SELECT
p.id,
p.title,
p.content,
p.created_at,
p.updated_at,
p.reply_to,
p.author_id,
u2.username AS author_name,
p.depth,
p3.title AS root_article_title
FROM post_votes pv
JOIN users u ON u.id = pv.user_id AND u.username = $1
LEFT JOIN posts p ON p.id = pv.post_id AND pv.type = `
	if voteType == model.VoteTypeUp {
		sqlStr += `'up' `
	} else {
		sqlStr += `'down' `
	}
	sqlStr += `
LEFT JOIN posts p3 ON p.root_article_id = p3.id
LEFT JOIN users u2 ON p.author_id = u2.id
WHERE p.deleted = false AND p.author_id != u.id
ORDER BY pv.created_at DESC`

	// fmt.Println("get vote post sql:", sqlStr)

	rows, err := u.dbPool.Query(context.Background(), sqlStr, username)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []*model.Article
	for rows.Next() {
		var item model.Article
		err = rows.Scan(
			&item.Id,
			&item.NullTitle,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyToId,
			&item.AuthorId,
			&item.AuthorName,
			&item.ReplyDepth,
			&item.NullReplyRootArticleTitle,
		)

		if err != nil {
			fmt.Printf("query user's saved posts error: %v", err)
			return nil, err
		}

		item.FormatNullValues()

		posts = append(posts, &item)
	}

	return posts, nil
}

// func (u *User) UpdateReputation(username string) error {
// 	user, err := u.ItemWithUsername(username)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("username:", user.Name)
// 	voteUpCount, voteDownCount, fadeOutCount, err := u.getVoteCount(user.Id)
// 	if err != nil {
// 		return err
// 	}

// 	thankedCount, happyCount, err := u.getReactCount(user.Id)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println("voteUp, voteDown, fadeOut, thanked, happy, banned", voteUpCount, voteDownCount, fadeOutCount, thankedCount, happyCount, user.BannedCount)

// 	reputation := voteUpCount*model.ReputationChangeValues[model.RPCTypeUpvoted] +
// 		voteDownCount*model.ReputationChangeValues[model.RPCTypeDownvoted] +
// 		fadeOutCount*model.ReputationChangeValues[model.RPCTypeFadeOut] +
// 		thankedCount*model.ReputationChangeValues[model.RPCTypeThanked] +
// 		happyCount*model.ReputationChangeValues[model.RPCTypeLaughed]

// 	// fmt.Println("before reputation:", reputation)
// 	// fmt.Println("math pow:", math.Pow(0.5, float64(user.BannedCount)))

// 	reputation = int(float64(reputation) * math.Pow(0.5, float64(user.BannedCount)))

// 	sqlStr := `UPDATE users SET reputation = $2 WHERE id = $1`
// 	_, err = u.dbPool.Exec(context.Background(), sqlStr, user.Id, reputation)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (u *User) getVoteCount(userId int) (int, int, int, error) {
// 	sqlStr := `SELECT COUNT(DISTINCT pv.id) AS vote_up_count, COUNT(DISTINCT pv1.id) AS vote_down_count, COUNT(DISTINCT p1.id) AS fade_out_count
// FROM posts p
// LEFT JOIN post_votes pv ON pv.post_id = p.id AND pv.type = 'up' AND pv.user_id != $1
// LEFT JOIN post_votes pv1 ON pv1.post_id = p.id AND pv1.type = 'down' AND pv1.user_id != $1
// FULL OUTER JOIN posts p1 ON p1.author_id = $1 AND p1.fade_out = true AND p1.deleted = false
// WHERE p.author_id = $1 AND p.deleted = false;`
// 	var voteUpCount, voteDownCount, fadeOutCount int
// 	err := u.dbPool.QueryRow(context.Background(), sqlStr, userId).Scan(&voteUpCount, &voteDownCount, &fadeOutCount)
// 	if err != nil {
// 		return 0, 0, 0, err
// 	}

// 	return voteUpCount, voteDownCount, fadeOutCount, nil
// }

// func (u *User) getReactCount(userId int) (int, int, error) {
// 	sqlStr := `SELECT COUNT(DISTINCT pr.id) AS react_thanks_count, COUNT(DISTINCT pr1.id) AS react_happy_count
// FROM posts p
// LEFT JOIN post_reacts pr ON pr.post_id = p.id AND pr.react_id IN (
//   SELECT id FROM reacts WHERE front_id = 'thanks'
// )
// LEFT JOIN post_reacts pr1 ON pr1.post_id = p.id AND pr1.react_id IN (
//   SELECT id FROM reacts WHERE front_id = 'happy'
// )
// WHERE p.author_id = $1 AND p.deleted = false;`
// 	var thanksCount, happyCount int
// 	err := u.dbPool.QueryRow(context.Background(), sqlStr, userId).Scan(&thanksCount, &happyCount)
// 	if err != nil {
// 		return 0, 0, err
// 	}

// 	return thanksCount, happyCount, nil
// }
