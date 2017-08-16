package models

// IsNotFoundError returns whether an error represents a "not found" error.
func IsNotFoundError(err error) bool {
	switch err.(type) {
	case UserNotFoundError:
		return true
	case RefreshTokenNotFoundError:
		return true
	case InstanceNotFoundError:
		return true
	}
	return false
}

// UserNotFoundError represents when a user is not found.
type UserNotFoundError struct{}

func (e UserNotFoundError) Error() string {
	return "User not found"
}

// RefreshTokenNotFoundError represents when a refresh token is not found.
type RefreshTokenNotFoundError struct{}

func (e RefreshTokenNotFoundError) Error() string {
	return "Refresh Token not found"
}

// InstanceNotFoundError represents when an instance is not found.
type InstanceNotFoundError struct{}

func (e InstanceNotFoundError) Error() string {
	return "Instance not found"
}
