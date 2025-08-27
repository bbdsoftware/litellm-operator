package interfaces

// SecretRefInterface defines the interface for different SecretRef types
type SecretRefInterface interface {
	GetSecretName() string
	GetNamespace() string
	GetKeys() KeysInterface
	HasKeys() bool
}

// InstanceRefInterface defines the interface for different InstanceRef types
type InstanceRefInterface interface {
	GetInstanceName() string
	GetNamespace() string
}

// ConnectionRefInterface defines the interface for different ConnectionRef types
type ConnectionRefInterface interface {
	GetSecretRef() interface{}
	GetInstanceRef() interface{}
	HasSecretRef() bool
	HasInstanceRef() bool
}

// KeysInterface defines the interface for different key structures
type KeysInterface interface {
	GetMasterKey() string
	GetURL() string
}
