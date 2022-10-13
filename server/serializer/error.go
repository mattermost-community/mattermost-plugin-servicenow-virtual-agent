package serializer

// Error struct to store error ids and error message.
type APIErrorResponse struct {
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (a *APIErrorResponse) Error() string {
	return a.Message
}
