package modeliamsession

// SessionsReq is the request payload for listing active sessions of the current user.
type SessionsReq struct{}

// SessionsRsp returns all active sessions of the current authenticated user.
type SessionsRsp struct {
	Items []CurrentSession `json:"items"`
	Total int64            `json:"total"`
}
