package config

// Serializable is a small interface that ensures certain objects can be freely represented in
// various encoded forms, usually for the purpose of transmitting in the network
type Serializable interface {
	ToBytes() ([]byte, error)
}
