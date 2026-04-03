package modeliamsession

// SessionsListReq is the request payload for listing active sessions of the current user.
type SessionsListReq struct{}

// SessionsListRsp returns all active sessions of the current authenticated user.
type SessionsListRsp struct {
	Items []CurrentSession `json:"items"`
	Total int64            `json:"total"`
}

// SessionsDeleteReq is the request payload for deleting a specified session of the current user.
type SessionsDeleteReq struct{}

// SessionsDeleteRsp returns the delete result for a specified session of the current user.
type SessionsDeleteRsp struct{}
