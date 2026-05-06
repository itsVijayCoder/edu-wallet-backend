package hasher

import "golang.org/x/crypto/bcrypt"

// Hasher abstracts password hashing for testability.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

type bcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a Hasher using bcrypt with the given cost.
// Default cost is 12 (not bcrypt.DefaultCost which is 10).
func NewBcryptHasher(cost int) Hasher {
	if cost <= 0 {
		cost = 12
	}
	return &bcryptHasher{cost: cost}
}

func (h *bcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (h *bcryptHasher) Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
