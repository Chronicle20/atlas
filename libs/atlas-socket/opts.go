package socket

type ServerConfigurator func(s *serverConfiguration)

//goland:noinspection GoUnusedExportedFunction
func SetIpAddress(ipAddress string) func(*serverConfiguration) {
	return func(s *serverConfiguration) {
		s.ipAddress = ipAddress
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetPort(port int) func(*serverConfiguration) {
	return func(s *serverConfiguration) {
		s.port = port
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetSessionCreator(creator SessionCreator) ServerConfigurator {
	return func(s *serverConfiguration) {
		s.creator = creator
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetSessionDestroyer(destroyer SessionDestroyer) ServerConfigurator {
	return func(s *serverConfiguration) {
		s.destroyer = destroyer
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetSessionMessageDecryptor(decryptor SessionMessageDecryptor) ServerConfigurator {
	return func(s *serverConfiguration) {
		s.decryptor = decryptor
	}
}
