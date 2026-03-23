package password

type HasherInterface interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

var _ HasherInterface = (*Hasher)(nil)
