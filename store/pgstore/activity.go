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

// List(userId int, actType string, page, pageSize int) ([]*model.Activity, error)
// Create(userId int, actType, targetModel string, targetId int, ipAddr, deviceInfo, details string) (int, error)

func (a *Activity) List(userId int, actType, action string, page, pageSize int) ([]*model.Activity, error) {
	sqlStr := `SELECT ua.id, ua.user_id, u.name as user_name, ua.type, ua.action, ua.target_model, ua.target_id, ua.ip_address, ua.device_info, ua.details, ua.created_at
FROM user_activities ua
LEFT JOIN users u ON u.id = user_id
`
	var args []any
	var conditions []string
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	args = append(args, userId)
	conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))

	if actType != "" {
		args = append(args, actType)
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}

	if action != "" {
		args = append(args, action)
		conditions = append(conditions, fmt.Sprintf("aciton = $%d", len(args)))
	}
	sqlStr += ` WHERE ` + strings.Join(conditions, " AND ")

	args = append(args, pageSize*(page-1), pageSize)
	sqlStr += fmt.Sprintf(" ORDER BY created_at DESC OFFSET $%d LIMIT $%d", len(args)-1, len(args))

	// fmt.Println("activity list sqlStr: ", sqlStr)
	// fmt.Println("activity list args: ", args)

	rows, err := a.dbPool.Query(context.Background(), sqlStr, args...)
	if err != nil {
		return nil, err
	}

	var list []*model.Activity
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
		)

		if err != nil {
			return nil, err
		}

		list = append(list, &item)
	}

	return list, nil
}

func (a *Activity) Create(userId int, actType, action, targetModel string, targetId int, ipAddr, deviceInfo, details string) (int, error) {
	var id int
	err := a.dbPool.QueryRow(
		context.Background(),
		`INSERT INTO user_activities
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