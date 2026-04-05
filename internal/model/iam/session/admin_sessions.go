package modeliamsession

// AdminSessionsListReq is the request payload for listing all sessions grouped by user.
type AdminSessionsListReq struct{}

// AdminSessionsListRsp returns all active sessions grouped by user for privileged administrators.
type AdminSessionsListRsp struct {
	Items        []AdminSessionUserView `json:"items"`
	Total        int64                  `json:"total"`
	SessionTotal int64                  `json:"session_total"`
}

// AdminSessionUserView describes a user together with all indexed sessions owned by the user.
type AdminSessionUserView struct {
	UserID             string        `json:"user_id"`
	Username           string        `json:"username"`
	Email              string        `json:"email"`
	FirstName          *string       `json:"first_name,omitempty"`
	LastName           *string       `json:"last_name,omitempty"`
	GroupID            string        `json:"group_id,omitempty"`
	GroupName          string        `json:"group_name,omitempty"`
	Status             string        `json:"status"`
	MustChangePassword bool          `json:"must_change_password"`
	SessionTotal       int64         `json:"session_total"`
	Sessions           []SessionView `json:"sessions"`
}
