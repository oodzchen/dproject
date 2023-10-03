package pgstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Activity struct {
	dbPool *pgxpool.Pool
}

func (a *Activity) List(userId int, userName, actType, action string, page, pageSize int) ([]*model.Activity, int, error) {
	sqlStr := `SELECT ua.id, ua.user_id, u.username as user_name, ua.type, ua.action, ua.target_model, ua.target_id, ua.ip_address, ua.device_info, ua.details, ua.created_at, COUNT(*) OVER() AS total
FROM activities ua
LEFT JOIN users u ON u.id = ua.user_id`

	var args []any
	var conditions []string
	conditionCount := 0
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	if userId > 0 {
		args = append(args, userId)
		conditions = append(conditions, fmt.Sprintf("ua.user_id = $%d", len(args)))
		conditionCount += 1
	}

	if len(userName) > 0 {
		bluredUserName := fmt.Sprintf("%s%s%s", "%%", userName, "%%")
		args = append(args, bluredUserName)
		sqlStr += fmt.Sprintf(" INNER JOIN users u1 ON u1.name ILIKE $%d AND u1.id = ua.user_id ", len(args))
	}

	if actType != "" {
		args = append(args, actType)
		conditions = append(conditions, fmt.Sprintf("ua.type = $%d", len(args)))
		conditionCount += 1
	}

	if action != "" {
		args = append(args, action)
		conditions = append(conditions, fmt.Sprintf("ua.action = $%d", len(args)))
		conditionCount += 1
	}

	if conditionCount > 0 {
		sqlStr += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	args = append(args, pageSize*(page-1), pageSize)
	sqlStr += fmt.Sprintf(" ORDER BY ua.created_at DESC OFFSET $%d LIMIT $%d", len(args)-1, len(args))

	// fmt.Println("activity list sqlStr: ", sqlStr)
	// fmt.Println("activity list args: ", args)

	rows, err := a.dbPool.Query(context.Background(), sqlStr, args...)
	if err != nil {
		return nil, 0, err
	}

	var list []*model.Activity
	var total int
	for rows.Next() {
		var item model.Activity
		// id, user_id, type, action, target_model, target_id, ip_address, device_info, details, created_at
		err := rows.Scan(
			&item.Id,
			&item.UserId,
			&item.UserName,
			&item.Type,
			&item.Action,
			&item.TargetModel,
			&item.TargetId,
			&item.IpAddr,
			&item.DeviceInfo,
			&item.Details,
			&item.CreatedAt,
			&total,
		)

		if err != nil {
			return nil, 0, err
		}

		list = append(list, &item)
	}

	fmt.Println("total: ", total)

	return list, total, nil
}

func (a *Activity) Create(userId int, actType, action, targetModel string, targetId int, ipAddr, deviceInfo, details string) (int, error) {
	var id int
	err := a.dbPool.QueryRow(
		context.Background(),
		`INSERT INTO activities
(user_id, type, action, target_model, target_id, ip_address, device_info, details)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING (id)`,
		userId,
		actType,
		action,
		targetModel,
		targetId,
		ipAddr,
		deviceInfo,
		details,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}
