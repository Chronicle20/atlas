package socket

type ServerConfigurator[E uint8 | uint16] func(s *serverConfiguration[E])

//goland:noinspection GoUnusedExportedFunction
func SetIpAddress[E uint8 | uint16](ipAddress string) func(*serverConfiguration[E]) {
	return func(s *serverConfiguration[E]) {
		s.ipAddress = ipAddress
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetPort[E uint8 | uint16](port int) func(*serverConfiguration[E]) {
	return func(s *serverConfiguration[E]) {
		s.port = port
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetSessionCreator[E uint8 | uint16](creator SessionCreator) ServerConfigurator[E] {
	return func(s *serverConfiguration[E]) {
		s.creator = creator
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetSessionDestroyer[E uint8 | uint16](destroyer SessionDestroyer) ServerConfigurator[E] {
	return func(s *serverConfiguration[E]) {
		s.destroyer = destroyer
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetSessionMessageDecryptor[E uint8 | uint16](decryptor SessionMessageDecryptor) ServerConfigurator[E] {
	return func(s *serverConfiguration[E]) {
		s.decryptor = decryptor
	}
}

func SetOpReader[E uint8 | uint16](reader OpReader[E]) ServerConfigurator[E] {
	return func(s *serverConfiguration[E]) {
		s.opReader = reader
	}
}
