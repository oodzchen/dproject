package web

type TipStatus int

const (
	TipStatusDeleteSuccess TipStatus = iota
	TipStatusDeleteFailed
)

var TipStatusStr = map[TipStatus]string{
	TipStatusDeleteSuccess: "Successfully deleted.",
	TipStatusDeleteFailed:  "Failed to delete",
}
