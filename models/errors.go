package models

func IsNotFoundError(err error) bool {
	switch err.(type) {
	case UserNotFoundError:
		return true
	case RefreshTokenNotFoundError:
		return true
	}
	return false
}

type UserNotFoundError struct{}

func (e UserNotFoundError) Error() string {
	return "User not found"
}

type RefreshTokenNotFoundError struct{}

func (e RefreshTokenNotFoundError) Error() string {
	return "Refresh Token not found"
}
